package proj

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func splitWKTName(secData string) (name, data string) {
	comma := strings.Index(secData, ",")
	name = secData[0:comma]
	data = secData[comma+1 : len(secData)]
	return
}

func (sr *SR) parseWKTProjCS(secName []string, secData string) error {
	if len(secName) == 1 {
		name, data := splitWKTName(secData)
		sr.SRSCode = name
		return sr.parseWKTSection(secName, data)
	}
	switch secName[1] {
	case "GEOGCS":
		sr.parseWKTGeogCS(secName, secData)
	case "PRIMEM":
		if err := sr.parseWKTPrimeM(secName, secData); err != nil {
			return err
		}
	case "PROJECTION":
		sr.parseWKTProjection(secName, secData)
	case "PARAMETER":
		if err := sr.parseWKTParameter(secName, secData); err != nil {
			return err
		}
	case "UNIT":
		if err := sr.parseWKTUnit(secName, secData); err != nil {
			return err
		}
	case "AUTHORITY": // This holds for example the ESPG number.
	case "AXIS":
	default:
		return fmt.Errorf("proj.parseWKTProjCS: unknown WKT section %#v", secName)
	}
	return nil
}

func stringInArray(s string, a []string) bool {
	for _, aa := range a {
		if aa == s {
			return true
		}
	}
	return false
}

func (sr *SR) parseWKTGeogCS(secName []string, secData string) error {
	if secName[len(secName)-1] == "GEOGCS" {
		name, data := splitWKTName(secData)
		// Set the datum name to the GEOCS name in case we don't find a datum.
		sr.DatumCode = strings.ToLower(name)
		sr.datumRename()
		return sr.parseWKTSection(secName, data)
	} else if stringInArray("DATUM", secName) {
		return sr.parseWKTDatum(secName, secData)
	} else if secName[len(secName)-1] == "PRIMEM" {
		return sr.parseWKTPrimeM(secName, secData)
	} else if secName[len(secName)-1] == "UNIT" && sr.Name == longlat {
		return sr.parseWKTUnit(secName, secData)
	} else if secName[len(secName)-1] == "AUTHORITY" {
		return nil // Don't do anything with authority for now.
	}
	return fmt.Errorf("proj.parseWKTGeogCS: unknown WKT section %v", secName)
}

func (sr *SR) parseWKTDatum(secName []string, secData string) error {
	switch secName[len(secName)-1] {
	case "DATUM":
		name, data := splitWKTName(secData)
		sr.DatumCode = strings.ToLower(strings.Trim(name, "\" "))
		sr.datumRename()
		return sr.parseWKTSection(secName, data)
	case "SPHEROID":
		if err := sr.parseWKTSpheroid(secName, secData); err != nil {
			return err
		}
	case "TOWGS84":
		s := strings.Split(secData, ",")
		sr.DatumParams = make([]float64, len(s))
		for i, ss := range s {
			var err error
			sr.DatumParams[i], err = strconv.ParseFloat(strings.TrimSpace(ss), 64)
			if err != nil {
				return err
			}
		}
	case "AUTHORITY": // Don't do anything with this for now.
	default:
		return fmt.Errorf("proj.parseWKTDatum: unknown WKT section %v", secName)
	}
	return nil
}

func (sr *SR) datumRename() {
	if sr.DatumCode[0:2] == "d_" {
		sr.DatumCode = sr.DatumCode[2:len(sr.DatumCode)]
	}
	if sr.DatumCode == "new_zealand_geodetic_datum_1949" ||
		sr.DatumCode == "new_zealand_1949" {
		sr.DatumCode = "nzgd49"
	}
	if sr.DatumCode == "wgs_1984" {
		if sr.Name == "Mercator_Auxiliary_Sphere" {
			sr.sphere = true
		}
		sr.DatumCode = "wgs84"
	}
	if strings.HasSuffix(sr.DatumCode, "_ferro") {
		sr.DatumCode = strings.TrimSuffix(sr.DatumCode, "_ferro")
	}
	if strings.HasSuffix(sr.DatumCode, "_jakarta") {
		sr.DatumCode = strings.TrimSuffix(sr.DatumCode, "_jakarta")
	}
	if strings.Contains(sr.DatumCode, "belge") {
		sr.DatumCode = "rnb72"
	}
}

func (sr *SR) parseWKTSpheroid(secName []string, secData string) error {
	d := strings.Split(secData, ",")
	sr.Ellps = strings.Replace(strings.Trim(d[0], "\""), "_19", "", -1)
	sr.Ellps = strings.Replace(sr.Ellps, "clarke_18", "clrk", -1)
	sr.Ellps = strings.Replace(sr.Ellps, "Clarke_18", "clrk", -1)
	if len(sr.Ellps) >= 13 && strings.ToLower(sr.Ellps[0:13]) == "international" {
		sr.Ellps = "intl"
	}
	a, err := strconv.ParseFloat(d[1], 64)
	if err != nil {
		return fmt.Errorf("in proj.parseWKTSpheroid a: '%v'", err)
	}
	sr.A = a
	sr.Rf, err = strconv.ParseFloat(d[2], 64)
	if err != nil {
		return fmt.Errorf("in proj.parseWKTSpheroid rf: '%v'", err)
	}
	if strings.Contains(sr.DatumCode, "osgb_1936") {
		sr.DatumCode = "osgb36"
	}
	if math.IsInf(sr.B, 0) {
		sr.B = sr.A
	}
	return nil
}

