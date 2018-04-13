/*
Copyright © 2017 the InMAP authors.
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

package aeputil

import (
	"os"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spatialmodel/inmap/emissions/aep"
)

func TestInventory(t *testing.T) {
	type config struct {
		Inventory InventoryConfig
	}
	r, err := os.Open("testdata/example_config.toml")
	if err != nil {
		t.Fatal(err)
	}

	c := new(config)

	// Read the configuration file into the configuration variable.
	if _, err = toml.DecodeReader(r, c); err != nil {
		t.Fatal(err)
	}
	_, report, err := c.Inventory.ReadEmissions()
	if err != nil {
		t.Fatal(err)
	}
	want := aep.Table{
		[]string{"Group", "File", "NH3 (kg)", "NOX (kg)", "PM2_5 (kg)", "SO2 (kg)", "VOC (kg)"},
		[]string{"othar", "testdata/testemis.csv", "34.056105917699995", "1.9697839276290547e+07", "1.3253413523899838e+06", "1.5806320939220862e+07", "650426.9504917137"},
	}
	if !reflect.DeepEqual(report.TotalsTable(), want) {
		t.Errorf("inventory report: have %v, want %v", report.TotalsTable(), want)
	}
}
