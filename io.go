/*
Copyright © 2013 the InMAP authors.
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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmap

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ctessum/aep"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/op"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
	goshp "github.com/jonas-p/go-shp"
)

// AddEmissionsFlux adds emissions to c.Cf and sets c.Ci equal to c.Cf.
// It should be run once for each timestep,
// and it should not be run in parallel with other CellManipulators.
func AddEmissionsFlux() CellManipulator {
	return func(c *Cell, Dt float64) {
		if c.EmisFlux != nil {
			for i := range PolNames {
				c.Cf[i] += c.EmisFlux[i] * Dt
				c.Ci[i] = c.Cf[i]
			}
		}
	}
}

// Emissions is a holder for input emissions data.
type Emissions struct {
	data *rtree.Rtree
}

// EmisRecord is a holder for an emissions record.
type EmisRecord struct {
	geom.Geom
	VOC, NOx, NH3, SOx float64 // emissions [μg/s]
	PM25               float64 `shp:"PM2_5"` // emissions [μg/s]
	Height             float64 // stack height [m]
	Diam               float64 // stack diameter [m]
	Temp               float64 // stack temperature [K]
	Velocity           float64 // stack velocity [m/s]
}

// add adds the emissions in o to the receiver.
func (e *EmisRecord) add(o *EmisRecord) {
	e.VOC += o.VOC
	e.NOx += o.NOx
	e.NH3 += o.NH3
	e.SOx += o.SOx
	e.PM25 += o.PM25
}

// NewEmissions Initializes a new emissions holder.
func NewEmissions() *Emissions {
	return &Emissions{
		data: rtree.NewTree(25, 50),
	}
}

// Add adds an emissions record to e.
func (e *Emissions) Add(er *EmisRecord) {
	e.data.Insert(er)
}

// ReadEmissionShapefiles returns the emissions data in the specified shapefiles,
// and converts them to the spatial reference gridSR. Input units are specified
// by units; options are tons/year and kg/year. Output units = μg/s.
// c is a channel over which status updates will be sent. If c is nil,
// no updates will be sent.
func ReadEmissionShapefiles(gridSR *proj.SR, units string, c chan string, shapefiles ...string) (*Emissions, error) {

	var emisConv float64
	switch units {
	case "tons/year":
		// Input units = tons/year; output units = μg/s
		const massConv = 907184740000. // μg per short ton
		const timeConv = 3600. * 8760. // seconds per year
		emisConv = massConv / timeConv // convert tons/year to μg/s
	case "kg/year":
		// Input units = kg/year; output units = μg/s
		const massConv = 1.e9          // μg per kg
		const timeConv = 3600. * 8760. // seconds per year
		emisConv = massConv / timeConv // convert kg/year to μg/s
	}

	// Add in emissions shapefiles
	// Load emissions into rtree for fast searching
	emis := NewEmissions()
	for _, fname := range shapefiles {
		if c != nil {
			c <- fmt.Sprintf("Loading emissions shapefile: %s.", fname)
		}
		fname = strings.Replace(fname, ".shp", "", -1)
		f, err := shp.NewDecoder(fname + ".shp")
		if err != nil {
			return nil, fmt.Errorf("there was a problem reading the emissions shapefile '%s'. "+
				"The error message was %v.", fname, err)
		}
		sr, err := f.SR()
		if err != nil {
			return nil, fmt.Errorf("there was a problem reading the projection information for "+
				"the emissions shapefile '%s'. The error message was %v.", fname, err)
		}
		trans, err := sr.NewTransform(gridSR)
		if err != nil {
			return nil, fmt.Errorf("there was a problem creating a spatial reprojector for "+
				"the emissions shapefile '%s'. The error message was %v.", fname, err)
		}
		for {
			var e EmisRecord
			if ok := f.DecodeRow(&e); !ok {
				break
			}

			e.Geom, err = e.Transform(trans)
			if err != nil {
				return nil, fmt.Errorf("there was a problem spatially reprojecting in "+
					"emissions file %s. The error message was %v", fname, err)
			}

			e.VOC *= emisConv
			e.NOx *= emisConv
			e.NH3 *= emisConv
			e.SOx *= emisConv
			e.PM25 *= emisConv

			if math.IsNaN(e.Height) {
				e.Height = 0.
			}
			if math.IsNaN(e.Diam) {
				e.Diam = 0.
			}
			if math.IsNaN(e.Temp) {
				e.Temp = 0.
			}
			if math.IsNaN(e.Velocity) {
				e.Velocity = 0.
			}
			emis.Add(&e)
		}
		f.Close()
		if err := f.Error(); err != nil {
			return nil, fmt.Errorf("problem reading emissions shapefile."+
				"\nfile: %s\nerror: %v", fname, err)
		}
	}
	return emis, nil
}

// FromAEP converts the given AEP (github.com/ctessum/aep) records to
// EmisRecords using the given SpatialProcessor and the SpatialProcessor
// grid index gi. VOC, NOx, NH3, SOx, and PM25 are lists of
// AEP Polluants that should be mapped to those InMAP species.
// The returned EmisRecords will be grouped as much as possible to minimize
// the number of records.
func FromAEP(r []aep.Record, sp *aep.SpatialProcessor, gi int, VOC, NOx, NH3, SOx, PM25 []aep.Pollutant) ([]*EmisRecord, error) {
	if gi > 0 || len(sp.Grids) <= gi {
		return nil, fmt.Errorf("inmap: converting AEP record to EmisRecord: invalid gi (%d)", gi)
	}

	checkDim := func(v *unit.Unit) float64 {
		if v == nil {
			return 0
		}
		if !v.Dimensions().Matches(unit.Kilogram) {
			panic(fmt.Errorf("bad dimensions: %v", v.Dimensions()))
		}
		return v.Value()
	}

	// Find the centroids of the grid cells.
	grid := sp.Grids[gi]
	centroids := make([]geom.Point, len(grid.Cells))
	for i, c := range grid.Cells {
		centroids[i] = c.Centroid()
	}

	var eRecs []*EmisRecord
	groundERecs := make(map[geom.Point]*EmisRecord)

	for _, rec := range r {
		gridSrg, _, inGrid, err := rec.Spatialize(sp, gi)
		if err != nil {
			return nil, err
		}
		if !inGrid {
			continue
		}
		for i, frac := range gridSrg.Elements {
			p := centroids[i]
			er := EmisRecord{
				Geom: p,
			}
			e := rec.GetEmissions().Totals()

			// Convert units.
			const (
				secPerYear        = 60 * 60 * 24 * 365
				ugPerKg           = 1.0e9
				kgPerYearToUgPerS = 1 * ugPerKg / secPerYear
			)

			// Add the emissions to the new record.
			for _, p := range VOC {
				er.VOC += checkDim(e[p]) * frac * kgPerYearToUgPerS
			}
			for _, p := range NOx {
				er.NOx += checkDim(e[p]) * frac * kgPerYearToUgPerS
			}
			for _, p := range NH3 {
				er.NH3 += checkDim(e[p]) * frac * kgPerYearToUgPerS
			}
			for _, p := range SOx {
				er.SOx += checkDim(e[p]) * frac * kgPerYearToUgPerS
			}
			for _, p := range PM25 {
				er.PM25 += checkDim(e[p]) * frac * kgPerYearToUgPerS
			}

			pointSource, ok := rec.(aep.PointSource)
			if !ok || pointSource.PointData().GroundLevel() {
				// For ground level sources, combine with other records
				// at the same point.
				if _, ok := groundERecs[p]; !ok {
					groundERecs[p] = &er
				} else {
					groundERecs[p].add(&er)
				}
			} else {
				stack := pointSource.PointData()
				er.Height = stack.StackHeight.Value()
				er.Diam = stack.StackDiameter.Value()
				er.Temp = stack.StackTemp.Value()
				er.Velocity = stack.StackVelocity.Value()
				eRecs = append(eRecs, &er)
			}
		}
	}
	for _, groundERec := range groundERecs {
		eRecs = append(eRecs, groundERec)
	}
	return eRecs, nil
}

// addEmisFlux calculates emissions flux given emissions array in units of μg/s
// and a scale for molecular mass conversion.
func (c *Cell) addEmisFlux(val float64, scale float64, iPol int) {
	fluxScale := 1. / c.Dx / c.Dy / c.Dz // μg/s /m/m/m = μg/m3/s
	c.EmisFlux[iPol] += val * scale * fluxScale
}

// calcWeightFactor calculates the fraction of emissions in e that should be
// allocated to the intersection between e and c based on the areas of lengths or areas.
func calcWeightFactor(e geom.Geom, c *Cell) float64 {
	var weightFactor float64
	switch e.(type) {
	case geom.Point:
		p := e.(geom.Point)
		in := p.Within(c)
		if in == geom.Inside {
			weightFactor = 1.
		} else if in == geom.OnEdge {
			onCorner := false
			for _, cp := range c.Polygons()[0][0] {
				if cp.Equals(p) {
					// If the point is located exactly on one of the corners of the
					// grid cell, we split the emissions evenly between this grid cell
					// and the three that it shares a corner with.
					onCorner = true
					weightFactor = 0.25
					break
				}
			}
			if !onCorner {
				// If the point is on the edge of the cell but not on the corner,
				// split the emissions between this cell and the cell that it shares
				// an edge with.
				weightFactor = 0.5
			}
		}
	case geom.Polygonal:
		poly := e.(geom.Polygonal)
		intersection := poly.Intersection(c)
		if intersection == nil {
			return 0.
		}
		weightFactor = intersection.Area() / poly.Area()
	case geom.Linear:
		var err error
		intersection, err := op.Construct(e, c.Polygonal, op.INTERSECTION)
		if err != nil {
			log.Fatalf("while allocating emissions to grid: %v", err)
		}
		if intersection == nil {
			return 0.
		}
		el := e.(geom.Linear)
		il := intersection.(geom.Linear)
		weightFactor = il.Length() / el.Length()
	default:
		log.Fatalf("unsupported geometry type: %#v in emissions file", e)
	}
	return weightFactor
}

// setEmissionsFlux sets the emissions flux for c based on the emissions in e.
func (c *Cell) setEmissionsFlux(e *Emissions) {
	c.EmisFlux = make([]float64, len(PolNames))
	for _, eTemp := range e.data.SearchIntersect(c.Bounds()) {
		e := eTemp.(*EmisRecord)
		if e.Height > 0. {
			// Figure out if this cell is at the right hight for the plume.
			in, _, err := c.IsPlumeIn(e.Height, e.Diam, e.Temp, e.Velocity)
			if err != nil {
				panic(err)
			}
			if !in {
				continue
			}
		} else if c.Layer != 0 {
			continue
		}
		weightFactor := calcWeightFactor(e.Geom, c)
		if weightFactor == 0 {
			continue
		}

		// Emissions: all except PM2.5 go to gas phase
		c.addEmisFlux(e.VOC, 1.*weightFactor, igOrg)
		c.addEmisFlux(e.NOx, NOxToN*weightFactor, igNO)
		c.addEmisFlux(e.NH3, NH3ToN*weightFactor, igNH)
		c.addEmisFlux(e.SOx, SOxToS*weightFactor, igS)
		c.addEmisFlux(e.PM25, 1.*weightFactor, iPM2_5)
	}
}

// Output returns a function that writes simulation results to a shapefile or
// shapefiles.
// If  allLayers` is true, the function writes out data for all of the vertical
// layers, otherwise only the ground-level layer is written.
// outputVariables is a map of the names of the variables for which data
// should be returned to expressions that define how the data should be calculated.
// These expressions can contain built-in InMAP variables, user-defined variables,
// and functions. For more information on the functions available for defining
// output variables see the documentation for the Results function.
func Output(fileName string, allLayers bool, outputVariables map[string]string) DomainManipulator {
	return func(d *InMAP) error {

		// Projection definition. This may need to be changed for a different
		// spatial domain.
		// TODO: Make this settable by the user, or at least check to make sure it
		// matches the InMAPProj configuration variable.
		const proj4 = `PROJCS["Lambert_Conformal_Conic",GEOGCS["GCS_unnamed ellipse",DATUM["D_unknown",SPHEROID["Unknown",6370997,0]],PRIMEM["Greenwich",0],UNIT["Degree",0.017453292519943295]],PROJECTION["Lambert_Conformal_Conic"],PARAMETER["standard_parallel_1",33],PARAMETER["standard_parallel_2",45],PARAMETER["latitude_of_origin",40],PARAMETER["central_meridian",-97],PARAMETER["false_easting",0],PARAMETER["false_northing",0],UNIT["Meter",1]]`

		// Create slice of output variable names
		outputVariableNames := make([]string, len(outputVariables))
		i := 0
		for k := range outputVariables {
			outputVariableNames[i] = k
			i++
		}

		results, err := d.Results(allLayers, true, outputVariables)
		if err != nil {
			return err
		}

		vars := make([]string, 0, len(results))
		for v := range results {
			vars = append(vars, v)
		}
		sort.Strings(vars)
		fields := make([]goshp.Field, len(vars))
		for i, v := range vars {
			fields[i] = goshp.FloatField(v, 14, 8)
		}

		// remove extension and replace it with .shp
		fileBase := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		fileName = fileBase + ".shp"
		shape, err := shp.NewEncoderFromFields(fileName, goshp.POLYGON, fields...)
		if err != nil {
			return fmt.Errorf("error creating output shapefile: %v", err)
		}
		cells := d.cells.array()
		for i, c := range cells[0:len(results[outputVariableNames[0]])] {
			outFields := make([]interface{}, len(vars))
			for j, v := range vars {
				outFields[j] = results[v][i]
			}
			err = shape.EncodeFields(c.Polygonal, outFields...)
			if err != nil {
				return fmt.Errorf("error writing output shapefile: %v", err)
			}
		}
		shape.Close()

		// Create .prj file
		f, err := os.Create(fileBase + ".prj")
		if err != nil {
			return fmt.Errorf("error creating output prj file: %v", err)
		}
		fmt.Fprint(f, proj4)
		f.Close()

		return nil
	}
}
