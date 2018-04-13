/*
Copyright © 2018 the InMAP authors.
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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package aep

import (
	"bitbucket.org/ctessum/sparse"
)

// SpatialAdjustRecord wraps around a Record to provide a spatially-explicit
// adjustment.
type SpatialAdjustRecord struct {
	// Record is the record to be adjusted.
	Record

	// SpatialAdjuster specifies the adjustment to occur.
	SpatialAdjuster
}

// Spatialize calls the Spatialize method of the contained Record and adjusts
// the output using the SpatialAdjuster field.
func (r *SpatialAdjustRecord) Spatialize(sp *SpatialProcessor, gi int) (
	gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error) {

	gridSrg, coveredByGrid, inGrid, err = r.Record.Spatialize(sp, gi)
	if gridSrg == nil || err != nil {
		return
	}

	adjustment, err := r.SpatialAdjuster.Adjustment()
	out := gridSrg.Copy()
	for i, v := range out.Elements {
		out.Elements[i] = v * adjustment.Elements[i]
	}
	return out, coveredByGrid, inGrid, err
}

// SpatialAdjuster is an interface for types that provide gridded adjustments
// to Records.
type SpatialAdjuster interface {
	Adjustment() (*sparse.DenseArray, error)
}
