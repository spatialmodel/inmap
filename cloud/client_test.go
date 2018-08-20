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
	"os"
	"reflect"
	"testing"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spatialmodel/inmap/inmaputil"
)

func TestClient_fake(t *testing.T) {
	c, err := NewFakeClient(t, true, "file://test")
	if err != nil {
		t.Fatal(err)
	}
	os.Mkdir("test", os.ModePerm)
	defer os.RemoveAll("test")

	jobSpec, err := inmaputil.CloudJobSpec("test_job", []string{"run", "steady"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), "user", "test_user")

	t.Run("RunJob", func(t *testing.T) {
		status, err := c.RunJob(ctx, jobSpec)
		if err != nil {
			t.Fatal(err)
		}
		wantStatus := &cloudrpc.JobStatus{
			Status: "&JobStatus{Conditions:[],StartTime:<nil>,CompletionTime:<nil>,Active:0,Succeeded:0,Failed:0,}",
		}
		if !reflect.DeepEqual(wantStatus, status) {
			t.Errorf("status:\n%+v\n!=\n%+v", status, wantStatus)
		}
	})

	t.Run("Status", func(t *testing.T) {
		status, err := c.Status(ctx, &cloudrpc.JobName{
			Version: inmap.Version,
			Name:    "test_job",
		})
		if err != nil {
			t.Fatal(err)
		}
		wantStatus := &cloudrpc.JobStatus{
			Status: "&JobStatus{Conditions:[],StartTime:<nil>,CompletionTime:<nil>,Active:0,Succeeded:0,Failed:0,}",
		}
		if !reflect.DeepEqual(wantStatus, status) {
			t.Errorf("status:\n%+v\n!=\n%+v", status, wantStatus)
		}
	})

	t.Run("Output", func(t *testing.T) {
		output, err := c.Output(ctx, &cloudrpc.JobName{
			Version: inmap.Version,
			Name:    "test_job",
		})
		if err != nil {
			t.Fatal(err)
		}
		wantFiles := map[string]int{
			"log.txt":        94100,
			"OutputFile.shp": 2276,
			"OutputFile.dbf": 561,
			"OutputFile.shx": 228,
			"OutputFile.prj": 431,
		}
		if len(output.Files) != len(wantFiles) {
			fmt.Errorf("wrong number of files: %d != %d", len(output.Files), len(wantFiles))
		}
		for name, data := range output.Files {
			if wantLen, ok := wantFiles[name]; ok {
				if len(data) != wantLen && name != "log.txt" {
					t.Errorf("wrong file length for %s: %d != %d", name, len(data), wantLen)
				}
			} else {
				t.Errorf("missing files '%s'", name)
			}
		}
	})
}
