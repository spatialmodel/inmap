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
	"os"
	"strings"
	"testing"

	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/proj"
	"github.com/gonum/floats"
	_ "github.com/lib/pq"
	"github.com/spatialmodel/inmap/internal/hash"
	"github.com/spatialmodel/inmap/internal/postgis"
)

func TestCreateSurrogates_osm(t *testing.T) {
	ctx := context.Background()
	postGISURL, postgresC := postgis.SetupTestDB(ctx, t, "testdata")
	defer postgresC.Terminate(ctx)

	inputSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("testdata/srgspec_osm.json")
	if err != nil {
		t.Fatal(err)
	}
	srgSpecs, err := ReadSrgSpecOSM(ctx, f, postGISURL)
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
		{0: 0.04886323779213095, 1: 0.4234115998508295, 2: 0.15919387877688768, 3: 0.08945252047016032, 4: 0.18993456550450022, 5: 0.008311450956844888, 6: 0.07115494071078621},
		{1: 0.6011955358239497, 3: 0.035471039348746576, 4: 0.03985223587634336, 6: 0.32348118895096034},
		{0: 0.017937219730941704, 1: 0.8834080717488813, 2: 0.04484304932735426, 3: 0.013452914798206277, 4: 0.020179372197309416, 6: 0.020179372197309416},
	}

	for i, code := range []string{"001", "002", "003"} {
		t.Run(code, func(t *testing.T) {
			srgSpec, err := srgSpecs.GetByCode(Global, code)
			if err != nil {
				t.Fatal(err)
			}
			sg := &srgGrid{srg: srgSpec, gridData: grid, loc: inputLoc, sp: sp}
			srgs := new(GriddedSrgData)
			if err := sg.Run(context.Background(), nil, (*griddedSrgDataHolder)(srgs)); err != nil {
				t.Fatalf("creating surrogate %s: %v", code, err)
			}
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
