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

func TestFF10Point(t *testing.T) {
	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}

	periods, err := ff10Periods("2005")

	rec := strings.Split(`"US","01001",,"10583011","62385813","50346112","83296814",,,,,"2275050011","100414",0.0042677399999999999,50,"Autauga County","1",1,2,3,4,5,"48811",-86.5104500000000058,32.4387800000000013,"002",,,,,"100","300",,,,,,"USEPA",8,"2011EPA_Air","UNK","orisFacility","orisBoiler",,"2011",20130210,,,,,"00000",,,,,,,,,,,,,,,,,,,,,,,,,,`, ",")
	if err != nil {
		t.Error(err)
	}

	rI, err := NewFF10Point(rec, start, end, periods, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r := rI.(*PointRecord)

	sdExpected := SourceData{
		FIPS:       "01001",
		SCC:        "2275050011",
		Country:    USA,
		SourceType: "",
	}
	if !reflect.DeepEqual(sdExpected, r.SourceData) {
		t.Errorf("want %v but have %v", sdExpected, r.SourceData)
	}
	pdExpected := PointSourceData{
		PlantID:          "10583011",
		PointID:          "62385813",
		StackID:          "50346112",
		Segment:          "83296814",
		Plant:            "Autauga County",
		ORISFacilityCode: "orisFacility",
		ORISBoilerID:     "orisBoiler",
		StackHeight:      badunit.Foot(1),
		StackDiameter:    badunit.Foot(2),
		StackTemp:        badunit.Fahrenheit(3),
		StackFlow:        badunit.Foot3PerSecond(4),
		StackVelocity:    badunit.FootPerSecond(5),
		Point:            geom.Point{X: -86.5104500000000058, Y: 32.4387800000000013},
		SR:               nad83,
	}
	if !reflect.DeepEqual(pdExpected, r.PointSourceData) {
		t.Errorf("want %v but have %v", pdExpected, r.PointSourceData)
	}

	edExpected := EconomicData{
		SIC:   "",
		NAICS: "488110",
	}
	if !reflect.DeepEqual(edExpected, r.EconomicData) {
		t.Errorf("want %v but have %v", edExpected, r.EconomicData)
	}

	cdExpected := ControlData{
		MACT: "",
		REff: 50,
	}
	if !reflect.DeepEqual(cdExpected, r.ControlData) {
		t.Errorf("want %v but have %v", cdExpected, r.ControlData)
	}

	emis := r.PeriodTotals(start, end)
	emisExpected := map[Pollutant]*unit.Unit{
		//"100414": badunit.Ton(0.0042677399999999999),
		Pollutant{Name: "100414"}: unit.New(3.8716297118999994, unit.Kilogram),
	}
	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}

	// now try monthly emissions
	rec = strings.Split(`"US","01001",,"10583011","62385813","50346112","83296814",,,,,"2275050011","100414",0.0042677399999999999,50,"Autauga County","1",1,2,3,4,5,"48811",-86.5104500000000058,32.4387800000000013,"002",,,,,"100","300",,,,,,"USEPA",8,"2011EPA_Air","UNK","orisFacility","orisBoiler",,"2011",20130210,,,,,"00000",,1,1,1,1,1,1,1,1,1,1,1,1,,,,,,,,,,,,,`, ",")
	if err != nil {
		t.Error(err)
	}
	rI, err = NewFF10Point(rec, start, end, periods, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r = rI.(*PointRecord)
	emis = r.PeriodTotals(start, end)
	emisExpected = map[Pollutant]*unit.Unit{
		//"100414": badunit.Ton(12),
		Pollutant{Name: "100414"}: unit.New(10886.219999999996, unit.Kilogram),
	}
	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}
	emisString := `100414: 2005-01-01 00:00:00 +0000 UTC -- 2005-02-01 00:00:00 +0000 UTC: 0.00033870407706093186 kg s^-1
100414: 2005-02-01 00:00:00 +0000 UTC -- 2005-03-01 00:00:00 +0000 UTC: 0.0003749937996031746 kg s^-1
100414: 2005-03-01 00:00:00 +0000 UTC -- 2005-04-01 00:00:00 +0000 UTC: 0.00033870407706093186 kg s^-1
100414: 2005-04-01 00:00:00 +0000 UTC -- 2005-05-01 00:00:00 +0000 UTC: 0.0003499942129629629 kg s^-1
100414: 2005-05-01 00:00:00 +0000 UTC -- 2005-06-01 00:00:00 +0000 UTC: 0.00033870407706093186 kg s^-1
100414: 2005-06-01 00:00:00 +0000 UTC -- 2005-07-01 00:00:00 +0000 UTC: 0.0003499942129629629 kg s^-1
100414: 2005-07-01 00:00:00 +0000 UTC -- 2005-08-01 00:00:00 +0000 UTC: 0.00033870407706093186 kg s^-1
100414: 2005-08-01 00:00:00 +0000 UTC -- 2005-09-01 00:00:00 +0000 UTC: 0.00033870407706093186 kg s^-1
100414: 2005-09-01 00:00:00 +0000 UTC -- 2005-10-01 00:00:00 +0000 UTC: 0.0003499942129629629 kg s^-1
100414: 2005-10-01 00:00:00 +0000 UTC -- 2005-11-01 00:00:00 +0000 UTC: 0.00033870407706093186 kg s^-1
100414: 2005-11-01 00:00:00 +0000 UTC -- 2005-12-01 00:00:00 +0000 UTC: 0.0003499942129629629 kg s^-1
100414: 2005-12-01 00:00:00 +0000 UTC -- 2006-01-01 00:00:00 +0000 UTC: 0.00033870407706093186 kg s^-1`
	if !reflect.DeepEqual(emisString, r.Emissions.String()) {
		t.Errorf("want:\n%v\nbut have:\n%v", emisString, r.Emissions.String())
	}
}

