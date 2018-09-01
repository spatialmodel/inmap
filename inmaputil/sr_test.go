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

package inmaputil_test

import (
	"context"
	"os"
	"testing"

	"github.com/spatialmodel/inmap/cloud"
	"github.com/spatialmodel/inmap/inmaputil"
)

func TestSR(t *testing.T) {
	output := "../cmd/inmap/testdata/tempSR.ncf"
	begin := 8
	end := 9
	layers := []int{0}
	cmds := []string{"run", "steady"}
	defer os.Remove(output)
	vgc, err := inmaputil.VarGridConfig(inmaputil.Cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), "user", "test_user")

	client, err := cloud.NewFakeClient(nil, nil, "file://test", inmaputil.Root, inmaputil.Cfg, inmaputil.InputFiles(), inmaputil.OutputFiles())
	if err != nil {
		t.Fatal(err)
	}
	c := cloud.FakeRPCClient{Client: client}
	os.Mkdir("test", os.ModePerm)
	defer os.RemoveAll("test")

	err = inmaputil.StartSR(ctx, "test_sr", cmds, 1,
		os.ExpandEnv(inmaputil.Cfg.GetString("VariableGridData")),
		vgc, begin, end, layers, c)
	if err != nil {
		t.Fatal(err)
	}
	err = inmaputil.SaveSR(ctx, "test_sr", output,
		os.ExpandEnv(inmaputil.Cfg.GetString("VariableGridData")),
		vgc, begin, end, layers, c)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSRPredict(t *testing.T) {
	inmaputil.Cfg.Set("SR.OutputFile", "../cmd/inmap/testdata/testSR_golden.ncf")
	inmaputil.Cfg.Set("OutputFile", "../cmd/inmap/testdata/output_SRPredict.shp")
	inmaputil.Cfg.Set("EmissionsShapefiles", []string{"../cmd/inmap/testdata/testEmisSR.shp"})

	inmaputil.Cfg.Set("config", "../cmd/inmap/configExample.toml")
	inmaputil.Root.SetArgs([]string{"sr", "predict"})
	if err := inmaputil.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestSRPredictAboveTop(t *testing.T) {
	cfg := inmaputil.Cfg
	cfg.Set("config", "../cmd/inmap/configExample.toml")
	cfg.Set("SR.OutputFile", "../cmd/inmap/testdata/testSR_golden.ncf")
	cfg.Set("OutputFile", "../cmd/inmap/testdata/output_SRPredict.shp")
	cfg.Set("EmissionsShapefiles", []string{"../cmd/inmap/testdata/testEmis.shp"})
	vcfg, err := inmaputil.VarGridConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := inmaputil.SRPredict(cfg.GetString("EmissionUnits"), cfg.GetString("SR.OutputFile"), cfg.GetString("OutputFile"), cfg.GetStringSlice("EmissionsShapefiles"), vcfg); err != nil {
		t.Fatal(err)
	}
}
