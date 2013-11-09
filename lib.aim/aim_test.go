package aim

import (
	"math"
	"runtime"
	"testing"
)

var d *AIMdata

const (
	j, i          = 219, 252 // Minneapolis!
	testTolerance = 1e-10
)

func init() {
	runtime.GOMAXPROCS(8)
	d = InitAIMdata("../wrf2aim/aimData.ncf")
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
	j, i := 118, 222 // center of domain
	Δt := 72.        // seconds
	nsteps := 1
	E := 0.01 // emissions
	for x := 0; x < nsteps; x++ {
		for k := 0; k < d.Nz; k++ {
			ii := d.getIndex(k, j, i)
			c := d.Data[ii]
			if k == 0 {
				c.Ci[0] += E // ground level emissions
				c.Cf[0] += E // ground level emissions
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
		sum += d.Data[ii].Cf[0]
		t.Logf("level %v=%.3v\n", k, d.Data[ii].Cf[0])
	}
	t.Logf("sum=%.8g (it should equal %v)\n", sum, E*float64(nsteps))
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
