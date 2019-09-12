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
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/spatialmodel/inmap/cloud"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
)

func TestCloud(t *testing.T) {
	cfg := InitializeConfig()
	checkRun := func(b []byte, err error) {
		if err != nil {
			log.Println(err)
		}
		t.Log(string(b))
	}
	client, err := cloud.NewFakeClient(nil, checkRun, "file://test", cfg.Root, cfg.Viper, cfg.InputFiles(), cfg.OutputFiles())
	if err != nil {
		t.Fatal(err)
	}
	c := cloud.FakeRPCClient{Client: client}
	ctx := context.WithValue(context.Background(), "user", "test_user")
	os.Mkdir("test", os.ModePerm)
	defer os.RemoveAll("test")

	t.Run("start", func(t *testing.T) {
		if err := CloudJobStart(ctx, c, cfg); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("status", func(t *testing.T) {
		status, err := CloudJobStatus(ctx, c, cfg)
		if err != nil {
			t.Fatal(err)
		}

		wantStatus := &cloudrpc.JobStatus{
			Status:         cloudrpc.Status_Complete,
			StartTime:      1359849600,
			CompletionTime: 1359936000,
		}
		if !reflect.DeepEqual(status, wantStatus) {
			t.Errorf("wrong status: %v != %v", status, wantStatus)
		}
	})

	t.Run("output", func(t *testing.T) {
		defer os.RemoveAll("test_job")
		err := CloudJobOutput(ctx, c, cfg)
		if err != nil {
			t.Fatal(err)
		}
		wantFiles := map[string]int64{
			"OutputFile.dbf": 465,
			"OutputFile.prj": 431,
			"OutputFile.shp": 2276,
			"OutputFile.shx": 228,
			"LogFile":        94169,
		}
		files, err := ioutil.ReadDir("test_job")
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != len(wantFiles) {
			t.Errorf("wrong number of files: %d != %d", len(files), len(wantFiles))
		}
		for _, file := range files {
			if wantLen, ok := wantFiles[file.Name()]; ok {
				if file.Size() != wantLen && file.Name() != "LogFile" {
					t.Errorf("%s: wrong file size: %d != %d", file.Name(), file.Size(), wantLen)
				}
			} else {
				t.Errorf("extra file %s", file.Name())
			}
		}
	})
}
