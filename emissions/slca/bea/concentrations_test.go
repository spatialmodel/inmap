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
	"context"
	"os"
	"sync"
	"testing"

	"gonum.org/v1/gonum/mat"
)

var s *SpatialEIO

var loadSpatialOnce sync.Once

func loadSpatial(t *testing.T) *SpatialEIO {
	loadSpatialOnce.Do(func() {
		f, err := os.Open("data/test_config.toml")
		if err != nil {
			t.Fatal(err)
		}
		s, err = NewSpatial(f)
		if err != nil {
			t.Fatal(err)
		}
	})
	if s == nil {
		t.Fatal("loadSpatial previously failed")
	}
	return s
}

func TestConcentrations(t *testing.T) {
	s := loadSpatial(t)

	demand, err := s.EIO.FinalDemand(All, nil, 2011, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	conc, err := s.Concentrations(ctx, demand, nil, TotalPM25, 2011, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	want := 0.6092829446666378
	have := mat.Sum(conc)
	if want != have {
		t.Errorf("have %g, want %g", have, want)
	}
}

func TestConcentrationMatrix(t *testing.T) {
	s := loadSpatial(t)

	demand, err := s.EIO.FinalDemand(All, nil, 2011, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	conc, err := s.ConcentrationMatrix(ctx, demand, TotalPM25, 2011, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	r, c := conc.Dims()
	wantR, wantC := 10, 188
	if r != wantR {
		t.Errorf("rows: %d !=  %d", r, wantR)
	}
	if c != wantC {
		t.Errorf("cols: %d !=  %d", c, wantC)
	}

	want := 0.6092829446666519
	have := mat.Sum(conc)
	if want != have {
		t.Errorf("have %g, want %g", have, want)
	}
}
