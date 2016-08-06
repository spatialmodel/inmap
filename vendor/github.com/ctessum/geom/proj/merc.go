package proj

import (
	"fmt"
	"math"
)

const (
	r2d    = 57.29577951308232088
	fortPi = math.Pi / 4
)

// Merc is a mercator projection.
func Merc(this *SR) (forward, inverse Transformer, err error) {
	if math.IsNaN(this.Long0) {
		this.Long0 = 0
	}
	var con = this.B / this.A
	this.Es = 1 - con*con
	if math.IsNaN(this.X0) {
		this.X0 = 0
	}
	if math.IsNaN(this.Y0) {
		this.Y0 = 0
	}
	this.E = math.Sqrt(this.Es)
	if !math.IsNaN(this.LatTS) {
		if this.sphere {
			this.K0 = math.Cos(this.LatTS)
		} else {
			this.K0 = msfnz(this.E, math.Sin(this.LatTS), math.Cos(this.LatTS))
		}
	} else {
		if math.IsNaN(this.K0) {
			if !math.IsNaN(this.K) {
				this.K0 = this.K
			} else {
				this.K0 = 1
			}
		}
	}

	// Mercator forward equations--mapping lat,long to x,y
	forward = func(lon, lat float64) (x, y float64, err error) {
		// convert to radians
		if math.IsNaN(lat) || math.IsNaN(lon) || lat*r2d > 90 || lat*r2d < -90 || lon*r2d > 180 || lon*r2d < -180 {
			err = fmt.Errorf("in proj.Merc forward: invalid longitude (%g) or latitude (%g)", lon, lat)
			return
		}

		if math.Abs(math.Abs(lat)-halfPi) <= epsln {
			err = fmt.Errorf("in proj.Merc forward, abs(lat)==pi/2")
			return
		}
		if this.sphere {
			x = this.X0 + this.A*this.K0*adjust_lon(lon-this.Long0)
			y = this.Y0 + this.A*this.K0*math.Log(math.Tan(fortPi+0.5*lat))
		} else {
			var sinphi = math.Sin(lat)
			var ts = tsfnz(this.E, lat, sinphi)
			x = this.X0 + this.A*this.K0*adjust_lon(lon-this.Long0)
			y = this.Y0 - this.A*this.K0*math.Log(ts)
		}
		return
	}

	// Mercator inverse equations--mapping x,y to lat/long
	inverse = func(x, y float64) (lon, lat float64, err error) {
		x -= this.X0
		y -= this.Y0

		if this.sphere {
			lat = halfPi - 2*math.Atan(math.Exp(-y/(this.A*this.K0)))
		} else {
			var ts = math.Exp(-y / (this.A * this.K0))
			lat, err = phi2z(this.E, ts)
			if err != nil {
				return
			}
		}
		lon = adjust_lon(this.Long0 + x/(this.A*this.K0))
		return
	}
	return
}

func init() {
	// Register this projection with the corresponding names.
	registerTrans(Merc, "Mercator", "Popular Visualisation Pseudo Mercator",
		"Mercator_1SP", "Mercator_Auxiliary_Sphere", "merc")
}
