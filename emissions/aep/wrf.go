/*
Copyright © 2017 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.*/

package aep

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/ctessum/geom/proj"
)

// WRFconfigData hold information about a WRF simulation configuration.
type WRFconfigData struct {
	MaxDom             int
	ParentID           []int
	ParentGridRatio    []float64
	IParentStart       []int
	JParentStart       []int
	EWE                []int
	ESN                []int
	Dx0                float64
	Dy0                float64
	MapProj            string
	RefLat             float64
	RefLon             float64
	TrueLat1           float64
	TrueLat2           float64
	StandLon           float64
	RefX               float64
	RefY               float64
	S                  []float64
	W                  []float64
	Dx                 []float64
	Dy                 []float64
	Nx                 []int
	Ny                 []int
	DomainNames        []string
	FramesPerAuxInput5 []int
	Kemit              int
	Nocolons           bool
	sr                 *proj.SR
}

// ParseWRFConfig extracts configuration information from a set of WRF namelists.
func ParseWRFConfig(wpsnamelist, wrfnamelist string) (d *WRFconfigData, err error) {
	e := new(wrfErrCat)
	d = new(WRFconfigData)
	d.parseWPSnamelist(wpsnamelist, e)
	d.parseWRFnamelist(wrfnamelist, e)
	d.projection(e)
	err = e.convertToError()
	return
}

// projection calculates the spatial projection of a WRF configuration.
func (d *WRFconfigData) projection(e *wrfErrCat) {
	const EarthRadius = 6370997.

	var mapProj string
	switch d.MapProj {
	case "lambert":
		mapProj = "lcc"
	case "lat-lon":
		mapProj = "longlat"
	case "merc":
		mapProj = "merc"
	default:
		e.Add(fmt.Errorf("ERROR: `lambert', `lat-lon', and `merc' "+
			"are the only map projections"+
			" that are currently supported (your projection is `%v').",
			d.MapProj))
	}
	d.sr = proj.NewSR()
	d.sr.Name = mapProj
	d.sr.Lat1 = d.TrueLat1
	d.sr.Lat2 = d.TrueLat2
	d.sr.Lat0 = d.RefLat
	d.sr.Long0 = d.RefLon
	d.sr.A = EarthRadius
	d.sr.B = EarthRadius
	d.sr.ToMeter = 1.
	d.sr.DeriveConstants()
}

// Grids creates grid definitions for the grids in WRF configuration d,
// where tzFile is a shapefile containing timezone information, and tzColumn
// is the data attribute column within that shapefile that contains the
// timezone offsets in hours.
func (d *WRFconfigData) Grids() []*GridDef {
	grids := make([]*GridDef, d.MaxDom)
	for i := 0; i < d.MaxDom; i++ {
		grids[i] = NewGridRegular(d.DomainNames[i], d.Nx[i], d.Ny[i],
			d.Dx[i], d.Dy[i], d.W[i], d.S[i], d.sr)
	}
	return grids
}

// Parse a WPS namelist
func (d *WRFconfigData) parseWPSnamelist(filename string, e *wrfErrCat) {
	file, err := os.Open(filename)
	if err != nil {
		e.Add(err)
		return
	}
	includesRefx := false
	includesRefy := false
	f := bufio.NewReader(file)
	for {
		line, err := f.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				e.Add(err)
				break
			} else {
				break
			}
		}
		i := strings.Index(line, "=")
		if i != -1 {
			name := strings.Trim(line[:i], " ,")
			val := strings.Trim(line[i+1:], " ,\n")
			switch name {
			case "max_dom":
				d.MaxDom = namelistInt(val)
			case "map_proj":
				d.MapProj = strings.Trim(val, " '")
			case "ref_lat":
				d.RefLat = namelistFloat(val)
			case "ref_lon":
				d.RefLon = namelistFloat(val)
			case "truelat1":
				d.TrueLat1 = namelistFloat(val)
			case "truelat2":
				d.TrueLat2 = namelistFloat(val)
			case "stand_lon":
				d.StandLon = namelistFloat(val)
			case "ref_x":
				d.RefX = namelistFloat(val)
				includesRefx = true
			case "ref_y":
				d.RefY = namelistFloat(val)
				includesRefy = true
			case "parent_id":
				d.ParentID = namelistIntList(val)
			case "parent_grid_ratio":
				d.ParentGridRatio = namelistFloatList(val)
			case "i_parent_start":
				d.IParentStart = namelistIntList(val)
			case "j_parent_start":
				d.JParentStart = namelistIntList(val)
			case "e_we":
				d.EWE = namelistIntList(val)
			case "e_sn":
				d.ESN = namelistIntList(val)
			case "dx":
				d.Dx0 = namelistFloat(val)
			case "dy":
				d.Dy0 = namelistFloat(val)
			}
		}
	}
	if !includesRefx {
		d.RefX = float64(d.EWE[0]) / 2.
	}
	if !includesRefy {
		d.RefY = float64(d.ESN[0]) / 2.
	}
	d.S = make([]float64, d.MaxDom)
	d.W = make([]float64, d.MaxDom)
	switch d.MapProj {
	case "lat-lon":
		d.S[0] = d.RefLat - (d.RefY-0.5)*d.Dy0
		d.W[0] = d.RefLon - (d.RefX-0.5)*d.Dx0
	default:
		d.S[0] = 0 - (d.RefY-0.5)*d.Dy0
		d.W[0] = 0 - (d.RefX-0.5)*d.Dx0
	}
	d.Dx = make([]float64, d.MaxDom)
	d.Dy = make([]float64, d.MaxDom)
	d.Dx[0] = d.Dx0
	d.Dy[0] = d.Dy0
	d.Nx = make([]int, d.MaxDom)
	d.Ny = make([]int, d.MaxDom)
	d.Nx[0] = d.EWE[0] - 1
	d.Ny[0] = d.ESN[0] - 1
	d.DomainNames = make([]string, d.MaxDom)
	for i := 0; i < d.MaxDom; i++ {
		parentID := d.ParentID[i] - 1
		d.DomainNames[i] = fmt.Sprintf("d%02v", i+1)
		d.S[i] = d.S[parentID] +
			float64(d.JParentStart[i]-1)*d.Dy[parentID]
		d.W[i] = d.W[parentID] +
			float64(d.IParentStart[i]-1)*d.Dx[parentID]
		d.Dx[i] = d.Dx[parentID] /
			d.ParentGridRatio[i]
		d.Dy[i] = d.Dy[parentID] /
			d.ParentGridRatio[i]
		d.Nx[i] = d.EWE[i] - 1
		d.Ny[i] = d.ESN[i] - 1
	}
}

