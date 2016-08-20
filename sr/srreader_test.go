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
	vars, err := sr.Variables("TotalPop", "MortalityRate", "Baseline Total PM2.5")
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]float64{
		"MortalityRate": []float64{0.0008000000013504179, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		"Baseline Total PM2.5": []float64{4.907700538635254, 4.907700538635254, 4.907700538635254,
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
			d: []float64{2.055840241155238e-06, 1.047539171850076e-06, 2.8155326958767546e-07,
				7.051808097457979e-07, 5.003935825698136e-07, 3.25464419859145e-08, 1.3483719385476434e-07,
				1.1449543535491102e-07, 2.8833353482582424e-08, 1.0711961984100071e-08},
		},
		{ // SOx
			d: []float64{3.282439250962277e-11, 2.9905126985863006e-11, 1.868233866220148e-11,
				2.1415506520905403e-11, 2.3892030714955936e-11, 3.853131789327557e-12,
				1.0379193164655742e-11, 1.1868565158446032e-11, 4.117327138952742e-12, 1.8555107537260307e-12},
		},
		{ // NH3
			d: []float64{4.531248976036295e-07, 2.3091041612133267e-07, 6.208613001490448e-08,
				1.5544650011634076e-07, 1.1032302182911735e-07, 8.556011010796283e-09,
				2.973595414346164e-08, 2.5256547075969138e-08, 3.881791776905175e-09, 1.8139122426319432e-09},
		},
		{ // NOx
			d: []float64{1.0414573381467562e-07, 5.3088761831077136e-08, 1.4289192939997974e-08,
				3.5740434611852834e-08, 2.5377987711294736e-08, 1.308883668116323e-09, 6.845049416170923e-09,
				5.817904824567677e-09, 3.490404354433707e-10, 3.0084079671865993e-10},
		},
		{ // VOC
			d: []float64{1.198987575889987e-08, 6.055075107980201e-09, 1.5783568896310385e-09,
				4.07068112551201e-09, 2.847663882477036e-09, 8.88231224682734e-11, 7.512397925957259e-10,
				6.246475559024134e-10, 1.406678662441152e-10, 1.7005208005627104e-11},
		},
		{ // PM25 100m
			d: []float64{5.763648378851758e-09, 5.823459944267615e-09, 4.280630652451855e-09,
				5.3620176265565675e-09, 6.437342548817897e-09, 1.0742635559344885e-09, 3.3860971494488653e-09,
				3.6380242796782233e-09, 1.2110653930769326e-09, 4.99774226355067e-10},
		},
		{ // PM25 200m
			d: []float64{2.8128233076074594e-09, 2.8373845495366368e-09, 2.078277994144173e-09,
				2.612094984755231e-09, 3.1337343830983855e-09, 6.175551181542005e-10, 1.6410512948539235e-09,
				1.7591890166812618e-09, 3.563900008440868e-10, 1.8423751413365608e-10},
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
