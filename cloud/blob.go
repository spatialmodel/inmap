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

package cloud

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/google/go-cloud/blob"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spatialmodel/inmap/inmaputil"
)

// readBlob reads the given blob from the given bucket.
func readBlob(ctx context.Context, bucket *blob.Bucket, key string) ([]byte, error) {
	var b bytes.Buffer
	r, err := bucket.NewReader(ctx, key)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	_, err = io.Copy(&b, r)
	return b.Bytes(), err
}

// writeBlob writes the given data to the given bucket.
func writeBlob(ctx context.Context, bucket *blob.Bucket, key string, data []byte) error {
	b := bytes.NewBuffer(data)
	w, err := bucket.NewWriter(ctx, key, &blob.WriterOptions{})
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, b)
	return err
}

// Output returns the output of the specified job.
func (c *Client) Output(ctx context.Context, job *cloudrpc.JobName) (*cloudrpc.JobOutput, error) {
	bucket, err := inmaputil.OpenBucket(ctx, c.bucketName)
	if err != nil {
		return nil, err
	}
	o := &cloudrpc.JobOutput{
		Files: make(map[string][]byte),
	}
	k8sJob, err := c.getk8sJob(ctx, job)
	addrs, err := c.jobOutputAddresses(ctx, job.Name, k8sJob.Spec.Template.Spec.Containers[0].Command)
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		for _, fname := range expandShp(addr) {
			url, err := url.Parse(fname)
			if err != nil {
				return nil, err
			}
			o.Files[filepath.Base(fname)], err = readBlob(ctx, bucket, strings.TrimLeft(url.Path, "/"))
			if err != nil {
				return nil, err
			}
		}
	}
	return o, nil
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
