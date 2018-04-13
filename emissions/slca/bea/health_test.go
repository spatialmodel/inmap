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
	want := 0.12827912797060476
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
	want := 0.12827912797060478
	have := mat.Sum(health)
	if different(want, have) {
		t.Errorf("have %g, want %g", have, want)
	}
}

func TestHealth_long(t *testing.T) {
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
		pop    string
		result float64
	}{
		{
			pol:    PNH4,
			pop:    "TotalPop",
			result: 12900.184318299018,
		},
		{
			pol:    PNO3,
			pop:    "TotalPop",
			result: 8325.437763238058,
		},
		{
			pol:    PSO4,
			pop:    "TotalPop",
			result: 12170.527563668516,
		},
		{
			pol:    SOA,
			pop:    "TotalPop",
			result: 2312.438101502105,
		},
		{
			pol:    PrimaryPM25,
			pop:    "TotalPop",
			result: 32889.795865085005,
		},
		{
			pol:    TotalPM25,
			pop:    "TotalPop",
			result: 68630.70194265452,
		},

		{
			pol:    PNH4,
			pop:    "WhiteNoLat",
			result: 8485.329964103656,
		},
		{
			pol:    PNO3,
			pop:    "WhiteNoLat",
			result: 5762.4782240421855,
		},
		{
			pol:    PSO4,
			pop:    "WhiteNoLat",
			result: 8403.29904605126,
		},
		{
			pol:    SOA,
			pop:    "WhiteNoLat",
			result: 1345.2876686767142,
		},
		{
			pol:    PrimaryPM25,
			pop:    "WhiteNoLat",
			result: 20124.44086743779,
		},
		{
			pol:    TotalPM25,
			pop:    "WhiteNoLat",
			result: 44143.78451136753,
		},

		{
			pol:    PNH4,
			pop:    "Black",
			result: 1618.1389342643881,
		},
		{
			pol:    PNO3,
			pop:    "Black",
			result: 1109.5442241162987,
		},
		{
			pol:    PSO4,
			pop:    "Black",
			result: 1946.5033686158447,
		},
		{
			pol:    SOA,
			pop:    "Black",
			result: 335.7989270761432,
		},
		{
			pol:    PrimaryPM25,
			pop:    "Black",
			result: 4997.549003764835,
		},
		{
			pol:    TotalPM25,
			pop:    "Black",
			result: 10012.060709391711,
		},

		{
			pol:    PNH4,
			pop:    "Native",
			result: 71.63370714746013,
		},
		{
			pol:    PNO3,
			pop:    "Native",
			result: 47.84151495241535,
		},
		{
			pol:    PSO4,
			pop:    "Native",
			result: 48.019436516370135,
		},
		{
			pol:    SOA,
			pop:    "Native",
			result: 14.389263969863453,
		},
		{
			pol:    PrimaryPM25,
			pop:    "Native",
			result: 179.10396995780567,
		},
		{
			pol:    TotalPM25,
			pop:    "Native",
			result: 361.09791993806607,
		},

		{
			pol:    PNH4,
			pop:    "Asian",
			result: 508.9767933829313,
		},
		{
			pol:    PNO3,
			pop:    "Asian",
			result: 278.5541286222845,
		},
		{
			pol:    PSO4,
			pop:    "Asian",
			result: 371.8572472817353,
		},
		{
			pol:    SOA,
			pop:    "Asian",
			result: 107.70880518844794,
		},
		{
			pol:    PrimaryPM25,
			pop:    "Asian",
			result: 1453.9224366265985,
		},
		{
			pol:    TotalPM25,
			pop:    "Asian",
			result: 2722.0026303739874,
		},

		{
			pol:    PNH4,
			pop:    "Latino",
			result: 1947.5336972252512,
		},
		{
			pol:    PNO3,
			pop:    "Latino",
			result: 955.4837890199625,
		},
		{
			pol:    PSO4,
			pop:    "Latino",
			result: 1168.0358202824243,
		},
		{
			pol:    SOA,
			pop:    "Latino",
			result: 458.7093392716181,
		},
		{
			pol:    PrimaryPM25,
			pop:    "Latino",
			result: 5423.999833522514,
		},
		{
			pol:    TotalPM25,
			pop:    "Latino",
			result: 9956.885478091597,
		},
	}
	for _, test := range tests {
		var failed bool
		t.Run(fmt.Sprintf("%v_%v", test.pol, test.pop), func(t *testing.T) {
			health, err := s.Health(ctx, demand, nil, test.pol, test.pop, year, Domestic, epi.NasariACS)
			if err != nil {
				failed = true
				t.Fatal(err)
			}
			have := mat.Sum(health)
			if test.result != have {
				t.Errorf("have %g, want %g", have, test.result)
			}
		})
		if failed {
			break
		}
	}
}
