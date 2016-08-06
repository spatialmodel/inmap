/*
Copyright © 2013 the InMAP authors.
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
	"reflect"
	"testing"

	"github.com/ctessum/geom"
	"github.com/gonum/floats"
)

func TestDynamicGrid(t *testing.T) {
	const (
		testTolerance      = 1.e-8
		gridMutateInterval = 3600. // interval between grid mutations in seconds.
	)

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := NewEmissions()
	emis.Add(&EmisRecord{
		SOx:  E,
		NOx:  E,
		PM25: E,
		VOC:  E,
		NH3:  E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions

	popConcMutator := NewPopConcMutator(cfg, popIndices)

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(
				UpwindAdvection(),
				Mixing(),
				MeanderMixing(),
				DryDeposition(),
				WetDeposition(),
				Chemistry(),
			),
			RunPeriodically(gridMutateInterval,
				cfg.MutateGrid(popConcMutator.Mutate(),
					ctmdata, pop, mr, emis, nil)),
			RunPeriodically(gridMutateInterval, SetTimestepCFL()),
			SteadyStateConvergenceCheck(-1, cfg.PopGridColumn, nil),
			popConcMutator.AdjustThreshold(cfg.PopConcThreshold, nil),
		},
	}

	if err := d.Init(); err != nil {
		t.Error(err)
	}
	if err := d.Run(); err != nil {
		t.Error(err)
	}

	cells := make([]int, d.nlayers)
	for _, c := range *d.cells {
		cells[c.Layer]++
	}

	wantCells := []int{10, 10, 10, 16, 16, 16, 16, 16, 13, 4}
	if !reflect.DeepEqual(cells, wantCells) {
		t.Errorf("dynamic grid should have %v cells but instead has %v", wantCells, cells)
	}

	r, err := d.Results(false, "TotalPop deaths")
	if err != nil {
		t.Error(err)
	}
	results := r["TotalPop deaths"]
	totald := floats.Sum(results)
	const expectedDeaths = 1.4976024171896891e-05
	if different(totald, expectedDeaths, testTolerance) {
		t.Errorf("Deaths (%v) doesn't equal %v", totald, expectedDeaths)
	}
}
