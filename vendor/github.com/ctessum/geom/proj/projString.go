package proj

import (
	"fmt"
	"strconv"
	"strings"
)

const deg2rad = 0.01745329251994329577

func projString(defData string) (*SR, error) {
	self := NewSR()
	var err error
	for i, a := range strings.Split(defData, "+") {
		if i == 0 {
			continue // skip everything to the left of the first +
		}
		a = strings.TrimSpace(a)
		split := strings.Split(a, "=")
		split = append(split, "true")
		paramName := strings.ToLower(split[0])
		paramVal := split[1]

		switch paramName {
		case "proj":
			self.Name = paramVal
		case "title":
			self.Title = paramVal
		case "datum":
			self.DatumCode = paramVal
		case "rf":
			self.Rf, err = strconv.ParseFloat(paramVal, 64)
		case "lat_0":
			self.Lat0, err = strconv.ParseFloat(paramVal, 64)
			self.Lat0 *= deg2rad
		case "lat_1":
			self.Lat1, err = strconv.ParseFloat(paramVal, 64)
			self.Lat1 *= deg2rad
		case "lat_2":
			self.Lat2, err = strconv.ParseFloat(paramVal, 64)
			self.Lat2 *= deg2rad
		case "lat_ts":
			self.LatTS, err = strconv.ParseFloat(paramVal, 64)
			self.LatTS *= deg2rad
		case "lon_0":
			self.Long0, err = strconv.ParseFloat(paramVal, 64)
			self.Long0 *= deg2rad
		case "lon_1":
			self.Long1, err = strconv.ParseFloat(paramVal, 64)
			self.Long1 *= deg2rad
		case "lon_2":
			self.Long2, err = strconv.ParseFloat(paramVal, 64)
			self.Long2 *= deg2rad
		case "alpha":
			self.Alpha, err = strconv.ParseFloat(paramVal, 64)
			self.Alpha *= deg2rad
		case "lonc":
			self.LongC, err = strconv.ParseFloat(paramVal, 64)
			self.LongC *= deg2rad
		case "x_0":
			self.X0, err = strconv.ParseFloat(paramVal, 64)
		case "y_0":
			self.Y0, err = strconv.ParseFloat(paramVal, 64)
		case "k_0", "k":
			self.K0, err = strconv.ParseFloat(paramVal, 64)
		case "a":
			self.A, err = strconv.ParseFloat(paramVal, 64)
		case "b":
			self.B, err = strconv.ParseFloat(paramVal, 64)
		case "ellps":
			self.Ellps = paramVal
		case "r_a":
			self.Ra = true
		case "zone":
			self.Zone, err = strconv.ParseFloat(paramVal, 64)
		case "south":
			self.UTMSouth = true
		case "no_defs":
			self.NoDefs = true
		case "towgs84":
			split := strings.Split(paramVal, ",")
			self.DatumParams = make([]float64, len(split))
			for i, s := range split {
				self.DatumParams[i], err = strconv.ParseFloat(s, 64)
				if err != nil {
					return nil, err
				}
			}
		case "to_meter":
			self.ToMeter, err = strconv.ParseFloat(paramVal, 64)
		case "units":
			self.Units = paramVal
			if u, ok := units[paramVal]; ok {
				self.ToMeter = u.to_meter
			}
		case "from_greenwich":
			self.FromGreenwich, err = strconv.ParseFloat(paramVal, 64)
			self.FromGreenwich *= deg2rad
		case "pm":
			if pm, ok := primeMeridian[paramVal]; ok {
				self.FromGreenwich = pm
			} else {
				self.FromGreenwich, err = strconv.ParseFloat(paramVal, 64)
				self.FromGreenwich *= deg2rad
			}
		case "nadgrids":
			if paramVal == "@null" {
				self.DatumCode = "none"
			} else {
				self.NADGrids = paramVal
			}
		case "axis":
			legalAxis := "ewnsud"
			if len(paramVal) == 3 && strings.Index(legalAxis, paramVal[0:1]) != -1 &&
				strings.Index(legalAxis, paramVal[1:2]) != -1 &&
				strings.Index(legalAxis, paramVal[2:3]) != -1 {
				self.Axis = paramVal
			}
		default:
			err = fmt.Errorf("proj: invalid field '%s'", paramName)
		}
		if err != nil {
			return nil, err
		}
	}
	if self.DatumCode != "WGS84" {
		self.DatumCode = strings.ToLower(self.DatumCode)
	}
	return self, nil
}
