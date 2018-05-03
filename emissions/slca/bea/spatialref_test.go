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

func TestRequirementsSCC(t *testing.T) {
	s := loadSpatial(t)

	r, err := s.requirementsSCC(s.totalRequirements[2011])
	if err != nil {
		t.Fatal(err)
	}
	rows, cols := r.Dims()
	if rows != 188 {
		t.Errorf("rows: %d != %d", rows, 188)
	}
	if cols != 389 {
		t.Errorf("cols: %d != %d", cols, 389)
	}
	for i := 0; i < rows; i++ {
		rowSum := mat.Sum(r.RowView(i))
		if rowSum <= 0 {
			t.Errorf("row %d: %g<=0", i, rowSum)
		}
	}
}
