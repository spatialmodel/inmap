package aim

import (
	"math"
	"math/rand"
	"runtime"
	"testing"
)

var d *AIMdata

const (
	testRow       = 25300 // somewhere in Chicago
	testTolerance = 1e-8
	Δt            = 6.   // seconds
	E             = 0.01 // emissions
)

func init() {
	runtime.GOMAXPROCS(8)
	d = InitAIMdata("../wrf2aim/aimData/aimData_[layer].geojson", 27, "8080")
	d.Dt = Δt
}

// Tests whether the cells correctly reference each other
func TestCellAlignment(t *testing.T) {
	for row, cell := range d.Data {
		if cell.Row != row {
			t.Logf("Failed for Row %v index", cell.Row)
			t.FailNow()
		}
		for _, w := range cell.West {
			if len(w.East) != 0 {
				pass := false
				for _, e := range w.East {
					if e.Row == cell.Row {
						pass = true
						break
					}
				}
				if !pass {
					t.Logf("Failed for Row %v West", cell.Row)
					t.FailNow()
				}
			}
		}
		for _, e := range cell.East {
			if len(e.West) != 0 {
				pass := false
				for _, w := range e.West {
					if w.Row == cell.Row {
						pass = true
						break
					}
				}
				if !pass {
					t.Logf("Failed for Row %v East", cell.Row)
					t.FailNow()
				}
			}
		}
		for _, n := range cell.North {
			if len(n.South) != 0 {
				pass := false
				for _, s := range n.South {
					if s.Row == cell.Row {
						pass = true
						break
					}
				}
				if !pass {
					t.Logf("Failed for Row %v North", cell.Row)
					t.FailNow()
				}
			}
		}
		for _, s := range cell.South {
			if len(s.North) != 0 {
				pass := false
				for _, n := range s.North {
					if n.Row == cell.Row {
						pass = true
						break
					}
				}
				if !pass {
					t.Logf("Failed for Row %v South", cell.Row)
					t.FailNow()
				}
			}
		}
		for _, a := range cell.Above {
			if len(a.Below) != 0 {
				pass := false
				for _, b := range a.Below {
					if b.Row == cell.Row {
						pass = true
						break
					}
				}
				if !pass {
					t.Logf("Failed for Row %v Above", cell.Row)
					t.FailNow()
				}
			}
		}
		for _, b := range cell.Below {
			if len(b.Above) != 0 {
				pass := false
				for _, a := range b.Above {
					if a.Row == cell.Row {
						pass = true
						break
					}
				}
				if !pass {
					t.Logf("Failed for Row %v Above", cell.Row)
					t.FailNow()
				}
			}
		}
		// Assume upper cells are never higher resolution than lower cells
		for _, g := range cell.GroundLevel {
			g2 := g
			pass := false
			for {
				if g2.Above == nil {
					pass = false
					break
				}
				g2 = g2.Above[0]
				if g2.Row == cell.Row {
					pass = true
					break
				}
			}
			if !pass {
				t.Logf("Failed for Row %v GroundLevel", cell.Row)
				t.FailNow()
			}
		}
	}
}

// Test whether vertical mixing mechanisms is properly conserving mass
func TestVerticalMixing(t *testing.T) {
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
	if different(sum, E*float64(nsteps)) {
		t.FailNow()
	}
	if !different(sum, maxval) {
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
		if different(finalSum, sum) {
			t.FailNow()
		}
		chemPrint(t, vals, c)
	}
}

func chemPrint(t *testing.T, vals []float64, c *AIMcell) {
	for i, val2 := range c.Cf {
		t.Logf("%v: initial=%.3g, final=%.3g\n", polNames[i], vals[i], val2)
	}
}

// Test whether mass is conserved during advection.
func TestAdvection(t *testing.T) {
	for _, c := range d.Data {
		c.Ci[0] = 0
		c.Cf[0] = 0
	}
	nsteps := 5
	for tt := 0; tt < nsteps; tt++ {
		c := d.Data[testRow]
		c.Ci[0] += E / c.Dz / c.Dy / c.Dx // ground level emissions
		c.Cf[0] += E / c.Dz / c.Dy / c.Dx // ground level emissions
		for _, c := range d.Data {
			c.RK3advectionPass1(d)
		}
		for _, c := range d.Data {
			c.RK3advectionPass2(d)
		}
		for _, c := range d.Data {
			c.RK3advectionPass3(d)
		}
		for _, c := range d.Data {
			c.Ci[0] = c.Cf[0]
		}
	}
	sum := 0.
	for _, c := range d.Data {
		val := c.Cf[0] * c.Dy * c.Dx * c.Dz
		sum += val
	}
	t.Logf("sum=%.12g (it should equal %v)\n", sum, E*float64(nsteps))
	if different(sum, E*float64(nsteps)) {
		t.FailNow()
	}
}

func different(a, b float64) bool {
	if math.Abs(a-b)/math.Abs(b) > testTolerance {
		return true
	} else {
		return false
	}
}
