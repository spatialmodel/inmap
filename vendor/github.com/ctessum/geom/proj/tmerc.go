package proj

import (
	"fmt"
	"math"
)

// TMerc is a transverse Mercator projection.
func TMerc(this *SR) (forward, inverse Transformer, err error) {

	e0 := e0fn(this.Es)
	e1 := e1fn(this.Es)
	e2 := e2fn(this.Es)
	e3 := e3fn(this.Es)
	ml0 := this.A * mlfn(e0, e1, e2, e3, this.Lat0)

	/**
	  Transverse Mercator Forward  - long/lat to x/y
	  long/lat in radians
	*/
	forward = func(lon, lat float64) (x, y float64, err error) {

		var delta_lon = adjust_lon(lon - this.Long0)
		var con float64
		var sin_phi = math.Sin(lat)
		var cos_phi = math.Cos(lat)

		if this.sphere {
			var b = cos_phi * math.Sin(delta_lon)
			if (math.Abs(math.Abs(b) - 1)) < 0.0000000001 {
				return math.NaN(), math.NaN(), fmt.Errorf("in proj.TMerc forward: b == 0")
			}
			x = 0.5 * this.A * this.K0 * math.Log((1+b)/(1-b))
			con = math.Acos(cos_phi * math.Cos(delta_lon) / math.Sqrt(1-b*b))
			if lat < 0 {
				con = -con
			}
			y = this.A * this.K0 * (con - this.Lat0)

		} else {
			var al = cos_phi * delta_lon
			var als = math.Pow(al, 2)
			var c = this.Ep2 * math.Pow(cos_phi, 2)
			var tq = math.Tan(lat)
			var t = math.Pow(tq, 2)
			con = 1 - this.Es*math.Pow(sin_phi, 2)
			var n = this.A / math.Sqrt(con)
			var ml = this.A * mlfn(e0, e1, e2, e3, lat)

			x = this.K0*n*al*(1+als/6*(1-t+c+als/20*(5-18*t+math.Pow(t, 2)+72*c-58*this.Ep2))) + this.X0
			y = this.K0*(ml-ml0+n*tq*(als*(0.5+als/24*(5-t+9*c+4*math.Pow(c, 2)+als/30*(61-58*t+math.Pow(t, 2)+600*c-330*this.Ep2))))) + this.Y0

		}
		return
	}

	/**
	  Transverse Mercator Inverse  -  x/y to long/lat
	*/
	inverse = func(x, y float64) (lon, lat float64, err error) {
		var con, phi float64
		var delta_phi float64
		const max_iter = 6

		if this.sphere {
			var f = math.Exp(x / (this.A * this.K0))
			var g = 0.5 * (f - 1/f)
			var temp = this.Lat0 + y/(this.A*this.K0)
			var h = math.Cos(temp)
			con = math.Sqrt((1 - h*h) / (1 + g*g))
			lat = asinz(con)
			if temp < 0 {
				lat = -lat
			}
			if (g == 0) && (h == 0) {
				lon = this.Long0
			} else {
				lon = adjust_lon(math.Atan2(g, h) + this.Long0)
			}
		} else { // ellipsoidal form
			var x = x - this.X0
			var y = y - this.Y0

			con = (ml0 + y/this.K0) / this.A
			phi = con
			i := 0
			for {
				delta_phi = ((con + e1*math.Sin(2*phi) - e2*math.Sin(4*phi) + e3*math.Sin(6*phi)) / e0) - phi
				phi += delta_phi
				if math.Abs(delta_phi) <= epsln {
					break
				}
				if i >= max_iter {
					return math.NaN(), math.NaN(), fmt.Errorf("in proj.TMerc inverse: i > max_iter")
				}
				i++
			}
			if math.Abs(phi) < halfPi {
				var sin_phi = math.Sin(phi)
				var cos_phi = math.Cos(phi)
				var tan_phi = math.Tan(phi)
				var c = this.Ep2 * math.Pow(cos_phi, 2)
				var cs = math.Pow(c, 2)
				var t = math.Pow(tan_phi, 2)
				var ts = math.Pow(t, 2)
				con = 1 - this.Es*math.Pow(sin_phi, 2)
				var n = this.A / math.Sqrt(con)
				var r = n * (1 - this.Es) / con
				var d = x / (n * this.K0)
				var ds = math.Pow(d, 2)
				lat = phi - (n*tan_phi*ds/r)*(0.5-ds/24*(5+3*t+10*c-4*cs-9*this.Ep2-ds/30*(61+90*t+298*c+45*ts-252*this.Ep2-3*cs)))
				lon = adjust_lon(this.Long0 + (d * (1 - ds/6*(1+2*t+c-ds/20*(5-2*c+28*t-3*cs+8*this.Ep2+24*ts))) / cos_phi))
			} else {
				lat = halfPi * sign(y)
				lon = this.Long0
			}
		}
		return
	}
	return
}

func init() {
	registerTrans(TMerc, "Transverse_Mercator", "Transverse Mercator", "tmerc")
}
