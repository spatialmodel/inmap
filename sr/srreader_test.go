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
			d: []float64{7.532266477205951e-11, 7.651135280895005e-11, 5.961255100661589e-11, 7.069076574106603e-11,
				8.60852084061392e-11, 7.538925213268488e-12, 4.855054205887832e-11, 5.725754939622796e-11,
				1.5217405460732358e-11, 2.237726497156256e-12},
			// This is a little wierd that primary PM2.5 leads to lower concentrations than
			// the other species.
		},
		{ // SOx
			d: []float64{2.859668279953098e-09, 2.9352951180783293e-09, 2.3388406766855496e-09,
				2.706592061585411e-09, 3.3283915623627536e-09, 7.088475362238e-10, 1.8985830685380733e-09,
				2.27681051789829e-09, 4.0662814781988743e-10, 2.2945549627007011e-10},
		},
		{ // NH3
			d: []float64{1.2973772278712659e-08, 1.3315153424287018e-08, 1.0606401978918711e-08,
				1.2277978633790099e-08, 1.5096759398147697e-08, 2.6964943611318404e-09,
				8.610092194771823e-09, 1.032253660268873e-08, 3.0211519952416666e-09, 1.3552794442261984e-09},
		},
		{ // NOx
			d: []float64{8.808173916803608e-13, 1.289281398655917e-12, 1.6517932854842599e-12,
				1.1649919310491308e-12, 1.825787134326906e-12, 5.900409826530006e-13, 1.3727305967284331e-12,
				2.0541151245226e-12, 7.06894612290121e-13, 3.699002909529625e-13},
		},
		{ // VOC
			d: []float64{9.522180377530276e-09, 6.3481002676724074e-09, 1.634761881419422e-09,
				4.223379423962115e-09, 3.080650623488168e-09, 7.221637871035469e-11, 8.101786463932115e-10,
				6.63807397938143e-10, 1.1118646281449429e-10, 1.4294213382393117e-11},
		},
		{ // PM25 100m
			d: []float64{7.133961404192659e-09, 7.321678581920725e-09, 5.832202132845418e-09,
				6.751361424670542e-09, 8.301340316530909e-09, 1.4827365769708483e-09, 4.734480001998119e-09,
				5.67611147622156e-09, 1.6612579030400784e-09, 7.452351590037889e-10},
		},
		{ // PM25 200m
			d: []float64{2.859668279953098e-09, 2.9352951180783293e-09, 2.3388406766855496e-09,
				2.706592061585411e-09, 3.3283915623627536e-09, 7.088475362238e-10, 1.8985830685380733e-09,
				2.27681051789829e-09, 4.0662814781988743e-10, 2.2945549627007011e-10},
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
