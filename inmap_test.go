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
	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	d.testCellAlignment2(t)
}

func (d *InMAP) testCellAlignment2(t *testing.T) {
	const testTolerance = 1.e-8
	for _, cell := range *d.cells {
		var westCoverage, eastCoverage, northCoverage, southCoverage float64
		var aboveCoverage, belowCoverage, groundLevelCoverage float64
		for _, w := range *cell.west {
			westCoverage += w.info.coverFrac
			if !w.boundary {
				pass := false
				for _, e := range *w.east {
					if e.Cell == cell.Cell {
						pass = true
						if different(w.info.diff, e.info.diff, testTolerance) {
							t.Errorf("Kxx doesn't match")
						}
						if different(w.info.centerDistance, e.info.centerDistance, testTolerance) {
							t.Errorf("Dx doesn't match")
							break
						}
					}
				}
				if !pass {
					t.Errorf("Failed for Cell %v West", cell)
				}
			}
		}
		for _, e := range *cell.east {
			eastCoverage += e.info.coverFrac
			if !e.boundary {
				pass := false
				for _, w := range *e.west {
					if w.Cell == cell.Cell {
						pass = true
						if different(e.info.diff, w.info.diff, testTolerance) {
							t.Errorf("Kxx doesn't match")
						}
						if different(e.info.centerDistance, w.info.centerDistance, testTolerance) {
							t.Errorf("Dx doesn't match")
						}
						break
					}
				}
				if !pass {
					t.Errorf("Failed for Cell %v East", cell)
				}
			}
		}
		for _, n := range *cell.north {
			northCoverage += n.info.coverFrac
			if !n.boundary {
				pass := false
				for _, s := range *n.south {
					if s.Cell == cell.Cell {
						pass = true
						if different(n.info.diff, s.info.diff, testTolerance) {
							t.Errorf("Kyy doesn't match")
						}
						if different(n.info.centerDistance, s.info.centerDistance, testTolerance) {
							t.Errorf("Dy doesn't match")
						}
						break
					}
				}
				if !pass {
					t.Errorf("Failed for Cell %v  North", cell)
				}
			}
		}
		for _, s := range *cell.south {
			southCoverage += s.info.coverFrac
			if !s.boundary {
				pass := false
				for _, n := range *s.north {
					if n.Cell == cell.Cell {
						pass = true
						if different(s.info.diff, n.info.diff, testTolerance) {
							t.Errorf("Kyy doesn't match")
						}
						if different(s.info.centerDistance, n.info.centerDistance, testTolerance) {
							t.Errorf("Dy doesn't match")
						}
						break
					}
				}
				if !pass {
					t.Errorf("Failed for Cell %v South", cell)
				}
			}
		}
		for _, a := range *cell.above {
			aboveCoverage += a.info.coverFrac
			if !a.boundary {
				pass := false
				for _, b := range *a.below {
					if b.Cell == cell.Cell {
						pass = true
						if different(a.info.diff, b.info.diff, testTolerance) {
							t.Errorf("Kzz doesn't match above (layer=%v, "+
								"KzzAbove=%v, KzzBelow=%v)", cell.Layer,
								b.info.diff, a.info.diff)
						}
						if different(a.info.centerDistance, b.info.centerDistance, testTolerance) {
							t.Errorf("Dz doesn't match")
						}
						break
					}
				}
				if !pass {
					t.Errorf("Failed for Cell %v Above", cell)
				}
			}
		}
		for _, b := range *cell.below {
			belowCoverage += b.info.coverFrac
			pass := false
			if cell.Layer == 0 && b.Cell == cell.Cell {
				pass = true
			} else {
				for _, a := range *b.above {
					if a.Cell == cell.Cell {
						pass = true
						if different(b.info.diff, a.info.diff, testTolerance) {
							t.Errorf("Kzz doesn't match below")
						}
						if different(b.info.centerDistance, a.info.centerDistance, testTolerance) {
							t.Errorf("Dz doesn't match")
						}
						break
					}
				}
			}
			if !pass {
				t.Errorf("Failed for Cell %v  Below", cell)
			}
		}
		// Assume upper cells are never higher resolution than lower cells
		for _, g := range *cell.groundLevel {
			groundLevelCoverage += g.info.coverFrac
			g2 := g
			pass := false
			for {
				if g2.above.len() == 0 {
					pass = false
					break
				}
				if g2.Cell == (*g2.above)[0].Cell {
					pass = false
					break
				}
				if g2.Cell == cell.Cell {
					pass = true
					break
				}
				g2 = (*g2.above)[0]
			}
			if !pass {
				t.Errorf("Failed for Cell %v GroundLevel", cell)
			}
		}
		const tolerance = 1.0e-10
		if different(westCoverage, 1, tolerance) {
			t.Errorf("cell %v, west coverage %g!=1", cell, westCoverage)
		}
		if different(eastCoverage, 1, tolerance) {
			t.Errorf("cell %v, east coverage %g!=1", cell, eastCoverage)
		}
		if different(southCoverage, 1, tolerance) {
			t.Errorf("cell %v, south coverage %g!=1", cell, southCoverage)
		}
		if different(northCoverage, 1, tolerance) {
			t.Errorf("cell %v, north coverage %g!=1", cell, northCoverage)
		}
		if different(belowCoverage, 1, tolerance) {
			t.Errorf("cell %v, below coverage %g!=1", cell, belowCoverage)
		}
		if different(aboveCoverage, 1, tolerance) {
			t.Errorf("cell %v, above coverage %g!=1", cell, aboveCoverage)
		}
		if different(groundLevelCoverage, 1, tolerance) {
			t.Errorf("cell %v, groundLevel coverage %g!=1", cell, groundLevelCoverage)
		}
	}
}

