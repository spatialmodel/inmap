package proj

import (
	"fmt"
	"math"
)

type datumType int

const (
	pjd3Param    datumType = 1
	pjd7Param    datumType = 2
	pjdGridShift datumType = 3
	pjdWGS84     datumType = 4 // WGS84 or equivalent
	pjdNoDatum   datumType = 5 // WGS84 or equivalent
)
const (
	secToRad = 4.84813681109535993589914102357e-6
	adc      = 1.0026000
	cos67p5  = 0.38268343236508977
)

type datum struct {
	datum_type    datumType
	datum_params  []float64
	a, b, es, ep2 float64
	nadGrids      string
}

func (proj *SR) getDatum() *datum {
	this := new(datum)
	this.datum_type = pjdWGS84 //default setting
	if proj.DatumCode == "" || proj.DatumCode == "none" {
		this.datum_type = pjdNoDatum
	}

	if len(proj.DatumParams) > 0 {
		this.datum_params = proj.DatumParams
		if this.datum_params[0] != 0 || this.datum_params[1] != 0 || this.datum_params[2] != 0 {
			this.datum_type = pjd3Param
		}
		if len(this.datum_params) > 3 {
			if this.datum_params[3] != 0 || this.datum_params[4] != 0 || this.datum_params[5] != 0 || this.datum_params[6] != 0 {
				this.datum_type = pjd7Param
				this.datum_params[3] *= secToRad
				this.datum_params[4] *= secToRad
				this.datum_params[5] *= secToRad
				this.datum_params[6] = (this.datum_params[6] / 1000000.0) + 1.0
			}
		}
	}

	// DGR 2011-03-21 : nadgrids support
	if proj.NADGrids != "" {
		this.datum_type = pjdGridShift
	}

	this.a = proj.A //datum object also uses these values
	this.b = proj.B
	this.es = proj.Es
	this.ep2 = proj.Ep2
	if this.datum_type == pjdGridShift {
		this.nadGrids = proj.NADGrids
	}
	return this
}

// compare_datums()
// Returns TRUE if the two datums match, otherwise FALSE.
func (this *datum) compare_datums(dest *datum) bool {
	if this.datum_type != dest.datum_type {
		return false // false, datums are not equal
	} else if this.a != dest.a || math.Abs(this.es-dest.es) > 0.000000000050 {
		// the tolerence for es is to ensure that GRS80 and WGS84
		// are considered identical
		return false
	} else if this.datum_type == pjd3Param {
		return (this.datum_params[0] == dest.datum_params[0] && this.datum_params[1] == dest.datum_params[1] && this.datum_params[2] == dest.datum_params[2])
	} else if this.datum_type == pjd7Param {
		return (this.datum_params[0] == dest.datum_params[0] && this.datum_params[1] == dest.datum_params[1] && this.datum_params[2] == dest.datum_params[2] && this.datum_params[3] == dest.datum_params[3] && this.datum_params[4] == dest.datum_params[4] && this.datum_params[5] == dest.datum_params[5] && this.datum_params[6] == dest.datum_params[6])
	} else if this.datum_type == pjdGridShift || dest.datum_type == pjdGridShift {
		//alert("ERROR: Grid shift transformations are not implemented.");
		//return false
		//DGR 2012-07-29 lazy ...
		return this.nadGrids == dest.nadGrids
	}
	return true // datums are equal
}

/*
 * The function Convert_Geodetic_To_Geocentric converts geodetic coordinates
 * (latitude, longitude, and height) to geocentric coordinates (X, Y, Z),
 * according to the current ellipsoid parameters.
 *
 *    Latitude  : Geodetic latitude in radians                     (input)
 *    Longitude : Geodetic longitude in radians                    (input)
 *    Height    : Geodetic height, in meters                       (input)
 *    X         : Calculated Geocentric X coordinate, in meters    (output)
 *    Y         : Calculated Geocentric Y coordinate, in meters    (output)
 *    Z         : Calculated Geocentric Z coordinate, in meters    (output)
 *
 */
func (this *datum) geodetic_to_geocentric(Longitude, Latitude, Height float64) (X, Y, Z float64, err error) {

	var Rn float64       //  Earth radius at location  */
	var Sin_Lat float64  /*  Math.sin(Latitude)  */
	var Sin2_Lat float64 /*  Square of Math.sin(Latitude)  */
	var Cos_Lat float64  /*  Math.cos(Latitude)  */

	/*
	 ** Don't blow up if Latitude is just a little out of the value
	 ** range as it may just be a rounding issue.  Also removed longitude
	 ** test, it should be wrapped by Math.cos() and Math.sin().  NFW for PROJ.4, Sep/2001.
	 */
	if Latitude < -halfPi && Latitude > -1.001*halfPi {
		Latitude = -halfPi
	} else if Latitude > halfPi && Latitude < 1.001*halfPi {
		Latitude = halfPi
	} else if (Latitude < -halfPi) || (Latitude > halfPi) {
		/* Latitude out of range */
		err = fmt.Errorf("proj.datum.geodetic_to_geocentric:lat out of range: %g", Latitude)
		return
	}

	if Longitude > math.Pi {
		Longitude -= (2 * math.Pi)
	}
	Sin_Lat = math.Sin(Latitude)
	Cos_Lat = math.Cos(Latitude)
	Sin2_Lat = Sin_Lat * Sin_Lat
	Rn = this.a / (math.Sqrt(1.0e0 - this.es*Sin2_Lat))
	X = (Rn + Height) * Cos_Lat * math.Cos(Longitude)
	Y = (Rn + Height) * Cos_Lat * math.Sin(Longitude)
	Z = ((Rn * (1 - this.es)) + Height) * Sin_Lat

	return
}

