/*
Copyright (C) 2019 the InMAP authors.
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

package aep

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/proj"
	"github.com/gonum/floats"
	"github.com/spatialmodel/inmap/internal/hash"
)

func TestCreateSurrogates_osm(t *testing.T) {
	inputSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("testdata/srgspec_osm.json")
	if err != nil {
		t.Fatal(err)
	}
	srgSpecs, err := ReadSrgSpecOSM(f, "", 10)
	if err != nil {
		t.Error(err)
	}
	gridRef, err := ReadGridRef(strings.NewReader(`000007;0010101011;001
000007;0010101012;002
000007;0010101013;003
  `), true)
	if err != nil {
		t.Fatal(err)
	}

	grid := NewGridRegular("test grid", 4, 4, 0.1, 0.1, -158, 21.25, inputSR)

	d, err := shp.NewDecoder("testdata/honolulu_hawaii.shp")
	if err != nil {
		t.Fatal(err)
	}
	g, _, _ := d.DecodeRowFields()
	if err := d.Error(); err != nil {
		t.Fatal(err)
	}
	sr, err := d.SR()
	if err != nil {
		t.Fatal(err)
	}

	inputLoc := &Location{Geom: g, SR: sr, Name: "input1"}

	key := hash.Hash(inputLoc)
	wantKey := "input1"
	if key != wantKey {
		t.Errorf("location key: have %s, want %s", key, wantKey)
	}

	matchFullSCC := true
	sp := NewSpatialProcessor(srgSpecs, []*GridDef{grid}, gridRef, inputSR, matchFullSCC)
	sp.load()

	want := []map[int]float64{
		map[int]float64{0: 0.04886323779213095, 1: 0.4234115998508295, 2: 0.15919387877688768, 3: 0.08945252047016032, 4: 0.18993456550450022, 5: 0.008311450956844888, 6: 0.07115494071078621},
		map[int]float64{1: 0.6011955358239497, 3: 0.035471039348746576, 4: 0.03985223587634336, 6: 0.32348118895096034},
		map[int]float64{0: 0.017937219730941704, 1: 0.8834080717488813, 2: 0.04484304932735426, 3: 0.013452914798206277, 4: 0.020179372197309416, 6: 0.020179372197309416},
	}

	for i, code := range []string{"001", "002", "003"} {
		t.Run(code, func(t *testing.T) {
			srgSpec, err := srgSpecs.GetByCode(Global, code)
			if err != nil {
				t.Fatal(err)
			}
			sg := &srgGrid{srg: srgSpec, gridData: grid, loc: inputLoc, sp: sp}
			srgsI, err := sg.Run(context.Background())
			if err != nil {
				t.Fatalf("creating surrogate %s: %v", code, err)
			}
			srgs := srgsI.(*GriddedSrgData)
			griddedSrg, covered := srgs.ToGrid()
			if covered {
				t.Errorf("srg %s should not cover", code)
			}
			sparseCompare(want[i], griddedSrg.Elements, t, 1.0e-10)
		})
	}
}

func sparseCompare(a, b map[int]float64, t *testing.T, tol float64) {
	for i, va := range a {
		if vb, ok := b[i]; ok {
			if !floats.EqualWithinAbsOrRel(va, vb, tol, tol) {
				t.Errorf("index %d: %g != %g", i, va, vb)
			}
		} else {
			t.Errorf("index %d not in b", i)
		}
	}
	for i := range b {
		if _, ok := a[i]; !ok {
			t.Errorf("index %d not in a", i)
		}
	}
}

// Test to make sure surrogate cache is working
func TestCreateSurrogates_osmSrgCache(t *testing.T) {
	inputSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("testdata/srgspec_osm.json")
	if err != nil {
		t.Fatal(err)
	}
	srgSpecs, err := ReadSrgSpecOSM(f, "", 10)
	if err != nil {
		t.Error(err)
	}
	gridRef, err := ReadGridRef(strings.NewReader(`000007;0010101011;001
000007;0010101012;002
000007;0010101013;003
  `), true)
	if err != nil {
		t.Fatal(err)
	}

	grid := NewGridRegular("test grid", 4, 4, 0.1, 0.1, -158, 21.25, inputSR)

	d, err := shp.NewDecoder("testdata/honolulu_hawaii.shp")
	if err != nil {
		t.Fatal(err)
	}
	g, _, _ := d.DecodeRowFields()
	if err := d.Error(); err != nil {
		t.Fatal(err)
	}
	sr, err := d.SR()
	if err != nil {
		t.Fatal(err)
	}

	inputLoc := &Location{Geom: g, SR: sr, Name: "input1"}

	key := hash.Hash(inputLoc)
	wantKey := "input1"
	if key != wantKey {
		t.Errorf("location key: have %s, want %s", key, wantKey)
	}

	matchFullSCC := true
	sp := NewSpatialProcessor(srgSpecs, []*GridDef{grid}, gridRef, inputSR, matchFullSCC)
	sp.load()

	srgSpec, err := srgSpecs.GetByCode(Global, "001")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			_, _, err := sp.Surrogate(srgSpec, grid, inputLoc)
			if err != nil {
				t.Fatal(err)
			}

			requestsWant := []int{i + 1, i + 1, 1}
			requests := sp.cache.Requests()
			if !reflect.DeepEqual(requests, requestsWant) {
				t.Errorf("%d: %v != %v", i, requests, requestsWant)
			}

			requestsWant = []int{1, 1, 1}
			requests = srgSpec.(*SrgSpecOSM).cache.Requests()
			if !reflect.DeepEqual(requests, requestsWant) {
				t.Errorf("%d: %v != %v", i, requests, requestsWant)
			}
		})
	}

	// Slightly perturb the input location.
	poly := g.(geom.Polygon)
	for i, r := range poly {
		for j, pt := range r {
			poly[i][j] = geom.Point{X: pt.X + 0.00000001, Y: pt.Y}
		}
	}
	inputLoc = &Location{Geom: poly, SR: sr, Name: "input2"}

	key = hash.Hash(inputLoc)
	wantKey = "input2"
	if key != wantKey {
		t.Errorf("location key: have %s, want %s", key, wantKey)
	}

	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			_, _, err := sp.Surrogate(srgSpec, grid, inputLoc)
			if err != nil {
				t.Fatal(err)
			}

			requestsWant := []int{i + 4, i + 4, 2}
			requests := sp.cache.Requests()
			if !reflect.DeepEqual(requests, requestsWant) {
				t.Errorf("%d: %v != %v", i, requests, requestsWant)
			}

			requestsWant = []int{2, 2, 1}
			requests = srgSpec.(*SrgSpecOSM).cache.Requests()
			if !reflect.DeepEqual(requests, requestsWant) {
				t.Errorf("%d: %v != %v", i, requests, requestsWant)
			}
		})
	}
}