func TestFF10DailyPoint(t *testing.T) {
	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}

	periods, err := ff10Periods("2005")

	rec := strings.Split(`"US","01033",,"7212811","10817213","10769412","61093014","20100101","NOX",,,,1,0.0307227831999999992,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,1,0,`, ",")
	if err != nil {
		t.Error(err)
	}

	rI, err := NewFF10DailyPoint(rec, start, end, periods, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r := rI.(*supplementalPointRecord)

	sdExpected := SourceData{
		FIPS:       "01033",
		SCC:        "0020100101",
		Country:    USA,
		SourceType: "",
	}
	if !reflect.DeepEqual(sdExpected, r.SourceData) {
		t.Errorf("want %v but have %v", sdExpected, r.SourceData)
	}
	pdExpected := PointSourceData{
		PlantID:          "7212811",
		PointID:          "10817213",
		StackID:          "10769412",
		Segment:          "61093014",
		Plant:            "",
		ORISFacilityCode: "",
		ORISBoilerID:     "",
	}
	if !reflect.DeepEqual(pdExpected, r.PointSourceData) {
		t.Errorf("want %v but have %v", pdExpected, r.PointSourceData)
	}

	emis := r.PeriodTotals(start, end)
	emisExpected := map[Pollutant]*unit.Unit{
		//"NOX": badunit.Ton(15),
		Pollutant{Name: "NOX"}: unit.New(13607.774999999994, unit.Kilogram),
	}
	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}

	emisString := `NOX: 2005-01-01 00:00:00 +0000 UTC -- 2005-01-02 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-02 00:00:00 +0000 UTC -- 2005-01-03 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-03 00:00:00 +0000 UTC -- 2005-01-04 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-04 00:00:00 +0000 UTC -- 2005-01-05 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-05 00:00:00 +0000 UTC -- 2005-01-06 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-06 00:00:00 +0000 UTC -- 2005-01-07 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-07 00:00:00 +0000 UTC -- 2005-01-08 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-08 00:00:00 +0000 UTC -- 2005-01-09 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-09 00:00:00 +0000 UTC -- 2005-01-10 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-10 00:00:00 +0000 UTC -- 2005-01-11 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-11 00:00:00 +0000 UTC -- 2005-01-12 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-12 00:00:00 +0000 UTC -- 2005-01-13 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-13 00:00:00 +0000 UTC -- 2005-01-14 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-14 00:00:00 +0000 UTC -- 2005-01-15 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-15 00:00:00 +0000 UTC -- 2005-01-16 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-16 00:00:00 +0000 UTC -- 2005-01-17 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-17 00:00:00 +0000 UTC -- 2005-01-18 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-18 00:00:00 +0000 UTC -- 2005-01-19 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-19 00:00:00 +0000 UTC -- 2005-01-20 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-20 00:00:00 +0000 UTC -- 2005-01-21 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-21 00:00:00 +0000 UTC -- 2005-01-22 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-22 00:00:00 +0000 UTC -- 2005-01-23 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-23 00:00:00 +0000 UTC -- 2005-01-24 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-24 00:00:00 +0000 UTC -- 2005-01-25 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-25 00:00:00 +0000 UTC -- 2005-01-26 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-26 00:00:00 +0000 UTC -- 2005-01-27 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-27 00:00:00 +0000 UTC -- 2005-01-28 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-28 00:00:00 +0000 UTC -- 2005-01-29 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-29 00:00:00 +0000 UTC -- 2005-01-30 00:00:00 +0000 UTC: 0 kg s^-1
NOX: 2005-01-30 00:00:00 +0000 UTC -- 2005-01-31 00:00:00 +0000 UTC: 0.010499826388888888 kg s^-1
NOX: 2005-01-31 00:00:00 +0000 UTC -- 2005-02-01 00:00:00 +0000 UTC: 0 kg s^-1`
	if !reflect.DeepEqual(emisString, r.Emissions.String()) {
		t.Errorf("want:\n%v\nbut have:\n%v", emisString, r.Emissions.String())
	}
}

