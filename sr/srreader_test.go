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
			Height: 400,
		},
	}

	type result struct {
		d   []float64
		err error
	}

	want := []result{
		{ // PM25
			d: []float64{7.532260232201438e-11, 7.651122096996588e-11, 5.961225263417802e-11, 7.069064777986966e-11,
				8.608497248374647e-11, 7.538811588880812e-12, 4.8550285319803876e-11, 5.725706020420773e-11,
				1.521722851893781e-11, 2.2376779248989287e-12},
			// This is a little wierd that primary PM2.5 leads to lower concentrations than
			// the other species.
		},
		{ // SOx
			d: []float64{2.8596671697300735e-09, 2.935293119676885e-09, 2.3388357917042413e-09,
				2.7065900631839668e-09, 3.32838778760447e-09, 7.088425402201892e-10, 1.898578627645975e-09,
				2.2768020802033107e-09, 4.066259273738382e-10, 2.2945302602384032e-10},
		},
		{ // NH3
			d: []float64{1.297376872599898e-08, 1.3315145430681241e-08, 1.0606383327171898e-08,
				1.2277971528362741e-08, 1.509674518729298e-08, 2.696477485741866e-09,
				8.61007443120343e-09, 1.0322503740667344e-08, 3.0211373402977415e-09, 1.3552663435945078e-09},
		},
		{ // NOx
			d: []float64{9.020272593099343e-13, 1.311029084773152e-12, 1.669059855602395e-12,
				1.1850466354740297e-12, 1.8504180392814318e-12, 5.943917775509677e-13, 1.386743801387591e-12,
				2.070863012321178e-12, 7.117823583140115e-13, 3.72074034993633e-13},
		},
		{ // VOC
			d: []float64{1.136567107096198e-08, 6.348142012058133e-09, 1.6347734277388781e-09,
				4.223407401582335e-09, 3.080671273636426e-09, 7.221717668315364e-11, 8.101846971086957e-10,
				6.638132266090224e-10, 1.1118769793760919e-10, 1.429444063116847e-11},
		},
		{ // PM25 100m
			d: []float64{7.133959450641932e-09, 7.321674186431589e-09, 5.832191876704102e-09,
				6.751357517569088e-09, 8.301332502328e-09, 1.482727297604896e-09, 4.734470234244485e-09,
				5.676093405877342e-09, 1.6612498446433303e-09, 7.452279552854837e-10},
		},
		{ // PM25 200m
			d: []float64{2.8596671697300735e-09, 2.935293119676885e-09, 2.3388357917042413e-09,
				2.7065900631839668e-09, 3.32838778760447e-09, 7.088425402201892e-10, 1.898578627645975e-09,
				2.2768020802033107e-09, 4.066259273738382e-10, 2.2945302602384032e-10},
		},
		{ // PM25 400m
			d:   []float64(nil),
			err: fmt.Errorf("plume height (400 m) is above the top layer in the SR matrix"),
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
