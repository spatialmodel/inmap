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
	"os"
	"testing"
)

func TestSR(t *testing.T) {
	Cfg.Set("SR.OutputFile", "../cmd/inmap/testdata/tempSR.ncf")
	Cfg.Set("begin", 8)
	Cfg.Set("end", 9)
	Cfg.Set("layers", []int{0})
	Cfg.Set("config", "../inmap/configExample.toml")
	defer os.Remove(Cfg.GetString("SR.OutputFile"))
	Root.SetArgs([]string{"sr"})
	if err := Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestWorkerInit(t *testing.T) {
	Cfg.Set("config", "../inmap/configExample.toml")
	if err := Root.PersistentPreRunE(nil, nil); err != nil {
		t.Fatal(err)
	}
	vgc, err := VarGridConfig(Cfg)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorker(
		os.ExpandEnv(Cfg.GetString("VariableGridData")),
		os.ExpandEnv(Cfg.GetString("InMAPData")),
		vgc,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Init(nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSRPredict(t *testing.T) {
	Cfg.Set("SR.OutputFile", "../cmd/inmap/testdata/testSR.ncf")
	Cfg.Set("OutputFile", "../cmd/inmap/testdata/output_SRPredict.shp")
	Cfg.Set("EmissionsShapefiles", []string{"../cmd/inmap/testdata/testEmisSR.shp"})

	Cfg.Set("config", "../inmap/configExample.toml")
	Root.SetArgs([]string{"sr", "predict"})
	if err := Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestSRPredictAboveTop(t *testing.T) {
	Cfg.Set("config", "../inmap/configExample.toml")
	Cfg.Set("SR.OutputFile", "../cmd/inmap/testdata/testSR.ncf")
	Cfg.Set("OutputFile", "../cmd/inmap/testdata/output_SRPredict.shp")
	Cfg.Set("EmissionsShapefiles", []string{"../cmd/inmap/testdata/testEmis.shp"})
	cfg, err := VarGridConfig(Cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := SRPredict(Cfg.GetString("EmissionUnits"), Cfg.GetString("SR.OutputFile"), Cfg.GetString("OutputFile"), Cfg.GetStringSlice("EmissionsShapefiles"), cfg); err != nil {
		t.Fatal(err)
	}
}
