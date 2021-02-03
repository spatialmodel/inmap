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

package inmap_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
)

func TestDynamicGrid(t *testing.T) {
	const (
		testTolerance      = 1.e-8
		gridMutateInterval = 3600. // interval between grid mutations in seconds.
	)

	cfg, ctmdata, pop, popIndices, mr, mortIndices := inmap.VarGridTestData()
	emis := inmap.NewEmissions()
	emis.Add(&inmap.EmisRecord{
		SOx:  E,
		NOx:  E,
		PM25: E,
		VOC:  E,
		NH3:  E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions

	popConcMutator := inmap.NewPopConcMutator(cfg, popIndices)
	var m simplechem.Mechanism
	drydep, err := m.DryDep("simple")
	if err != nil {
		t.Fatal(err)
	}
	wetdep, err := m.WetDep("emep")
	if err != nil {
		t.Fatal(err)
	}

	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			inmap.SetTimestepCFL(),
		},
		RunFuncs: []inmap.DomainManipulator{
			inmap.Calculations(inmap.AddEmissionsFlux()),
			inmap.Calculations(
				inmap.UpwindAdvection(),
				inmap.Mixing(),
				inmap.MeanderMixing(),
				drydep,
				wetdep,
				m.Chemistry(),
			),
			inmap.RunPeriodically(gridMutateInterval,
				cfg.MutateGrid(popConcMutator.Mutate(), ctmdata, pop, mr, emis, m, nil)),
			inmap.RunPeriodically(gridMutateInterval, inmap.SetTimestepCFL()),
			inmap.SteadyStateConvergenceCheck(-1, cfg.PopGridColumn, m, nil),
		},
	}

	if err = d.Init(); err != nil {
		t.Error(err)
	}
	if err = d.Run(); err != nil {
		t.Error(err)
	}

	cells := make([]int, 10)
	for _, c := range d.Cells() {
		cells[c.Layer]++
	}

	wantCells := []int{16, 16, 16, 16, 16, 16, 16, 16, 13, 4}
	if !reflect.DeepEqual(cells, wantCells) {
		t.Errorf("dynamic grid should have %v cells but instead has %v", wantCells, cells)
	}

	o, err := inmap.NewOutputter("", false, map[string]string{
		"TotalPopD": "(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * AllCause / 100000",
		"Latino":    "Latino", "LatinoMort": "LatinoMort"}, nil, m)
	if err != nil {
		t.Error(err)
	}

	r, err := d.Results(o)
	if err != nil {
		t.Error(err)
	}
	deaths := r["TotalPopD"]
	latino := r["Latino"]
	latinoMort := r["LatinoMort"]
	expectedDeaths := []float64{17.062006521104145, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	expectedLatinoPops := []float64{25000.00000157229, 4999.999998427711, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	expectedLatinoMorts := []float64{480.00000002012524, 800, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	for i := 0; i < wantCells[0]; i++ {
		if different(deaths[i], expectedDeaths[i], testTolerance) {
			t.Errorf("Deaths (%v) doesn't equal %v", deaths[i], expectedDeaths[i])
		}
		if different(latino[i], expectedLatinoPops[i], testTolerance) {
			t.Errorf("Latino population (%v) doesn't equal %v", latino[i], expectedLatinoPops[i])
		}
		if different(latinoMort[i], expectedLatinoMorts[i], testTolerance) {
			t.Errorf("Latino mortality rate (%v) doesn't equal %v", latinoMort[i], expectedLatinoMorts[i])
		}
	}
}

func different(a, b, tolerance float64) bool {
	if 2*math.Abs(a-b)/math.Abs(a+b) > tolerance || math.IsNaN(a) || math.IsNaN(b) {
		return true
	}
	return false
}
