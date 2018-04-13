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

	"github.com/spatialmodel/inmap/emissions/slca"

	"gonum.org/v1/gonum/mat"
)

func TestEmissions(t *testing.T) {
	s := loadSpatial(t)

	demand, err := s.EIO.FinalDemand(All, nil, 2011, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	emis, err := s.Emissions(ctx, demand, nil, slca.PM25, 2011, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	want := 4.56429973463053e+08
	have := mat.Sum(emis)
	if want != have {
		t.Errorf("have %g, want %g", have, want)
	}
}

func TestEmissionsMatrix(t *testing.T) {
	s := loadSpatial(t)

	demand, err := s.EIO.FinalDemand(All, nil, 2011, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	emis, err := s.EmissionsMatrix(ctx, demand, slca.PM25, 2011, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	want := 4.5642997346305287e+08
	have := mat.Sum(emis)
	if different(want, have) {
		t.Errorf("have %g, want %g", have, want)
	}
}

func TestEmissions_long(t *testing.T) {
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
		pol    slca.Pollutant
		result float64
	}{
		{
			pol:    slca.PM25,
			result: 1.2705912373324739e+11,
		},
		{
			pol:    slca.NH3,
			result: 9.860117148616118e+10,
		},
		{
			pol:    slca.NOx,
			result: 2.935171868956635e+11,
		},
		{
			pol:    slca.SOx,
			result: 1.3190282760274142e+11,
		},
		{
			pol:    slca.VOC,
			result: 2.4668897575804257e+11,
		},
	}
	for _, test := range tests {
		var failed bool
		t.Run(fmt.Sprintf("%v", test.pol), func(t *testing.T) {
			emis, err := s.Emissions(ctx, demand, nil, test.pol, year, Domestic)
			if err != nil {
				failed = true
				t.Fatal(err)
			}
			have := mat.Sum(emis)
			if different(test.result, have) {
				t.Errorf("have %g, want %g", have, test.result)
			}
		})
		if failed {
			break
		}
	}
}
