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
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"gocloud.dev/blob"
)

// readBlob reads the given blob from the given bucket.
func readBlob(ctx context.Context, bucket *blob.Bucket, key string) ([]byte, error) {
	var b bytes.Buffer
	r, err := bucket.NewReader(ctx, key, nil)
	if err != nil {
		return nil, fmt.Errorf("reading blob key %s: %v", key, err)
	}
	defer r.Close()
	_, err = io.Copy(&b, r)
	if err != nil {
		return nil, fmt.Errorf("Reading blob key %s: %v", key, err)
	}
	return b.Bytes(), nil
}

// writeBlob writes the given data to the given bucket.
func writeBlob(ctx context.Context, bucket *blob.Bucket, key string, data []byte) error {
	b := bytes.NewBuffer(data)
	w, err := bucket.NewWriter(ctx, key, &blob.WriterOptions{})
	if err != nil {
		return fmt.Errorf("inmap/cloud: creating writer for blob %s: %v", key, err)
	}
	_, err = io.Copy(w, b)
	if err != nil {
		return fmt.Errorf("inmap/cloud: copying blob %s: %v", key, err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("inmap/cloud: writing blob %s: %v", key, err)
	}
	return nil
}

// deleteBlobDir deletes all blobs in the the specified directory
// of the specified bucket
func deleteBlobDir(ctx context.Context, bucketName, user, jobName string) error {
	bucket, err := OpenBucket(ctx, bucketName)
	if err != nil {
		return err
	}

	url, err := url.Parse(bucketName)
	if err != nil {
		return fmt.Errorf("cloud: parsing bucket name: %v", err)
	}

	prefix := fmt.Sprintf("%s/%s/%s/", strings.TrimLeft(url.Path, "/"), user, jobName)
	iter := bucket.List(&blob.ListOptions{
		Prefix:    prefix,
		Delimiter: "/",
	})
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("cloud: listing blob %s to delete: %v", obj.Key, err)
		}
		if err = bucket.Delete(ctx, obj.Key); err != nil {
			return fmt.Errorf("cloud: deleting blob %s: %v", obj.Key, err)
		}
	}
	return nil
}

// Output returns the output of the specified job.
func (c *Client) Output(ctx context.Context, job *cloudrpc.JobName) (*cloudrpc.JobOutput, error) {
	bucket, err := OpenBucket(ctx, c.bucketName)
	if err != nil {
		return nil, err
	}
	o := &cloudrpc.JobOutput{
		Files: make(map[string][]byte),
	}
	k8sJob, err := c.getk8sJob(ctx, job)
	if err != nil {
		return nil, err
	}
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
