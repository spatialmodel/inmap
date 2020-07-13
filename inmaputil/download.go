/*
Copyright © 2018 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmaputil

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spatialmodel/inmap/cloud"
)

// download checks if the input is an existing file locally.
// If not, it checks if the file is a URL.
// If it's a URL, it downloads the file and
// returns the path to the downloaded file.
// For shapefiles, it downloads all associated files and
// returns the path to the file with the ".shp" extension.
// c, if not nil, is a channel across which error and
// logging messages will be sent.
func maybeDownload(ctx context.Context, path string, c chan string) string {
	// Check if local file exists. If it does, return the given path.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path
	}

	// If the path starts with one of these prefixes, download the file and
	// return the location it was downloaded to.
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return downloadHTTP(path, c)
	}

	if IsBlob(path) {
		return downloadBlob(ctx, path, c)
	}

	return path
}

// downloadHTTP downloads a file from the specified URL and returns
// the path to the downloaded file.
func downloadHTTP(path string, c chan string) string {
	// Prepare a temporary directory for the downloads.
	dir, err := ioutil.TempDir("", "inmap")
	if err != nil {
		panic(fmt.Errorf("inmaputil: failed creating temporary download directory: %v", err))
	}

	fnames := expandShp(path)
	for _, fname := range fnames {
		w, err := os.Create(filepath.Join(dir, filepath.Base(fname)))
		if err != nil {
			panic(fmt.Errorf("inmaputil: failed creating file for download: %v", err))
		}
		fname = fname[0:len(fname)-4] + filepath.Ext(fname)
		resp, err := http.Get(fname)
		if err != nil {
			c <- err.Error()
			return path
		}
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			c <- err.Error()
			return path
		}
		resp.Body.Close()
		w.Close()
	}
	return filepath.Join(dir, filepath.Base(fnames[0]))
}

// IsBlob returns whether the given filename represents a blob.
// (i.e., if it starts with `gs://`, 's3://', or 'file://').
func IsBlob(path string) bool {
	return strings.HasPrefix(path, "gs://") || strings.HasPrefix(path, "s3://") || strings.HasPrefix(path, "file://")
}

// downloadBlob download the specified file from blob storage.
func downloadBlob(ctx context.Context, path string, c chan string) string {
	url, err := url.Parse(path)
	if err != nil {
		c <- err.Error()
		return path
	}
	bucket, err := cloud.OpenBucket(ctx, url.Scheme+"://"+url.Host)
	if err != nil {
		c <- err.Error()
		return path
	}
	dir, err := ioutil.TempDir("", "inmap")
	if err != nil {
		panic(fmt.Errorf("inmaputil: failed creating temporary download directory: %v", err))
	}
	ext := filepath.Ext(url.Path)
	fnames := expandShp(url.Path)
	for _, fname := range fnames {
		w, err := os.Create(filepath.Join(dir, filepath.Base(fname)))
		if err != nil {
			panic(fmt.Errorf("inmaputil: failed creating file for download: %v", err))
		}
		bucketPath := strings.TrimPrefix(url.Path, "/")
		bucketPath = bucketPath[0:len(bucketPath)-len(ext)] + filepath.Ext(fname)
		r, err := bucket.NewReader(ctx, bucketPath, nil)
		if err != nil {
			c <- err.Error()
			return path
		}
		_, err = io.Copy(w, r)
		if err != nil {
			c <- err.Error()
			return path
		}
		r.Close()
		w.Close()
	}
	return filepath.Join(dir, filepath.Base(fnames[0]))
}

// expandShp returns the given file + associated [.dbf, .shx, .prj]
// files if the given file has the .shp extension, and returns the given
// file otherwise
func expandShp(filename string) []string {
	o := []string{filename}
	ext := filepath.Ext(filename)
	if ext != ".shp" {
		return o
	}
	for _, newExt := range []string{".dbf", ".shx", ".prj"} {
		o = append(o, filename[0:len(filename)-4]+newExt)
	}
	return o
}
