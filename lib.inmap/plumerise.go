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
	"bitbucket.org/ctessum/atmos/plumerise"
)

// Calculates plume rise when given stack information
// (see bitbucket.org/ctessum/atmos/plumerise for required units)
// and the index of the (ground level) grid cell (called `row`).
// Returns the index of the cell the emissions should be added to.
// This function assumes that when one grid cell is above another
// grid cell, the upper cell is never smaller than the lower cell.
func (d *InMAPdata) CalcPlumeRise(stackHeight, stackDiam, stackTemp,
	stackVel float64, row int) (plumeRow int, err error) {
	layerHeights := make([]float64, d.Nlayers+1)
	temperature := make([]float64, d.Nlayers)
	windSpeed := make([]float64, d.Nlayers)
	sClass := make([]float64, d.Nlayers)
	s1 := make([]float64, d.Nlayers)

	cell := d.Data[row]
	for i := 0; i < d.Nlayers; i++ {
		layerHeights[i+1] = layerHeights[i] + cell.Dz
		windSpeed[i] = cell.WindSpeed
		sClass[i] = cell.SClass
		s1[i] = cell.S1
		cell = cell.Above[0]
	}
	var kPlume int
	kPlume, err = plumerise.PlumeRiseASME(stackHeight, stackDiam, stackTemp,
		stackVel, layerHeights, temperature, windSpeed,
		sClass, s1)
	if err != nil {
		return
	}

	plumeCell := d.Data[row]
	for i := 0; i < kPlume; i++ {
		plumeCell = plumeCell.Above[0]
	}
	plumeRow = plumeCell.Row
	return
}
