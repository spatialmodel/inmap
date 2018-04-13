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
	"fmt"
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
	want := 0.5574290770354347
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
	want := 0.5574290770354343
	have := mat.Sum(conc)
	if want != have {
		t.Errorf("have %g, want %g", have, want)
	}
}

func TestConcentrations_long(t *testing.T) {
	if testing.Short() {
		return
	}
	f, err := os.Open("data/example_config.toml")
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSpatial(f)
	if err != nil {
		t.Fatal(err)
	}
	const year = 2014
	demand, err := s.EIO.FinalDemand(All, nil, year, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tests := []struct {
		pol    Pollutant
		result float64
	}{
		{
			pol:    PNH4,
			result: 34146.995227718384,
		},
		{
			pol:    PNO3,
			result: 19366.49435073656,
		},
		{
			pol:    PSO4,
			result: 27890.642109846598,
		},
		{
			pol:    SOA,
			result: 6908.045187570745,
		},
		{
			pol:    PrimaryPM25,
			result: 89144.0790637183,
		},
		{
			pol:    TotalPM25,
			result: 177456.25593959185,
		},
	}
	for _, test := range tests {
		var failed bool
		t.Run(fmt.Sprintf("%v", test.pol), func(t *testing.T) {
			conc, err := s.Concentrations(ctx, demand, nil, test.pol, year, Domestic)
			if err != nil {
				failed = true
				t.Fatal(err)
			}
			have := mat.Sum(conc)
			if test.result != have {
				t.Errorf("have %g, want %g", have, test.result)
			}
		})
		if failed {
			break
		}
	}
}
