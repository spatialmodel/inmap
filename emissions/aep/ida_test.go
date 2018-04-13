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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.*/

package aep

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ctessum/geom"
	"github.com/ctessum/unit"
	"github.com/ctessum/unit/badunit"
)

func TestIDAPoint(t *testing.T) {
	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Fatal(err)
	}

	rec := ` 2  102001001       1              1            ORIS  BLR   XINDUSTRIA NAVAL DE CALIFORNIA S.A. DE C.31499900          29.81.8044141.  37.08449 14.50131                    0                                                     342931.863611-116.6108          4.94   0.01353425 25     75          0.0                              26     76                         0.07  1.917808E-4                    0.0            9.78   0.02679452                    0.0            9.42   0.02580822                    0.0            0.07  1.917808E-4                    0.0          7750.4     21.23397                    0.0   `
	pollutants := strings.Split("CO NH3 NOX PM10 PM2_5 SO2 VOC", " ")
	rI, err := NewIDAPoint(rec, pollutants, Mexico, start, end, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r := rI.(*pointRecordIDA)

	sdExpected := SourceData{
		FIPS:       "02001",
		SCC:        "0031499900",
		Country:    Mexico,
		SourceType: "",
	}
	if !reflect.DeepEqual(sdExpected, r.SourceData) {
		t.Errorf("want %v but have %v", sdExpected, r.SourceData)
	}
	pdExpected := PointSourceData{
		PlantID:          "02001001",
		PointID:          "1",
		StackID:          "1",
		Segment:          "X",
		Plant:            "INDUSTRIA NAVAL DE CALIFORNIA S.A. DE C.",
		ORISFacilityCode: "ORIS",
		ORISBoilerID:     "BLR",
		StackHeight:      badunit.Foot(29.8),
		StackDiameter:    badunit.Foot(1.8044),
		StackTemp:        badunit.Fahrenheit(141.),
		StackFlow:        unit.New(1.050114086432, unit.Meter3PerSecond),
		StackVelocity:    badunit.FootPerSecond(14.50131),
		Point:            geom.Point{X: -116.6108, Y: 31.863611},
		SR:               longlat,
	}
	if !reflect.DeepEqual(pdExpected, r.PointSourceData) {
		t.Errorf("want %v but have %v", pdExpected, r.PointSourceData)
	}

	edExpected := EconomicData{
		SIC:   "3429",
		NAICS: "",
	}
	if !reflect.DeepEqual(edExpected, r.EconomicData) {
		t.Errorf("want %v but have %v", edExpected, r.EconomicData)
	}

	cdExpected := map[string]ControlData{
		"CO": ControlData{
			MACT: "",
			CEff: 25,
			REff: 75,
		},
		"NH3": ControlData{
			MACT: "",
			CEff: 26,
			REff: 76,
		},
		"NOX": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
		"PM10": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
		"PM2_5": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
		"SO2": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
		"VOC": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
	}
	if !reflect.DeepEqual(cdExpected, r.ControlData) {
		t.Errorf("want %v but have %v", cdExpected, r.ControlData)
	}

	emis := r.PeriodTotals(start, end)
	emisExpected := map[Pollutant]*unit.Unit{
		Pollutant{Name: "PM2_5"}: unit.New(8545.6827, unit.Kilogram),
		Pollutant{Name: "SO2"}:   unit.New(63.502950000000006, unit.Kilogram),
		Pollutant{Name: "VOC"}:   unit.New(7.031046623999999e+06, unit.Kilogram),
		Pollutant{Name: "CO"}:    unit.New(4481.4939, unit.Kilogram),
		Pollutant{Name: "NH3"}:   unit.New(0, unit.Kilogram),
		Pollutant{Name: "NOX"}:   unit.New(63.502950000000006, unit.Kilogram),
		Pollutant{Name: "PM10"}:  unit.New(8872.269299999998, unit.Kilogram),
	}

	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}
}

func TestIDAArea(t *testing.T) {
	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}
	rec := ` 2  121020040004.157395360.01139012            25     75 99          0.0       0.0                           19.95549770.05467259                           0.831479070.00227802                           0.19955498 5.4672E-4                           4.961953160.01359439                           0.16629581  4.556E-4                           `
	pollutants := strings.Split("CO NH3 NOX PM10 PM2_5 SO2 VOC", " ")
	rI, err := NewIDAArea(rec, pollutants, Mexico, start, end, badunit.Ton)
	if err != nil {
		t.Fatal(err)
	}
	r := rI.(*polygonRecordIDA)

	sdExpected := SourceData{
		FIPS:       "02001",
		SCC:        "2102004000",
		Country:    Mexico,
		SourceType: "",
	}
	if !reflect.DeepEqual(sdExpected, r.SourceData) {
		t.Errorf("want %v but have %v", sdExpected, r.SourceData)
	}

	cdExpected := map[string]ControlData{
		"CO": ControlData{
			MACT: "",
			CEff: 25,
			REff: 75,
			RPen: 99,
		},
		"NH3": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
		"NOX": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
		"PM10": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
		"PM2_5": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
		"SO2": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
		"VOC": ControlData{
			MACT: "",
			CEff: 0,
			REff: 0,
		},
	}
	if !reflect.DeepEqual(cdExpected, r.ControlData) {
		t.Errorf("want %v but have %v", cdExpected, r.ControlData)
	}

	emis := r.PeriodTotals(start, end)
	emisExpected := map[Pollutant]*unit.Unit{
		Pollutant{Name: "PM2_5"}: badunit.Ton(0.19955498),
		Pollutant{Name: "SO2"}:   badunit.Ton(4.96195316),
		Pollutant{Name: "VOC"}:   badunit.Ton(0.16629581),
		Pollutant{Name: "CO"}:    badunit.Ton(4.15739536),
		Pollutant{Name: "NH3"}:   unit.New(0, unit.Kilogram),
		Pollutant{Name: "NOX"}:   badunit.Ton(19.9554977),
		Pollutant{Name: "PM10"}:  badunit.Ton(0.83147907),
	}

	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}
}

func TestIDAMobile(t *testing.T) {
	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Fatal(err)
	}
	rec := ` 2  1          2201001000  7697.581  21.08926  18.363470.05031088  321.4992 0.8808197  20.069510.05498496  18.30982 0.0501639  35.085230.09612392  1053.695  2.886834`
	pollutants := strings.Split("CO NH3 NOX PM10 PM2_5 SO2 VOC", " ")
	rI, err := NewIDAMobile(rec, pollutants, Mexico, start, end, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r := rI.(*mobilePolygonRecordIDA)

	sdExpected := SourceData{
		FIPS:       "02001",
		SCC:        "2201001000",
		Country:    Mexico,
		SourceType: "",
	}
	if !reflect.DeepEqual(sdExpected, r.SourceData) {
		t.Errorf("want %v but have %v", sdExpected, r.SourceData)
	}

	emis := r.PeriodTotals(start, end)
	emisExpected := map[Pollutant]*unit.Unit{
		Pollutant{Name: "PM2_5"}: badunit.Ton(18.30982),
		//"SO2":   badunit.Ton(35.08523),
		Pollutant{Name: "SO2"}:  unit.New(31828.794377549995, unit.Kilogram),
		Pollutant{Name: "VOC"}:  badunit.Ton(1053.695),
		Pollutant{Name: "CO"}:   badunit.Ton(7697.581),
		Pollutant{Name: "NH3"}:  badunit.Ton(18.36347),
		Pollutant{Name: "NOX"}:  badunit.Ton(321.4992),
		Pollutant{Name: "PM10"}: badunit.Ton(20.06951),
	}

	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}
}
