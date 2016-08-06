package proj

import (
	"fmt"
	"math"
)

// LCC is a Lambert Conformal Conic projection.
func LCC(this *SR) (forward, inverse Transformer, err error) {

	//double c_lat;                   /* center latitude                      */
	//double c_lon;                   /* center longitude                     */
	//double lat1;                    /* first standard parallel              */
	//double lat2;                    /* second standard parallel             */
	//double r_maj;                   /* major axis                           */
	//double r_min;                   /* minor axis                           */
	//double false_east;              /* x offset in meters                   */
	//double false_north;             /* y offset in meters                   */

	if math.IsNaN(this.Lat2) {
		this.Lat2 = this.Lat1
	} //if lat2 is not defined
	if math.IsNaN(this.K0) {
		this.K0 = 1
	}
	if math.IsNaN(this.X0) {
		this.X0 = 0
	}
	if math.IsNaN(this.Y0) {
		this.Y0 = 0
	}
	// Standard Parallels cannot be equal and on opposite sides of the equator
	if math.Abs(this.Lat1+this.Lat2) < epsln {
		err = fmt.Errorf("proj.LCC: standard Parallels cannot be equal and on opposite sides of the equator")
		return
	}

	temp := this.B / this.A
	this.E = math.Sqrt(1 - temp*temp)

	var sin1 = math.Sin(this.Lat1)
	var cos1 = math.Cos(this.Lat1)
	var ms1 = msfnz(this.E, sin1, cos1)
	var ts1 = tsfnz(this.E, this.Lat1, sin1)

	var sin2 = math.Sin(this.Lat2)
	var cos2 = math.Cos(this.Lat2)
	var ms2 = msfnz(this.E, sin2, cos2)
	var ts2 = tsfnz(this.E, this.Lat2, sin2)

	var ts0 = tsfnz(this.E, this.Lat0, math.Sin(this.Lat0))

	var NS float64
	if math.Abs(this.Lat1-this.Lat2) > epsln {
		NS = math.Log(ms1/ms2) / math.Log(ts1/ts2)
	} else {
		NS = sin1
	}
	if math.IsNaN(NS) {
		NS = sin1
	}
	F0 := ms1 / (NS * math.Pow(ts1, NS))
	RH := this.A * F0 * math.Pow(ts0, NS)
	if this.Title == "" {
		this.Title = "Lambert Conformal Conic"
	}

	// Lambert Conformal conic forward equations--mapping lat,long to x,y
	// -----------------------------------------------------------------
	forward = func(lon, lat float64) (x, y float64, err error) {

		// singular cases :
		if math.Abs(2*math.Abs(lat)-math.Pi) <= epsln {
			lat = sign(lat) * (halfPi - 2*epsln)
		}
		con := math.Abs(math.Abs(lat) - halfPi)
		var ts, rh1 float64
		if con > epsln {
			ts = tsfnz(this.E, lat, math.Sin(lat))
			rh1 = this.A * F0 * math.Pow(ts, NS)
		} else {
			con = lat * NS
			if con <= 0 {
				err = fmt.Errorf("proj.LCC: con <= 0")
				return
			}
			rh1 = 0
		}
		var theta = NS * adjust_lon(lon-this.Long0)
		x = this.K0*(rh1*math.Sin(theta)) + this.X0
		y = this.K0*(RH-rh1*math.Cos(theta)) + this.Y0

		return
	}

	// Lambert Conformal Conic inverse equations--mapping x,y to lat/long
	// -----------------------------------------------------------------
	inverse = func(x, y float64) (lon, lat float64, err error) {

		var rh1, con, ts float64
		x = (x - this.X0) / this.K0
		y = (RH - (y-this.Y0)/this.K0)
		if NS > 0 {
			rh1 = math.Sqrt(x*x + y*y)
			con = 1
		} else {
			rh1 = -math.Sqrt(x*x + y*y)
			con = -1
		}
		var theta = 0.
		if rh1 != 0 {
			theta = math.Atan2((con * x), (con * y))
		}
		if (rh1 != 0) || (NS > 0) {
			con = 1 / NS
			ts = math.Pow((rh1 / (this.A * F0)), con)
			lat, err = phi2z(this.E, ts)
			if err != nil {
				return
			}
		} else {
			lat = -halfPi
		}
		lon = adjust_lon(theta/NS + this.Long0)

		return
	}
	return
}

func init() {
	registerTrans(LCC, "Lambert Tangential Conformal Conic Projection", "Lambert_Conformal_Conic", "Lambert_Conformal_Conic_2SP", "lcc")
}
