package proj

import (
	"fmt"
	"math"
)

// Krovak is a Krovak projection.
func Krovak(this *SR) (forward, inverse Transformer, err error) {
	this.A = 6377397.155
	this.Es = 0.006674372230614
	this.E = math.Sqrt(this.Es)
	if math.IsNaN(this.Lat0) {
		this.Lat0 = 0.863937979737193
	}
	if math.IsNaN(this.Long0) {
		this.Long0 = 0.7417649320975901 - 0.308341501185665
	}
	/* if scale not set default to 0.9999 */
	if math.IsNaN(this.K0) {
		this.K0 = 0.9999
	}
	const S45 = 0.785398163397448 /* 45 */
	const S90 = 2 * S45
	Fi0 := this.Lat0
	E2 := this.Es
	this.E = math.Sqrt(E2)
	Alfa := math.Sqrt(1 + (E2*math.Pow(math.Cos(Fi0), 4))/(1-E2))
	const Uq = 1.04216856380474
	U0 := math.Asin(math.Sin(Fi0) / Alfa)
	G := math.Pow((1+this.E*math.Sin(Fi0))/(1-this.E*math.Sin(Fi0)), Alfa*this.E/2)
	K := math.Tan(U0/2+S45) / math.Pow(math.Tan(Fi0/2+S45), Alfa) * G
	K1 := this.K0
	N0 := this.A * math.Sqrt(1-E2) / (1 - E2*math.Pow(math.Sin(Fi0), 2))
	const S0 = 1.37008346281555
	N := math.Sin(S0)
	Ro0 := K1 * N0 / math.Tan(S0)
	Ad := S90 - Uq

	/* ellipsoid */
	/* calculate xy from lat/lon */
	/* Constants, identical to inverse transform function */
	forward = func(lon, lat float64) (x, y float64, err error) {
		var gfi, u, deltav, s, d, eps, ro float64
		delta_lon := adjust_lon(lon - this.Long0)
		/* Transformation */
		gfi = math.Pow(((1 + this.E*math.Sin(lat)) / (1 - this.E*math.Sin(lat))), (Alfa * this.E / 2))
		u = 2 * (math.Atan(K*math.Pow(math.Tan(lat/2+S45), Alfa)/gfi) - S45)
		deltav = -delta_lon * Alfa
		s = math.Asin(math.Cos(Ad)*math.Sin(u) + math.Sin(Ad)*math.Cos(u)*math.Cos(deltav))
		d = math.Asin(math.Cos(u) * math.Sin(deltav) / math.Cos(s))
		eps = N * d
		ro = Ro0 * math.Pow(math.Tan(S0/2+S45), N) / math.Pow(math.Tan(s/2+S45), N)
		y = ro * math.Cos(eps) / 1
		x = ro * math.Sin(eps) / 1

		if !this.Czech {
			y *= -1
			x *= -1
		}
		return
	}

	/* calculate lat/lon from xy */
	inverse = func(x, y float64) (lon, lat float64, err error) {
		var u, deltav, s, d, eps, ro, fi1 float64
		var ok int

		/* Transformation */
		/* revert y, x*/
		x, y = y, x
		if !this.Czech {
			y *= -1
			x *= -1
		}
		ro = math.Sqrt(x*x + y*y)
		eps = math.Atan2(y, x)
		d = eps / math.Sin(S0)
		s = 2 * (math.Atan(math.Pow(Ro0/ro, 1/N)*math.Tan(S0/2+S45)) - S45)
		u = math.Asin(math.Cos(Ad)*math.Sin(s) - math.Sin(Ad)*math.Cos(s)*math.Cos(d))
		deltav = math.Asin(math.Cos(s) * math.Sin(d) / math.Cos(u))
		x = this.Long0 - deltav/Alfa
		fi1 = u
		ok = 0
		var iter = 0
		for {
			if !(ok == 0 && iter < 15) {
				break
			}
			y = 2 * (math.Atan(math.Pow(K, -1/Alfa)*math.Pow(math.Tan(u/2+S45), 1/Alfa)*math.Pow((1+this.E*math.Sin(fi1))/(1-this.E*math.Sin(fi1)), this.E/2)) - S45)
			if math.Abs(fi1-y) < 0.0000000001 {
				ok = 1
			}
			fi1 = y
			iter++
		}
		if iter >= 15 {
			err = fmt.Errorf("proj.Krovak: iter >= 15")
			return
		}

		return
	}
	return
}

func init() {
	registerTrans(Krovak, "Krovak", "krovak")
}
