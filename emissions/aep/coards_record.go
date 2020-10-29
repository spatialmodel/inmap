/*
Copyright © 2019 the InMAP authors.
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
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/ctessum/cdf"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
)

// gridPointsToGridSpacing returns the size of the grid cell at index
// i when given the grid center points.
func gridPointsToGridSpacing(gridPoints []float64, i int) float64 {
	if i == 0 {
		return gridPoints[1] - gridPoints[0]
	} else if i == len(gridPoints)-1 {
		return gridPoints[len(gridPoints)-1] - gridPoints[len(gridPoints)-2]
	}
	return (gridPoints[i+1] - gridPoints[i-1]) / 2
}

// readCOARDSVar eads a floating point variable from a COARDS file.
// It will return nil if the variable is not floating point.
func readCOARDSVar(nc *cdf.File, v string) ([]float64, error) {
	r := nc.Reader(v, nil, nil)
	dataI := r.Zero(-1)
	switch dataI.(type) {
	case []float32, []float64:
	default:
		return nil, nil
	}
	_, err := r.Read(dataI)
	if err != nil {
		return nil, err
	}
	var data []float64
	switch dataI.(type) {
	case []float64:
		data = dataI.([]float64)
	case []float32:
		dat32 := dataI.([]float32)
		data = make([]float64, len(dat32))
		for i, v := range dat32 {
			data[i] = float64(v)
		}
	default:
		panic("this shouldn't happen")
	}

	noDataI := nc.Header.GetAttribute(v, "_FillValue")
	if noDataI != nil {
		var noData float64
		switch noDataI.(type) {
		case []float32:
			noData = float64(noDataI.([]float32)[0])
		case []float64:
			noData = noDataI.([]float64)[0]
		default:
			return nil, fmt.Errorf("invalid type for COARDS FillValue: %T", noDataI)
		}
		for i, d := range data {
			if d == noData {
				data[i] = math.NaN()
			}
		}
	}
	return data, nil
}

// ReadCOARDSFile reads a COARDS-compliant NetCDF file
// (NetCDF 4 and greater not supported) and returns a record generator.
// The generator will return io.EOF after the last record.
// All floating point variables that have dimensions [lat, lon] are
// assumed to be emissions variables.
// begin and end specify the time period when the emissions occur.
// units represents the input units of the emissions.
// SourceData specifies additional information to be included in each
// emissions record.
// Data in the COARDS file are assumed to be row-major (i.e., latitude-major).
// Information regarding the COARDS NetCDF conventions are
// available here: https://ferret.pmel.noaa.gov/Ferret/documentation/coards-netcdf-conventions.
func ReadCOARDSFile(file string, begin, end time.Time, units InputUnits, sourceData SourceData) (func() (Record, error), error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("aep: opening COARDS file %s: %v", file, err)
	}
	defer f.Close()
	nc, err := cdf.Open(f)
	if err != nil {
		return nil, fmt.Errorf("aep: opening COARDS file %s: %v", file, err)
	}

	sr, err := proj.Parse("+proj=longlat")
	if err != nil {
		panic(err)
	}

	// Read in emissions variables.
	variables := make(map[string][]float64)
	for _, v := range nc.Header.Variables() {
		dims := nc.Header.Dimensions(v)
		if len(dims) != 2 || dims[0] != "lat" || dims[1] != "lon" {
			continue
		}
		data, err := readCOARDSVar(nc, v)
		if err != nil {
			return nil, fmt.Errorf("aep: reading variable %s from COARDS file %s: %v", v, file, err)
		}
		if data != nil {
			variables[v] = data
		}
	}
	lons, err := readCOARDSVar(nc, "lon")
	if err != nil {
		return nil, fmt.Errorf("aep: reading variable %s from COARDS file %s: %v", "lon", file, err)
	}
	lats, err := readCOARDSVar(nc, "lat")
	if err != nil {
		return nil, fmt.Errorf("aep: reading variable %s from COARDS file %s: %v", "lat", file, err)
	}
	if len(lons) < 2 || len(lats) < 2 {
		return nil, fmt.Errorf("aep: reading from COARDS file %s: lat and lon variables must be length >= 2 but are %d and %d", file, len(lats), len(lons))
	}

	convert := units.Conversion(1)
	durationSeconds := end.Sub(begin).Seconds()
	duration := unit.New(durationSeconds, unit.Second)

	var i, j int
	generator := func() (Record, error) {
		if j == len(lats) {
			return nil, io.EOF
		}
		dy := gridPointsToGridSpacing(lats, j)
		dx := gridPointsToGridSpacing(lons, i)
		y := lats[j]
		x := lons[i]
		min := geom.Point{X: x - dx/2, Y: y - math.Abs(dy/2)}
		max := geom.Point{X: x + dx/2, Y: y + math.Abs(dy/2)}

		r := &basicPolygonRecord{
			Polygonal:    &geom.Bounds{Min: min, Max: max},
			SourceData:   sourceData,
			SR:           sr,
			LocationName: fmt.Sprintf("%d_%d", j, i),
		}

		e := new(Emissions)
		for name, data := range variables {
			rate := unit.Div(
				convert(data[len(lons)*j+i]),
				duration,
			)
			e.Add(begin, end, name, "", rate)
		}
		r.Emissions = *e

		i++
		if i == len(lons) {
			i = 0
			j++
		}
		return r, nil
	}
	return generator, nil
}
