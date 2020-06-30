/*
Copyright © 2013 the InMAP authors.
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
	"os"
	"testing"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud"
)

func TestSR(t *testing.T) {
	cfg := InitializeConfig()
	output := "../cmd/inmap/testdata/tempSR.ncf"
	begin := 8
	end := 9
	layers := []int{0}
	cmds := []string{"run", "steady"}
	defer os.Remove(output)
	vgc, err := VarGridConfig(cfg.Viper)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), "user", "test_user")

	client, err := cloud.NewFakeClient(nil, nil, "file://test", cfg.Root, cfg.Viper, cfg.InputFiles(), cfg.OutputFiles())
	if err != nil {
		t.Fatal(err)
	}
	c := cloud.FakeRPCClient{Client: client}
	os.Mkdir("test", os.ModePerm)
	defer os.RemoveAll("test")

	err = StartSR(ctx, "test_sr", cmds, 1,
		os.ExpandEnv(cfg.GetString("VariableGridData")),
		vgc, begin, end, layers, c, cfg)
	if err != nil {
		t.Fatal(err)
	}
	err = SaveSR(ctx, "test_sr", output,
		os.ExpandEnv(cfg.GetString("VariableGridData")),
		vgc, begin, end, layers, c)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSRPredict(t *testing.T) {
	cfg := InitializeConfig()
	cfg.Set("SR.OutputFile", "../cmd/inmap/testdata/testSR_golden.ncf")
	cfg.Set("OutputFile", "../cmd/inmap/testdata/output_SRPredict.shp")
	cfg.Set("OutputVariables", `{"TotalPM25": "PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA",
"TotalPopD": "(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * allcause / 100000"}`)
	cfg.Set("EmissionsShapefiles", []string{"../cmd/inmap/testdata/testEmisSR.shp"})
	defer os.Remove(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_SRPredict.log"))
	defer inmap.DeleteShapefile(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_SRPredict.shp"))

	cfg.Set("config", "../cmd/inmap/configExample.toml")
	cfg.Root.SetArgs([]string{"srpredict"})
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestSRPredictAboveTop(t *testing.T) {
	cfg := InitializeConfig()
	cfg.Set("config", "../cmd/inmap/configExample.toml")
	cfg.Set("SR.OutputFile", "../cmd/inmap/testdata/testSR_golden.ncf")
	cfg.Set("OutputFile", "../cmd/inmap/testdata/output_SRPredict.shp")
	defer os.Remove(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_SRPredict.log"))
	defer inmap.DeleteShapefile(os.ExpandEnv("$INMAP_ROOT_DIR/cmd/inmap/testdata/output_SRPredict.shp"))
	cfg.Set("OutputVariables", `{"PNH4": "pNH4",
	"PNO3": "pNO3",
	"PSO4": "pSO4",
	"SOA": "SOA",
	"PrimPM25": "PrimaryPM25",
	"TotalPM25": "PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA"}`)
	cfg.Set("EmissionsShapefiles", []string{"../cmd/inmap/testdata/testEmis.shp"})
	outputVars, err := checkOutputVars(GetStringMapString("OutputVariables", cfg.Viper))
	if err != nil {
		t.Fatal(err)
	}
	vcfg, err := VarGridConfig(cfg.Viper)
	if err != nil {
		t.Fatal(err)
	}
	mask, err := parseMask(cfg.GetString("EmissionMaskGeoJSON"))
	if err != nil {
		t.Fatal(err)
	}
	if err := SRPredict(cfg.GetString("EmissionUnits"), cfg.GetString("SR.OutputFile"), cfg.GetString("OutputFile"), outputVars, cfg.GetStringSlice("EmissionsShapefiles"), mask, vcfg); err != nil {
		t.Fatal(err)
	}
}
