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

package simplechem

import (
	"math"
	"testing"

	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap"
)

const E = 1000000. // emissions

// Test whether mass is conserved during chemical reactions.
func TestChemistry(t *testing.T) {
	const (
		testTolerance = 1.e-8
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

	mutator, err := inmap.PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	m := Mechanism{}
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
			inmap.SetTimestepCFL(),
		},
		RunFuncs: []inmap.DomainManipulator{
			inmap.Calculations(inmap.AddEmissionsFlux()),
			inmap.Calculations(m.Chemistry()),
			inmap.SteadyStateConvergenceCheck(1, cfg.PopGridColumn, m, nil),
		},
	}
	if err = d.Init(); err != nil {
		t.Error(err)
	}
	if err = d.Run(); err != nil {
		t.Error(err)
	}

	c := d.Cells()[0]
	sum := 0.
	sum += c.Cf[igOrg] + c.Cf[ipOrg]
	sum += (c.Cf[igNO] + c.Cf[ipNO]) / NOxToN
	sum += (c.Cf[igNH] + c.Cf[ipNH]) / NH3ToN
	sum += (c.Cf[igS] + c.Cf[ipS]) / SOxToS
	sum += c.Cf[iPM2_5]
	sum *= c.Volume

	if c.Cf[ipOrg] == 0 || c.Cf[ipS] == 0 || c.Cf[ipNH] == 0 || c.Cf[ipNO] == 0 {
		t.Error("chemistry appears not to have occured")
	}
	if different(sum, 5*E*d.Dt, testTolerance) {
		t.Errorf("different: %g != %g", sum, 5*E*d.Dt)
	}

	v, err := m.Value(c, "SOxEmissions")
	if err != nil {
		t.Error(err)
	}
	want := E * SOxToS / c.Dx / c.Dy / c.Dz
	if v != want {
		t.Errorf("have %g, want %g", v, want)
	}
	v, err = m.Value(c, "TotalPM25")
	if err != nil {
		t.Error(err)
	}
	want = 2.706460494064197
	if v != want {
		t.Errorf("have %g, want %g", v, want)
	}
	_, err = m.Value(c, "xxxxx")
	if err == nil {
		t.Error("should be an error")
	}

}

func TestDryDep(t *testing.T) {
	m := Mechanism{}
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
	m := Mechanism{}
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
	m := Mechanism{}
	u, err := m.Units("VOCEmissions")
	if err != nil {
		t.Error(err)
	}
	if u != "μg/m³/s" {
		t.Errorf("want: 'μg/m³/s'; have '%s'", u)
	}
	u, err = m.Units("SOA")
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