func TestFF10DailyPointBlankRecord(t *testing.T) {
	rec := []string{"US", "36091", "", "7819511", "3429213", "58898412", "18074414", "10500206", "NOX", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", ""}

	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}

	periods, err := ff10Periods("2005")
	if err != nil {
		t.Error(err)
	}

	_, err = NewFF10DailyPoint(rec, start, end, periods, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
}

func TestFF10Nonpoint(t *testing.T) {
	start, end, err := Annual.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}

	periods, err := ff10Periods("2005")

	rec := strings.Split(`"US","54015",,,,"2102002000",,"92524",6.98020000000000053e-08,50,,,,,,"63DDDDD&63DDDDD&63DDDDD&63DDDDD",5,2011,20140828,"2011EPA_HAP-Aug",,,,,,,,,,,,,,,,,,,,,,,,,`, ",")
	if err != nil {
		t.Error(err)
	}

	rI, err := NewFF10Nonpoint(rec, start, end, periods, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r := rI.(*nobusinessPolygonRecord)

	sdExpected := SourceData{
		FIPS:       "54015",
		SCC:        "2102002000",
		Country:    USA,
		SourceType: "",
	}
	if !reflect.DeepEqual(sdExpected, r.SourceData) {
		t.Errorf("want %v but have %v", sdExpected, r.SourceData)
	}

	cdExpected := ControlData{
		MACT: "",
		REff: 50,
	}
	if !reflect.DeepEqual(cdExpected, r.ControlData) {
		t.Errorf("want %v but have %v", cdExpected, r.ControlData)
	}

	emis := r.PeriodTotals(start, end)
	emisExpected := map[Pollutant]*unit.Unit{
		Pollutant{Name: "92524"}: badunit.Ton(6.98020000000000053e-08),
		//"92524": unit.New(3.8716297118999994, unit.Kilogram),
	}
	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}

	// now try monthly emissions
	rec = strings.Split(`"US","54015",,,,"2102002000",,"92524",6.98020000000000053e-08,,,,,,,"63DDDDD&63DDDDD&63DDDDD&63DDDDD",5,2011,20140828,"2011EPA_HAP-Aug",1,1,1,1,1,1,1,1,1,1,1,1,,,,,,,,,,,,,`, ",")
	if err != nil {
		t.Error(err)
	}
	rI, err = NewFF10Nonpoint(rec, start, end, periods, badunit.Ton)
	if err != nil {
		t.Error(err)
	}
	r = rI.(*nobusinessPolygonRecord)
	emis = r.PeriodTotals(start, end)
	emisExpected = map[Pollutant]*unit.Unit{
		//"92524": badunit.Ton(12),
		Pollutant{Name: "92524"}: unit.New(10886.219999999996, unit.Kilogram),
	}
	if !reflect.DeepEqual(emisExpected, emis) {
		t.Errorf("want %v but have %v", emisExpected, emis)
	}
}
