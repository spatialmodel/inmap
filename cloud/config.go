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
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spf13/pflag"
)

// jobOutputAddresses returns the locations of where the output files of the job
// with the given name, belonging to the given user, with the given command arguments,
// will be stored.
func (c *Client) jobOutputAddresses(ctx context.Context, name string, cmd []string) (map[string]string, error) {
	outputFiles := make(map[string]struct{})
	for _, f := range c.outputFileArgs {
		outputFiles[f] = struct{}{}
	}
	user, err := getUser(ctx)
	if err != nil {
		return nil, err
	}
	o := make(map[string]string)
	execCmd, _, err := c.root.Find(cmd[1:])
	if err != nil {
		return nil, fmt.Errorf("cloud: couldn't find command %v: %v", cmd[1:], err)
	}
	flags := execCmd.InheritedFlags()
	flags.AddFlagSet(execCmd.LocalFlags())
	flags.VisitAll(func(f *pflag.Flag) {
		if _, ok := outputFiles[f.Name]; ok { // Is this an output file?
			ext := filepath.Ext(f.Value.String())
			o[f.Name] = fmt.Sprintf("%s/%s/%s/%s%s", c.bucketName, user, name, strings.Replace(f.Name, ".", "_", -1), ext)
		}
	})
	return o, nil
}

func (c *Client) checkOutputs(ctx context.Context, name string, cmd []string) error {
	addrs, err := c.jobOutputAddresses(ctx, name, cmd)
	if err != nil {
		return err
	}
	bucket, err := OpenBucket(ctx, c.bucketName)
	if err != nil {
		return fmt.Errorf("cloud: opening bucket %s: %v", c.bucketName, err)
	}
	for _, addr := range addrs {
		for _, fname := range expandShp(addr) {
			url, err := url.Parse(fname)
			if err != nil {
				return fmt.Errorf("cloud: parsing URL %s: %v", fname, err)
			}
			key := strings.TrimLeft(url.Path, "/")
			r, err := bucket.NewReader(ctx, key, nil)
			if err != nil {
				return fmt.Errorf("cloud: opening reader for `%s`: %v", key, err)
			}
			if r.Size() == 0 {
				return fmt.Errorf("cloud: output file `%s` is zero-length: %v", key, err)
			}
			r.Close()
		}
	}
	return nil
}

// setOutputPaths changes the paths of the output files in the given
// job specification so that they match
// the locations where the files should be stored.
func (c *Client) setOutputPaths(ctx context.Context, job *cloudrpc.JobSpec) error {
	addrs, err := c.jobOutputAddresses(ctx, job.Name, job.Cmd)
	if err != nil {
		return err
	}
	for i, arg := range job.Args {
		if addr, ok := addrs[strings.TrimLeft(arg, "--")]; ok {
			job.Args[i+1] = addr
		}
	}
	return nil
}

// stageInputs stages the input data in blob storage and replaces the input
// file locations with the actual locations of the staged input files.
func (c *Client) stageInputs(ctx context.Context, job *cloudrpc.JobSpec) error {
	bucket, err := OpenBucket(ctx, c.bucketName)
	if err != nil {
		return err
	}
	url, err := url.Parse(c.bucketName)
	if err != nil {
		return fmt.Errorf("inmap/cloud: staging inputs: %v", err)
	}

	user, err := getUser(ctx)
	if err != nil {
		return err
	}
	for fname, data := range job.FileData {
		filePath := strings.TrimPrefix(url.Path+"/"+user+"/"+job.Name+"/"+fname, "/")
		if err := writeBlob(ctx, bucket, filePath, data); err != nil {
			return err
		}
		for i, arg := range job.Args {
			if fname == arg {
				job.Args[i] = url.Scheme + "://" + url.Hostname() + "/" + filePath
			}
		}
	}
	return nil
}