// Parse a WRF namelist
func (d *WRFconfigData) parseWRFnamelist(filename string, e *wrfErrCat) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	f := bufio.NewReader(file)
	for {
		line, err := f.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				panic(err)
			} else {
				break
			}
		}
		i := strings.Index(line, "=")
		if i != -1 {
			name := strings.Trim(line[:i], " ,")
			val := strings.Trim(line[i+1:], " ,\n")
			switch name {
			case "max_dom":
				e.compare(d.MaxDom, namelistInt(val), name)
			case "parent_id":
				e.compare(d.ParentID, namelistIntList(val), name)
			case "parent_grid_ratio":
				e.compare(d.ParentGridRatio, namelistFloatList(val), name)
			case "i_parent_start":
				e.compare(d.IParentStart, namelistIntList(val), name)
			case "j_parent_start":
				e.compare(d.JParentStart, namelistIntList(val), name)
			case "e_we":
				e.compare(d.EWE, namelistIntList(val), name)
			case "e_sn":
				e.compare(d.ESN, namelistIntList(val), name)
			case "dx":
				e.compare(d.Dx0, namelistFloatList(val)[0], name)
			case "dy":
				e.compare(d.Dy0, namelistFloatList(val)[0], name)
			case "frames_per_auxinput5":
				// Interval will be 60 minutes regardless of input file
				// All domains will have the same number of frames per file
				d.FramesPerAuxInput5 = namelistIntList(val)
			case "kemit":
				d.Kemit = namelistInt(val)
			case "nocolons":
				d.Nocolons = namelistBool(val)
			}
		}
	}
}

func namelistInt(str string) (out int) {
	out, err := strconv.Atoi(strings.Trim(str, " "))
	if err != nil {
		panic(err)
	}
	return
}
func namelistIntList(str string) (out []int) {
	out = make([]int, 0)
	for _, ival := range strings.Split(str, ",") {
		xval, err := strconv.Atoi(strings.Trim(ival, " "))
		if err != nil {
			panic(err)
		}
		out = append(out, xval)
	}
	return
}
func namelistFloat(str string) (out float64) {
	out, err := strconv.ParseFloat(strings.Trim(str, " "), 64)
	if err != nil {
		panic(err)
	}
	return
}
func namelistFloatList(str string) (out []float64) {
	out = make([]float64, 0)
	for _, ival := range strings.Split(str, ",") {
		xval, err := strconv.ParseFloat(strings.Trim(ival, " "), 64)
		if err != nil {
			panic(err)
		}
		out = append(out, xval)
	}
	return
}
func namelistBool(str string) (out bool) {
	out, err := strconv.ParseBool(strings.Trim(str, " ."))
	if err != nil {
		panic(err)
	}
	return
}

// The ErrCat type and methods collect errors while the program is running
// and then print them later so that all errors can be seen and fixed at once,
// instead of just the first one.
type wrfErrCat struct {
	str string
}

func (e *wrfErrCat) Add(err error) {
	if err != nil && strings.Index(e.str, err.Error()) == -1 {
		e.str += err.Error() + "\n"
	}
	return
}
func (e *wrfErrCat) convertToError() error {
	if e.str != "" {
		return errors.New(e.str)
	}
	return nil
}

func (e *wrfErrCat) compare(val1, val2 interface{}, name string) {
	errFlag := false
	switch val1.(type) {
	case int:
		if val1.(int) != val2.(int) {
			errFlag = true
		}
	case float64:
		errFlag = floatcompare(val1.(float64), val2.(float64))
	case []int:
		for i := 0; i < min(len(val1.([]int)), len(val2.([]int))); i++ {
			if val1.([]int)[i] != val2.([]int)[i] {
				errFlag = true
				break
			}
		}
	case []float64:
		for i := 0; i < min(len(val1.([]float64)), len(val2.([]float64))); i++ {
			if floatcompare(val1.([]float64)[i], val2.([]float64)[i]) {
				errFlag = true
				break
			}
		}
	case string:
		if val1.(string) != val2.(string) {
			errFlag = true
		}
	default:
		panic("Unknown type")
	}
	if errFlag {
		e.Add(fmt.Errorf("WRF variable mismatch for %v, WPS namelist=%v; "+
			"WRF namelist=%v.", name, val1, val2))
	}
}

func floatcompare(val1, val2 float64) bool {
	return math.Abs((val1-val2)/val2) > 1.e-8
}

func min(val1, val2 int) int {
	if val1 > val2 {
		return val2
	}
	return val1
}
