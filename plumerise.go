/*
Copyright (C) 2013-2014 Regents of the University of Minnesota.
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

package inmap

import (
	"fmt"

	"github.com/ctessum/atmos/plumerise"
)

// CalcPlumeRise calculates plume rise when given stack information
// (see github.com/ctessum/atmos/plumerise for required units)
// and the index of the (ground level) grid cell (called `row`).
// Returns the index of the cell the emissions should be added to.
// This function assumes that when one grid cell is above another
// grid cell, the upper cell is never smaller than the lower cell.
func (d *InMAPdata) CalcPlumeRise(stackHeight, stackDiam, stackTemp,
	stackVel float64, row int) (plumeRow int, plumeHeight float64, err error) {
	layerHeights := make([]float64, d.Nlayers+1)
	temperature := make([]float64, d.Nlayers)
	windSpeed := make([]float64, d.Nlayers)
	windSpeedInverse := make([]float64, d.Nlayers)
	windSpeedMinusThird := make([]float64, d.Nlayers)
	windSpeedMinusOnePointFour := make([]float64, d.Nlayers)
	sClass := make([]float64, d.Nlayers)
	s1 := make([]float64, d.Nlayers)

	cell := d.Cells[row]
	for i := 0; i < d.Nlayers; i++ {
		layerHeights[i+1] = layerHeights[i] + cell.Dz
		windSpeed[i] = cell.WindSpeed
		windSpeedInverse[i] = cell.WindSpeedInverse
		windSpeedMinusThird[i] = cell.WindSpeedMinusThird
		windSpeedMinusOnePointFour[i] = cell.WindSpeedMinusOnePointFour
		sClass[i] = cell.SClass
		s1[i] = cell.S1
		if len(cell.Above) > 0 {
			cell = cell.Above[0]
		} else {
			err = fmt.Errorf("Plume rise is above top layer for height=%g, "+
				"diameter=%g, temperature=%g, velocity=%g.", stackHeight,
				stackDiam, stackTemp, stackVel)
			return
		}
	}
	var plumeIndex int
	plumeIndex, plumeHeight, err = plumerise.ASMEPrecomputed(stackHeight, stackDiam,
		stackTemp, stackVel, layerHeights, temperature, windSpeed,
		sClass, s1, windSpeedMinusOnePointFour, windSpeedMinusThird,
		windSpeedInverse)
	if err != nil {
		if err == plumerise.ErrAboveModelTop {
			plumeIndex = d.Nlayers - 1
			err = nil
		} else {
			return
		}
	}

	plumeCell := d.Cells[row]
	for i := 0; i < plumeIndex; i++ {
		plumeCell = plumeCell.Above[0]
	}
	plumeRow = plumeCell.Row
	return
}
