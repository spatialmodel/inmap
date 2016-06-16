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
	"math"
	"testing"
	"time"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
	"github.com/gonum/floats"
)

const E = 1000000. // emissions

// Tests whether the cells correctly reference each other
func TestCellAlignment(t *testing.T) {

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	d.testCellAlignment2(t)
}

func (d *InMAP) testCellAlignment2(t *testing.T) {
	const testTolerance = 1.e-8
	for _, cell := range d.Cells {
		for i, w := range cell.west {
			if !w.boundary && len(w.east) != 0 {
				pass := false
				for j, e := range w.east {
					if e == cell {
						pass = true
						if different(w.kxxEast[j], cell.kxxWest[i], testTolerance) {
							t.Logf("Kxx doesn't match")
							t.FailNow()
						}
						if different(w.dxPlusHalf[j], cell.dxMinusHalf[i], testTolerance) {
							t.Logf("Dx doesn't match")
							t.FailNow()
						}
						break
					}
				}
				if !pass {
					t.Logf("Failed for Cell %v (layer %v) West",
						cell.Polygonal, cell.Layer)
					t.FailNow()
				}
			}
		}
		for i, e := range cell.east {
			if !e.boundary && len(e.west) != 0 {
				pass := false
				for j, w := range e.west {
					if w == cell {
						pass = true
						if different(e.kxxWest[j], cell.kxxEast[i], testTolerance) {
							t.Logf("Kxx doesn't match")
							t.FailNow()
						}
						if different(e.dxMinusHalf[j], cell.dxPlusHalf[i], testTolerance) {
							t.Logf("Dx doesn't match")
							t.FailNow()
						}
						break
					}
				}
				if !pass {
					t.Logf("Failed for Cell %v (layer %v) East",
						cell.Polygonal, cell.Layer)
					t.FailNow()
				}
			}
		}
		for i, n := range cell.north {
			if !n.boundary && len(n.south) != 0 {
				pass := false
				for j, s := range n.south {
					if s == cell {
						pass = true
						if different(n.kyySouth[j], cell.kyyNorth[i], testTolerance) {
							t.Logf("Kyy doesn't match")
							t.FailNow()
						}
						if different(n.dyMinusHalf[j], cell.dyPlusHalf[i], testTolerance) {
							t.Logf("Dy doesn't match")
							t.FailNow()
						}
						break
					}
				}
				if !pass {
					t.Logf("Failed for Cell %v (layer %v) North",
						cell.Polygonal, cell.Layer)
					t.FailNow()
				}
			}
		}
		for i, s := range cell.south {
			if !s.boundary && len(s.north) != 0 {
				pass := false
				for j, n := range s.north {
					if n == cell {
						pass = true
						if different(s.kyyNorth[j], cell.kyySouth[i], testTolerance) {
							t.Logf("Kyy doesn't match")
							t.FailNow()
						}
						if different(s.dyPlusHalf[j], cell.dyMinusHalf[i], testTolerance) {
							t.Logf("Dy doesn't match")
							t.FailNow()
						}
						break
					}
				}
				if !pass {
					t.Logf("Failed for Cell %v (layer %v) South",
						cell.Polygonal, cell.Layer)
					t.FailNow()
				}
			}
		}
		for i, a := range cell.above {
			if !a.boundary && len(a.below) != 0 {
				pass := false
				for j, b := range a.below {
					if b == cell {
						pass = true
						if different(a.kzzBelow[j], cell.kzzAbove[i], testTolerance) {
							t.Logf("Kzz doesn't match above (layer=%v, "+
								"KzzAbove=%v, KzzBelow=%v)", cell.Layer,
								cell.kzzAbove[i], a.kzzBelow[j])
							t.Fail()
						}
						if different(a.dzMinusHalf[j], cell.dzPlusHalf[i], testTolerance) {
							t.Logf("Dz doesn't match")
							t.FailNow()
						}
						break
					}
				}
				if !pass {
					t.Logf("Failed for Cell %v (layer %v) Above",
						cell.Polygonal, cell.Layer)
					t.FailNow()
				}
			}
		}
		for i, b := range cell.below {
			pass := false
			if cell.Layer == 0 && b == cell {
				pass = true
			} else if len(b.above) != 0 {
				for j, a := range b.above {
					if a == cell {
						pass = true
						if different(b.kzzAbove[j], cell.kzzBelow[i], testTolerance) {
							t.Logf("Kzz doesn't match below")
							t.FailNow()
						}
						if different(b.dzPlusHalf[j], cell.dzMinusHalf[i], testTolerance) {
							t.Logf("Dz doesn't match")
							t.FailNow()
						}
						break
					}
				}
			} else {
				pass = true
			}
			if !pass {
				t.Logf("Failed for Cell %v (layer %v) Below",
					cell, cell.Layer)
				t.FailNow()
			}
		}
		// Assume upper cells are never higher resolution than lower cells
		for _, g := range cell.groundLevel {
			g2 := g
			pass := false
			for {
				if len(g2.above) == 0 {
					pass = false
					break
				}
				if g2 == g2.above[0] {
					pass = false
					break
				}
				if g2 == cell {
					pass = true
					break
				}
				g2 = g2.above[0]
			}
			if !pass {
				t.Logf("Failed for Cell %v (layer %v) GroundLevel",
					cell.Polygonal, cell.Layer)
				t.FailNow()
			}
		}
	}
}

