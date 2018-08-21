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
	"fmt"
	"reflect"
	"testing"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/inmaputil"
)

func TestRunInputFromViper(t *testing.T) {
	js, err := inmaputil.CloudJobSpec("test_job", []string{"run", "steady"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if js.Version != inmap.Version {
		t.Errorf("version: %s != %s", js.Version, inmap.Version)
	}
	wantCmd := []string{"inmap", "run", "steady"}
	if !reflect.DeepEqual(js.Cmd, wantCmd) {
		t.Errorf("cmd: %s != %s", js.Cmd, wantCmd)
	}

	wantArgs := map[string]string{
		"--VarGrid.MortalityRateFile":    "764874ad5081665459c67d40607f68df6fc689aa695b4822e012aef84cba5394.shp",
		"--VarGrid.VariableGridDx":       "4000",
		"--NumIterations":                "0",
		"--VarGrid.CensusPopColumns":     "TotalPop,WhiteNoLat,Black,Native,Asian,Latino",
		"--VariableGridData":             "6cd7b21b88adfaac1ac16cf4d5a746d6818b17eaa1cbc629899020a0ef2e9ece.gob",
		"--OutputVariables":              "{\"TotalPM25\":\"PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA\",\"TotalPopD\":\"(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * AllCause / 100000\"}\n",
		"--OutputFile":                   "inmap_output.shp",
		"--VarGrid.PopThreshold":         "40000",
		"--VarGrid.Ynests":               "2,2,2",
		"--static":                       "false",
		"--VarGrid.MortalityRateColumns": "{\"AllCause\":\"TotalPop\",\"AsianMort\":\"Asian\",\"BlackMort\":\"Black\",\"LatinoMort\":\"Latino\",\"NativeMort\":\"Native\",\"WhNoLMort\":\"WhiteNoLat\"}\n",
		"--creategrid":                   "false",
		"--VarGrid.Xnests":               "2,2,2",
		"--EmissionsShapefiles":          "258bbcefe8c0073d6f323351463be9e9685e74bb92e367ca769b9536ed247213.shp",
		"--VarGrid.PopGridColumn":        "TotalPop",
		"--VarGrid.GridProj":             "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1",
		"--VarGrid.PopConcThreshold":     "1e-09",
		"--VarGrid.CensusFile":           "72f6717ef5f6f9600378fe5b192776ba142b3e93311c3dfd0b67bfecbe399990.shp",
		"--VarGrid.VariableGridYo":       "-4000",
		"--InMAPData":                    "434bf26e3fda1ef9cef7e1fa6cc6b5174d11a22b19cbe10d256adc83b2a97d44.ncf",
		"--VarGrid.VariableGridXo":       "-4000",
		"--OutputAllLayers":              "false",
		"--VarGrid.HiResLayers":          "1",
		"--VarGrid.PopDensityThreshold":  "0.0055",
		"--VarGrid.VariableGridDy":       "4000",
		"--EmissionUnits":                "tons/year",
		"--LogFile":                      "",
	}
	if len(js.Args) != len(wantArgs)*2 {
		t.Errorf("wrong number of arguments: %d != %d", len(js.Args)/2, len(wantArgs))
	}
	for i := 0; i < len(js.Args); i += 2 {
		key, val := js.Args[i], js.Args[i+1]
		if wantVal, ok := wantArgs[key]; ok {
			if val != wantVal {
				t.Errorf("invalid argument val for key %s: %s != %s", key, val, wantVal)
			}
		} else {
			t.Errorf("missing argument key '%s'", key)
		}
	}

	wantFiles := map[string]int{
		"258bbcefe8c0073d6f323351463be9e9685e74bb92e367ca769b9536ed247213.shp": 620,
		"258bbcefe8c0073d6f323351463be9e9685e74bb92e367ca769b9536ed247213.dbf": 869,
		"258bbcefe8c0073d6f323351463be9e9685e74bb92e367ca769b9536ed247213.prj": 432,
		"258bbcefe8c0073d6f323351463be9e9685e74bb92e367ca769b9536ed247213.shx": 140,
		"72f6717ef5f6f9600378fe5b192776ba142b3e93311c3dfd0b67bfecbe399990.shp": 236,
		"72f6717ef5f6f9600378fe5b192776ba142b3e93311c3dfd0b67bfecbe399990.dbf": 353,
		"72f6717ef5f6f9600378fe5b192776ba142b3e93311c3dfd0b67bfecbe399990.shx": 108,
		"72f6717ef5f6f9600378fe5b192776ba142b3e93311c3dfd0b67bfecbe399990.prj": 432,
		"764874ad5081665459c67d40607f68df6fc689aa695b4822e012aef84cba5394.shp": 236,
		"764874ad5081665459c67d40607f68df6fc689aa695b4822e012aef84cba5394.shx": 108,
		"764874ad5081665459c67d40607f68df6fc689aa695b4822e012aef84cba5394.dbf": 341,
		"764874ad5081665459c67d40607f68df6fc689aa695b4822e012aef84cba5394.prj": 432,
		"6c3122217a2817d29cbe795c72f6cb83c43321e70d922802db10b5ea4cf5a16e.txt": 244,
		"6cd7b21b88adfaac1ac16cf4d5a746d6818b17eaa1cbc629899020a0ef2e9ece.gob": 27389,
		"434bf26e3fda1ef9cef7e1fa6cc6b5174d11a22b19cbe10d256adc83b2a97d44.ncf": 14284,
	}
	if len(js.FileData) != len(wantFiles) {
		fmt.Errorf("incorrect number of files: %d != %d", len(js.FileData), len(wantFiles))
	}
	for name, b := range js.FileData {
		if wantB, ok := wantFiles[name]; ok {
			if len(b) != wantB {
				t.Errorf("file %s: length %d != %d", name, len(b), wantB)
			}
		} else {
			t.Errorf("missing file %s", name)
		}
	}

	if js.MemoryGB != 1 {
		t.Errorf("memory: %d != 1", js.MemoryGB)
	}
}