// Test whether convective mixing coefficients are balanced in
// a way that conserves mass
func TestConvectiveMixing(t *testing.T) {
	const testTolerance = 1.e-8

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := NewEmissions()

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	for _, c := range *d.cells {
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

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}
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
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(Mixing()),
			SteadyStateConvergenceCheck(numTimesteps, cfg.PopGridColumn, nil),
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
	cells := d.cells.array()
	expectedMass := cells[0].EmisFlux[iPM2_5] * cells[0].Volume * d.Dt * numTimesteps
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
	)
	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}
	emis.data.Insert(&EmisRecord{
		SOx:  E,
		NOx:  E,
		PM25: E,
		VOC:  E,
		NH3:  E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(Chemistry()),
			SteadyStateConvergenceCheck(1, cfg.PopGridColumn, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	if err := d.Run(); err != nil {
		t.Error(err)
	}

	c := d.cells.array()[0]
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

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(UpwindAdvection()),
			SteadyStateConvergenceCheck(1, cfg.PopGridColumn, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	var cellGroups = []*cellList{d.cells, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary}

	for _, testCell := range *d.cells {
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

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(MeanderMixing()),
			SteadyStateConvergenceCheck(nsteps, cfg.PopGridColumn, nil),
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

func TestConverge(t *testing.T) {
	const (
		testTolerance = 1.e-8
		timeout       = 10 * time.Second
	)

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}
	emis.data.Insert(&EmisRecord{
		SOx:  E,
		NOx:  E,
		PM25: E,
		VOC:  E,
		NH3:  E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions

	convergences := []DomainManipulator{SteadyStateConvergenceCheck(2, cfg.PopGridColumn, nil),
		SteadyStateConvergenceCheck(-1, cfg.PopGridColumn, nil)}
	convergenceNames := []string{"fixed", "criterion"}
	expectedConcentration := []float64{0.46486263752954793, 83.7425598603494}

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
		results := r["Primary PM2.5"]
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
	emis.Add(&EmisRecord{
		SOx:  E,
		NOx:  E,
		PM25: E,
		VOC:  E,
		NH3:  E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		b.Error(err)
	}
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
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
			SteadyStateConvergenceCheck(1000, cfg.PopGridColumn, nil),
		},
	}
	if err = d.Init(); err != nil {
		b.Error(err)
	}
	if err = d.Run(); err != nil {
		b.Error(err)
	}

	r, err := d.Results(false, "TotalPop deaths")
	if err != nil {
		b.Error(err)
	}
	results := r["TotalPop deaths"]
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

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(DryDeposition()),
			SteadyStateConvergenceCheck(1, cfg.PopGridColumn, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	for _, c := range *d.cells {
		for i := range c.Ci {
			c.Cf[i] = 1 // set concentrations to 1
		}
	}
	if err := d.Run(); err != nil {
		t.Error(err)
	}

	for _, c := range *d.cells {
		for ii, cc := range c.Cf {
			if c.Layer == 0 {
				if cc >= 1 || cc <= 0.98 {
					t.Errorf("ground-level cell %v pollutant %d should equal be between 0.98 and 1 but is %g", c, ii, cc)
				}
			} else if cc != 1 {
				t.Errorf("above-ground cell %v pollutant %d should equal 1 but equals %g", c, ii, cc)
			}
		}
	}
}

func TestWetDeposition(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(WetDeposition()),
			SteadyStateConvergenceCheck(1, cfg.PopGridColumn, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	for _, c := range *d.cells {
		for i := range c.Ci {
			c.Cf[i] = 1 // set concentrations to 1
		}
	}
	if err := d.Run(); err != nil {
		t.Error(err)
	}

	for _, c := range *d.cells {
		for ii, cc := range c.Cf {
			if cc > 1 || cc <= 0.99 {
				t.Errorf("ground-level cell %v pollutant %d should equal be between 0.99 and 1 but is %g", c, ii, cc)
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