// Test whether convective mixing coefficients are balanced in
// a way that conserves mass
func TestConvectiveMixing(t *testing.T) {
	const testTolerance = 1.e-8

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	for i, c := range d.Cells {
		val := c.M2u - c.M2d + c.above[0].M2d*c.above[0].Dz/c.Dz
		if absDifferent(val, 0, testTolerance) {
			t.Error(i, c.Layer, val, c.M2u, c.M2d, c.above[0].M2d)
		}
	}
}

// Test whether the mixing mechanisms are properly conserving mass
func TestMixing(t *testing.T) {
	const (
		testTolerance = 1.e-8
		numTimesteps  = 5
	)

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}
	emis.data.Insert(emisRecord{
		PM25: E,
		Geom: geom.LineString{
			geom.Point{X: -3999, Y: -3999.},
			geom.Point{X: -3500, Y: -3500.},
		},
	}) // ground level emissions

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(Mixing()),
			SteadyStateConvergenceCheck(numTimesteps, nil),
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
	for _, group := range [][]*Cell{d.Cells, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary} {
		for _, cell := range group {
			sum += cell.Cf[iPM2_5] * cell.Volume
			maxval = max(maxval, cell.Cf[iPM2_5])
		}
	}
	expectedMass := d.Cells[0].EmisFlux[iPM2_5] * d.Cells[0].Volume * d.Dt * numTimesteps
	if different(sum, expectedMass, testTolerance) {
		t.Errorf("sum=%g (it should equal %g)\n", sum, expectedMass)
	}
	if !different(sum, maxval, testTolerance) {
		t.Error("All of the mass is in one cell--it didn't mix")
	}
}

