package proj

import (
	"fmt"
	"math"
)

// UTM is a universal transverse Mercator projection.
func UTM(this *SR) (forward, inverse Transformer, err error) {

	if math.IsNaN(this.Zone) {
		err = fmt.Errorf("in proj.UTM: zone is not specified")
		return
	}
	this.Lat0 = 0
	this.Long0 = ((6 * math.Abs(this.Zone)) - 183) * deg2rad
	this.X0 = 500000
	if this.UTMSouth {
		this.Y0 = 10000000
	} else {
		this.Y0 = 0
	}
	this.K0 = 0.9996

	return TMerc(this)
}

func init() {
	registerTrans(UTM, "Universal Transverse Mercator System", "utm")
}
