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
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/proj"
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
			d: []float64{2.291749524374609e-06, 1.0526249525355524e-06, 2.9358676556512364e-07, 7.0943650598565e-07,
				5.076755087429774e-07, 3.374364254682405e-08, 1.4487922328498826e-07, 1.2993119469228986e-07,
				2.987489722272585e-08, 1.0531102212496535e-08},
		},
		{ // SOx
			d: []float64{8.102691434475062e-11, 7.153899000966746e-11, 4.6963259670018687e-11, 5.1436865183829283e-11,
				5.775954714515308e-11, 9.677107205841029e-12, 2.7410114109005512e-11, 3.392639641441875e-11,
				1.0323465172989987e-11, 4.801651264096929e-12},
		},
		{ // NH3
			d: []float64{5.051084031038044e-07, 2.320321925708413e-07, 6.47412576881834e-08, 1.5638531181139115e-07,
				1.1192977922291902e-07, 8.87124596005151e-09, 3.195175324322008e-08, 2.8664521423138467e-08,
				4.022360222677435e-09, 1.7840074972852449e-09},
		},
		{ // NOx
			d: []float64{1.1608678818220142e-07, 5.3347690709415474e-08, 1.490171186446787e-08, 3.595710396098184e-08,
				2.5748777332523787e-08, 1.3573143720080338e-09, 7.356107278866375e-09, 6.604432556400752e-09,
				3.617471044936593e-10, 2.9599717121797653e-10},
		},
		{ // VOC
			d: []float64{1.3390037523208775e-08, 6.081683601166787e-09, 1.6415258041746483e-09, 4.092947314404682e-09,
				2.885646832595512e-09, 9.169261877550738e-11, 8.041544652392929e-10, 7.038127303182762e-10,
				1.4493828359718464e-10, 1.6445405801035484e-11},
		},
		{ // PM25 100m
			d: []float64{1.0384019515664409e-06, 4.806791708367362e-07, 1.3779439660174915e-07, 3.255841114416068e-07,
				2.3568557381549462e-07, 1.6861964589935698e-08, 6.997043397091758e-08, 6.392484788112892e-08,
				1.5372791544187306e-08, 5.5777951037364604e-09},
		},
		{ // PM25 200m
			d: []float64{1.2420400707924273e-08, 1.2488357015172369e-08, 1.0263853766900866e-08, 1.136522786993055e-08,
				1.3036516754993954e-08, 3.0427411701339224e-09, 8.650623328776419e-09, 9.89251436323002e-09,
				3.5014693366974825e-09, 1.5230525729492683e-09},
		},
		{ // PM25 800m
			d: []float64{3.835875714286452e-11, 6.099905996981292e-11, 8.292511816110348e-11, 5.7506100575865915e-11,
				8.120508432352125e-11, 4.1053983129701876e-11, 7.843166394128076e-11, 1.0826384233553199e-10,
				4.4888457534364434e-11, 2.513591244868163e-11},
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
		if !reflect.DeepEqual(want[i].d, c.TotalPM25()) {
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

func TestOutput(t *testing.T) {
	r, err := os.Open("../cmd/inmap/testdata/testSR_golden.ncf")
	if err != nil {
		t.Fatal(err)
	}
	sr, err := NewReader(r)
	if err != nil {
		t.Fatal(err)
	}

	e := []*inmap.EmisRecord{
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
	}

	want := []float64{2.9264157800981683e-06, 1.3441580584069857e-06, 3.749182241816196e-07, 9.059233059376115e-07,
		6.482974716781609e-07, 4.407357260486494e-08, 1.85018648386423e-07, 1.6593788779856178e-07,
		3.441426629866712e-08, 1.2632353938064889e-08}

	c, err := sr.Concentrations(e...)
	if err != nil {
		t.Fatalf("calculating concentrations: %v", err)
	}
	totalPM25 := c.TotalPM25()
	t.Run("check concentrations", func(t *testing.T) {
		if !reflect.DeepEqual(want, totalPM25) {
			for j, v := range totalPM25 {
				w := want[j]
				if math.Abs(w-v)*2/(w+v) > 1.e-8 {
					t.Errorf("row %d: want %v but have %v", j, w, v)
				}
			}
		}
	})
	if err = sr.SetConcentrations(c); err != nil {
		t.Fatalf("setting concentrations: %v", err)
	}

	sRef, err := proj.Parse("+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1")
	if err != nil {
		t.Fatal(err)
	}

	const TestOutputFilename = "testOutput.shp"

	if err = sr.Output(TestOutputFilename, map[string]string{
		"TotalPop":   "TotalPop",
		"WhiteNoLat": "WhiteNoLat",
		"NPctWNoLat": "{sum(WhiteNoLat) / sum(TotalPop)}",
		"NPctOther":  "{(sum(TotalPop) - sum(WhiteNoLat)) / sum(TotalPop)}",
		"NPctRatio":  "NPctWNoLat / NPctOther",
		"TotalPM25":  "PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA",
		"TotalPopD":  "(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * allcause / 100000",
		"PrimPM25":   "PrimaryPM25",
		"PNH4":       "pNH4",
		"PSO4":       "pSO4",
		"PNO3":       "pNO3",
		"SOA":        "SOA",
		"BasePM25":   "BaselineTotalPM25",
		"WindSpeed":  "WindSpeed"},
		nil, sRef); err != nil {
		t.Fatal(err)
	}

	type outData struct {
		BasePM25                        float64 `shp:"BasePM25"`
		TotalPM25                       float64
		PrimPM25, PNH4, PSO4, PNO3, SOA float64
		TotalPop                        float64
		WhiteNoLat                      float64
		NPctWNoLat                      float64
		NPctOther                       float64
		NPctRatio                       float64
		Deaths                          float64 `shp:"TotalPopD"`
		WindSpeed                       float64
	}
	dec, err := shp.NewDecoder(TestOutputFilename)
	if err != nil {
		t.Fatal(err)
	}
	var recs []outData
	for {
		var rec outData
		if more := dec.DecodeRow(&rec); !more {
			break
		}
		recs = append(recs, rec)
	}
	if err := dec.Error(); err != nil {
		t.Fatal(err)
	}

	shpWant := []outData{
		{BasePM25: 4.90770054, TotalPM25: 2.9264157801e-06, PrimPM25: 2.2917495244e-06, PNH4: 5.051084031e-07, PSO4: 8.102691434e-11, PNO3: 1.16086788182e-07, SOA: 1.33900375232e-08, TotalPop: 100000, WhiteNoLat: 50000, NPctWNoLat: 0.5, NPctOther: 0.5, NPctRatio: 1, Deaths: 1.75836556e-05, WindSpeed: 2.16334701},
		{BasePM25: 4.90770054, TotalPM25: 1.3441580584e-06, PrimPM25: 1.0526249525e-06, PNH4: 2.3203219257e-07, PSO4: 7.153899001e-11, PNO3: 5.3347690709e-08, SOA: 6.0816836012e-09, TotalPop: 0, WhiteNoLat: 0, NPctWNoLat: 0.5, NPctOther: 0.5, NPctRatio: 1, Deaths: 0, WindSpeed: 2.16334701},
		{BasePM25: 4.90770054, TotalPM25: 3.749182242e-07, PrimPM25: 2.935867656e-07, PNH4: 6.474125769e-08, PSO4: 4.696325967e-11, PNO3: 1.4901711864e-08, SOA: 1.6415258042e-09, TotalPop: 0, WhiteNoLat: 0, NPctWNoLat: 0.5, NPctOther: 0.5, NPctRatio: 1, Deaths: 0, WindSpeed: 2.16334701},
		{BasePM25: 4.90770054, TotalPM25: 9.059233059e-07, PrimPM25: 7.09436506e-07, PNH4: 1.5638531181e-07, PSO4: 5.143686518e-11, PNO3: 3.5957103961e-08, SOA: 4.0929473144e-09, TotalPop: 0, WhiteNoLat: 0, NPctWNoLat: 0.5, NPctOther: 0.5, NPctRatio: 1, Deaths: 0, WindSpeed: 2.16334701},
		{BasePM25: 4.90770054, TotalPM25: 6.482974717e-07, PrimPM25: 5.076755087e-07, PNH4: 1.1192977922e-07, PSO4: 5.775954715e-11, PNO3: 2.5748777333e-08, SOA: 2.8856468326e-09, TotalPop: 0, WhiteNoLat: 0, NPctWNoLat: 0.5, NPctOther: 0.5, NPctRatio: 1, Deaths: 0, WindSpeed: 2.16334701},
		{BasePM25: 10.34742928, TotalPM25: 4.40735726e-08, PrimPM25: 3.37436425e-08, PNH4: 8.87124596e-09, PSO4: 9.67710721e-12, PNO3: 1.357314372e-09, SOA: 9.16926188e-11, TotalPop: 0, WhiteNoLat: 0, NPctWNoLat: 0.5, NPctOther: 0.5, NPctRatio: 1, Deaths: 0, WindSpeed: 1.88434911},
		{BasePM25: 4.90770054, TotalPM25: 1.850186484e-07, PrimPM25: 1.448792233e-07, PNH4: 3.195175324e-08, PSO4: 2.741011411e-11, PNO3: 7.356107279e-09, SOA: 8.041544652e-10, TotalPop: 0, WhiteNoLat: 0, NPctWNoLat: 0.5, NPctOther: 0.5, NPctRatio: 1, Deaths: 0, WindSpeed: 2.16334701},
		{BasePM25: 4.90770054, TotalPM25: 1.659378878e-07, PrimPM25: 1.299311947e-07, PNH4: 2.866452142e-08, PSO4: 3.392639641e-11, PNO3: 6.604432556e-09, SOA: 7.038127303e-10, TotalPop: 0, WhiteNoLat: 0, NPctWNoLat: 0.5, NPctOther: 0.5, NPctRatio: 1, Deaths: 0, WindSpeed: 2.16334701},
		{BasePM25: 4.2574172, TotalPM25: 3.44142663e-08, PrimPM25: 2.98748972e-08, PNH4: 4.02236022e-09, PSO4: 1.032346517e-11, PNO3: 3.61747104e-10, SOA: 1.449382836e-10, TotalPop: 0, WhiteNoLat: 0, NPctWNoLat: 0.5, NPctOther: 0.5, NPctRatio: 1, Deaths: 0, WindSpeed: 2.7272017},
		{BasePM25: 5.36232233, TotalPM25: 1.26323539e-08, PrimPM25: 1.05311022e-08, PNH4: 1.7840075e-09, PSO4: 4.80165126e-12, PNO3: 2.95997171e-10, SOA: 1.64454058e-11, TotalPop: 0, WhiteNoLat: 0, NPctWNoLat: 0.5, NPctOther: 0.5, NPctRatio: 1, Deaths: 0, WindSpeed: 2.56135321},
	}

	if len(recs) != len(shpWant) {
		t.Errorf("want %d records but have %d", len(shpWant), len(recs))
	}
	for i, w := range shpWant {
		if i >= len(recs) {
			continue
		}
		h := recs[i]
		if !reflect.DeepEqual(w, h) {
			t.Errorf("record %d: want %+v but have %+v", i, w, h)
		}
	}
	dec.Close()
	inmap.DeleteShapefile(TestOutputFilename)
}
