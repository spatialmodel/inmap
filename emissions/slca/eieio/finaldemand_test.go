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
)

func TestLoadFinalDemand(t *testing.T) {
	e := loadSpatial(t).EIO
	allDemand, err := e.FinalDemand(context.Background(), &eieiorpc.FinalDemandInput{
		FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
		Year:            2007,
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := len(allDemand.Data)
	if r != 389 {
		t.Fatalf("length should be 389 but is %d", r)
	}
	v := allDemand.Data[388]
	want := 1.696e+03 * 1.0e6 // total from spreadsheet (but different because imports)
	if different(v, want) {
		t.Errorf("have %v but want %v", v, want)
	}

	fd1997, err := e.loadFinalDemand("data/IOUse_Before_Redefinitions_PRO_2007_Detail.xlsx", "data/IOUse_Before_Redefinitions_PRO_1997-2015_Summary.xlsx", 1997, 2007, false)
	if err != nil {
		t.Fatal(err)
	}
	pc := fd1997[PersonalConsumption]
	r, _ = pc.Dims()
	if r != 389 {
		t.Fatalf("length should be 389 but is %d", r)
	}
	v = pc.At(1, 0) // Grain farming

	const (
		pc2007Detail  = 1096.0
		pc2007Summary = 52756.0
		pc1997Summary = 35926.0
		pcWant        = pc2007Detail * pc1997Summary / pc2007Summary * 1.0e6
	)
	if different(v, pcWant) {
		t.Errorf("have %v but want %v", v, pcWant)
	}
}
