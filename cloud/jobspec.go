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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lnashier/viper"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// JobSpec initializes a cloudrpc.JobSpec object from the given
// configuration information. memoryGB and storageGB are the required
// amounts of RAM and hard-disk storage, respectively, in gigabytes.
// name is the user-specified job name, cmdArgs is a list of InMAP sub-commands
// (e.g., "run steady"), and inputFiles is a list of the configuration arguments
// that represent input files.
func JobSpec(root *cobra.Command, config *viper.Viper, name string, cmdArgs, inputFiles []string, memoryGB int32) (*cloudrpc.JobSpec, error) {
	inputFields := make(map[string]struct{})
	for _, f := range inputFiles {
		inputFields[f] = struct{}{}
	}

	js := &cloudrpc.JobSpec{
		Version:  inmap.Version,
		Name:     name,
		Cmd:      append([]string{"inmap"}, cmdArgs...),
		MemoryGB: memoryGB,
		FileData: make(map[string][]byte),
	}

	execCmd, _, err := root.Find(cmdArgs)
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
		v := config.Get(f.Name)
		if v == nil || f.Name == "config" || f.Name == "addr" || f.Name == "job_name" {
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
			vals := stringsFromInterface(config.Get(f.Name))
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
		if argVal != "false" {
			if argVal == "true" {
				js.Args = append(js.Args, fmt.Sprintf("--%s", f.Name), "true")
			} else {
				js.Args = append(js.Args, fmt.Sprintf("--%s", f.Name), argVal)
			}
		}
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
	case []interface{}:
		valSlice := val.([]interface{})
		s := make([]string, len(valSlice))
		for i, v := range valSlice {
			s[i] = fmt.Sprint(v)
		}
		return s
	default:
		panic(fmt.Errorf("cloud.JobSpec: invalid file field type %T", t))
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
		return nil, "", fmt.Errorf("cloud: opening input file: %v", err)
	}
	if _, err := io.Copy(&dst, src); err != nil {
		return nil, "", err
	}
	if err := src.Close(); err != nil {
		return nil, "", err
	}
	sumBytes := sha256.Sum256(dst.Bytes())
	return dst.Bytes(), fmt.Sprintf("%x", sumBytes[0:sha256.Size]), nil
}