func (this *datum) geocentric_to_geodetic(X, Y, Z float64) (Longitude, Latitude, Height float64) {
	/* local defintions and variables */
	/* end-criterium of loop, accuracy of sin(Latitude) */
	const (
		genau   = 1e-12
		genau2  = (genau * genau)
		maxiter = 30
	)
	var P float64  /* distance between semi-minor axis and location */
	var RR float64 /* distance between center and location */
	var CT float64 /* sin of geocentric latitude */
	var ST float64 /* cos of geocentric latitude */
	var RX float64
	var RK float64
	var RN float64    /* Earth radius at location */
	var CPHI0 float64 /* cos of start or old geodetic latitude in iterations */
	var SPHI0 float64 /* sin of start or old geodetic latitude in iterations */
	var CPHI float64  /* cos of searched geodetic latitude */
	var SPHI float64  /* sin of searched geodetic latitude */
	var SDPHI float64 /* end-criterium: addition-theorem of sin(Latitude(iter)-Latitude(iter-1)) */
	//	var At_Pole bool  /* indicates location is in polar region */
	var iter int /* # of continous iteration, max. 30 is always enough (s.a.) */

	//	At_Pole = false
	P = math.Sqrt(X*X + Y*Y)
	RR = math.Sqrt(X*X + Y*Y + Z*Z)

	/*      special cases for latitude and longitude */
	if P/this.a < genau {

		/*  special case, if P=0. (X=0., Y=0.) */
		//	At_Pole = true
		Longitude = 0.0

		/*  if (X,Y,Z)=(0.,0.,0.) then Height becomes semi-minor axis
		 *  of ellipsoid (=center of mass), Latitude becomes PI/2 */
		if RR/this.a < genau {
			Latitude = halfPi
			Height = -this.b
			return
		}
	} else {
		/*  ellipsoidal (geodetic) longitude
		 *  interval: -PI < Longitude <= +PI */
		Longitude = math.Atan2(Y, X)
	}

	/* --------------------------------------------------------------
	 * Following iterative algorithm was developped by
	 * "Institut for Erdmessung", University of Hannover, July 1988.
	 * Internet: www.ife.uni-hannover.de
	 * Iterative computation of CPHI,SPHI and Height.
	 * Iteration of CPHI and SPHI to 10**-12 radian resp.
	 * 2*10**-7 arcsec.
	 * --------------------------------------------------------------
	 */
	CT = Z / RR
	ST = P / RR
	RX = 1.0 / math.Sqrt(1.0-this.es*(2.0-this.es)*ST*ST)
	CPHI0 = ST * (1.0 - this.es) * RX
	SPHI0 = CT * RX
	iter = 0

	/* loop to find sin(Latitude) resp. Latitude
	 * until |sin(Latitude(iter)-Latitude(iter-1))| < genau */
	for {
		iter++
		RN = this.a / math.Sqrt(1.0-this.es*SPHI0*SPHI0)

		/*  ellipsoidal (geodetic) height */
		Height = P*CPHI0 + Z*SPHI0 - RN*(1.0-this.es*SPHI0*SPHI0)

		RK = this.es * RN / (RN + Height)
		RX = 1.0 / math.Sqrt(1.0-RK*(2.0-RK)*ST*ST)
		CPHI = ST * (1.0 - RK) * RX
		SPHI = CT * RX
		SDPHI = SPHI*CPHI0 - CPHI*SPHI0
		CPHI0 = CPHI
		SPHI0 = SPHI
		if !(SDPHI*SDPHI > genau2 && iter < maxiter) {
			break
		}
	}
	/*      ellipsoidal (geodetic) latitude */
	Latitude = math.Atan(SPHI / math.Abs(CPHI))
	return
}

/** Convert_Geocentric_To_Geodetic
 * The method used here is derived from 'An Improved Algorithm for
 * Geocentric to Geodetic Coordinate Conversion', by Ralph Toms, Feb 1996
 */
