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
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
	"github.com/gonum/floats"
)

const E = 0.01 // emissions

// Tests whether the cells correctly reference each other
func TestCellAlignment(t *testing.T) {
	const testTolerance = 1.e-8

	cfg, ctmdata, pop, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAPdata{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, mr, emis),
			cfg.StaticVariableGrid(ctmdata, pop, mr, emis),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	for _, cell := range d.Cells {
		for i, w := range cell.West {
			if !w.Boundary && len(w.East) != 0 {
				pass := false
				for j, e := range w.East {
					if e == cell {
						pass = true
						if different(w.KxxEast[j], cell.KxxWest[i], testTolerance) {
							t.Logf("Kxx doesn't match")
							t.FailNow()
						}
						if different(w.DxPlusHalf[j], cell.DxMinusHalf[i], testTolerance) {
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
		for i, e := range cell.East {
			if !e.Boundary && len(e.West) != 0 {
				pass := false
				for j, w := range e.West {
					if w == cell {
						pass = true
						if different(e.KxxWest[j], cell.KxxEast[i], testTolerance) {
							t.Logf("Kxx doesn't match")
							t.FailNow()
						}
						if different(e.DxMinusHalf[j], cell.DxPlusHalf[i], testTolerance) {
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
		for i, n := range cell.North {
			if !n.Boundary && len(n.South) != 0 {
				pass := false
				for j, s := range n.South {
					if s == cell {
						pass = true
						if different(n.KyySouth[j], cell.KyyNorth[i], testTolerance) {
							t.Logf("Kyy doesn't match")
							t.FailNow()
						}
						if different(n.DyMinusHalf[j], cell.DyPlusHalf[i], testTolerance) {
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
		for i, s := range cell.South {
			if !s.Boundary && len(s.North) != 0 {
				pass := false
				for j, n := range s.North {
					if n == cell {
						pass = true
						if different(s.KyyNorth[j], cell.KyySouth[i], testTolerance) {
							t.Logf("Kyy doesn't match")
							t.FailNow()
						}
						if different(s.DyPlusHalf[j], cell.DyMinusHalf[i], testTolerance) {
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
		for i, a := range cell.Above {
			if !a.Boundary && len(a.Below) != 0 {
				pass := false
				for j, b := range a.Below {
					if b == cell {
						pass = true
						if different(a.KzzBelow[j], cell.KzzAbove[i], testTolerance) {
							t.Logf("Kzz doesn't match above (layer=%v, "+
								"KzzAbove=%v, KzzBelow=%v)", cell.Layer,
								cell.KzzAbove[i], a.KzzBelow[j])
							t.Fail()
						}
						if different(a.DzMinusHalf[j], cell.DzPlusHalf[i], testTolerance) {
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
		for i, b := range cell.Below {
			pass := false
			if cell.Layer == 0 && b == cell {
				pass = true
			} else if len(b.Above) != 0 {
				for j, a := range b.Above {
					if a == cell {
						pass = true
						if different(b.KzzAbove[j], cell.KzzBelow[i], testTolerance) {
							t.Logf("Kzz doesn't match below")
							t.FailNow()
						}
						if different(b.DzPlusHalf[j], cell.DzMinusHalf[i], testTolerance) {
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
		for _, g := range cell.GroundLevel {
			g2 := g
			pass := false
			for {
				if len(g2.Above) == 0 {
					pass = false
					break
				}
				if g2 == g2.Above[0] {
					pass = false
					break
				}
				if g2 == cell {
					pass = true
					break
				}
				g2 = g2.Above[0]
			}
			if !pass {
				t.Logf("Failed for Cell %v (layer %v) GroundLevel",
					cell.Polygonal, cell.Layer)
				t.FailNow()
			}
		}
	}
}

// Test whether convective mixing coeffecients are balanced in
// a way that conserves mass
func TestConvectiveMixing(t *testing.T) {
	const testTolerance = 1.e-8

	cfg, ctmdata, pop, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAPdata{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, mr, emis),
			cfg.StaticVariableGrid(ctmdata, pop, mr, emis),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	for i, c := range d.Cells {
		val := c.M2u - c.M2d + c.Above[0].M2d*c.Above[0].Dz/c.Dz
		if absDifferent(val, 0, testTolerance) {
			t.Error(i, c.Layer, val, c.M2u, c.M2d, c.Above[0].M2d)
		}
	}
}

// Test whether the mixing mechanisms are properly conserving mass
func TestMixing(t *testing.T) {
	const (
		testTolerance = 1.e-8
		testRow       = 0
		numTimesteps  = 5
	)

	cfg, ctmdata, pop, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}
	emis.data.Insert(emisRecord{
		PM25: E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions

	d := &InMAPdata{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, mr, emis),
			cfg.StaticVariableGrid(ctmdata, pop, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(Mixing()),
			SteadyStateConvergenceCheck(numTimesteps),
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
	if different(sum, E*d.Dt*numTimesteps, testTolerance) {
		t.Errorf("sum=%.12g (it should equal %v)\n", sum, E*d.Dt*numTimesteps)
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
	cfg, ctmdata, pop, mr := VarGridData()
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

	d := &InMAPdata{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, mr, emis),
			cfg.StaticVariableGrid(ctmdata, pop, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(Chemistry()),
			SteadyStateConvergenceCheck(1),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	d.sort()
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

	cfg, ctmdata, pop, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAPdata{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, mr, emis),
			cfg.StaticVariableGrid(ctmdata, pop, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(UpwindAdvection()),
			SteadyStateConvergenceCheck(1),
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

	cfg, ctmdata, pop, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAPdata{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, mr, emis),
			cfg.StaticVariableGrid(ctmdata, pop, mr, emis),
			SetTimestepCFL(),
		},
		RunFuncs: []DomainManipulator{
			Calculations(AddEmissionsFlux()),
			Calculations(MeanderMixing()),
			SteadyStateConvergenceCheck(nsteps),
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

func BenchmarkRun(b *testing.B) {
	const (
		testTolerance = 1.e-8
		testRow       = 2
	)

	nprocsStr := os.Getenv("$GOMAXPROCS")
	if nprocsStr != "" {
		nprocs, err := strconv.ParseInt(nprocsStr, 10, 64)
		if err != nil {
			b.Error(err)
		}
		runtime.GOMAXPROCS(int(nprocs))
	}

	cfg, ctmdata, pop, mr := VarGridData()
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

	d := &InMAPdata{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, mr, emis),
			cfg.StaticVariableGrid(ctmdata, pop, mr, emis),
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
			SteadyStateConvergenceCheck(1000),
		},
	}
	if err := d.Init(); err != nil {
		b.Error(err)
	}
	if err := d.Run(); err != nil {
		b.Error(err)
	}

	results := d.Results(false, "TotalPop deaths")["TotalPop deaths"][0]
	totald := floats.Sum(results)
	const expectedDeaths = 7.191501683235596e-10
	if different(totald, expectedDeaths, testTolerance) {
		b.Errorf("Deaths (%v) doesn't equal %v", totald, expectedDeaths)
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
