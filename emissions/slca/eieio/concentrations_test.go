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

package eieio

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/gonum/floats"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"
	"github.com/spatialmodel/inmap/epi"
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
		c := new(SpatialConfig)
		if _, err := toml.DecodeReader(f, c); err != nil {
			t.Fatal(err)
		}
		s, err = NewSpatial(c, epi.NasariACS)
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

	demand, err := s.EIO.FinalDemand(context.Background(), &eieiorpc.FinalDemandInput{
		FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
		Year:            2011,
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	conc, err := s.Concentrations(ctx, &eieiorpc.ConcentrationInput{
		Demand:    demand,
		Pollutant: eieiorpc.Pollutant_TotalPM25,
		Year:      2011,
		Location:  eieiorpc.Location_Domestic,
		AQM:       "isrm",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := 0.6092829446666379
	have := floats.Sum(conc.Data)
	if want != have {
		t.Errorf("have %g, want %g", have, want)
	}
}

func TestConcentrationMatrix(t *testing.T) {
	s := loadSpatial(t)

	demand, err := s.EIO.FinalDemand(context.Background(), &eieiorpc.FinalDemandInput{
		FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
		Year:            2011,
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	concRPC, err := s.ConcentrationMatrix(ctx, &eieiorpc.ConcentrationMatrixInput{
		Demand:    demand,
		Pollutant: eieiorpc.Pollutant_TotalPM25,
		Year:      2011,
		Location:  eieiorpc.Location_Domestic,
		AQM:       "isrm",
	})
	if err != nil {
		t.Fatal(err)
	}
	conc := rpc2mat(concRPC)
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
