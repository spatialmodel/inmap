package proj

import "math"

const (
	epsln = 1.0e-10
	// ellipoid pj_set_ell.c
	sixth = 0.1666666666666666667
	/* 1/6 */
	ra4 = 0.04722222222222222222
	/* 17/360 */
	ra6 = 0.02215608465608465608
)

// DeriveConstants calculates some properties of the spatial reference based
// on other properties
func (json *SR) DeriveConstants() {
	// DGR 2011-03-20 : nagrids -> nadgrids
	if json.DatumCode != "" && json.DatumCode != "none" {
		datumDef, ok := datumDefs[json.DatumCode]
		if ok {
			json.DatumParams = make([]float64, len(datumDef.towgs84))
			for i, p := range datumDef.towgs84 {
				json.DatumParams[i] = p
			}
			json.Ellps = datumDef.ellipse
			if datumDef.datumName != "" {
				json.DatumName = datumDef.datumName
			} else {
				json.DatumName = json.DatumCode
			}
		}
	}
	if math.IsNaN(json.A) { // do we have an ellipsoid?
		ellipse, ok := ellipsoidDefs[json.Ellps]
		if !ok {
			ellipse = ellipsoidDefs["WGS84"]
		}
		if ellipse.a != 0 {
			json.A = ellipse.a
		}
		if ellipse.b != 0 {
			json.B = ellipse.b
		}
		if ellipse.rf != 0 {
			json.Rf = ellipse.rf
		}
		json.EllipseName = ellipse.ellipseName
	}
	if !math.IsNaN(json.Rf) && math.IsNaN(json.B) {
		json.B = (1.0 - 1.0/json.Rf) * json.A
	}
	if json.Rf == 0 || math.Abs(json.A-json.B) < epsln {
		json.sphere = true
		json.B = json.A
	}
	json.A2 = json.A * json.A               // used in geocentric
	json.B2 = json.B * json.B               // used in geocentric
	json.Es = (json.A2 - json.B2) / json.A2 // e ^ 2
	json.E = math.Sqrt(json.Es)             // eccentricity
	if json.Ra {
		json.A *= 1 - json.Es*(sixth+json.Es*(ra4+json.Es*ra6))
		json.A2 = json.A * json.A
		json.B2 = json.B * json.B
		json.Es = 0
	}
	json.Ep2 = (json.A2 - json.B2) / json.B2 // used in geocentric
	if math.IsNaN(json.K0) {
		json.K0 = 1.0 //default value
	}
	//DGR 2010-11-12: axis
	if json.Axis == "" {
		json.Axis = enu
	}

	if json.datum == nil {
		json.datum = json.getDatum()
	}
}
