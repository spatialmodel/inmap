// +build FORTRAN

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

package mosaic

import (
	"fmt"
	"log"
	"math"
	"testing"

	"net/http"
	_ "net/http/pprof"

	"github.com/ctessum/geom"
	"github.com/gonum/floats"
	"github.com/spatialmodel/inmap"
)

func TestRun(t *testing.T) {
	const testTolerance = 1.e-8
	const E = 1000000. // emissions

	if testing.Short() {
		return
	}

	go func() {
		log.Println(http.ListenAndServe("localhost:6061", nil))
	}()

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

	m := NewMechanism()
	drydep, err := m.DryDep("simple")
	if err != nil {
		t.Fatal(err)
	}
	wetdep, err := m.WetDep("emep")
	if err != nil {
		t.Fatal(err)
	}

	mutator, err := inmap.PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	chan1 := make(chan *inmap.SimulationStatus)
	chan2 := make(chan inmap.ConvergenceStatus)
	go func() {
		for {
			select {
			case m := <-chan1:
				fmt.Println(m)
			case m := <-chan2:
				fmt.Println(m)
			}
		}
	}()
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
			inmap.Log(chan1),
			inmap.SteadyStateConvergenceCheck(0, cfg.PopGridColumn, m, chan2),
		},
	}
	if err = d.Init(); err != nil {
		t.Error(err)
	}
	if err = d.Run(); err != nil {
		t.Error(err)
	}

	o, err := inmap.NewOutputter("", false, map[string]string{
		"TotalPM25":      "SO4+PNO3+Cl+NH4+PMSA+Aro1+Aro2+Alk1+Ole1+PApi1+PApi2+Lim1+Lim2+CO3+Na+Ca+Oin+OC+BC",
		"TotalPopDeaths": "coxHazard(loglogRR(TotalPM25), TotalPop, MortalityRate)",
	}, nil, m)
	if err != nil {
		t.Error(err)
	}

	r, err := d.Results(o)
	if err != nil {
		t.Error(err)
	}
	results := r["TotalPopDeaths"]
	totald := floats.Sum(results)
	const expectedDeaths = 3.695015627615384

	if different(totald, expectedDeaths, testTolerance) {
		t.Errorf("Deaths (%v) doesn't equal %v", totald, expectedDeaths)
	}
}

func TestDryDep(t *testing.T) {
	m := NewMechanism()
	_, err := m.DryDep("simple")
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.DryDep("XXX")
	if err == nil {
		t.Fatal("should be an error")
	}
}

func TestWetDep(t *testing.T) {
	m := NewMechanism()
	_, err := m.WetDep("emep")
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.WetDep("XXX")
	if err == nil {
		t.Fatal("should be an error")
	}
}

func TestUnits(t *testing.T) {
	m := NewMechanism()
	u, err := m.Units("ISOP")
	if err != nil {
		t.Error(err)
	}
	if u != "ppb" {
		t.Errorf("want: 'ppb'; have '%s'", u)
	}
	u, err = m.Units("BC")
	if err != nil {
		t.Error(err)
	}
	if u != "μg/m³" {
		t.Errorf("want: 'μg/m³'; have '%s'", u)
	}
	_, err = m.Units("xxxx")
	if err == nil {
		t.Error("should be an error")
	}
}

func different(a, b, tolerance float64) bool {
	if 2*math.Abs(a-b)/math.Abs(a+b) > tolerance || math.IsNaN(a) || math.IsNaN(b) {
		return true
	}
	return false
}