func (sr *SR) parseWKTProjection(secName []string, secData string) {
	if strings.Contains(secData, ",") {
		// Sometimes the projection has an authority after it, which we aren't
		// currently interested in.
		sr.Name = strings.Trim(strings.Split(secData, ",")[0], "\" ")
	} else {
		sr.Name = strings.Trim(secData, "\"")
	}
}

func (sr *SR) parseWKTParameter(secName []string, secData string) error {
	v := strings.Split(secData, ",")
	name := strings.Trim(strings.ToLower(v[0]), "\"")
	val, err := strconv.ParseFloat(strings.TrimSpace(v[1]), 64)
	if err != nil {
		return fmt.Errorf("in proj.parseWKTParameter: %v", err)
	}
	switch name {
	case "standard_parallel_1":
		sr.Lat1 = val * deg2rad
	case "standard_parallel_2":
		sr.Lat2 = val * deg2rad
	case "false_easting":
		sr.X0 = val
	case "false_northing":
		sr.Y0 = val
	case "latitude_of_origin":
		sr.Lat0 = val * deg2rad
	case "central_parallel":
		sr.Lat0 = val * deg2rad
	case "scale_factor":
		sr.K0 = val
	case "latitude_of_center":
		sr.Lat0 = val * deg2rad
	case "longitude_of_center":
		sr.LongC = val * deg2rad
	case "central_meridian":
		sr.Long0 = val * deg2rad
	case "azimuth":
		sr.Alpha = val * deg2rad
	case "auxiliary_sphere_type", "rectified_grid_angle":
		// TODO: Figure out if this is important.
	default:
		return fmt.Errorf("proj.parseWKTParameter: unknown name %v", name)
	}
	return nil
}

func (sr *SR) parseWKTPrimeM(secName []string, secData string) error {
	v := strings.Split(secData, ",")
	name := strings.ToLower(strings.Trim(v[0], "\""))
	if name != "greenwich" {
		return fmt.Errorf("in proj.parseWTKPrimeM: prime meridian is %s but"+
			"only greenwich is supported", name)
	}
	return nil
}

func (sr *SR) parseWKTUnit(secName []string, secData string) error {
	v := strings.Split(secData, ",")
	sr.Units = strings.Trim(strings.ToLower(v[0]), "\"")
	if sr.Units == "metre" {
		sr.Units = "meter"
	}
	if len(v) > 1 {
		convert, err := strconv.ParseFloat(strings.TrimSpace(v[1]), 64)
		if err != nil {
			return fmt.Errorf("in proj.parseWKTUnit: %v", err)
		}
		if sr.Name == longlat {
			sr.ToMeter = convert * sr.A
		} else {
			sr.ToMeter = convert
		}
	}
	return nil
}

// wkt parses a WKT specification.
func wkt(wkt string) (*SR, error) {
	sr := NewSR()
	err := sr.parseWKTSection([]string{}, wkt)

	// Convert units to meters.
	sr.X0 *= sr.ToMeter
	sr.Y0 *= sr.ToMeter
	if math.IsNaN(sr.Lat0) {
		sr.Lat0 = sr.Lat1
	}

	return sr, err
}

// parseWKTSection is a recursive function to parse a WKT specification.
func (sr *SR) parseWKTSection(secName []string, secData string) error {
	open, close := findWKTSections(secData)
	if len(open) != len(close) {
		return fmt.Errorf("proj: malformed WKT section '%s'", secData)
	}
	for i, o := range open {
		c := close[i]
		name := strings.Trim(secData[0:o], ", ")
		if strings.Contains(name, ",") {
			comma := strings.LastIndex(name, ",")
			name = strings.TrimSpace(name[comma+1 : len(name)])
		}
		secNameO := append(secName, name)
		secDataO := secData[o+1 : c]
		var err error
		switch secNameO[0] {
		case "PROJCS":
			err = sr.parseWKTProjCS(secNameO, secDataO)
		case "GEOGCS":
			// This should only happen if there is no PROJCS.
			sr.Name = longlat
			err = sr.parseWKTGeogCS(secNameO, secDataO)
		case "LOCAL_CS":
			sr.Name = "identity"
			sr.local = true
		default:
			err = fmt.Errorf("proj: unknown WKT section name '%s'", secNameO)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// findWKTSections steps through all or part of a WKT specifications
// to find matching outermost-level brackets.
func findWKTSections(secData string) (open, close []int) {
	nest := 0
	for i := 0; i < len(secData); i++ {
		if secData[i] == '[' {
			if nest == 0 {
				open = append(open, i)
			}
			nest++
		} else if secData[i] == ']' {
			nest--
			if nest == 0 {
				close = append(close, i)
			}
		}
	}
	return
}