// Test whether mass is conserved during chemical reactions.
func TestChemistry(t *testing.T) {
	const (
		testTolerance = 1.e-8
		testRow       = 2
	)
	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}
	emis.data.Insert(emisRecord{
		SOx:  E,
		NOx:  E,
		PM25: E,
		VOC:  E,
		NH3:  E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(Chemistry()),
			SteadyStateConvergenceCheck(1, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	if err := d.Run(); err != nil {
		t.Error(err)
	}

	c := d.Cells[0]
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
		t.Error("different")
	}
}

// Test whether mass is conserved during advection.
func TestAdvection(t *testing.T) {
	const tolerance = 1.e-8

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(UpwindAdvection()),
			SteadyStateConvergenceCheck(1, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	var cellGroups = [][]*Cell{d.Cells, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary}

	for testRow := 0; testRow < len(d.Cells); testRow++ {
		ResetCells()(d)

		// Add emissions
		c := d.Cells[testRow]
		c.Ci[0] += E / c.Dz / c.Dy / c.Dx
		c.Cf[0] += E / c.Dz / c.Dy / c.Dx
		// Calculate advection

		if err := d.Run(); err != nil {
			t.Error(err)
		}

		sum := 0.
		layerSum := make(map[int]float64)
		for _, cellGroup := range cellGroups {
			for _, c := range cellGroup {
				val := c.Cf[0] * c.Dy * c.Dx * c.Dz
				if val < 0 {
					t.Fatalf("row %d emis: negative concentration", testRow)
				}
				sum += val
				layerSum[c.Layer] += val
			}
		}
		if different(sum, E, tolerance) {
			t.Errorf("row %d emis: sum=%.12g (it should equal %v)\n", testRow, sum, E)
		}
	}
}

// Test whether mass is conserved during meander mixing.
func TestMeanderMixing(t *testing.T) {
	const tolerance = 1.e-8
	nsteps := 10

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(MeanderMixing()),
			SteadyStateConvergenceCheck(nsteps, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	var cellGroups = [][]*Cell{d.Cells, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary}
	// Test emissions from every thirtieth row.
	for testRow := 0; testRow < len(d.Cells); testRow++ {
		for _, group := range cellGroups {
			for _, c := range group {
				c.Ci[0] = 0
				c.Cf[0] = 0
			}
		}
		ResetCells()(d)
		for tt := 0; tt < nsteps; tt++ {

			c := d.Cells[testRow]
			c.Ci[0] += E / c.Dz / c.Dy / c.Dx // ground level emissions
			c.Cf[0] += E / c.Dz / c.Dy / c.Dx // ground level emissions

			if err := d.Run(); err != nil {
				t.Error(err)
			}
		}
		sum := 0.
		layerSum := make(map[int]float64)
		for _, group := range cellGroups {
			for _, c := range group {
				val := c.Cf[0] * c.Dy * c.Dx * c.Dz
				if val < 0 {
					t.Fatalf("row %d emis: negative concentration", testRow)
				}
				sum += val
				layerSum[c.Layer] += val
			}
		}
		if different(sum, E*float64(nsteps), tolerance) {
			t.Errorf("row %d emis: sum=%.12g (it should equal %v)\n", testRow, sum, E*float64(nsteps))
		}
	}
}

func TestConverge(t *testing.T) {
	const (
		testTolerance = 1.e-8
		timeout       = 10 * time.Second
	)

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}
	emis.data.Insert(emisRecord{
		SOx:  E,
		NOx:  E,
		PM25: E,
		VOC:  E,
		NH3:  E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions

	convergences := []DomainManipulator{SteadyStateConvergenceCheck(2, nil), SteadyStateConvergenceCheck(-1, nil)}
	convergenceNames := []string{"fixed", "criterion"}
	expectedConcentration := []float64{0.46486263752954793, 83.22031969811773}

	for i, conv := range convergences {

		iterations := 0

		d := &InMAP{
			InitFuncs: []DomainManipulator{
				cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
				SetTimestepCFL(),
			},
			RunFuncs: []DomainManipulator{
				Calculations(AddEmissionsFlux()),
				Calculations(
					DryDeposition(),
					WetDeposition(),
				),
				conv,
				func(_ *InMAP) error {
					iterations++
					return nil
				},
			},
		}
		if err := d.Init(); err != nil {
			t.Error(err)
		}
		timeoutChan := time.After(timeout)
		doneChan := make(chan int)
		go func() {
			if err := d.Run(); err != nil {
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

		r, err := d.Results(false, "Primary PM2.5")
		if err != nil {
			t.Error(err)
		}
		results := r["Primary PM2.5"][0]
		total := floats.Sum(results)
		if different(total, expectedConcentration[i], testTolerance) {
			t.Errorf("%s concentration (%v) doesn't equal %v", convergenceNames[i], total, expectedConcentration[i])
		}
	}
}

func BenchmarkRun(b *testing.B) {
	const testTolerance = 1.e-8

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}
	emis.data.Insert(emisRecord{
		SOx:  E,
		NOx:  E,
		PM25: E,
		VOC:  E,
		NH3:  E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
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
			SteadyStateConvergenceCheck(1000, nil),
		},
	}
	if err := d.Init(); err != nil {
		b.Error(err)
	}
	if err := d.Run(); err != nil {
		b.Error(err)
	}

	r, err := d.Results(false, "TotalPop deaths")
	if err != nil {
		b.Error(err)
	}
	results := r["TotalPop deaths"][0]
	totald := floats.Sum(results)
	const expectedDeaths = 1.1582659761054755e-06
	if different(totald, expectedDeaths, testTolerance) {
		b.Errorf("Deaths (%v) doesn't equal %v", totald, expectedDeaths)
	}
}

func TestDryDeposition(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(DryDeposition()),
			SteadyStateConvergenceCheck(1, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	for _, c := range d.Cells {
		for i := range c.Ci {
			c.Cf[i] = 1 // set concentrations to 1
		}
	}
	if err := d.Run(); err != nil {
		t.Error(err)
	}

	for i, c := range d.Cells {
		for ii, cc := range c.Cf {
			if c.Layer == 0 {
				if cc >= 1 || cc <= 0.98 {
					t.Errorf("ground-level cell %d pollutant %d should equal be between 0.98 and 1 but is %g", i, ii, cc)
				}
			} else if cc != 1 {
				t.Errorf("above-ground cell %d pollutant %d should equal 1 but equals %g", i, ii, cc)
			}
		}
	}
}

func TestWetDeposition(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(WetDeposition()),
			SteadyStateConvergenceCheck(1, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	for _, c := range d.Cells {
		for i := range c.Ci {
			c.Cf[i] = 1 // set concentrations to 1
		}
	}
	if err := d.Run(); err != nil {
		t.Error(err)
	}

	for i, c := range d.Cells {
		for ii, cc := range c.Cf {
			if cc > 1 || cc <= 0.99 {
				t.Errorf("ground-level cell %d pollutant %d should equal be between 0.99 and 1 but is %g", i, ii, cc)
			}
		}
	}
}

func different(a, b, tolerance float64) bool {
	if 2*math.Abs(a-b)/math.Abs(a+b) > tolerance || math.IsNaN(a) || math.IsNaN(b) {
		return true
	}
	return false
}

func absDifferent(a, b, tolerance float64) bool {
	if math.Abs(a-b) > tolerance {
		return true
	}
	return false
}
