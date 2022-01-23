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
	"testing"

	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"

	"gonum.org/v1/gonum/mat"
)

func TestEmissions(t *testing.T) {
	s := loadSpatial(t)

	ctx := context.Background()
	demand, err := s.EIO.FinalDemand(ctx, &eieiorpc.FinalDemandInput{
		FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
		Year:            2011,
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		t.Fatal(err)
	}
	emisRPC, err := s.Emissions(ctx, &eieiorpc.EmissionsInput{
		Demand:   demand,
		Emission: eieiorpc.Emission_PM25,
		Year:     2011,
		Location: eieiorpc.Location_Domestic,
		AQM:      "isrm",
	})
	if err != nil {
		t.Fatal(err)
	}
	emis := array2vec(emisRPC.Data)

	want := 4.988885756456615e+08 // ug/s; ~= 17342.6039028 ton/year *  17342.6039028 ug/s / (ton/year)
	have := mat.Sum(emis)
	if want != have {
		t.Errorf("have %g, want %g", have, want)
	}
}

func TestEmissionsMatrix(t *testing.T) {
	s := loadSpatial(t)

	ctx := context.Background()
	demand, err := s.EIO.FinalDemand(ctx, &eieiorpc.FinalDemandInput{
		FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
		Year:            2011,
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		t.Fatal(err)
	}
	emisRPC, err := s.EmissionsMatrix(ctx, &eieiorpc.EmissionsMatrixInput{
		Demand:   demand,
		Emission: eieiorpc.Emission_PM25,
		Year:     2011,
		Location: eieiorpc.Location_Domestic,
		AQM:      "isrm",
	})
	if err != nil {
		t.Fatal(err)
	}
	emis := rpc2mat(emisRPC)
	r, c := emis.Dims()
	wantR, wantC := 10, 188
	if r != wantR {
		t.Errorf("rows: %d !=  %d", r, wantR)
	}
	if c != wantC {
		t.Errorf("cols: %d !=  %d", c, wantC)
	}

	want := 4.9888857564566237e+08
	have := mat.Sum(emis)
	if different(want, have) {
		t.Errorf("have %g, want %g", have, want)
	}
}
