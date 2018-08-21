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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NewCloudClient creates a new RPC client based on the information in Cfg.
func NewCloudClient() (cloudrpc.CloudRPCClient, error) {
	conn, err := grpc.Dial(Cfg.GetString("addr"), grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	if err != nil {
		return nil, err
	}
	return cloudrpc.NewCloudRPCClient(conn), nil
}

// CloudJobStart starts a new cloud job based on the information in Cfg.
func CloudJobStart(ctx context.Context, c cloudrpc.CloudRPCClient) error {
	in, err := CloudJobSpec(
		Cfg.GetString("job_name"),
		Cfg.GetStringSlice("cmds"),
		int32(Cfg.GetInt("memory_gb")),
	)
	if err != nil {
		return err
	}
	_, err = c.RunJob(ctx, in)
	if err != nil {
		return err
	}
	return nil
}

// CloudJobStatus checks the status of a cloud job
// based on the information in Cfg.
func CloudJobStatus(ctx context.Context, c cloudrpc.CloudRPCClient) (*cloudrpc.JobStatus, error) {
	in := &cloudrpc.JobName{
		Version: inmap.Version,
		Name:    Cfg.GetString("job_name"),
	}
	return c.Status(ctx, in)
}

// CloudJobOutput retrieves and saves the output of a cloud job
// based on the information in Cfg. The files will be saved
// in `current_dir/job_name`, where current_dir is the directory
// the command is run in.
func CloudJobOutput(ctx context.Context, c cloudrpc.CloudRPCClient) error {
	name := Cfg.GetString("job_name")
	in := &cloudrpc.JobName{
		Version: inmap.Version,
		Name:    name,
	}
	output, err := c.Output(ctx, in)
	if err != nil {
		return err
	}
	os.Mkdir(name, os.ModePerm)
	for fname, data := range output.Files {
		w, err := os.Create(filepath.Join(name, fname))
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		if err != nil {
			return err
		}
	}
	return nil
}

// CloudJobSpec initializes a cloudrpc.JobSpec object from the given
// configuration information. memoryGB and storageGB are the required
// amounts of RAM and hard-disk storage, respectively, in gigabytes.
// name is the job name and cmd is a list of InMAP sub-commands
// (e.g., "run steady").
func CloudJobSpec(name string, cmd []string, memoryGB int32) (*cloudrpc.JobSpec, error) {
	inputFields := make(map[string]struct{})
	for _, f := range InputFiles() {
		inputFields[f] = struct{}{}
	}

	js := &cloudrpc.JobSpec{
		Version:  inmap.Version,
		Name:     name,
		Cmd:      append([]string{"inmap"}, cmd...),
		MemoryGB: memoryGB,
		FileData: make(map[string][]byte),
	}

	execCmd, _, err := Root.Find(cmd)
	if err != nil {
		return nil, err
	}
	flags := execCmd.InheritedFlags()
	flags.AddFlagSet(execCmd.LocalFlags())
	var visitErr error
	flags.VisitAll(func(f *pflag.Flag) {
		if visitErr != nil {
			return
		}
		var val string
		v := Cfg.Get(f.Name)
		if v == nil || f.Name == "config" {
			return
		}
		switch v.(type) {
		case []string:
			val = strings.Join(v.([]string), ",")
		case []interface{}:
			val = strings.TrimPrefix(strings.TrimSuffix(fmt.Sprintf("%#v", v), "}"), "[]interface {}{")
			val = strings.Replace(val, ", ", ",", -1)
		case map[string]string, map[string]interface{}:
			var b bytes.Buffer
			e := json.NewEncoder(&b)
			if err := e.Encode(v); err != nil {
				panic(err)
			}
			val = b.String()
		default:
			val = strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%v", v), "["), "]")
		}
		argVal := val
		if _, ok := inputFields[f.Name]; ok {
			argVal = ""
			vals := stringsFromInterface(Cfg.Get(f.Name))
			for i, val := range vals {
				val, visitErr = localFileToRunInput(val, js)
				if visitErr != nil {
					return
				}
				if i == 0 {
					argVal += val
				} else {
					argVal += "," + val
				}
			}
		}
		js.Args = append(js.Args, fmt.Sprintf("--%s", f.Name), argVal)
	})
	if visitErr != nil {
		return nil, visitErr
	}
	return js, nil
}

func stringsFromInterface(val interface{}) []string {
	switch t := val.(type) {
	case string:
		return []string{val.(string)}
	case []string:
		return val.([]string)
	default:
		panic(fmt.Errorf("dist.RunInputFromViper: invalid file field type %T", t))
	}
}

// localFileToRunInput checks if filePath represents a local file (i.e., it doesn't
// start with http://, https://, gs://, or s3://) and if so copies its contents
// to the FileData field of ri using 'sha256checksum.ext' as the new file path,
// and returns the new file path of the file.
// As a special case, if the file has the extension '.shp', the function
// will copy the corresponding '.dbf', '.shx', and '.prj' files but only
// return the path of the original '.shp' file.
// filePath can contain environment variables.
func localFileToRunInput(filePath string, js *cloudrpc.JobSpec) (string, error) {
	if filePath == "" ||
		strings.HasPrefix(filePath, "http://") ||
		strings.HasPrefix(filePath, "https://") ||
		strings.HasPrefix(filePath, "gs://") ||
		strings.HasPrefix(filePath, "s3://") {
		return filePath, nil
	}
	filePath = os.ExpandEnv(filePath)
	ext := filepath.Ext(filePath)
	data, sum, err := fileContentsAndSum(filePath)
	if err != nil {
		return "", err
	}
	newPath := sum + ext
	js.FileData[newPath] = data
	if ext == ".shp" {
		for _, newExt := range []string{".dbf", ".shx", ".prj"} {
			data, _, err := fileContentsAndSum(filePath[0:len(filePath)-4] + newExt)
			if err != nil {
				return "", err
			}
			js.FileData[sum+newExt] = data
		}
	}
	return newPath, nil
}

// fileContentsAndSum returns the contents and sha256 checksum of a file.
func fileContentsAndSum(filePath string) ([]byte, string, error) {
	var dst bytes.Buffer
	src, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("dist: opening input file: %v", err)
	}
	if _, err := io.Copy(&dst, src); err != nil {
		return nil, "", err
	}
	sumBytes := sha256.Sum256(dst.Bytes())
	return dst.Bytes(), fmt.Sprintf("%x", sumBytes[0:sha256.Size]), nil
}
