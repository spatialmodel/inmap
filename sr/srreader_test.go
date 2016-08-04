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
			d: []float64{7.361675852246563e-11, 7.25222104591694e-11, 5.275072739729936e-11,
				6.637591865032988e-11, 7.456263384497674e-11, 6.623934040161927e-12, 4.146669466131847e-11,
				4.1918510329530534e-11, 1.3805411674949752e-11, 1.7925821417519305e-12},
			// This is a little wierd that primary PM2.5 leads to lower concentrations than
			// the other species.
		},
		{ // SOx
			d: []float64{2.7906648103481757e-09, 2.7757851572829395e-09, 2.0536543576810118e-09,
				2.5344293330675782e-09, 2.866508141963209e-09, 6.171398947429907e-10, 1.6097677635329433e-09,
				1.6469000607699513e-09, 3.6538386205542395e-10, 1.8184596883852322e-10},
		},
		{ // NH3
			d: []float64{1.2660957615651114e-08, 1.2591940823369896e-08, 9.314028659446194e-09,
				1.149738615424667e-08, 1.3002747323298536e-08, 2.3481301347771932e-09, 7.301005133797389e-09,
				7.46819761587858e-09, 2.7152833315113867e-09, 1.0744598544221162e-09},
		},
		{ // NOx
			d: []float64{7.73883181929419e-13, 1.123560771330856e-12, 1.2631224191594903e-12,
				9.923681884918545e-13, 1.3749463807083417e-12, 4.5181618320809525e-13, 1.0186144462631663e-12,
				1.2619373861849636e-12, 5.59143931242595e-13, 2.5981238102774917e-13},
		},
		{ // VOC
			d: []float64{2.983310265491923e-09, 6.054876600103398e-09, 1.578298935989153e-09,
				4.0705465664814255e-09, 2.8475652946724495e-09, 8.881933383220186e-11, 7.512100941298172e-10,
				6.246204109494613e-10, 1.406614685839358e-10, 1.7004517585683665e-11},
		},
		{ // PM25 100m
			d: []float64{6.961952239471292e-09, 6.924001586276673e-09, 5.121557519786114e-09,
				6.322132631237578e-09, 7.149894075532943e-09, 1.2911795731922923e-09, 4.014644909544477e-09,
				4.106579983524113e-09, 1.4930681741834906e-09, 5.90819305837425e-10},
		},
		{ // PM25 200m
			d: []float64{2.7906648103481757e-09, 2.7757851572829395e-09, 2.0536543576810118e-09,
				2.5344293330675782e-09, 2.866508141963209e-09, 6.171398947429907e-10, 1.6097677635329433e-09,
				1.6469000607699513e-09, 3.6538386205542395e-10, 1.8184596883852322e-10},
		},
		{ // PM25 400m
			d:   []float64(nil),
			err: fmt.Errorf("plume height (400 m) is above the top layer in the SR matrix"),
		},
	}

	for i, ee := range e {
		c, err := sr.Concentrations(&ee)
		fmt.Printf("%#v,\n", c)
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
