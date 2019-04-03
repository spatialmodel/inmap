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

package cloud_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spatialmodel/inmap/inmaputil"
)

// Set up directory location for configuration files.
func init() {
	os.Setenv("INMAP_ROOT_DIR", "../")
}

func TestClient_fake(t *testing.T) {
	checkConfig := func(cmd []string) {
		wantCmd := []string{"inmap", "run", "steady",
			"--EmissionUnits=tons/year",
			"--EmissionsShapefiles=file://test/test/test_user/test_job/258bbcefe8c0073d6f323351463be9e9685e74bb92e367ca769b9536ed247213.shp",
			"--InMAPData=file://test/test/test_user/test_job/434bf26e3fda1ef9cef7e1fa6cc6b5174d11a22b19cbe10d256adc83b2a97d44.ncf",
			"--LogFile=file://test/test/test_user/test_job/LogFile",
			"--NumIterations=0",
			"--OutputFile=file://test/test/test_user/test_job/OutputFile.shp",
			"--OutputVariables={\"TotalPM25\":\"PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA\",\"TotalPopD\":\"(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * AllCause / 100000\"}\n",
			"--VarGrid.CensusFile=file://test/test/test_user/test_job/72f6717ef5f6f9600378fe5b192776ba142b3e93311c3dfd0b67bfecbe399990.shp",
			"--VarGrid.CensusPopColumns=TotalPop,WhiteNoLat,Black,Native,Asian,Latino",
			"--VarGrid.GridProj=+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1",
			"--VarGrid.HiResLayers=1",
			"--VarGrid.MortalityRateColumns={\"AllCause\":\"TotalPop\",\"AsianMort\":\"Asian\",\"BlackMort\":\"Black\",\"LatinoMort\":\"Latino\",\"NativeMort\":\"Native\",\"WhNoLMort\":\"WhiteNoLat\"}\n",
			"--VarGrid.MortalityRateFile=file://test/test/test_user/test_job/764874ad5081665459c67d40607f68df6fc689aa695b4822e012aef84cba5394.shp",
			"--VarGrid.PopConcThreshold=1e-09", "--VarGrid.PopDensityThreshold=0.0055",
			"--VarGrid.PopGridColumn=TotalPop", "--VarGrid.PopThreshold=40000", "--VarGrid.VariableGridDx=4000",
			"--VarGrid.VariableGridDy=4000", "--VarGrid.VariableGridXo=-4000", "--VarGrid.VariableGridYo=-4000",
			"--VarGrid.Xnests=2,2,2", "--VarGrid.Ynests=2,2,2",
			"--VariableGridData=file://test/test/test_user/test_job/3c7e1a672dad2c3e41c76a2d3b1bf3b528510f354231cd06ddd374ebdf2a010d.gob",
		}
		if len(cmd) != len(wantCmd) {
			t.Errorf("wrong command length: %d != %d", len(cmd), len(wantCmd))
		}
		for i, a := range cmd {
			if i >= len(wantCmd) {
				t.Errorf("command element %d: '%s' != ''", i, a)
			} else if a != wantCmd[i] {
				t.Errorf("command element %d: '%s' != '%s'", i, a, wantCmd[i])
			}
		}
	}

	checkRun := func(o []byte, err error) {
		if err != nil {
			t.Error(err)
		}
		for _, l := range strings.Split(string(o), "\n") {
			if strings.Contains(strings.ToLower(l), "error") {
				t.Log(l)
			}
		}
	}
	cfg := inmaputil.InitializeConfig()

	c, err := cloud.NewFakeClient(checkConfig, checkRun, "file://test/test", cfg.Root, cfg.Viper, cfg.InputFiles(), cfg.OutputFiles())
	if err != nil {
		t.Fatal(err)
	}
	os.Mkdir("test", os.ModePerm)
	defer os.RemoveAll("test")

	jobSpec, err := cloud.JobSpec(cfg.Root, cfg.Viper, "test_job", []string{"run", "steady"}, cfg.InputFiles(), 1)
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
			Status:         cloudrpc.Status_Complete,
			StartTime:      1359849600,
			CompletionTime: 1359936000,
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
			Status:         cloudrpc.Status_Complete,
			StartTime:      1359849600,
			CompletionTime: 1359936000,
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
			"LogFile":        94100,
			"OutputFile.shp": 2276,
			"OutputFile.dbf": 465,
			"OutputFile.shx": 228,
			"OutputFile.prj": 431,
		}
		if len(output.Files) != len(wantFiles) {
			t.Errorf("wrong number of files: %d != %d", len(output.Files), len(wantFiles))
		}
		for name, data := range output.Files {
			if wantLen, ok := wantFiles[name]; ok {
				if len(data) != wantLen && name != "LogFile" {
					t.Errorf("wrong file length for %s: %d != %d", name, len(data), wantLen)
				}
			} else {
				t.Errorf("missing files '%s'", name)
			}
		}
	})
	t.Run("Delete", func(t *testing.T) {
		_, err := c.Delete(ctx, &cloudrpc.JobName{
			Version: inmap.Version,
			Name:    "test_job",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Ensure the directory is empty
		err = filepath.Walk("test", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			fmt.Println(path)
			return fmt.Errorf("found file %s in directory that should have been deleted", path)
		})
		if err != nil {
			t.Error(err)
		}
	})
}