func (this *datum) geocentric_to_geodetic_noniter(X, Y, Z float64) (Longitude, Latitude, Height float64) {

	var W float64       /* distance from Z axis */
	var W2 float64      /* square of distance from Z axis */
	var T0 float64      /* initial estimate of vertical component */
	var T1 float64      /* corrected estimate of vertical component */
	var S0 float64      /* initial estimate of horizontal component */
	var S1 float64      /* corrected estimate of horizontal component */
	var Sin_B0 float64  /* Math.sin(B0), B0 is estimate of Bowring aux variable */
	var Sin3_B0 float64 /* cube of Math.sin(B0) */
	var Cos_B0 float64  /* Math.cos(B0) */
	var Sin_p1 float64  /* Math.sin(phi1), phi1 is estimated latitude */
	var Cos_p1 float64  /* Math.cos(phi1) */
	var Rn float64      /* Earth radius at location */
	var Sum float64     /* numerator of Math.cos(phi1) */
	var At_Pole = false /* indicates location is in polar region */

	if X != 0.0 {
		Longitude = math.Atan2(Y, X)
	} else {
		if Y > 0 {
			Longitude = halfPi
		} else if Y < 0 {
			Longitude = -halfPi
		} else {
			At_Pole = true
			Longitude = 0.0
			if Z > 0.0 { /* north pole */
				Latitude = halfPi
			} else if Z < 0.0 { /* south pole */
				Latitude = -halfPi
			} else { /* center of earth */
				Latitude = halfPi
				Height = -this.b
				return
			}
		}
	}
	W2 = X*X + Y*Y
	W = math.Sqrt(W2)
	T0 = Z * adc
	S0 = math.Sqrt(T0*T0 + W2)
	Sin_B0 = T0 / S0
	Cos_B0 = W / S0
	Sin3_B0 = Sin_B0 * Sin_B0 * Sin_B0
	T1 = Z + this.b*this.ep2*Sin3_B0
	Sum = W - this.a*this.es*Cos_B0*Cos_B0*Cos_B0
	S1 = math.Sqrt(T1*T1 + Sum*Sum)
	Sin_p1 = T1 / S1
	Cos_p1 = Sum / S1
	Rn = this.a / math.Sqrt(1.0-this.es*Sin_p1*Sin_p1)
	if Cos_p1 >= cos67p5 {
		Height = W/Cos_p1 - Rn
	} else if Cos_p1 <= -cos67p5 {
		Height = W/-Cos_p1 - Rn
	} else {
		Height = Z/Sin_p1 + Rn*(this.es-1.0)
	}
	if At_Pole == false {
		Latitude = math.Atan(Sin_p1 / Cos_p1)
	}

	return
}

/****************************************************************/
// pj_geocentic_to_wgs84( p )
//  p = point to transform in geocentric coordinates (x,y,z)
func (this *datum) geocentric_to_wgs84(x, y, z float64) (float64, float64, float64) {

	if this.datum_type == pjd3Param {
		// if( x[io] === HUGE_VAL )
		//    continue;
		x += this.datum_params[0]
		y += this.datum_params[1]
		z += this.datum_params[2]
	} else if this.datum_type == pjd7Param {
		var Dx_BF = this.datum_params[0]
		var Dy_BF = this.datum_params[1]
		var Dz_BF = this.datum_params[2]
		var Rx_BF = this.datum_params[3]
		var Ry_BF = this.datum_params[4]
		var Rz_BF = this.datum_params[5]
		var M_BF = this.datum_params[6]
		// if( x[io] === HUGE_VAL )
		//    continue;
		var x_out = M_BF*(x-Rz_BF*y+Ry_BF*z) + Dx_BF
		var y_out = M_BF*(Rz_BF*x+y-Rx_BF*z) + Dy_BF
		var z_out = M_BF*(-Ry_BF*x+Rx_BF*y+z) + Dz_BF
		return x_out, y_out, z_out
	}
	return x, y, z
}

/****************************************************************/
// pj_geocentic_from_wgs84()
//  coordinate system definition,
//  point to transform in geocentric coordinates (x,y,z)
func (this *datum) geocentric_from_wgs84(x, y, z float64) (float64, float64, float64) {

	if this.datum_type == pjd3Param {
		//if( x[io] === HUGE_VAL )
		//    continue;
		x -= this.datum_params[0]
		y -= this.datum_params[1]
		z -= this.datum_params[2]
	} else if this.datum_type == pjd7Param {
		var Dx_BF = this.datum_params[0]
		var Dy_BF = this.datum_params[1]
		var Dz_BF = this.datum_params[2]
		var Rx_BF = this.datum_params[3]
		var Ry_BF = this.datum_params[4]
		var Rz_BF = this.datum_params[5]
		var M_BF = this.datum_params[6]
		var x_tmp = (x - Dx_BF) / M_BF
		var y_tmp = (y - Dy_BF) / M_BF
		var z_tmp = (z - Dz_BF) / M_BF
		//if( x[io] === HUGE_VAL )
		//    continue;

		x = x_tmp + Rz_BF*y_tmp - Ry_BF*z_tmp
		y = -Rz_BF*x_tmp + y_tmp + Rx_BF*z_tmp
		z = Ry_BF*x_tmp - Rx_BF*y_tmp + z_tmp
	} //cs_geocentric_from_wgs84()
	return x, y, z
}
