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
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap"
)

func TestLayerFracs(t *testing.T) {
	r, err := os.Open("../cmd/inmap/testdata/testSR_golden.ncf")
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
	r, err := os.Open("../cmd/inmap/testdata/testSR_golden.ncf")
	if err != nil {
		t.Fatal(err)
	}
	sr, err := NewReader(r)
	if err != nil {
		t.Fatal(err)
	}
	vars, err := sr.Variables("TotalPop", "allcause", "BaselineTotalPM25")
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]float64{
		"allcause": {800, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		"BaselineTotalPM25": {4.907700538635254, 4.907700538635254, 4.907700538635254,
			4.907700538635254, 4.907700538635254, 10.347429275512695, 4.907700538635254, 4.907700538635254,
			4.25741720199585, 5.3623223304748535},
		"TotalPop": {100000, 0, 0, 0, 0, 0, 0, 0, 0, 0},
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
	r, err := os.Open("../cmd/inmap/testdata/testSR_golden.ncf")
	if err != nil {
		t.Fatal(err)
	}
	sr, err := NewReader(r)
	if err != nil {
		t.Fatal(err)
	}
	g := sr.Geometry()

	want := []geom.Polygonal{geom.Polygon{
		[]geom.Point{
			{X: -4000, Y: -4000},
			{X: -3000, Y: -4000},
			{X: -3000, Y: -3000},
			{X: -4000, Y: -3000},
			{X: -4000, Y: -4000},
		}},
		geom.Polygon{[]geom.Point{
			{X: -4000, Y: -3000},
			{X: -3000, Y: -3000},
			{X: -3000, Y: -2000},
			{X: -4000, Y: -2000},
			{X: -4000, Y: -3000},
		}},
		geom.Polygon{[]geom.Point{
			{X: -4000, Y: -2000},
			{X: -2000, Y: -2000},
			{X: -2000, Y: 0},
			{X: -4000, Y: 0},
			{X: -4000, Y: -2000},
		}},
		geom.Polygon{[]geom.Point{
			{X: -3000, Y: -4000},
			{X: -2000, Y: -4000},
			{X: -2000, Y: -3000},
			{X: -3000, Y: -3000},
			{X: -3000, Y: -4000},
		}},
		geom.Polygon{[]geom.Point{
			{X: -3000, Y: -3000},
			{X: -2000, Y: -3000},
			{X: -2000, Y: -2000},
			{X: -3000, Y: -2000},
			{X: -3000, Y: -3000},
		}},
		geom.Polygon{[]geom.Point{
			{X: -4000, Y: 0},
			{X: 0, Y: 0},
			{X: 0, Y: 4000},
			{X: -4000, Y: 4000},
			{X: -4000, Y: 0},
		}},
		geom.Polygon{[]geom.Point{
			{X: -2000, Y: -4000},
			{X: 0, Y: -4000},
			{X: 0, Y: -2000},
			{X: -2000, Y: -2000},
			{X: -2000, Y: -4000},
		}},
		geom.Polygon{[]geom.Point{
			{X: -2000, Y: -2000},
			{X: 0, Y: -2000},
			{X: 0, Y: 0},
			{X: -2000, Y: 0},
			{X: -2000, Y: -2000},
		}},
		geom.Polygon{[]geom.Point{
			{X: 0, Y: -4000},
			{X: 4000, Y: -4000},
			{X: 4000, Y: 0},
			{X: 0, Y: 0},
			{X: 0, Y: -4000},
		}},
		geom.Polygon{[]geom.Point{
			{X: 0, Y: 0},
			{X: 4000, Y: 0},
			{X: 4000, Y: 4000},
			{X: 0, Y: 4000},
			{X: 0, Y: 0},
		}},
	}

	if !reflect.DeepEqual(want, g) {
		t.Errorf("geometry doesn't match")
	}
}

func TestSR_Source(t *testing.T) {
	r, err := os.Open("../cmd/inmap/testdata/testSR_golden.ncf")
	if err != nil {
		t.Fatal(err)
	}
	sr, err := NewReader(r)
	if err != nil {
		t.Fatal(err)
	}

	_, err = sr.Source("xxxx", 0, 0)
	if err == nil {
		t.Errorf("should have an error")
	}
	if !strings.Contains(err.Error(), "valid pollutant") {
		t.Errorf("error should be about invalid pollutant")
	}

	_, err = sr.Source("PrimaryPM25", 0, 1000000000)
	if err == nil {
		t.Errorf("should have an error")
	}
	if !strings.Contains(err.Error(), "grid cells") {
		t.Errorf("error should be about too many grid cells")
	}

	_, err = sr.Source("PrimaryPM25", 100, 0)
	if err == nil {
		t.Errorf("should have an error")
	}
	if !strings.Contains(err.Error(), "layer") {
		t.Errorf("error should be about too many layers")
	}
}

func TestConcentrations(t *testing.T) {
	r, err := os.Open("../cmd/inmap/testdata/testSR_golden.ncf")
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
			d: []float64{2.29174747801153e-06, 1.0526232472329866e-06, 2.935857708052936e-07,
				7.094353691172728e-07, 5.076742581877625e-07, 3.374343648943068e-08, 1.4487866906165436e-07,
				1.2993052678211825e-07, 2.987471603432823e-08, 1.0531014282832984e-08},
		},
		{ // SOx
			d: []float64{3.6036743034095764e-11, 3.181685123698763e-11, 2.0886658722019114e-11,
				2.2876445529562695e-11, 2.5688359425735108e-11, 4.303799070598524e-12, 1.219046391609524e-11,
				1.5088473873103858e-11, 4.591254893632213e-12, 2.135475390269148e-12},
		},
		{ // NH3
			d: []float64{5.051080051998724e-07, 2.3203187993203755e-07, 6.474105873621738e-08, 1.56385070226861e-07,
				1.1192953053296151e-07, 8.871197998416847e-09, 3.195164310909604e-08, 2.866438997273235e-08,
				4.022338462306152e-09, 1.7839962840326962e-09},
		},
		{ // NOx
			d: []float64{1.160867242333552e-07, 5.334764097142397e-08, 1.490167989004476e-08,
				3.595706843384505e-08, 2.574873825267332e-08, 1.3573092649821206e-09, 7.356089959387191e-09,
				6.6044112401186794e-09, 3.6174571671487854e-10, 2.959957001724689e-10},
		},
		{ // VOC
			d: []float64{1.338964050745517e-08, 6.081382952771719e-09, 1.6413496117806403e-09,
				4.092734595673164e-09, 2.8854207911876983e-09, 9.16747094903414e-11, 8.040578203249993e-10,
				7.03697711212925e-10, 1.4490984800996642e-10, 1.6441238995246188e-11},
		},
		{ // PM25 100m
			d: []float64{1.0384010211692146e-06, 4.806783900520519e-07, 1.3779393223094143e-07,
				3.255835884765454e-07, 2.3568499479334203e-07, 1.6861865123153858e-08, 6.997017131502968e-08,
				6.392452770298667e-08, 1.537270302735229e-08, 5.577751434206565e-09},
		},
		{ // PM25 200m
			d: []float64{1.2420383832534299e-08, 1.2488333034355037e-08, 1.0263823568834596e-08,
				1.1365207441826897e-08, 1.3036487445106104e-08, 3.0427289576806515e-09,
				8.650599347959087e-09, 9.892478836093233e-09, 3.5014566801550018e-09, 1.5230451344550033e-09},
		},
		{ // PM25 800m
			d: []float64{3.835866693724377e-11, 6.099889343635923e-11, 8.292484754424123e-11,
				5.750595138964698e-11, 8.120485534002242e-11, 4.1053837412929894e-11, 7.843142801888803e-11,
				1.0826346763526118e-10, 4.48883083481455e-11, 2.5135810099996547e-11},
			err: AboveTopErr{PlumeHeight: 800},
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
			for j, v := range c.TotalPM25() {
				w := want[i].d[j]
				if math.Abs(w-v)*2/(w+v) > 1.e-8 {
					t.Errorf("test %d, row %d: want %v but have %v", i, j, w, v)
				}
			}
		}
	}
}

func BenchmarkConcentrations(b *testing.B) {
	r, err := os.Open("../cmd/inmap/testdata/testSR_golden.ncf")
	if err != nil {
		b.Fatal(err)
	}
	sr, err := NewReader(r)
	if err != nil {
		b.Fatal(err)
	}

	for _, n := range []int{10, 100, 1000, 10000, 100000} {
		r := make([]*inmap.EmisRecord, n)
		for i := 0; i < n; i++ {
			r[i] = &inmap.EmisRecord{
				Geom:   geom.Point{X: rand.Float64()*7000 - 3500, Y: rand.Float64()*7000 - 3500},
				PM25:   1,
				NOx:    1,
				NH3:    1,
				SOx:    1,
				VOC:    1,
				Height: rand.Float64() * 400,
			}
		}
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			_, err := sr.Concentrations(r...)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}
