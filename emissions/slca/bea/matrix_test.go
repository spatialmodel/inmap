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

package bea

import (
	"fmt"
	"reflect"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestMatrixFromExcel(t *testing.T) {
	if testing.Short() {
		return
	}
	t.Parallel()

	e := new(EIO)
	m, err := e.matrixFromExcel("data/IxC_TR_1997-2015_Summary.xlsx", "2009", 7, 78, 2, 75)
	if err != nil {
		t.Fatal(err)
	}
	r, c := m.Dims()
	if r != 71 || c != 73 {
		t.Errorf("(r,c) should be (71,73) but is (%d,%d)", r, c)
	}
	s := mat.Sum(m)
	wantSum := 135.4597329 // Sum within Excel.
	if different(s, wantSum) {
		t.Errorf("sum should be %g but is %g", wantSum, s)
	}
}

func TestTotalRequirementsSummary(t *testing.T) {
	if testing.Short() {
		return
	}
	t.Parallel()

	e := new(EIO)
	m, err := e.totalRequirementsSummary("data/IxC_TR_1997-2015_Summary.xlsx", 2009)
	if err != nil {
		t.Fatal(err)
	}
	r, c := m.Dims()
	if r != 71 || c != 73 {
		t.Errorf("(r,c) should be (71,73) but is (%d,%d)", r, c)
	}
	s := mat.Sum(m)
	wantSum := 135.4597329 // Sum within Excel.
	if different(s, wantSum) {
		t.Errorf("sum should be %g but is %g", wantSum, s)
	}
}

func TestTotalRequirementsDetail(t *testing.T) {
	if testing.Short() {
		return
	}
	t.Parallel()

	e := new(EIO)
	m, err := e.totalRequirementsDetail("data/IxC_TR_2007_Detail.xlsx", 2007)
	if err != nil {
		t.Fatal(err)
	}
	r, c := m.Dims()
	if r != 389 || c != 389 {
		t.Errorf("(r,c) should be (389,389) but is (%d,%d)", r, c)
	}
	s := mat.Sum(m)
	wantSum := 852.2414756 // Sum within Excel.
	if different(s, wantSum) {
		t.Errorf("sum should be %g but is %g", wantSum, s)
	}
}

func TestTextColumnFromExcel(t *testing.T) {
	if testing.Short() {
		return
	}
	t.Parallel()

	e := new(EIO)
	s, err := e.textColumnFromExcel("data/IxC_TR_1997-2015_Summary.xlsx", "2009", 1, 7, 11)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Farms",
		"Forestry, fishing, and related activities",
		"Oil and gas extraction",
		"Mining, except oil and gas",
	}
	if !reflect.DeepEqual(s, want) {
		t.Errorf("have %v but want %v", s, want)
	}
}

func TestTextRowFromExcel(t *testing.T) {
	if testing.Short() {
		return
	}
	t.Parallel()

	e := new(EIO)
	s, err := e.textRowFromExcel("data/IxC_TR_1997-2015_Summary.xlsx", "2009", 6, 2, 6)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Farms",
		"Forestry, fishing, and related activities",
		"Oil and gas extraction",
		"Mining, except oil and gas",
	}
	if !reflect.DeepEqual(s, want) {
		t.Errorf("have %v but want %v", s, want)
	}
}

func TestReadIndustriesAndCommodities(t *testing.T) {
	if testing.Short() {
		return
	}
	t.Parallel()

	e := new(EIO)
	tests := []struct {
		name          string
		numWant       int
		wantCodeFirst string
		wantDescFirst string
		wantCodeLast  string
		wantDescLast  string
		f             func(string) ([]string, []string, error)
		fileName      string
	}{
		{
			name:          "industriesSummary",
			numWant:       71,
			wantCodeFirst: "111CA",
			wantDescFirst: "Farms",
			wantCodeLast:  "GSLE",
			wantDescLast:  "State and local government enterprises",
			f:             e.industriesSummary,
			fileName:      "data/IxC_TR_1997-2015_Summary.xlsx",
		},
		{
			name:          "industriesDetail",
			numWant:       389,
			wantCodeFirst: "1111A0",
			wantDescFirst: "Oilseed farming",
			wantCodeLast:  "S00203",
			wantDescLast:  "Other state and local government enterprises",
			f:             e.industriesDetail,
			fileName:      "data/IxC_Domestic_2007_Detail.xlsx",
		},
		{
			name:          "commoditiesSummary",
			numWant:       73,
			wantCodeFirst: "111CA",
			wantDescFirst: "Farms",
			wantCodeLast:  "Other",
			wantDescLast:  "Noncomparable imports and rest-of-the-world adjustment",
			f:             e.commoditiesSummary,
			fileName:      "data/IxC_TR_1997-2015_Summary.xlsx",
		},
		{
			name:          "commoditiesDetail",
			numWant:       389,
			wantCodeFirst: "1111A0",
			wantDescFirst: "Oilseed farming",
			wantCodeLast:  "S00900",
			wantDescLast:  "Rest of the world adjustment",
			f:             e.commoditiesDetail,
			fileName:      "data/IxC_Domestic_2007_Detail.xlsx",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			codes, descriptions, err := test.f(test.fileName)
			if err != nil {
				t.Fatal(err)
			}
			if len(codes) != test.numWant || len(descriptions) != test.numWant {
				t.Fatalf("number of codes (%d) or descriptions (%d) != %d", len(codes), len(descriptions), test.numWant)
			}
			if codes[0] != test.wantCodeFirst {
				t.Errorf("want %s, have %s", test.wantCodeFirst, codes[0])
			}
			if descriptions[0] != test.wantDescFirst {
				t.Errorf("want %s, have %s", test.wantDescFirst, descriptions[0])
			}

			if codes[len(codes)-1] != test.wantCodeLast {
				t.Errorf("want %s, have %s", test.wantCodeLast, codes[len(codes)-1])
			}
			if descriptions[len(codes)-1] != test.wantDescLast {
				t.Errorf("want %s, have %s", test.wantDescLast, descriptions[len(codes)-1])
			}
		})
	}
}

