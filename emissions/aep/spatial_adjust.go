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
	"fmt"
	"reflect"
	"time"

	"github.com/ctessum/sparse"
	"github.com/ctessum/unit"
)

// RecordGriddedAdjusted wraps around a RecordGridded to provide a spatially-explicit
// adjustment.
type RecordGriddedAdjusted struct {
	// RecordGridded is the record to be adjusted.
	RecordGridded

	// SpatialAdjuster specifies the adjustment to occur.
	SpatialAdjuster
}

// GridFactors calls the GridFactors method of the contained Record and adjusts
// the output using the SpatialAdjuster field.
func (r *RecordGriddedAdjusted) GridFactors(gi int) (
	gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error) {

	gridSrg, coveredByGrid, inGrid, err = r.RecordGridded.GridFactors(gi)
	if gridSrg == nil || err != nil {
		return
	}

	adjustment, err := r.SpatialAdjuster.Adjustment()
	if adjustment == nil || err != nil {
		return
	}

	if !reflect.DeepEqual(gridSrg.Shape, adjustment.Shape) {
		err = fmt.Errorf("aep.RecordGriddedAdjustment: adjustment shape (%v) doesn't match grid shape (%v)", adjustment.Shape, gridSrg.Shape)
		return
	}

	out := gridSrg.Copy()
	for i, v := range out.Elements {
		out.Elements[i] = v * adjustment.Elements[i]
	}
	return out, coveredByGrid, inGrid, nil
}

// GriddedEmissions calls the GriddedEmissions method of the contained Record
// and adjusts the output using the SpatialAdjuster field.
func (r *RecordGriddedAdjusted) GriddedEmissions(begin, end time.Time, gi int) (
	emis map[Pollutant]*sparse.SparseArray, units map[Pollutant]unit.Dimensions, err error) {

	var gridSrg *sparse.SparseArray
	gridSrg, _, _, err = r.GridFactors(gi)
	if err != nil || gridSrg == nil {
		return
	}

	emis = make(map[Pollutant]*sparse.SparseArray)
	units = make(map[Pollutant]unit.Dimensions)
	periodEmis := r.PeriodTotals(begin, end)
	for pol, data := range periodEmis {
		emis[pol] = gridSrg.ScaleCopy(data.Value())
		units[pol] = data.Dimensions()
	}
	return
}

// SpatialAdjuster is an interface for types that provide gridded adjustments
// to Records.
type SpatialAdjuster interface {
	Adjustment() (*sparse.DenseArray, error)
}
