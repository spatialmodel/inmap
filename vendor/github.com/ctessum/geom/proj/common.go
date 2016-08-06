package proj

import (
	"fmt"
	"math"
)

func msfnz(eccent, sinphi, cosphi float64) float64 {
	var con = eccent * sinphi
	return cosphi / (math.Sqrt(1 - con*con))
}

func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}

const (
	twoPi = math.Pi * 2
	// SPI is slightly greater than Math.PI, so values that exceed the -180..180
	// degree range by a tiny amount don't get wrapped. This prevents points that
	// have drifted from their original location along the 180th meridian (due to
	// floating point error) from changing their sign.
	sPi    = 3.14159265359
	halfPi = math.Pi / 2
)

func adjust_lon(x float64) float64 {
	if math.Abs(x) <= sPi {
		return x
	}
	return (x - (sign(x) * twoPi))
}

func tsfnz(eccent, phi, sinphi float64) float64 {
	var con = eccent * sinphi
	var com = 0.5 * eccent
	con = math.Pow(((1 - con) / (1 + con)), com)
	return (math.Tan(0.5*(halfPi-phi)) / con)
}

func phi2z(eccent, ts float64) (float64, error) {
	var eccnth = 0.5 * eccent
	phi := halfPi - 2*math.Atan(ts)
	for i := 0; i <= 15; i++ {
		con := eccent * math.Sin(phi)
		dphi := halfPi - 2*math.Atan(ts*(math.Pow(((1-con)/(1+con)), eccnth))) - phi
		phi += dphi
		if math.Abs(dphi) <= 0.0000000001 {
			return phi, nil
		}
	}
	return math.NaN(), fmt.Errorf("phi2z has no convergence")
}

func e0fn(x float64) float64 {
	return (1 - 0.25*x*(1+x/16*(3+1.25*x)))
}

func e1fn(x float64) float64 {
	return (0.375 * x * (1 + 0.25*x*(1+0.46875*x)))
}

func e2fn(x float64) float64 {
	return (0.05859375 * x * x * (1 + 0.75*x))
}

func e3fn(x float64) float64 {
	return (x * x * x * (35 / 3072))
}

func mlfn(e0, e1, e2, e3, phi float64) float64 {
	return (e0*phi - e1*math.Sin(2*phi) + e2*math.Sin(4*phi) - e3*math.Sin(6*phi))
}

func asinz(x float64) float64 {
	if math.Abs(x) > 1 {
		if x > 1 {
			x = 1
		} else {
			x = -1
		}
	}
	return math.Asin(x)
}

func qsfnz(eccent, sinphi float64) float64 {
	var con float64
	if eccent > 1.0e-7 {
		con = eccent * sinphi
		return ((1 - eccent*eccent) * (sinphi/(1-con*con) - (0.5/eccent)*math.Log((1-con)/(1+con))))
	} else {
		return (2 * sinphi)
	}
}
