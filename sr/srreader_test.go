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

package sr

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap"
)

func TestLayerFracs(t *testing.T) {
	r, err := os.Open("../inmap/testdata/testSR.ncf")
	if err != nil {
		t.Fatal(err)
	}
	sr, err := NewReader(r)
	if err != nil {
		t.Fatal(err)
	}
	layers, fracs, err := sr.layerFracs(sr.d.Cells()[10], 100)
	if err != nil {
		t.Fatal(err)
	}
	wantLayers := []int{0, 1}
	wantFracs := []float64{0.4501243546645219, 0.5498756453354781}

	if !reflect.DeepEqual(wantLayers, layers) {
		t.Errorf("layers: want %v but have %v", wantLayers, layers)
	}
	if !reflect.DeepEqual(wantFracs, fracs) {
		t.Errorf("fractions: want %v but have %v", wantFracs, fracs)
	}
}

func TestVariable(t *testing.T) {
	r, err := os.Open("../inmap/testdata/testSR.ncf")
	if err != nil {
		t.Fatal(err)
	}
	sr, err := NewReader(r)
	if err != nil {
		t.Fatal(err)
	}
	vars, err := sr.Variables("TotalPop", "MortalityRate", "Baseline TotalPM25")
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]float64{
		"MortalityRate": []float64{0.0008000000013504179, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		"Baseline TotalPM25": []float64{4.907700538635254, 4.907700538635254, 4.907700538635254,
			4.907700538635254, 4.907700538635254, 10.347429275512695, 4.907700538635254, 4.907700538635254,
			4.25741720199585, 5.3623223304748535},
		"TotalPop": []float64{100000, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}

	if len(vars) != len(want) {
		t.Errorf("incorrect number of variables: %d != %d", len(vars), len(want))
	}

	for v, d := range want {
		if !reflect.DeepEqual(d, vars[v]) {
			t.Errorf("%s: want %v but have %v", v, d, vars[v])
		}
	}
}

func TestGeometry(t *testing.T) {
	r, err := os.Open("../inmap/testdata/testSR.ncf")
	if err != nil {
		t.Fatal(err)
	}
	sr, err := NewReader(r)
	if err != nil {
		t.Fatal(err)
	}
	g := sr.Geometry()

	want := []geom.Polygonal{
		geom.Polygon{[]geom.Point{
			geom.Point{X: -4000, Y: -4000},
			geom.Point{X: -4000, Y: -3000},
			geom.Point{X: -3000, Y: -3000},
			geom.Point{X: -3000, Y: -4000},
		}},
		geom.Polygon{[]geom.Point{
			geom.Point{X: -4000, Y: -3000},
			geom.Point{X: -4000, Y: -2000},
			geom.Point{X: -3000, Y: -2000},
			geom.Point{X: -3000, Y: -3000},
		}},
		geom.Polygon{[]geom.Point{
			geom.Point{X: -4000, Y: -2000},
			geom.Point{X: -4000, Y: 0},
			geom.Point{X: -2000, Y: 0},
			geom.Point{X: -2000, Y: -2000}}},
		geom.Polygon{[]geom.Point{
			geom.Point{X: -3000, Y: -4000},
			geom.Point{X: -3000, Y: -3000},
			geom.Point{X: -2000, Y: -3000},
			geom.Point{X: -2000, Y: -4000}}},
		geom.Polygon{[]geom.Point{
			geom.Point{X: -3000, Y: -3000},
			geom.Point{X: -3000, Y: -2000},
			geom.Point{X: -2000, Y: -2000},
			geom.Point{X: -2000, Y: -3000}}},
		geom.Polygon{[]geom.Point{
			geom.Point{X: -4000, Y: 0},
			geom.Point{X: -4000, Y: 4000},
			geom.Point{X: 0, Y: 4000},
			geom.Point{X: 0, Y: 0}}},
		geom.Polygon{[]geom.Point{
			geom.Point{X: -2000, Y: -4000},
			geom.Point{X: -2000, Y: -2000},
			geom.Point{X: 0, Y: -2000},
			geom.Point{X: 0, Y: -4000}}},
		geom.Polygon{[]geom.Point{
			geom.Point{X: -2000, Y: -2000},
			geom.Point{X: -2000, Y: 0},
			geom.Point{X: 0, Y: 0},
			geom.Point{X: 0, Y: -2000}}},
		geom.Polygon{[]geom.Point{
			geom.Point{X: 0, Y: -4000},
			geom.Point{X: 0, Y: 0},
			geom.Point{X: 4000, Y: 0},
			geom.Point{X: 4000, Y: -4000}}},
		geom.Polygon{[]geom.Point{
			geom.Point{X: 0, Y: 0},
			geom.Point{X: 0, Y: 4000},
			geom.Point{X: 4000, Y: 4000},
			geom.Point{X: 4000, Y: 0}}},
	}

	if !reflect.DeepEqual(want, g) {
		t.Errorf("geometry doesn't match")
	}
}

func TestConcentrations(t *testing.T) {
	r, err := os.Open("../inmap/testdata/testSR.ncf")
	if err != nil {
		t.Fatal(err)
	}
	sr, err := NewReader(r)
	if err != nil {
		t.Fatal(err)
	}

	e := []inmap.EmisRecord{
		{
			Geom: geom.Point{X: -3500, Y: -3500},
			PM25: 1,
		},
		{
			Geom: geom.Point{X: -3500, Y: -3500},
			SOx:  1,
		},
		{
			Geom: geom.Point{X: -3500, Y: -3500},
			NH3:  1,
		},
		{
			Geom: geom.Point{X: -3500, Y: -3500},
			NOx:  1,
		},
		{
			Geom: geom.Point{X: -3500, Y: -3500},
			VOC:  1,
		},
		{
			Geom:   geom.Point{X: -3500, Y: -3500},
			PM25:   1,
			Height: 100,
		},
		{
			Geom:   geom.Point{X: -3500, Y: -3500},
			PM25:   1,
			Height: 200,
		},
		{
			Geom:   geom.Point{X: -3500, Y: -3500},
			PM25:   1,
			Height: 800,
		},
	}

	type result struct {
		d   []float64
		err error
	}

	want := []result{
		{ // PM25
			d: []float64{2.058471864074818e-06, 1.0526232472329866e-06, 2.935857708052936e-07,
				7.094353691172728e-07, 5.076742581877625e-07, 3.374343648943068e-08, 1.4487866906165436e-07,
				1.2993052678211825e-07, 2.987471603432823e-08, 1.0531014282832984e-08},
		},
		{ // SOx
			d: []float64{3.324798075743374e-11, 3.083564306671782e-11, 2.0613025175642008e-11,
				2.2215146389115148e-11, 2.5215158086511202e-11, 4.2723537381494925e-12, 1.2055437377533895e-11,
				1.49673884397572e-11, 4.563417786013213e-12, 2.1256640111294578e-12},
		},
		{ // NH3
			d: []float64{4.5370552470558323e-07, 2.3203244836622616e-07, 6.474122216104661e-08,
				1.5638546813079302e-07, 1.1192980764462845e-07, 8.871220202877339e-09, 3.195172126879697e-08,
				2.8664461027005927e-08, 4.0223486763579785e-09, 1.7840007249247947e-09},
		},
		{ // NOx
			d: []float64{1.0427964269865697e-07, 5.3347715578411226e-08, 1.4901701206326834e-08,
				3.5957121724550234e-08, 2.5748775556166947e-08, 1.3573112633835649e-09, 7.356100617528227e-09,
				6.604421010081296e-09, 3.6174624407081524e-10, 2.9599611650610314e-10},
		},
		{ // VOC
			d: []float64{1.200379973198551e-08, 6.081353198794659e-09, 1.641341618174863e-09,
				4.092715055747931e-09, 2.885406802377588e-09, 9.167426540113155e-11, 8.040539345444131e-10,
				7.036943250326999e-10, 1.4490914024278823e-10, 1.6441159197966293e-11},
		},
		{ // PM25 100m
			d: []float64{5.611809754041488e-09, 5.65484465381776e-09, 4.6655953143699175e-09,
				5.147377787511767e-09, 5.912693070554996e-09, 1.3921809138359893e-09, 3.936972981057887e-09,
				4.512377092445574e-09, 1.6007739160615037e-09, 6.993812771173818e-10},
		},
		{ // PM25 200m
			d: []float64{2.737697180066334e-09, 2.7530715485113433e-09, 2.2632349327977863e-09,
				2.5054136543189998e-09, 2.874128046670421e-09, 7.997795758996062e-10, 1.907466185002704e-09,
				2.1818344908552945e-09, 4.712191747913153e-10, 2.5783555845926287e-10},
		},
		{ // PM25 800m
			d:   []float64(nil),
			err: fmt.Errorf("plume height (800 m) is above the top layer in the SR matrix"),
		},
	}

	for i, ee := range e {
		c, err := sr.Concentrations(&ee)
		if err != nil {
			if want[i].err == nil {
				t.Errorf("test %d: %v", i, err)
			} else if err.Error() != want[i].err.Error() {
				t.Errorf("test %d error: want %v, have %v", i, want[i].err, err)
			}
		}
		if !reflect.DeepEqual(want[i].d, c) {
			for j, v := range c {
				w := want[i].d[j]
				if math.Abs(w-v)*2/(w+v) > 1.e-8 {
					t.Errorf("test %d, row %d: want %v but have %v", i, j, w, v)
				}
			}
		}
	}
}
