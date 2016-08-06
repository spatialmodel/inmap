package proj

import (
	"fmt"
	"math"
)

// AEA is an Albers Conical Equal Area projection.
func AEA(this *SR) (forward, inverse Transformer, err error) {

	if math.Abs(this.Lat1+this.Lat2) < epsln {
		err = fmt.Errorf("proj.AEA: standard Parallels cannot be equal and on opposite sides of the equator")
	}
	temp := this.B / this.A
	es := 1 - math.Pow(temp, 2)
	e3 := math.Sqrt(es)

	sin_po := math.Sin(this.Lat1)
	cos_po := math.Cos(this.Lat1)
	//t1 := sin_po
	con := sin_po
	ms1 := msfnz(e3, sin_po, cos_po)
	qs1 := qsfnz(e3, sin_po)

	sin_po = math.Sin(this.Lat2)
	cos_po = math.Cos(this.Lat2)
	//t2 := sin_po
	ms2 := msfnz(e3, sin_po, cos_po)
	qs2 := qsfnz(e3, sin_po)

	sin_po = math.Sin(this.Lat0)
	//cos_po = math.Cos(this.Lat0)
	//t3 := sin_po
	qs0 := qsfnz(e3, sin_po)

	var ns0 float64
	if math.Abs(this.Lat1-this.Lat2) > epsln {
		ns0 = (ms1*ms1 - ms2*ms2) / (qs2 - qs1)
	} else {
		ns0 = con
	}
	c := ms1*ms1 + ns0*qs1
	rh := this.A * math.Sqrt(c-ns0*qs0) / ns0

	/* Albers Conical Equal Area forward equations--mapping lat,long to x,y
	   -------------------------------------------------------------------*/
	forward = func(lon, lat float64) (x, y float64, err error) {

		sin_phi := math.Sin(lat)
		//cos_phi := math.Cos(lat)

		var qs = qsfnz(e3, sin_phi)
		var rh1 = this.A * math.Sqrt(c-ns0*qs) / ns0
		var theta = ns0 * adjust_lon(lon-this.Long0)
		x = rh1*math.Sin(theta) + this.X0
		y = rh - rh1*math.Cos(theta) + this.Y0
		return
	}

	inverse = func(x, y float64) (lon, lat float64, err error) {
		var rh1, qs, con, theta float64

		x -= this.X0
		y = rh - y + this.Y0
		if ns0 >= 0 {
			rh1 = math.Sqrt(x*x + y*y)
			con = 1
		} else {
			rh1 = -math.Sqrt(x*x + y*y)
			con = -1
		}
		theta = 0
		if rh1 != 0 {
			theta = math.Atan2(con*x, con*y)
		}
		con = rh1 * ns0 / this.A
		if this.sphere {
			lat = math.Asin((c - con*con) / (2 * ns0))
		} else {
			qs = (c - con*con) / ns0
			lat, err = aeaPhi1z(e3, qs)
			if err != nil {
				return
			}
		}

		lon = adjust_lon(theta/ns0 + this.Long0)
		return
	}
	return
}

// aeaPhi1z is a function to compute phi1, the latitude for the inverse of the
//   Albers Conical Equal-Area projection.
func aeaPhi1z(eccent, qs float64) (float64, error) {
	var sinphi, cosphi, con, com, dphi float64
	var phi = asinz(0.5 * qs)
	if eccent < epsln {
		return phi, nil
	}

	var eccnts = eccent * eccent
	for i := 1; i <= 25; i++ {
		sinphi = math.Sin(phi)
		cosphi = math.Cos(phi)
		con = eccent * sinphi
		com = 1 - con*con
		dphi = 0.5 * com * com / cosphi * (qs/(1-eccnts) - sinphi/com + 0.5/eccent*math.Log((1-con)/(1+con)))
		phi = phi + dphi
		if math.Abs(dphi) <= 1e-7 {
			return phi, nil
		}
	}
	return math.NaN(), fmt.Errorf("proj.aeaPhi1z: didn't converge")
}

func init() {
	registerTrans(AEA, "Albers_Conic_Equal_Area", "Albers", "aea")
}
