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
	"testing"
	"time"

	"github.com/ctessum/geom"
	"github.com/gonum/floats"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
)

const E = 1000000. // emissions

func TestConverge(t *testing.T) {
	const (
		testTolerance = 1.e-8
		timeout       = 10 * time.Second
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

	var m simplechem.Mechanism
	convergences := []inmap.DomainManipulator{inmap.SteadyStateConvergenceCheck(2, cfg.PopGridColumn, m, nil),
		inmap.SteadyStateConvergenceCheck(-1, cfg.PopGridColumn, m, nil)}
	convergenceNames := []string{"fixed", "criterion"}
	expectedConcentration := []float64{0.3489647639076775, 83.89644268259369}

	for i, conv := range convergences {

		iterations := 0
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
					drydep,
					wetdep,
				),
				conv,
				func(_ *inmap.InMAP) error {
					iterations++
					return nil
				},
			},
		}
		if err = d.Init(); err != nil {
			t.Error(err)
		}
		timeoutChan := time.After(timeout)
		doneChan := make(chan int)
		go func() {
			if err = d.Run(); err != nil {
				t.Error(err)
			}
			doneChan <- 0
		}()
		select {
		case <-timeoutChan:
			t.Errorf("%s timed out after %d iterations.", convergenceNames[i], iterations)
		case <-doneChan:
			t.Logf("%s completed after %d iterations.", convergenceNames[i], iterations)
		}

		o, err := inmap.NewOutputter("", false, map[string]string{"PrimPM25": "PrimaryPM25"}, nil, m)
		if err != nil {
			t.Error(err)
		}

		r, err := d.Results(o)
		if err != nil {
			t.Error(err)
		}
		results := r["PrimPM25"]
		total := floats.Sum(results)
		if different(total, expectedConcentration[i], testTolerance) {
			t.Errorf("%s concentration (%v) doesn't equal %v", convergenceNames[i], total, expectedConcentration[i])
		}
	}
}

func BenchmarkRun(b *testing.B) {
	const testTolerance = 1.e-8

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

	var m simplechem.Mechanism
	drydep, err := m.DryDep("simple")
	if err != nil {
		b.Fatal(err)
	}
	wetdep, err := m.WetDep("emep")
	if err != nil {
		b.Fatal(err)
	}

	mutator, err := inmap.PopulationMutator(cfg, popIndices)
	if err != nil {
		b.Error(err)
	}
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
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
			inmap.SteadyStateConvergenceCheck(1000, cfg.PopGridColumn, m, nil),
		},
	}
	if err = d.Init(); err != nil {
		b.Error(err)
	}
	if err = d.Run(); err != nil {
		b.Error(err)
	}

	o, err := inmap.NewOutputter("", false, map[string]string{"TotalPopDeaths": "(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * MortalityRate / 100000"}, nil, m)
	if err != nil {
		b.Error(err)
	}

	r, err := d.Results(o)
	if err != nil {
		b.Error(err)
	}
	results := r["TotalPopDeaths"]
	totald := floats.Sum(results)
	const expectedDeaths = 6.614182415997713e-06

	if different(totald, expectedDeaths, testTolerance) {
		b.Errorf("Deaths (%v) doesn't equal %v", totald, expectedDeaths)
	}
}

// TestBigM2d checks whether the model can run stably with a high rate of
// convective mixing.
func TestBigM2d(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr, mortIndices := inmap.VarGridTestData()
	ctmdata.Data["M2d"].Data.Scale(100)
	ctmdata.Data["M2u"].Data.Scale(100)

	emis := inmap.NewEmissions()
	emis.Add(&inmap.EmisRecord{
		SOx:  E,
		NOx:  E,
		PM25: E,
		VOC:  E,
		NH3:  E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions
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
			inmap.SteadyStateConvergenceCheck(-1, cfg.PopGridColumn, m, nil),
		},
	}
	if err = d.Init(); err != nil {
		t.Error(err)
	}
	if err = d.Run(); err != nil {
		t.Error(err)
	}

	o, err := inmap.NewOutputter("", false, map[string]string{"TotalPM25": "TotalPM25"}, nil, m)
	if err != nil {
		t.Error(err)
	}

	r, err := d.Results(o)
	if err != nil {
		t.Error(err)
	}
	results := r["TotalPM25"]
	sum := floats.Sum(results)
	if math.IsNaN(sum) {
		t.Errorf("concentration sum is NaN")
	}
}

// Tests whether the cells correctly reference each other
func TestCellAlignment(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr, mortIndices := inmap.VarGridTestData()
	emis := inmap.NewEmissions()
	mutator, err := inmap.PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	var m simplechem.Mechanism

	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	d.TestCellAlignment2(t)
}
