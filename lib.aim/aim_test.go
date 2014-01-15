package aim

import (
	"math"
	"math/rand"
	"runtime"
	"testing"
)

var d *AIMdata

const (
	j, i          = 219, 252 // Minneapolis!
	testTolerance = 1e-8
	Δt            = 72.  // seconds
	E             = 0.01 // emissions
)

func init() {
	runtime.GOMAXPROCS(8)
	d = InitAIMdata("../wrf2aim/aimData.ncf", "8080")
	d.Dt = Δt
}

// Tests whether the cells correctly reference each other
func TestCellAlignment(t *testing.T) {
	var previousAbove, previous int
	for k := 0; k < d.Nz; k++ {
		ii := d.getIndex(k, j, i)
		c := d.Data[ii]
		if k == 0 {
			if c.ii != c.GroundLevel.ii {
				t.FailNow()
			}
		} else {
			if c.ii != previousAbove {
				t.FailNow()
			}
			if c.Below.ii != previous {
				t.FailNow()
			}
		}
		if c.k != k {
			t.FailNow()
		}
		previousAbove = c.Above.ii
		previous = c.ii
	}
}

// Test whether vertical mixing mechanisms is properly conserving mass
func TestVerticalMixing(t *testing.T) {
	nsteps := 100
	for tt := 0; tt < nsteps; tt++ {
		for k := 0; k < d.Nz; k++ {
			ii := d.getIndex(k, j, i)
			c := d.Data[ii]
			if k == 0 {
				c.Ci[0] += E / c.Dz // ground level emissions
				c.Cf[0] += E / c.Dz // ground level emissions
			}
		}
		for k := 0; k < d.Nz; k++ {
			ii := d.getIndex(k, j, i)
			d.Data[ii].VerticalMixing(Δt)
		}
		for k := 0; k < d.Nz; k++ {
			ii := d.getIndex(k, j, i)
			c := d.Data[ii]
			c.Ci[0] = c.Cf[0]
		}
	}
	sum := 0.
	for k := 0; k < d.Nz; k++ {
		ii := d.getIndex(k, j, i)
		sum += d.Data[ii].Cf[0] * d.Data[ii].Dz
		t.Logf("level %v=%.3v\n", k, d.Data[ii].Cf[0]*d.Data[ii].Dz)
	}
	t.Logf("sum=%.12g (it should equal %v)\n", sum, E*float64(nsteps))
	if different(sum, E*float64(nsteps)) {
		t.FailNow()
	}
}

// Test whether mass is conserved during chemical reactions.
func TestChemistry(t *testing.T) {
	c := d.Data[d.getIndex(0, j, i)]
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
		c.COBRAchemistry(d)
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
		ii := d.getIndex(0, j, i)
		c := d.Data[ii]
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
