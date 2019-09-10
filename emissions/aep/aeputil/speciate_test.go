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
	"io"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spatialmodel/inmap/emissions/aep"
)

func TestSpeciate(t *testing.T) {
	type config struct {
		Inventory InventoryConfig
		Speciate  SpeciateConfig
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

	c.Speciate.Speciation = c.Inventory.PolsToKeep

	emis, _, err := c.Inventory.ReadEmissions()
	if err != nil {
		t.Fatal(err)
	}
	iter := c.Speciate.Iterator(IteratorFromMap(emis))
	for {
		_, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
	}
	report := iter.Report()
	table := report.TotalsTable()
	want := aep.Table{
		[]string{"Group", "File", "ALK3 (kmol)", "ALK4 (kmol)", "Ammonia (kmol)", "Elemental Carbon (kg)", "Nitrate (kg)", "Nitrogen Dioxide (kmol)", "Nitrogen Monoxide (Nitric Oxide) (kmol)", "Organic carbon (kg)", "Other Unspeciated PM2.5 (kg)", "Particulate Non-Carbon Organic Matter (kg)", "Sulfate (kg)", "Sulfur (kg)", "Sulfur dioxide (kmol)"},
		[]string{"", "Speciation", "7676.472685560194", "3223.2914382084273", "1.9997713398532004", "492788.36977318383", "27233.04148746542", "61173.41390152345", "562795.4078940157", "324202.87485077884", "207489.83990449845", "127087.5269415053", "110228.97744926481", "36310.72198328722", "246747.06580861437"},
	}
	compareTables(table, want, 1.0e-14, t)

	droppedTable := report.DroppedTotalsTable()
	droppedWant := aep.Table{[]string{"Group", "File"}, []string{"", "Speciation"}}
	compareTables(droppedTable, droppedWant, 1.0e-14, t)
}

func compareTables(table, want aep.Table, tolerance float64, t *testing.T) {
	if !reflect.DeepEqual(table[0], want[0]) {
		t.Errorf("inventory report header: have %v, want %v", table[0], want[0])
	}
	if len(want[1]) != len(table[1]) {
		t.Fatalf("line 1: length %d != %d", len(want[1]), len(table[1]))
	}
	for i := 2; i < len(want[1]); i++ {
		v1, err := strconv.ParseFloat(table[1][i], 64)
		if err != nil {
			t.Fatal(err)
		}
		v2, err := strconv.ParseFloat(want[1][i], 64)
		if err != nil {
			t.Fatal(err)
		}
		if different(v1, v2, tolerance) {
			t.Errorf("%s: %g != %g", table[0][i], v1, v2)
		}
	}
}

func different(v1, v2, tolerance float64) bool {
	if 2*(v2-v1)/(v2+v1) > tolerance {
		return true
	}
	return false
}
