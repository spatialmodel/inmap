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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/google/go-cloud/blob"
	"github.com/google/go-cloud/blob/fileblob"
	"github.com/google/go-cloud/blob/gcsblob"
	"github.com/google/go-cloud/blob/s3blob"
	"github.com/google/go-cloud/gcp"
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
		return path
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

// OpenBucket returns the blob storage bucket specified by bucketName,
// where bucketName must be in the format 'provider://name' where provider
// is the name of the storage provider and name is the name of the bucket.
// Even if name contains subdirectories, only the base directory name will be
// used when opening the bucket.
// The currently accepted storage providers are "file" for the local filesystem
// (e.g., for testing), "gs" for Google Cloud Storage, and "s3" for AWS S3.
func OpenBucket(ctx context.Context, bucketName string) (*blob.Bucket, error) {
	url, err := url.Parse(bucketName)
	if err != nil {
		return nil, fmt.Errorf("inmaputil.OpenBucket: %v", err)
	}
	switch url.Scheme {
	case "file":
		return fileblob.NewBucket(url.Hostname())
	case "gs":
		return gsBucket(ctx, url.Hostname())
	case "s3":
		return s3Bucket(ctx, url.Hostname())
	default:
		return nil, fmt.Errorf("cloud.OpenBucket: invalid provider %s", url.Scheme)
	}
}

func gsBucket(ctx context.Context, name string) (*blob.Bucket, error) {
	// See here for information on credentials:
	// https://cloud.google.com/docs/authentication/getting-started
	creds, err := gcp.DefaultCredentials(ctx)
	if err != nil {
		return nil, err
	}
	c, err := gcp.NewHTTPClient(gcp.DefaultTransport(), gcp.CredentialsTokenSource(creds))
	if err != nil {
		return nil, err
	}
	return gcsblob.OpenBucket(ctx, name, c)
}

// s3Bucket opens an s3 storage bucket. It assumes the following
// environment variables are set: AWS_REGION, AWS_ACCESS_KEY_ID, and
// AWS_SECRET_ACCESS_KEY.
func s3Bucket(ctx context.Context, name string) (*blob.Bucket, error) {
	region := os.ExpandEnv("AWS_REGION")
	if region == "" {
		region = "us-east-2"
	}
	c := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewEnvCredentials(),
	}
	s := session.Must(session.NewSession(c))
	return s3blob.OpenBucket(ctx, s, name)
}

// downloadBlob download the specified file from blob storage.
func downloadBlob(ctx context.Context, path string, c chan string) string {
	url, err := url.Parse(path)
	if err != nil {
		c <- err.Error()
		return path
	}
	bucket, err := OpenBucket(ctx, url.Scheme+"://"+url.Host)
	if err != nil {
		c <- err.Error()
		return path
	}
	dir, err := ioutil.TempDir("", "inmap")
	if err != nil {
		panic(fmt.Errorf("inmaputil: failed creating temporary download directory: %v", err))
	}
	fnames := expandShp(url.Path)
	for _, fname := range fnames {
		w, err := os.Create(filepath.Join(dir, filepath.Base(fname)))
		if err != nil {
			panic(fmt.Errorf("inmaputil: failed creating file for download: %v", err))
		}
		bucketPath := strings.TrimPrefix(url.Path, "/")
		bucketPath = bucketPath[0:len(bucketPath)-4] + filepath.Ext(fname)
		r, err := bucket.NewReader(ctx, bucketPath)
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
