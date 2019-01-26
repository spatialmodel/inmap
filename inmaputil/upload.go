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
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gocloud.dev/blob"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud"
)

type uploader struct {
	// files is a set of file path pairs. The first of each pair
	// is a local file path and the second is a blob storage
	// path where it should be uploaded to.
	files [][2]string
	err   error
	dir   string
}

func (u *uploader) uploadOutput(d *inmap.InMAP) error {
	if u.err != nil {
		return u.err
	}
	ctx := context.TODO()
	for _, files := range u.files {
		r, err := os.Open(files[0])
		if err != nil {
			return fmt.Errorf("inmaputil: opening file '%s' for upload: %s", files[0], err)
		}
		defer r.Close()
		url, err := url.Parse(files[1])
		if err != nil {
			return fmt.Errorf("inmaputil: parsing url '%s' for upload: %s", files[1], err)
		}
		bucket, err := cloud.OpenBucket(ctx, url.Scheme+"://"+url.Host)
		if err != nil {
			return fmt.Errorf("inmaputil: opening bucket to upload file '%s': %s", files[1], err)
		}
		w, err := bucket.NewWriter(ctx, strings.TrimPrefix(url.Path, "/"), &blob.WriterOptions{})
		if err != nil {
			return fmt.Errorf("inmaputil: opening writer to upload file '%s': %s", files[1], err)
		}
		defer w.Close()
		if _, err := io.Copy(w, r); err != nil {
			return fmt.Errorf("inmaputil: uploading file '%s' to '%s': %s", files[0], files[1], err)
		}
	}
	return nil
}

// maybeUpload checks whether the given output file path refers to
// a blob storage location. If it does, then a temporary file location
// is returned. The file will then be uploaded to blob storage when
// uploadOutput method is run.
func (u *uploader) maybeUpload(path string) string {
	if u.err != nil {
		return ""
	}
	if !IsBlob(path) {
		return path
	}
	if u.dir == "" {
		u.dir, u.err = ioutil.TempDir("", "inmap")
		if u.err != nil {
			return ""
		}
	}
	files := expandShp(path)
	for _, f := range files {
		u.files = append(u.files, [2]string{
			filepath.Join(u.dir, filepath.Base(f)),
			f,
		})
	}
	return filepath.Join(u.dir, filepath.Base(files[0]))
}
