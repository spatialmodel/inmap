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

import "github.com/ctessum/atmos/plumerise"

// IsPlumeIn calculates whether the plume rise from an emission is at the height
// of c when given stack information
// (see github.com/ctessum/atmos/plumerise for required units).
func (c *Cell) IsPlumeIn(stackHeight, stackDiam, stackTemp, stackVel float64) (bool, error) {

	// Find the cells in the vertical column below c.
	var cellStack []*Cell
	cc := c
	for {
		if cc.GroundLevel[0] != cc {
			cellStack = append(cellStack, cc)
		} else {
			break
		}
		cc = cc.Below[0]
	}
	// reverse the order of the stack so it starts at ground level.
	for left, right := 0, len(cellStack)-1; left < right; left, right = left+1, right-1 {
		cellStack[left], cellStack[right] = cellStack[right], cellStack[left]
	}

	layerHeights := make([]float64, len(cellStack)+1)
	temperature := make([]float64, len(cellStack))
	windSpeed := make([]float64, len(cellStack))
	windSpeedInverse := make([]float64, len(cellStack))
	windSpeedMinusThird := make([]float64, len(cellStack))
	windSpeedMinusOnePointFour := make([]float64, len(cellStack))
	sClass := make([]float64, len(cellStack))
	s1 := make([]float64, len(cellStack))

	for i, cell := range cellStack {
		layerHeights[i+1] = layerHeights[i] + cell.Dz
		windSpeed[i] = cell.WindSpeed
		windSpeedInverse[i] = cell.WindSpeedInverse
		windSpeedMinusThird[i] = cell.WindSpeedMinusThird
		windSpeedMinusOnePointFour[i] = cell.WindSpeedMinusOnePointFour
		sClass[i] = cell.SClass
		s1[i] = cell.S1
	}

	plumeIndex, _, err := plumerise.ASMEPrecomputed(stackHeight, stackDiam,
		stackTemp, stackVel, layerHeights, temperature, windSpeed,
		sClass, s1, windSpeedMinusOnePointFour, windSpeedMinusThird,
		windSpeedInverse)
	if err != nil {
		if err == plumerise.ErrAboveModelTop {
			// If the plume is above the top of our stack, return true if c is
			// in the top model layer (because we want to put the plume in the
			// top layer even if it should technically go above it),
			//  otherwise return false.
			if c.Above[0].Boundary {
				return true, nil
			}
			return false, nil
		}
		return false, err
	}

	// if the index of the plume is at the end of the cell stack,
	// that means that the plume should go in this cell.
	if plumeIndex == len(cellStack)-1 {
		return true, nil
	}
	return false, nil
}
