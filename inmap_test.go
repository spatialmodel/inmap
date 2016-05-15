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
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

const E = 0.01 // emissions

// Tests whether the cells correctly reference each other
func TestCellAlignment(t *testing.T) {
	const testTolerance = 1.e-8
	d := CreateVarGrid()

	for row, cell := range d.Cells {
		if cell.Row != row {
			t.Logf("Failed for Row %v (layer %v) index", cell.Row, cell.Layer)
			t.FailNow()
		}
		for i, w := range cell.West {
			if !w.Boundary && len(w.East) != 0 {
				pass := false
				for j, e := range w.East {
					if e.Row == cell.Row {
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
					t.Logf("Failed for Row %v (layer %v) West",
						cell.Row, cell.Layer)
					t.FailNow()
				}
			}
		}
		for i, e := range cell.East {
			if !e.Boundary && len(e.West) != 0 {
				pass := false
				for j, w := range e.West {
					if w.Row == cell.Row {
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
					t.Logf("Failed for Row %v (layer %v) East",
						cell.Row, cell.Layer)
					t.FailNow()
				}
			}
		}
		for i, n := range cell.North {
			if !n.Boundary && len(n.South) != 0 {
				pass := false
				for j, s := range n.South {
					if s.Row == cell.Row {
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
					t.Logf("Failed for Row %v (layer %v) North",
						cell.Row, cell.Layer)
					t.FailNow()
				}
			}
		}
		for i, s := range cell.South {
			if !s.Boundary && len(s.North) != 0 {
				pass := false
				for j, n := range s.North {
					if n.Row == cell.Row {
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
					t.Logf("Failed for Row %v (layer %v) South",
						cell.Row, cell.Layer)
					t.FailNow()
				}
			}
		}
		for i, a := range cell.Above {
			if !a.Boundary && len(a.Below) != 0 {
				pass := false
				for j, b := range a.Below {
					if b.Row == cell.Row {
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
					t.Logf("Failed for Row %v (layer %v) Above",
						cell.Row, cell.Layer)
					t.FailNow()
				}
			}
		}
		for i, b := range cell.Below {
			pass := false
			if cell.Layer == 0 && b.Row == cell.Row {
				pass = true
			} else if len(b.Above) != 0 {
				for j, a := range b.Above {
					if a.Row == cell.Row {
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
				t.Logf("Failed for Row %v (layer %v) Below",
					cell.Row, cell.Layer)
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
				if g2.Row == g2.Above[0].Row {
					pass = false
					break
				}
				if g2.Row == cell.Row {
					pass = true
					break
				}
				g2 = g2.Above[0]
			}
			if !pass {
				t.Logf("Failed for Row %v (layer %v) GroundLevel",
					cell.Row, cell.Layer)
				t.FailNow()
			}
		}
	}
}

// Test whether convective mixing coeffecients are balanced in
// a way that conserves mass
func TestConvectiveMixing(t *testing.T) {
	const testTolerance = 1.e-8
	d := CreateVarGrid()
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
	)
	d := CreateVarGrid()
	d.Cells[testRow].Ci[0] += E / d.Cells[testRow].Volume // ground level emissions
	d.Cells[testRow].Cf[0] += E / d.Cells[testRow].Volume // ground level emissions
	for _, cell := range d.Cells {
		cell.Mixing(d.Dt)
	}
	for _, cell := range d.Cells {
		cell.Ci[0] = cell.Cf[0]
	}
	sum := 0.
	maxval := 0.
	for _, group := range [][]*Cell{d.Cells, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary} {
		for _, cell := range group {
			sum += cell.Cf[0] * cell.Volume
			maxval = max(maxval, cell.Cf[0])
		}
	}
	if different(sum, E, testTolerance) {
		t.Errorf("sum=%.12g (it should equal %v)\n", sum, E)
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
	d := CreateVarGrid()

	c := d.Cells[testRow]
	nsteps := 10
	vals := make([]float64, len(polNames))
	for tt := 0; tt < nsteps; tt++ {
		sum := 0.
		for i := 0; i < len(vals); i++ {
			vals[i] = rand.Float64() * 10.
			sum += vals[i]
		}
		for i, v := range vals {
			c.Cf[i] = v
		}
		c.Chemistry(d)
		finalSum := 0.
		for _, val := range c.Cf {
			finalSum += val
			if val < 0 {
				chemPrint(t, vals, c)
				t.Fail()
			}
		}
		if different(finalSum, sum, testTolerance) {
			t.Error("different")
		}
		//chemPrint(t, vals, c)
	}
}

func chemPrint(t *testing.T, vals []float64, c *Cell) {
	for i, val2 := range c.Cf {
		t.Logf("%v: initial=%.3g, final=%.3g\n", polNames[i], vals[i], val2)
	}
}

// Test whether mass is conserved during advection.
func TestAdvection(t *testing.T) {
	d := CreateVarGrid()
	const tolerance = 1.e-8
	var cellGroups = [][]*Cell{d.Cells, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary}

	for testRow := 0; testRow < len(d.Cells); testRow++ {
		for _, cellGroup := range cellGroups {
			for _, c := range cellGroup {
				c.Ci[0] = 0
				c.Cf[0] = 0
			}
		}

		// Add emissions
		c := d.Cells[testRow]
		c.Ci[0] += E / c.Dz / c.Dy / c.Dx
		c.Cf[0] += E / c.Dz / c.Dy / c.Dx
		// Calculate advection
		for _, c := range d.Cells {
			c.UpwindAdvection(d.Dt)
		}
		for _, cellGroup := range cellGroups {
			for _, c := range cellGroup {
				c.Ci[0] = c.Cf[0]
			}
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
	const tolerance = 1.e-3
	nsteps := 10
	d := CreateVarGrid()

	var cellGroups = [][]*Cell{d.Cells, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary}
	// Test emissions from every thirtieth row.
	for testRow := 0; testRow < len(d.Cells); testRow += 30 {
		for _, group := range cellGroups {
			for _, c := range group {
				c.Ci[0] = 0
				c.Cf[0] = 0
			}
		}
		for tt := 0; tt < nsteps; tt++ {
			c := d.Cells[testRow]
			c.Ci[0] += E / c.Dz / c.Dy / c.Dx // ground level emissions
			c.Cf[0] += E / c.Dz / c.Dy / c.Dx // ground level emissions
			for _, c := range d.Cells {
				c.MeanderMixing(d.Dt)
			}
			for _, c := range d.Cells {
				c.Ci[0] = c.Cf[0]
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
			fmt.Println("qqq", testRow, d.Cells[testRow].Polygonal, d.Cells[testRow].Layer)
			t.Errorf("row %d emis: sum=%.12g (it should equal %v)\n", testRow, sum, E*float64(nsteps))
		}
	}
}

func BenchmarkRun(b *testing.B) {
	const (
		testTolerance = 1.e-8
		testRow       = 2
	)
	d := CreateVarGrid()

	var timing []time.Duration
	var procs = []int{1, 2, 4, 8, 16, 24, 36, 48}
	for _, nprocs := range procs {
		runtime.GOMAXPROCS(nprocs)
		emissions := make(map[string][]float64)
		emissions["SOx"] = make([]float64, len(d.Cells))
		emissions["SOx"][testRow] = 100.
		var results []float64
		start := time.Now()
		results = d.Run(emissions, false)["TotalPop deaths"][0]
		timing = append(timing, time.Since(start))
		totald := 0.
		for _, v := range results {
			totald += v
		}
		const expectedDeaths = 7.191501683235596e-10
		if different(totald+1, expectedDeaths+1, testTolerance) {
			b.Errorf("Deaths (%v) doesn't equal %v", totald, expectedDeaths)
		}
	}
	for i, p := range procs {
		fmt.Printf("For %v procs\ttime = %v\tscale eff = %.3g\n",
			p, timing[i], timing[0].Seconds()/
				timing[i].Seconds()/float64(p))
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
