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

var d *InMAPdata

const (
	testRow          = 240 // in the middle of the grid
	testTolerance    = 1e-3
	Δt               = 6.   // seconds
	E                = 0.01 // emissions
	numRunIterations = 100  // number of iterations for Run to run

	dataPath = "testdata/inmapData_[layer].gob"
	//dataURL  = "https://github.com/ctessum/inmap/releases/download/v1.0.0/inmapData_48_24_12_4_2_1_40000.zip"
)

func init() {
	//dataPath := os.Getenv("inmapdata")
	var err error
	//if dataPath != "" {
	d, err = InitInMAPdata(UseFileTemplate(dataPath, 27), numRunIterations, "")
	//} else {
	//	d, err = InitInMAPdata(UseWebArchive(dataURL,
	//		"inmapData_[layer].gob", 27), numRunIterations, "")
	//}
	if err != nil {
		panic(err)
	}
	d.Dt = Δt
}

// Tests whether the cells correctly reference each other
func TestCellAlignment(t *testing.T) {
	for row, cell := range d.Data {
		if cell.Row != row {
			t.Logf("Failed for Row %v (layer %v) index", cell.Row, cell.Layer)
			t.FailNow()
		}
		for i, w := range cell.West {
			if len(w.East) != 0 {
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
			if len(e.West) != 0 {
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
			if len(n.South) != 0 {
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
			if len(s.North) != 0 {
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
			if len(a.Below) != 0 {
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
	for i, c := range d.Data {
		val := c.M2u - c.M2d + c.Above[0].M2d*c.Above[0].Dz/c.Dz
		if absDifferent(val, 0) {
			t.Log(i, c.Layer, val, c.M2u, c.M2d, c.Above[0].M2d)
			t.FailNow()
		}
	}
}

// Test whether the mixing mechanisms are properly conserving mass
func TestMixing(t *testing.T) {
	nsteps := 100
	for tt := 0; tt < nsteps; tt++ {
		d.Data[testRow].Ci[0] += E / d.Data[testRow].Dz // ground level emissions
		d.Data[testRow].Cf[0] += E / d.Data[testRow].Dz // ground level emissions
		for _, cell := range d.Data {
			cell.Mixing(Δt)
		}
		for _, cell := range d.Data {
			cell.Ci[0] = cell.Cf[0]
		}
	}
	sum := 0.
	maxval := 0.
	for _, cell := range d.Data {
		sum += cell.Cf[0] * cell.Dz
		maxval = max(maxval, cell.Cf[0])
	}
	t.Logf("sum=%.12g (it should equal %v)\n", sum, E*float64(nsteps))
	if different(sum, E*float64(nsteps), testTolerance) {
		t.FailNow()
	}
	if !different(sum, maxval, testTolerance) {
		t.Log("All of the mass is in one cell--it didn't mix")
		t.FailNow()
	}
}

// Test whether mass is conserved during chemical reactions.
func TestChemistry(t *testing.T) {
	c := d.Data[testRow]
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
				t.FailNow()
			}
		}
		if different(finalSum, sum, testTolerance) {
			t.FailNow()
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
	const tolerance = 1.e-8
	var cellGroups = [][]*Cell{d.Data, d.westBoundary, d.eastBoundary,
		d.northBoundary, d.southBoundary, d.topBoundary}
	for _, cellGroup := range cellGroups {
		for _, c := range cellGroup {
			c.Ci[0] = 0
			c.Cf[0] = 0
		}
	}
	nsteps := 1000
	for tt := 0; tt < nsteps; tt++ {
		c := d.Data[testRow]
		c.Ci[0] += E / c.Dz / c.Dy / c.Dx // ground level emissions
		c.Cf[0] += E / c.Dz / c.Dy / c.Dx // ground level emissions
		for _, c := range d.Data {
			c.UpwindAdvection(Δt)
		}
		for _, c := range d.Data {
			c.Ci[0] = c.Cf[0]
		}
	}
	sum := 0.
	layerSum := make(map[int]float64)
	for _, cellGroup := range cellGroups {
		for _, c := range cellGroup {
			val := c.Cf[0] * c.Dy * c.Dx * c.Dz
			if val < 0 {
				t.Fatalf("negative concentration")
			}
			sum += val
			layerSum[c.Layer] += val
		}
	}
	t.Logf("sum=%.12g (it should equal %v)\n", sum, E*float64(nsteps))
	if different(sum, E*float64(nsteps), tolerance) {
		t.FailNow()
	}
}

func BenchmarkRun(b *testing.B) {
	var timing []time.Duration
	var procs = []int{1, 2, 4, 8, 16, 24, 36, 48}
	for _, nprocs := range procs {
		runtime.GOMAXPROCS(nprocs)
		emissions := make(map[string][]float64)
		emissions["SOx"] = make([]float64, len(d.Data))
		emissions["SOx"][25000] = 100.
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

func absDifferent(a, b float64) bool {
	if math.Abs(a-b) > testTolerance {
		return true
	}
	return false
}
