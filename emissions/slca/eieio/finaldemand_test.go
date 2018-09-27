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

	"github.com/gonum/floats"
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
	want := 1.26269e+5 * 1.0e6 // total from spreadsheet (but different because imports and negative values)
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

func TestFinalDemand_sum(t *testing.T) {
	e := loadSpatial(t).EIO

	t.Run("negative", func(t *testing.T) {
		allDemand, err := e.FinalDemand(context.Background(), &eieiorpc.FinalDemandInput{
			FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
			Year:            2015,
			Location:        eieiorpc.Location_Domestic,
		})
		if err != nil {
			t.Fatal(err)
		}
		for i, v := range allDemand.Data {
			if v < 0 {
				t.Errorf("final demand index %d < 0 = %g", i, v)
			}
		}
	})

	allDemand, err := e.FinalDemand(context.Background(), &eieiorpc.FinalDemandInput{
		FinalDemandType: eieiorpc.FinalDemandType_NonExport,
		Year:            2007,
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		t.Fatal(err)
	}

	var demandSum *eieiorpc.Vector
	for _, fd := range []eieiorpc.FinalDemandType{eieiorpc.FinalDemandType_DefenseConsumption,
		eieiorpc.FinalDemandType_DefenseStructures, eieiorpc.FinalDemandType_DefenseEquipment,
		eieiorpc.FinalDemandType_DefenseIP, eieiorpc.FinalDemandType_NondefenseConsumption,
		eieiorpc.FinalDemandType_NondefenseStructures, eieiorpc.FinalDemandType_NondefenseEquipment,
		eieiorpc.FinalDemandType_NondefenseIP, eieiorpc.FinalDemandType_LocalConsumption,
		eieiorpc.FinalDemandType_LocalStructures, eieiorpc.FinalDemandType_LocalEquipment,
		eieiorpc.FinalDemandType_LocalIP,

		eieiorpc.FinalDemandType_PersonalConsumption,
		eieiorpc.FinalDemandType_PrivateResidential,
		eieiorpc.FinalDemandType_PrivateStructures,
		eieiorpc.FinalDemandType_PrivateEquipment,
		eieiorpc.FinalDemandType_PrivateIP,
		eieiorpc.FinalDemandType_InventoryChange,
	} {
		temp, err := e.FinalDemand(context.Background(), &eieiorpc.FinalDemandInput{
			FinalDemandType: fd,
			Year:            2007,
			Location:        eieiorpc.Location_Domestic,
		})
		if err != nil {
			t.Fatal(err)
		}
		if demandSum == nil {
			demandSum = temp
		} else {
			floats.Add(demandSum.Data, temp.Data)
		}
	}
	for i, v := range allDemand.Data {
		if !floats.EqualWithinAbsOrRel(v, demandSum.Data[i], 1e-10, 1e-10) {
			t.Errorf("%d: %g != %g", i, v, demandSum.Data[i])
		}
	}
}
