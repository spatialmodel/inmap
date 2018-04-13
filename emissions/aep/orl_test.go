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

func TestORLPoint(t *testing.T) {
	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}

	rec := strings.Split(`"10001","1096","3159","31591096","EC","SECA_C3","2280003200","02","03",65.620000000000005,2.625,539.60000000000002,0,82.019999999999996,"SIC","MACT","NAICS","L",-75.492564130000005,39.364973204000002,,"NOX",12.878310506,,25,75,,,,"orisFacility","orisBoiler",,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,`, ",")
	rI, err := NewORLPoint(rec, USA, start, end, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r := rI.(*PointRecord)

	sdExpected := SourceData{
		FIPS:       "10001",
		SCC:        "2280003200",
		Country:    USA,
		SourceType: "03",
	}
	if !reflect.DeepEqual(sdExpected, r.SourceData) {
		t.Errorf("want %v but have %v", sdExpected, r.SourceData)
	}
	pdExpected := PointSourceData{
		PlantID:          "1096",
		PointID:          "3159",
		StackID:          "31591096",
		Segment:          "EC",
		Plant:            "SECA_C3",
		ORISFacilityCode: "orisFacility",
		ORISBoilerID:     "orisBoiler",
		StackHeight:      badunit.Foot(65.62),
		StackDiameter:    badunit.Foot(2.625),
		StackTemp:        badunit.Fahrenheit(539.6),
		StackFlow:        unit.New(12.56935955809148, unit.Meter3PerSecond),
		StackVelocity:    badunit.FootPerSecond(82.02),
		Point:            geom.Point{X: -75.49256413, Y: 39.364973204},
		SR:               longlat,
	}
	if !reflect.DeepEqual(pdExpected, r.PointSourceData) {
		t.Errorf("want %v but have %v", pdExpected, r.PointSourceData)
	}

	edExpected := EconomicData{
		SIC:   "SIC0",
		NAICS: "NAICS0",
	}
	if !reflect.DeepEqual(edExpected, r.EconomicData) {
		t.Errorf("want %v but have %v", edExpected, r.EconomicData)
	}

	cdExpected := ControlData{
		MACT: "MACT",
		CEff: 25,
		REff: 75,
	}
	if !reflect.DeepEqual(cdExpected, r.ControlData) {
		t.Errorf("want %v but have %v", cdExpected, r.ControlData)
	}

	emis := r.PeriodTotals(start, end)
	emisExpected := map[Pollutant]*unit.Unit{
		Pollutant{Name: "NOX"}: unit.New(11683.010116385609, unit.Kilogram),
	}
	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}
}

func TestORLNonpoint(t *testing.T) {
	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}

	rec := strings.Split(`"01001","2102002000","SIC","0107-1","02","NAICS","CO",0.01,,33,66,99,,,"S-02-X","2002","000","SCC-D","03","20020101","20021231",25,25,25,25,6,0,0,0,0,0,0,0,,,,`, ",")

	rI, err := NewORLNonpoint(rec, USA, start, end, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r := rI.(*PolygonRecord)

	sdExpected := SourceData{
		FIPS:       "01001",
		SCC:        "2102002000",
		Country:    USA,
		SourceType: "02",
	}
	if !reflect.DeepEqual(sdExpected, r.SourceData) {
		t.Errorf("want %v but have %v", sdExpected, r.SourceData)
	}

	edExpected := EconomicData{
		SIC:   "SIC0",
		NAICS: "NAICS0",
	}
	if !reflect.DeepEqual(edExpected, r.EconomicData) {
		t.Errorf("want %v but have %v", edExpected, r.EconomicData)
	}

	cdExpected := ControlData{
		MACT: "0107-1",
		CEff: 33,
		REff: 66,
		RPen: 99,
	}
	if !reflect.DeepEqual(cdExpected, r.ControlData) {
		t.Errorf("want %v but have %v", cdExpected, r.ControlData)
	}

	emis := r.PeriodTotals(start, end)
	emisExpected := map[Pollutant]*unit.Unit{
		Pollutant{Name: "CO"}: badunit.Ton(0.01),
	}
	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}
}

func TestORLNonroad(t *testing.T) {
	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}

	rec := strings.Split(`"06067","2270002003","EXH__VOC",0,0.025497026700000001,10,20,30,"03","S","2005","000",,,,,,,,,,,,,,,,,,`, ",")

	rI, err := NewORLNonroad(rec, USA, start, end, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r := rI.(*nobusinessPolygonRecord)

	sdExpected := SourceData{
		FIPS:       "06067",
		SCC:        "2270002003",
		Country:    USA,
		SourceType: "03",
	}
	if !reflect.DeepEqual(sdExpected, r.SourceData) {
		t.Errorf("want %v but have %v", sdExpected, r.SourceData)
	}

	cdExpected := ControlData{
		MACT: "",
		CEff: 10,
		REff: 20,
		RPen: 30,
	}
	if !reflect.DeepEqual(cdExpected, r.ControlData) {
		t.Errorf("want %v but have %v", cdExpected, r.ControlData)
	}

	emis := r.PeriodTotals(start, end)
	emisExpected := map[Pollutant]*unit.Unit{
		Pollutant{Name: "VOC", Prefix: "EXH"}: badunit.Ton(0.025497026700000001 * 365),
	}
	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}
}

func TestORLMobile(t *testing.T) {
	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}

	rec := strings.Split(`"01001","2201001110","NAPHTH_72",,0.00022282850000000001,"04","E","2005",,10,20,30,,,,`, ",")

	rI, err := NewORLMobile(rec, USA, start, end, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r := rI.(*nobusinessPolygonRecord)

	sdExpected := SourceData{
		FIPS:       "01001",
		SCC:        "2201001110",
		Country:    USA,
		SourceType: "04",
	}
	if !reflect.DeepEqual(sdExpected, r.SourceData) {
		t.Errorf("want %v but have %v", sdExpected, r.SourceData)
	}

	cdExpected := ControlData{
		MACT: "",
		CEff: 10,
		REff: 20,
		RPen: 30,
	}
	if !reflect.DeepEqual(cdExpected, r.ControlData) {
		t.Errorf("want %v but have %v", cdExpected, r.ControlData)
	}

	emis := r.PeriodTotals(start, end)
	emisExpected := map[Pollutant]*unit.Unit{
		//"NAPHTH_72": badunit.Ton(0.00022282850000000001 * 365),
		Pollutant{Name: "NAPHTH_72"}: unit.New(73.78353556196251, unit.Kilogram),
	}
	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}
}
