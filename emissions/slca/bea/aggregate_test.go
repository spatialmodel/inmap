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
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestAggregator(t *testing.T) {
	e := loadSpatial(t).EIO

	a, err := e.NewAggregator("data/aggregates.xlsx")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("names", func(t *testing.T) {
		if len(a.Names()) != 11 {
			t.Errorf("have %d names, want 11", len(a.Names()))
		}
	})

	t.Run("abbreviations", func(t *testing.T) {
		if len(a.Abbreviations()) != 11 {
			t.Errorf("have %d abbreviations, want 11", len(a.Abbreviations()))
		}
	})

	t.Run("abbreviation", func(t *testing.T) {
		abbrev, err := a.Abbreviation("Animal Production")
		if err != nil {
			t.Fatal(err)
		}
		if abbrev != "Anml." {
			t.Errorf("have %s, want 'Anml.'", abbrev)
		}
	})

	t.Run("industry mask", func(t *testing.T) {
		mask := a.IndustryMask("Elec.")
		r := len(e.Industries)
		v := mat.NewVecDense(r, nil)
		for i := 0; i < r; i++ {
			v.SetVec(i, float64(i))
		}
		mask.Mask(v)
		sum := mat.Sum(v)
		const sumWant float64 = 21 + 383 + 387
		if sum != sumWant {
			t.Errorf("sum: want %g, have %g", sumWant, sum)
		}
	})

	t.Run("commodity mask", func(t *testing.T) {
		mask := a.CommodityMask("Elec.")
		r := len(e.Commodities)
		v := mat.NewVecDense(r, nil)
		for i := 0; i < r; i++ {
			v.SetVec(i, float64(i))
		}
		mask.Mask(v)
		sum := mat.Sum(v)
		const sumWant float64 = 21
		if sum != sumWant {
			t.Errorf("sum: want %g, have %g", sumWant, sum)
		}
	})

	t.Run("single industry mask", func(t *testing.T) {
		m, err := e.IndustryMask("Other state and local government enterprises")
		if err != nil {
			t.Fatal(err)
		}
		v := (mat.VecDense)(*m)
		if v.At(388, 0) != 1 {
			t.Error("wrong mask index")
		}
		if mat.Sum(&v) != 1 {
			t.Errorf("wrong mask sum")
		}
	})

	t.Run("single commodity mask", func(t *testing.T) {
		m, err := e.CommodityMask("Rest of the world adjustment")
		if err != nil {
			t.Fatal(err)
		}
		v := (mat.VecDense)(*m)
		if v.At(388, 0) != 1 {
			t.Error("wrong mask index")
		}
		if mat.Sum(&v) != 1 {
			t.Errorf("wrong mask sum")
		}
	})
}