func TestCodeCrosswalk(t *testing.T) {
	if testing.Short() {
		return
	}
	t.Parallel()

	e := new(EIO)
	x, err := e.codeCrosswalk("data/IxC_TR_1997-2015_Summary.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	codes111CA := []string{"1111A0", "1111B0", "111200", "111300", "111400",
		"111900", "1121A0", "112120", "112A00", "112300"}
	codesOther := []string{"S00300", "S00900"}
	if !reflect.DeepEqual(x["111CA"], codes111CA) {
		t.Errorf("111CA: have %v, want %v", x["111CA"], codes111CA)
	}
	if !reflect.DeepEqual(x["Other"], codesOther) {
		t.Errorf("Other: have %v, want %v", x["Other"], codesOther)
	}
}

func TestExpandMatrix(t *testing.T) {
	if testing.Short() {
		return
	}
	t.Parallel()

	m := mat.NewDense(2, 2, []float64{0, 1, 2, 3})
	rowCodes := []string{"a", "b"}
	colCodes := []string{"c", "d"}
	expandedRowCodes := []string{"a1", "a2", "b1", "b2"}
	expandedColCodes := []string{"c1", "c2", "d1", "d2"}
	codeCrosswalk := map[string][]string{
		"a": []string{"a1", "a2"},
		"b": []string{"b1", "b2"},
		"c": []string{"c1", "c2"},
		"d": []string{"d1", "d2"},
	}
	o := expandMatrix(m, rowCodes, colCodes, expandedRowCodes, expandedColCodes, codeCrosswalk)

	want := `⎡0  0  1  1⎤
⎢0  0  1  1⎥
⎢2  2  3  3⎥
⎣2  2  3  3⎦`

	fo := mat.Formatted(o, mat.Prefix(""), mat.Squeeze())
	have := fmt.Sprintf("%v", fo)

	if want != have {
		t.Errorf("want:\n%s\nhave:\n%s", want, have)
	}
}

func TestTotalRequirementsAdjusted(t *testing.T) {
	if testing.Short() {
		return
	}
	t.Parallel()

	e := new(EIO)
	m, err := e.totalRequirementsAdjusted("data/IxC_TR_2007_Detail.xlsx", "data/IxC_TR_1997-2015_Summary.xlsx", 1997, 2007)
	if err != nil {
		t.Fatal(err)
	}
	r, c := m.Dims()
	if r != 389 || c != 389 {
		t.Fatalf("want r,c == (389,389) but have (%d,%d)", r, c)
	}
	const (
		// 2007 detail value for Other state and local government enterprises,
		// Rest of the world adjustment
		lowerRightDetail2007 = 0.0031446

		// 2007 summary value for State and local government enterprises,
		// Noncomparable imports and rest-of-the-world adjustment
		lowerRightSummary2007 = 0.0038666

		// 1997 summary value for State and local government enterprises,
		// Noncomparable imports and rest-of-the-world adjustment
		lowerRightSummary1997 = 0.0039846

		lowerRightWant = lowerRightDetail2007 * lowerRightSummary1997 / lowerRightSummary2007
	)
	lowerRightHave := m.At(388, 388)
	if different(lowerRightWant, lowerRightHave) {
		t.Errorf("want %g but have %g", lowerRightWant, lowerRightHave)
	}
}
