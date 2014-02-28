package aim

import (
	"bitbucket.org/ctessum/atmos/plumerise"
)

// Calculates plume rise when given stack information
// (see bitbucket.org/ctessum/atmos/plumerise for required units)
// and the index of the (ground level) grid cell (called `row`).
// Returns the index of the cell the emissions should be added to.
// This function assumes that when one grid cell is above another
// grid cell, the upper cell is never smaller than the lower cell.
func (d *AIMdata) CalcPlumeRise(stackHeight, stackDiam, stackTemp,
	stackVel float64, row int) (plumeRow int, err error) {
	layerHeights := make([]float64, d.nLayers)
	temperature := make([]float64, d.nLayers)
	windSpeed := make([]float64, d.nLayers)
	sClass := make([]float64, d.nLayers)
	s1 := make([]float64, d.nLayers)

	cell := d.Data[row]
	for i := 0; i < d.nLayers; i++ {
		layerHeights[i] = cell.LayerHeight
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
