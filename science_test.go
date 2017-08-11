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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmap

import (
	"math"
	"testing"

	"github.com/ctessum/geom"
)

// Test whether convective mixing coefficients are balanced in
// a way that conserves mass
func TestConvectiveMixing(t *testing.T) {
	const testTolerance = 1.e-8

	cfg, ctmdata, pop, popIndices, mr, mortIndices := VarGridTestData()
	emis := NewEmissions()

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	var m Mech
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	for _, c := range d.Cells() {
		val := c.M2u - c.M2d + (*c.above)[0].M2d*(*c.above)[0].Dz/c.Dz
		if absDifferent(val, 0, testTolerance) {
			t.Error(c.Layer, val, c.M2u, c.M2d, (*c.above)[0].M2d)
		}
	}
}

// Test whether the mixing mechanisms are properly conserving mass
func TestMixing(t *testing.T) {
	const (
		testTolerance = 1.e-8
		numTimesteps  = 5
	)

	cfg, ctmdata, pop, popIndices, mr, mortIndices := VarGridTestData()
	emis := NewEmissions()
	emis.Add(&EmisRecord{
		PM25: E,
		Geom: geom.LineString{
			geom.Point{X: -3999, Y: -3999.},
			geom.Point{X: -3500, Y: -3500.},
		},
	}) // ground level emissions

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	var m Mech
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(Mixing()),
			SteadyStateConvergenceCheck(numTimesteps, cfg.PopGridColumn, m, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	if err := d.Run(); err != nil {
		t.Error(err)
	}

	sum := 0.
	maxval := 0.
	for _, group := range []*cellList{d.cells, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary} {
		for _, cell := range *group {
			sum += cell.Cf[iPM2_5] * cell.Volume
			maxval = max(maxval, cell.Cf[iPM2_5])
		}
	}
	cells := d.Cells()
	expectedMass := cells[0].EmisFlux[iPM2_5] * cells[0].Volume * d.Dt * numTimesteps
	if different(sum, expectedMass, testTolerance) {
		t.Errorf("sum=%g (it should equal %g)\n", sum, expectedMass)
	}
	if !different(sum, maxval, testTolerance) {
		t.Error("All of the mass is in one cell--it didn't mix")
	}
}

// Test whether mass is conserved during advection.
func TestAdvection(t *testing.T) {
	const tolerance = 1.e-8

	cfg, ctmdata, pop, popIndices, mr, mortIndices := VarGridTestData()
	emis := NewEmissions()

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	var m Mech
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(UpwindAdvection()),
			SteadyStateConvergenceCheck(1, cfg.PopGridColumn, m, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	var cellGroups = []*cellList{d.cells, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary}

	for _, testCell := range d.Cells() {
		ResetCells()(d)

		// Add emissions
		testCell.Ci[0] += E / testCell.Dz / testCell.Dy / testCell.Dx
		testCell.Cf[0] += E / testCell.Dz / testCell.Dy / testCell.Dx
		// Calculate advection

		if err := d.Run(); err != nil {
			t.Error(err)
		}

		sum := 0.
		layerSum := make(map[int]float64)
		for _, cellGroup := range cellGroups {
			for _, c := range *cellGroup {
				val := c.Cf[0] * c.Dy * c.Dx * c.Dz
				if val < 0 {
					t.Fatalf("cell %v emis: negative concentration", testCell)
				}
				sum += val
				layerSum[c.Layer] += val
			}
		}
		if different(sum, E, tolerance) {
			t.Errorf("cell %v emis: sum=%.12g (it should equal %v)\n", testCell, sum, E)
		}
	}
}

// Test whether mass is conserved during meander mixing.
func TestMeanderMixing(t *testing.T) {
	const tolerance = 1.e-8
	nsteps := 10

	cfg, ctmdata, pop, popIndices, mr, mortIndices := VarGridTestData()
	emis := NewEmissions()

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	var m Mech
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(MeanderMixing()),
			SteadyStateConvergenceCheck(nsteps, cfg.PopGridColumn, m, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	var cellGroups = []*cellList{d.cells, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary}
	for _, testCell := range *d.cells {
		for _, group := range cellGroups {
			for _, c := range *group {
				c.Ci[0] = 0
				c.Cf[0] = 0
			}
		}
		ResetCells()(d)
		for tt := 0; tt < nsteps; tt++ {

			testCell.Ci[0] += E / testCell.Dz / testCell.Dy / testCell.Dx // ground level emissions
			testCell.Cf[0] += E / testCell.Dz / testCell.Dy / testCell.Dx // ground level emissions

			if err := d.Run(); err != nil {
				t.Error(err)
			}
		}
		sum := 0.
		layerSum := make(map[int]float64)
		for _, group := range cellGroups {
			for _, c := range *group {
				val := c.Cf[0] * c.Dy * c.Dx * c.Dz
				if val < 0 {
					t.Fatalf("cell %v emis: negative concentration", testCell)
				}
				sum += val
				layerSum[c.Layer] += val
			}
		}
		if different(sum, E*float64(nsteps), tolerance) {
			t.Errorf("cell %v emis: sum=%.12g (it should equal %v)\n", testCell, sum, E*float64(nsteps))
		}
	}
}

func absDifferent(a, b, tolerance float64) bool {
	if math.Abs(a-b) > tolerance {
		return true
	}
	return false
}
