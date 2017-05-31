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
	configFile := "../inmap/configExample.toml"
	Cfg.SetConfigFile(configFile)
	cfg, err := LoadConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	cfg.SR.OutputFile = "../inmap/testdata/tempSR.ncf"
	begin := 8
	end := 9
	layers := []int{0}
	if err := RunSR(cfg, configFile, begin, end, layers); err != nil {
		t.Fatal(err)
	}
	os.Remove(cfg.SR.OutputFile)
}

func TestWorkerInit(t *testing.T) {
	Cfg.SetConfigFile("../inmap/configExample.toml")
	cfg, err := LoadConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorker(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Init(nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSRPredict(t *testing.T) {
	Cfg.SetConfigFile("../inmap/configExample.toml")
	cfg, err := LoadConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	cfg.SR.OutputFile = "../inmap/testdata/testSR.ncf"
	cfg.OutputFile = "../inmap/testdata/output_SRPredict.shp"
	cfg.EmissionsShapefiles = []string{"../inmap/testdata/testEmisSR.shp"}

	if err := SRPredict(cfg); err != nil {
		t.Fatal(err)
	}
}
