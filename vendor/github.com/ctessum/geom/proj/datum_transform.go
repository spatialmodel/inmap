package proj

import (
	"fmt"
	"math"
)

const (
	srsWGS84SemiMajor = 6378137              // only used in grid shift transforms
	srsWGS84ESquared  = 0.006694379990141316 //DGR: 2012-07-29
)

func checkDatumParams(fallback datumType) bool {
	return (fallback == pjd3Param || fallback == pjd7Param)
}

func datumTransform(source, dest *datum, x, y, z float64) (float64, float64, float64, error) {
	var err error

	// Short cut if the datums are identical.
	if source.compare_datums(dest) {
		return x, y, z, nil // in this case, zero is sucess,
		// whereas cs_compare_datums returns 1 to indicate TRUE
		// confusing, should fix this
	}

	// Explicitly skip datum transform by setting 'datum=none' as parameter for either source or dest
	if source.datum_type == pjdNoDatum || dest.datum_type == pjdNoDatum {
		return x, y, z, nil
	}

	//DGR: 2012-07-29 : add nadgrids support (begin)
	var src_a = source.a
	var src_es = source.es

	var dst_a = dest.a
	var dst_es = dest.es

	var fallback = source.datum_type
	// If this datum requires grid shifts, then apply it to geodetic coordinates.
	if fallback == pjdGridShift {
		err := fmt.Errorf("in proj.datumTransform: gridshift not supported")
		return math.NaN(), math.NaN(), math.NaN(), err
		/*if this.apply_gridshift(source, 0, x, y, z) == 0 {
			source.a = SRS_WGS84_SEMIMAJOR
			source.es = SRS_WGS84_ESQUARED
		} else {
			// try 3 or 7 params transformation or nothing ?
			if len(source.datum_params) == 0 {
				source.a = src_a
				source.es = source.es
				return x, y, z, nil
			}
			wp = 1
			for i := 0; i < len(source.datum_params); i++ {
				wp *= source.datum_params[i]
			}
			if wp == 0 {
				source.a = src_a
				source.es = source.es
				return x, y, z, nil
			}
			if len(source.datum_params) > 3 {
				fallback = PJD_7PARAM
			} else {
				fallback = PJD_3PARAM
			}
		}*/
	}
	if dest.datum_type == pjdGridShift {
		dest.a = srsWGS84SemiMajor
		dest.es = srsWGS84ESquared
	}
	// Do we need to go through geocentric coordinates?
	if source.es != dest.es || source.a != dest.a || checkDatumParams(fallback) ||
		checkDatumParams(dest.datum_type) {
		//DGR: 2012-07-29 : add nadgrids support (end)
		// Convert to geocentric coordinates.
		x, y, z, err = source.geodetic_to_geocentric(x, y, z)
		if err != nil {
			return math.NaN(), math.NaN(), math.NaN(), err
		}
		// CHECK_RETURN;
		// Convert between datums
		if checkDatumParams(source.datum_type) {
			x, y, z = source.geocentric_to_wgs84(x, y, z)
			// CHECK_RETURN;
		}
		if checkDatumParams(dest.datum_type) {
			x, y, z = dest.geocentric_from_wgs84(x, y, z)
			// CHECK_RETURN;
		}
		// Convert back to geodetic coordinates
		x, y, z = dest.geocentric_to_geodetic(x, y, z)
		// CHECK_RETURN;
	}
	// Apply grid shift to destination if required
	if dest.datum_type == pjdGridShift {
		err := fmt.Errorf("in proj.datumTransform: gridshift not supported")
		return math.NaN(), math.NaN(), math.NaN(), err
		//this.apply_gridshift(dest, 1, x, y, z)
		// CHECK_RETURN;
	}

	source.a = src_a
	source.es = src_es
	dest.a = dst_a
	dest.es = dst_es

	return x, y, z, nil
}
