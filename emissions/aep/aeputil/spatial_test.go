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
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
	"github.com/spatialmodel/inmap/emissions/aep"
)

func TestSpatial(t *testing.T) {
	type config struct {
		Inventory InventoryConfig
		Spatial   SpatialConfig
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

	sr, err := proj.Parse(c.Spatial.OutputSR)
	if err != nil {
		t.Fatal(err)
	}
	grid := aep.NewGridRegular("test", 111, 84, 48000, 48000, -2736000.0, -2088000.0, sr)
	g := make([]geom.Polygonal, len(grid.Cells))
	for i, c := range grid.Cells {
		g[i] = c.Polygonal
	}
	c.Spatial.GridCells = g

	records, _, err := c.Inventory.ReadEmissions()
	if err != nil {
		t.Fatal(err)
	}

	wantEmis := map[aep.Pollutant]float64{
		aep.Pollutant{Name: "NOX"}:   1.9694509976996027e+07 + 3329.29929452133,
		aep.Pollutant{Name: "VOC"}:   650426.9504917137,
		aep.Pollutant{Name: "PM2_5"}: 1.3251549508572659e+06 + 186.401532717915,
		aep.Pollutant{Name: "SO2"}:   1.5804381260919824e+07 + 1939.6783010388299,
		aep.Pollutant{Name: "NH3"}:   34.056105917699995,
	}

	wantUnits := map[aep.Pollutant]unit.Dimensions{
		aep.Pollutant{Name: "PM2_5"}: unit.Dimensions{4: 1},
		aep.Pollutant{Name: "NH3"}:   unit.Dimensions{4: 1},
		aep.Pollutant{Name: "SO2"}:   unit.Dimensions{4: 1},
		aep.Pollutant{Name: "NOX"}:   unit.Dimensions{4: 1},
		aep.Pollutant{Name: "VOC"}:   unit.Dimensions{4: 1},
	}
	iter := c.Spatial.Iterator(IteratorFromMap(records), 0)
	for {
		_, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
	}
	emis, units := iter.SpatialTotals()
	for pol, grid := range emis {
		if grid.Sum() != wantEmis[pol] {
			t.Errorf("emissions for %v: have %g but want %g", pol, grid.Sum(), wantEmis[pol])
		}
	}
	for pol, u := range units {
		if !u.Matches(wantUnits[pol]) {
			t.Errorf("units for %v: have %v but want %v", pol, wantUnits[pol], units)
		}
	}
	report := iter.Report()

	t.Run("totals", func(t *testing.T) {
		totals := report.TotalsTable()
		totalsWant := aep.Table{
			[]string{"Group", "File", "NH3 (kg)", "NOX (kg)", "PM2_5 (kg)", "SO2 (kg)", "VOC (kg)"},
			[]string{"", "Spatial", "34.056105917699995", "1.9697839276290547e+07", "1.3253413523899838e+06", "1.5806320939220862e+07", "650426.9504917137"},
		}
		compareTables(totals, totalsWant, 1.0e-14, t)
	})
	t.Run("totals", func(t *testing.T) {
		droppedTotals := report.DroppedTotalsTable()
		droppedTotalsWant := aep.Table{
			[]string{"Group", "File", "NH3 (kg)", "NOX (kg)", "PM2_5 (kg)", "SO2 (kg)", "VOC (kg)"},
			[]string{"", "Spatial", "0", "0", "0", "0", "0"},
		}
		compareTables(droppedTotals, droppedTotalsWant, 1.0e-14, t)
	})
}

func TestSpatial_coards(t *testing.T) {
	type config struct {
		Inventory InventoryConfig
		Spatial   SpatialConfig
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

	c.Inventory.NEIFiles = nil
	c.Inventory.COARDSFiles = map[string][]string{
		"all": {"../testdata/emis_coards_hawaii.nc"},
	}
	c.Inventory.COARDSYear = 2016

	c.Spatial.SrgSpec = "../testdata/srgspec_osm.json"
	c.Spatial.SrgSpecType = "OSM"
	c.Spatial.GridRef = []string{"testdata/gridref_osm.txt"}
	c.Spatial.OutputSR = "+proj=longlat"

	sr, err := proj.Parse(c.Spatial.OutputSR)
	if err != nil {
		t.Fatal(err)
	}
	grid := aep.NewGridRegular("test grid", 4, 4, 0.1, 0.1, -158, 21.25, sr)
	g := make([]geom.Polygonal, len(grid.Cells))
	for i, c := range grid.Cells {
		g[i] = c.Polygonal
	}
	c.Spatial.GridCells = g

	records, report, err := c.Inventory.ReadEmissions()
	if err != nil {
		t.Fatal(err)
	}

	wantEmis := map[aep.Pollutant]float64{
		aep.Pollutant{Name: "NOx"}:   1.3984131235786172e+07,
		aep.Pollutant{Name: "VOC"}:   2.7990005393761573e+06,
		aep.Pollutant{Name: "PM2_5"}: 5.988116747096776e+06,
		aep.Pollutant{Name: "SOx"}:   5.494101046635956e+06,
		aep.Pollutant{Name: "NH3"}:   1.2303264126897848e+06,
	}

	wantUnits := map[aep.Pollutant]unit.Dimensions{
		aep.Pollutant{Name: "PM2_5"}: unit.Dimensions{4: 1},
		aep.Pollutant{Name: "NH3"}:   unit.Dimensions{4: 1},
		aep.Pollutant{Name: "SOx"}:   unit.Dimensions{4: 1},
		aep.Pollutant{Name: "NOx"}:   unit.Dimensions{4: 1},
		aep.Pollutant{Name: "VOC"}:   unit.Dimensions{4: 1},
	}
	iter := c.Spatial.Iterator(IteratorFromMap(records), 0)
	for {
		_, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
	}
	emis, units := iter.SpatialTotals()
	for pol, grid := range emis {
		if different(grid.Sum(), wantEmis[pol], 1e-10) {
			t.Errorf("emissions for %v: have %g but want %g", pol, grid.Sum(), wantEmis[pol])
		}
	}
	for pol, u := range units {
		if !u.Matches(wantUnits[pol]) {
			t.Errorf("units for %v: have %v but want %v", pol, wantUnits[pol], units)
		}
	}
	report = iter.Report()

	t.Run("totals", func(t *testing.T) {
		totals := report.TotalsTable()
		totalsWant := aep.Table{
			[]string{"Group", "File", "NH3 (kg)", "NOx (kg)", "PM2_5 (kg)", "SOx (kg)", "VOC (kg)"},
			[]string{"", "Spatial", "1.230326412689783e+06", "1.3984131235786151e+07", "5.988116747096768e+06", "5.494101046635947e+06", "2.799000539376153e+06"},
		}
		compareTables(totals, totalsWant, 1.0e-14, t)
	})
	t.Run("totals", func(t *testing.T) {
		droppedTotals := report.DroppedTotalsTable()
		droppedTotalsWant := aep.Table{
			[]string{"Group", "File", "NH3 (kg)", "NOx (kg)", "PM2_5 (kg)", "SOx (kg)", "VOC (kg)"},
			[]string{"", "Spatial", "0", "0", "0", "0", "0"},
		}
		compareTables(droppedTotals, droppedTotalsWant, 1.0e-14, t)
	})
}
