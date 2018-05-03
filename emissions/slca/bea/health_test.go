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
	"testing"

	"github.com/spatialmodel/epi"
	"gonum.org/v1/gonum/mat"
)

func TestHealth(t *testing.T) {
	s := loadSpatial(t)

	demand, err := s.EIO.FinalDemand(All, nil, 2011, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	health, err := s.Health(ctx, demand, nil, TotalPM25, "TotalPop", 2011, Domestic, epi.NasariACS)
	if err != nil {
		t.Fatal(err)
	}
	want := 0.14021207010740525
	have := mat.Sum(health)
	if different(want, have) {
		t.Errorf("have %g, want %g", have, want)
	}
}

func TestHealthMatrix(t *testing.T) {
	s := loadSpatial(t)

	demand, err := s.EIO.FinalDemand(All, nil, 2011, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	health, err := s.HealthMatrix(ctx, demand, TotalPM25, "TotalPop", 2011, Domestic, epi.NasariACS)
	if err != nil {
		t.Fatal(err)
	}
	r, c := health.Dims()
	wantR, wantC := 10, 188
	if r != wantR {
		t.Errorf("rows: %d !=  %d", r, wantR)
	}
	if c != wantC {
		t.Errorf("cols: %d !=  %d", c, wantC)
	}

	want := 0.1402120701074049
	have := mat.Sum(health)
	if different(want, have) {
		t.Errorf("have %g, want %g", have, want)
	}
}
