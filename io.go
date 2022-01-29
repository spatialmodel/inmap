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
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
	goshp "github.com/jonas-p/go-shp"
	"github.com/spatialmodel/inmap/emissions/aep"
	"gonum.org/v1/gonum/floats"
)

// AddEmissionsFlux adds emissions to c.Cf and sets c.Ci equal to c.Cf.
// It should be run once for each timestep,
// and it should not be run in parallel with other CellManipulators.
func AddEmissionsFlux() CellManipulator {
	return func(c *Cell, Dt float64) {
		if c.EmisFlux != nil {
			for i := range c.EmisFlux {
				c.Cf[i] += c.EmisFlux[i] * Dt
				c.Ci[i] = c.Cf[i]
			}
		}
	}
}

// Emissions is a holder for input emissions data.
type Emissions struct {
	data      *rtree.Rtree
	dataSlice []*EmisRecord

	// Mask specifies the region that emissions should be clipped
	// to. It is assumed to use the same spatial reference as the
	// InMAP computational grid. It is ignored if nil.
	Mask geom.Polygon
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

// Add adds an emissions record to the receiver, clipping
// it to the Mask if necessary.
func (e *Emissions) Add(er *EmisRecord) {
	if e.Mask == nil {
		e.data.Insert(er)
		e.dataSlice = append(e.dataSlice, er)
		return
	}

	if !er.Bounds().Overlaps(e.Mask.Bounds()) {
		return
	}

	var g geom.Geom  // g is the intersection of the emission geometry and the mask.
	var frac float64 // Frac is the fraction of the geometry overlapping the mask.
	switch t := er.Geom.(type) {
	case geom.Polygonal:
		p := t.Intersection(e.Mask)
		frac = p.Area() / t.Area()
		g = p
	case geom.Linear:
		l := t.Clip(e.Mask)
		g = l
		frac = l.Length() / t.Length()
	case geom.Point:
		if w := t.Within(e.Mask); w == geom.Inside || w == geom.OnEdge {
			g = t
			frac = 1
		}
	default:
		panic(fmt.Errorf("invalid geometry %T", t))
	}
	if g != nil {
		er2 := er
		er2.Geom = g
		er2.VOC *= frac
		er2.NOx *= frac
		er2.NH3 *= frac
		er2.SOx *= frac
		er2.PM25 *= frac
		e.data.Insert(er2)
		e.dataSlice = append(e.dataSlice, er2)
	}
}

// EmisRecords returns all EmisRecords stored in the
// receiver.
func (e *Emissions) EmisRecords() []*EmisRecord { return e.dataSlice }

// emisConversionFactor returns the conversion factor to μg/s
// for the given units.
func emisConversionFactor(units string) (float64, error) {
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
	case "ug/s", "μg/s":
		// Input units = μg/s; output units = μg/s
		emisConv = 1
	default:
		return math.NaN(), fmt.Errorf("inmap: invalid emissions units '%s'", units)
	}
	return emisConv, nil
}

// ReadEmissionShapefiles returns the emissions data in the specified shapefiles,
// and converts them to the spatial reference gridSR. Input units are specified
// by units; options are tons/year, kg/year, ug/s, and μg/s. Output units = μg/s.
// c is a channel over which status updates will be sent. If c is nil,
// no updates will be sent.
// mask specifies the region that emissions should be clipped to, assumed to
// use the same spatial reference as the InMAP grid. If mask is nil
// it will be ignored.
func ReadEmissionShapefiles(gridSR *proj.SR, units string, c chan string, mask geom.Polygon, shapefiles ...string) (*Emissions, error) {
	emisConv, err := emisConversionFactor(units)
	if err != nil {
		return nil, err
	}
	// Add in emissions shapefiles
	// Load emissions into rtree for fast searching
	emis := NewEmissions()
	emis.Mask = mask
	for _, fname := range shapefiles {
		if c != nil {
			c <- fmt.Sprintf("Loading emissions shapefile: %s.", fname)
		}
		fname = strings.Replace(fname, ".shp", "", -1)
		f, err := shp.NewDecoder(fname + ".shp")
		if err != nil {
			return nil, fmt.Errorf("there was a problem reading the emissions shapefile '%s' "+
				"The error message was %v", fname, err)
		}
		sr, err := f.SR()
		if err != nil {
			return nil, fmt.Errorf("there was a problem reading the projection information for "+
				"the emissions shapefile '%s'. The error message was %v", fname, err)
		}
		trans, err := sr.NewTransform(gridSR)
		if err != nil {
			return nil, fmt.Errorf("there was a problem creating a spatial reprojector for "+
				"the emissions shapefile '%s'. The error message was %v", fname, err)
		}
		for {
			var e EmisRecord
			if ok := f.DecodeRow(&e); !ok {
				break
			}

			if e.Geom == nil {
				continue
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

// FromAEP converts the given AEP (github.com/spatialmodel/inmap/emissions/aep) records to
// EmisRecords using the given grid definitions and
// grid index gi. VOC, NOx, NH3, SOx, and PM25 are lists of
// AEP Polluants that should be mapped to those InMAP species.
// The returned EmisRecords will be grouped as much as possible to minimize
// the number of records.
func FromAEP(r []aep.RecordGridded, grids []*aep.GridDef, gi int, VOC, NOx, NH3, SOx, PM25 []aep.Pollutant) ([]*EmisRecord, error) {
	if gi < 0 || len(grids) <= gi {
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
	grid := grids[gi]

	var eRecs []*EmisRecord
	groundERecs := make(map[int]*EmisRecord)

	for _, rec := range r {
		gridSrg, _, inGrid, err := rec.GridFactors(gi)
		if err != nil {
			return nil, err
		}
		if !inGrid {
			continue
		}
		e := rec.GetEmissions().Totals()
		for i, frac := range gridSrg.Elements {
			er := EmisRecord{
				Geom: grid.Cells[i].Polygonal,
			}

			// Convert units.
			const (
				secPerYear        = 60 * 60 * 24 * 365
				ugPerKg           = 1.0e9
				kgPerYearToUgPerS = 1 * ugPerKg / secPerYear
			)

			// Add the emissions to the new record.
			for pRec, v := range e {
				var found bool
				for _, p := range VOC {
					if pRec.Name == p.Name {
						er.VOC += checkDim(v) * frac * kgPerYearToUgPerS
						found = true
						break
					}
				}
				if found {
					continue
				}
				for _, p := range NOx {
					if pRec.Name == p.Name {
						er.NOx += checkDim(e[pRec]) * frac * kgPerYearToUgPerS
						found = true
						break
					}
				}
				if found {
					continue
				}
				for _, p := range NH3 {
					if pRec.Name == p.Name {
						er.NH3 += checkDim(e[pRec]) * frac * kgPerYearToUgPerS
						found = true
						break
					}
				}
				if found {
					continue
				}
				for _, p := range SOx {
					if pRec.Name == p.Name {
						er.SOx += checkDim(e[pRec]) * frac * kgPerYearToUgPerS
						found = true
						break
					}
				}
				if found {
					continue
				}
				for _, p := range PM25 {
					if pRec.Name == p.Name {
						er.PM25 += checkDim(e[pRec]) * frac * kgPerYearToUgPerS
						found = true
						break
					}
				}
			}

			if ptRec, ok := rec.Parent().(aep.RecordElevated); ok && !ptRec.GroundLevel() {
				StackHeight, StackDiameter, StackTemp, _, StackVelocity := ptRec.StackParameters()
				er.Height = StackHeight.Value()
				er.Diam = StackDiameter.Value()
				er.Temp = StackTemp.Value()
				er.Velocity = StackVelocity.Value()
				eRecs = append(eRecs, &er)
			} else {
				// For ground level sources, combine with other records
				// at the same point.
				if _, ok := groundERecs[i]; !ok {
					groundERecs[i] = &er
				} else {
					groundERecs[i].add(&er)
				}
			}
		}
	}
	for _, groundERec := range groundERecs {
		eRecs = append(eRecs, groundERec)
	}
	return eRecs, nil
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
		intersection := poly.Intersection(c.Polygonal)
		if intersection == nil {
			return 0.
		}
		weightFactor = intersection.Area() / poly.Area()
	case geom.Linear:
		intersection := e.(geom.Linear).Clip(c.Polygonal)
		if intersection == nil {
			return 0.
		}
		el := e.(geom.Linear)
		il := intersection
		weightFactor = il.Length() / el.Length()
	default:
		log.Fatalf("unsupported geometry type: %#v in emissions file", e)
	}
	return weightFactor
}

// SetEmissionsFlux sets the emissions flux for the receiver based on the emissions in e.
func (c *Cell) SetEmissionsFlux(e *Emissions, m Mechanism) error {
	c.EmisFlux = make([]float64, m.Len())
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

		if err := m.AddEmisFlux(c, "VOC", e.VOC*weightFactor); err != nil {
			return err
		}
		if err := m.AddEmisFlux(c, "NOx", e.NOx*weightFactor); err != nil {
			return err
		}
		if err := m.AddEmisFlux(c, "NH3", e.NH3*weightFactor); err != nil {
			return err
		}
		if err := m.AddEmisFlux(c, "SOx", e.SOx*weightFactor); err != nil {
			return err
		}
		if err := m.AddEmisFlux(c, "PM2_5", e.PM25*weightFactor); err != nil {
			return err
		}
	}
	return nil
}

// Outputter is a holder for output parameters.
//
// fileName contains the path where the output will be saved.
//
// If allLayers is true, output will contain data for all of the vertical
// layers, otherwise only the ground-level layer is returned.
//
// outputVariables maps the names of the variables for which data
// should be returned to expressions that define how the
// requested data should be calculated. These expressions can utilize variables
// built into the model, user-defined variables, and functions.
//
// modelVariables is automatically generated based on the model variables that
// are required to calculate the requested output variables.
//
// Functions are defined in the outputFunctions variable.
type Outputter struct {
	fileName        string
	allLayers       bool
	outputVariables map[string]string
	modelVariables  []string
	outputFunctions map[string]govaluate.ExpressionFunction
	m               Mechanism
}

// NewOutputter initializes a new Outputter holder and adds a set of default
// output functions. Default functions include:
//
// 'exp(x)' which applies the exponental function e^x.
//
// 'log(x)' which applies the natural logarithm function log(e).
//
// 'log10(x)' which applies the base-10 logarithm function log10(e).
//
// 'sum(x)' which sums a variable across all grid cells.
func NewOutputter(fileName string, allLayers bool, outputVariables map[string]string, outputFunctions map[string]govaluate.ExpressionFunction, m Mechanism) (*Outputter, error) {
	defaultOutputFuncs := map[string]govaluate.ExpressionFunction{
		"exp": func(arg ...interface{}) (interface{}, error) {
			if len(arg) != 1 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'exp', but need 1", len(arg))
			}
			return (float64)(math.Exp(arg[0].(float64))), nil
		},
		"log": func(arg ...interface{}) (interface{}, error) {
			if len(arg) != 1 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'exp', but need 1", len(arg))
			}
			return (float64)(math.Log(arg[0].(float64))), nil
		},
		"log10": func(arg ...interface{}) (interface{}, error) {
			if len(arg) != 1 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'exp', but need 1", len(arg))
			}
			return (float64)(math.Log(arg[0].(float64))), nil
		},
		"sum": func(arg ...interface{}) (interface{}, error) {
			if len(arg) != 1 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'sum', but need 1", len(arg))
			}
			return floats.Sum(arg[0].([]float64)), nil
		},
	}

	for key, val := range outputFunctions {
		defaultOutputFuncs[key] = val
	}

	o := Outputter{
		fileName:        fileName,
		allLayers:       allLayers,
		outputVariables: outputVariables,
		outputFunctions: defaultOutputFuncs,
		m:               m,
	}

	for _, val := range o.outputVariables {
		regx := regexp.MustCompile(`{(.*?)}`)
		matches := regx.FindAllString(val, -1)
		if len(matches) > 0 {
			for _, m := range matches {
				if strings.Count(m, "{") > 1 || strings.Count(m, "}") > 1 {
					fmt.Println("inmap o.outputVariables: unsupported use of braces {}")
				}
				o.outputVariables[m] = m[1 : len(m)-1]
			}
		}
	}

	err := o.checkForDerivatives()

	for k1, v1 := range o.outputVariables {
		if strings.Contains(k1, "{") {
			for k2, v2 := range o.outputVariables {
				if k1 != k2 {
					o.outputVariables[k2] = strings.Replace(v2, v1, "{"+v1+"}", -1)
				}
			}
			delete(o.outputVariables, k1)
		}
	}

	return &o, err
}

// removeDuplicates removes all duplicated strings from a slice, returning a
// slice that contains only unique strings.
func removeDuplicates(s []string) []string {
	result := make([]string, 0, len(s))
	seen := make(map[string]string)
	for _, val := range s {
		if _, ok := seen[val]; !ok {
			result = append(result, val)
			seen[val] = val
		}
	}
	return result
}

func checkPrefix(s string) (bool, error) {
	var isPrefix bool
	var err error
	if string(s) != "" {
		isPrefix, err = regexp.MatchString("[a-zA-Z0-9_]", string(s[0]))
		if err != nil {
			return false, err
		}
	} else {
		isPrefix = false
	}
	return isPrefix, nil
}

func checkSuffix(s string) (bool, error) {
	var isSuffix bool
	var err error
	if string(s) != "" {
		isSuffix, err = regexp.MatchString("[a-zA-Z0-9_]", string(s[len(s)-1]))
		if err != nil {
			return false, err
		}
	} else {
		isSuffix = false
	}
	return isSuffix, nil
}

// checkForDerivatives identifies the unique input variables that are required
// to calculate the requested output variables.
// Inputs:
// (1) Map of requested output variable names to their corresponding expressions.
// (2) Map of all function names to function definitions that are used in expressions.
// Outputs:
// (1) Map of output variable names to revised expressions where any user-defined
// output variable showing up in a subsequent expression is replaced by its
// corresponding user-defined expression.
// (2) Slice of all unique input variables required to calculate the requested
// output variables.
func (o *Outputter) checkForDerivatives() error {
	o.modelVariables = make([]string, 0, len(o.outputVariables))
	for key, val := range o.outputVariables {
		o.outputVariables[key] = strings.Replace(val, "{", "", -1)
		o.outputVariables[key] = strings.Replace(o.outputVariables[key], "}", "", -1)
		expression, err := govaluate.NewEvaluableExpressionWithFunctions(o.outputVariables[key], o.outputFunctions)
		if err != nil {
			return fmt.Errorf("inmap o.outputVariables: %v", err)
		}
		uniqueVars := removeDuplicates(expression.Vars())
		o.modelVariables = append(o.modelVariables, uniqueVars...)
		// For each variable name identified in an output variable expression,
		// check if the variable is defined in terms of other variables within a
		// separate expression. If so, any instance of the variable name in the
		// current will be replaced by the expression that defines it.
		var isSuffix bool
		var isPrefix bool
		for _, uniqueVar := range uniqueVars {
			if o.outputVariables[uniqueVar] != "" && o.outputVariables[uniqueVar] != uniqueVar {
				// In order to verify that an instance of a variable name is not part of
				// a longer variable name, the text preceding and following the variable
				// name is analyzed. For example, 'White' is not a standalone variable
				// in an expression if it appears as 'PctWhite'.
				splitVal := strings.Split(val, uniqueVar)
				for i := 0; i < len(splitVal)-1; i++ {
					isSuffix, err = checkSuffix(splitVal[i])
					if err != nil {
						return fmt.Errorf("inmap o.outputVariables: %v", err)
					}
					isPrefix, err = checkPrefix(splitVal[i+1])
					if err != nil {
						return fmt.Errorf("inmap o.outputVariables: %v", err)
					}
					splitVal[i] = splitVal[i] + uniqueVar
					// For every instance of the variable name that is not part of a
					// longer variable name, replace it by the expression that defines it.
					if !isSuffix && !isPrefix {
						splitVal[i] = strings.Replace(splitVal[i], uniqueVar, "("+o.outputVariables[uniqueVar]+")", -1)
					}
				}
				o.outputVariables[key] = strings.Join(splitVal, "")
				return o.checkForDerivatives()
			}
		}
	}
	o.modelVariables = removeDuplicates(o.modelVariables)
	return nil
}

// CheckModelVars checks whether the unique input variables required to calculate
// the user-requested output variables are available in the model.
func (d *InMAP) checkModelVars(m Mechanism, g ...string) error {
	outputOps, _, _ := d.OutputOptions(m)
	mapOutputOps := make(map[string]struct{})
	for _, n := range outputOps {
		mapOutputOps[n] = struct{}{}
	}
	for _, v := range g {
		if _, ok := mapOutputOps[v]; !ok {
			return fmt.Errorf("inmap: undefined variable name '%s'", v)
		}
	}
	return nil
}

// checkOutputNames checks (1) if any output variable names exceed 10 characters
// and (2) if any output variable names include characters that are unsupported
// in shapefile field names.
func checkOutputNames(o map[string]string) error {
	for key := range o {
		long := len(key) > 10
		noCharError, err := regexp.MatchString("^[A-Za-z]\\w*$", key)
		if err != nil {
			panic(err)
		}
		if long && !noCharError {
			return fmt.Errorf("inmap: output variable name '%s' exceeds 10 characters and includes unsupported character(s)", key)
		} else if long {
			return fmt.Errorf("inmap: output variable name '%s' exceeds 10 characters", key)
		} else if !noCharError {
			return fmt.Errorf("inmap: output variable name '%s' includes unsupported characters", key)
		}
	}
	return nil
}

// CheckOutputVars ensures that the requested output variables are all valid.
func (o *Outputter) CheckOutputVars(m Mechanism) DomainManipulator {
	return func(d *InMAP) error {
		if err := d.checkModelVars(m, o.modelVariables...); err != nil {
			return err
		} else if err := checkOutputNames(o.outputVariables); err != nil {
			return err
		} else {
			return nil
		}
	}
}

// Output writes the simulation results to a shapefile.
// SR is the spatial reference of the model grid.
func (o *Outputter) Output(sr *proj.SR) DomainManipulator {
	return func(d *InMAP) error {
		// Projection definition. This may need to be changed for a different
		// spatial domain.
		// TODO: Make this settable by the user, or at least check to make sure it
		// matches the InMAPProj configuration variable.
		var wkt string
		switch sr.Name {
		case "lcc":
			wkt = fmt.Sprintf("PROJCS[\"Lambert_Conformal_Conic\",GEOGCS[\"GCS_unnamed ellipse\","+
				"DATUM[\"D_unknown\",SPHEROID[\"Unknown\",%f,0]],PRIMEM[\"Greenwich\",0],"+
				"UNIT[\"Degree\",0.017453292519943295]],PROJECTION[\"Lambert_Conformal_Conic\"],"+
				"PARAMETER[\"standard_parallel_1\",%g],PARAMETER[\"standard_parallel_2\",%g],"+
				"PARAMETER[\"latitude_of_origin\",%g],PARAMETER[\"central_meridian\",%g],"+
				"PARAMETER[\"false_easting\",0],PARAMETER[\"false_northing\",0],UNIT[\"Meter\",1]]",
				sr.A, sr.Lat1/math.Pi*180, sr.Lat2/math.Pi*180, sr.Lat0/math.Pi*180,
				sr.Long0/math.Pi*180)
		case "longlat":
			wkt = `GEOGCS["GCS_WGS_1984",DATUM["D_WGS_1984",SPHEROID["WGS_1984",6378137,298.257223563]],PRIMEM["Greenwich",0],UNIT["Degree",0.017453292519943295]]`
		default:
			return fmt.Errorf("only `lcc` and `longlat` projections are supported, not %s", sr.Name)
		}

		// Create slice of output variable names
		outputVariableNames := make([]string, len(o.outputVariables))
		i := 0
		for k := range o.outputVariables {
			outputVariableNames[i] = k
			i++
		}

		results, err := d.Results(o)
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
			fields[i] = shpFieldFromArray(v, results[v])
		}

		// remove extension and replace it with .shp
		fileBase := strings.TrimSuffix(o.fileName, filepath.Ext(o.fileName))
		o.fileName = fileBase + ".shp"
		shape, err := shp.NewEncoderFromFields(o.fileName, goshp.POLYGON, fields...)
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
		fmt.Fprint(f, wkt)
		f.Close()

		return nil
	}
}

// shpFieldFromArray creates a shapefile field from the given array,
// ensuring that all values in the array will have a minimum of 9 significant
// digits.
func shpFieldFromArray(name string, d []float64) goshp.Field {
	const minPrecision = 9
	minExp := math.Inf(+1)
	maxExp := math.Inf(-1)
	minVal := math.Inf(1)
	for _, v := range d {
		if v == 0 {
			continue
		}
		exp := math.Log10(math.Abs(v))
		if exp < minExp {
			minExp = exp
		}
		if exp > maxExp {
			maxExp = exp
		}
		if v < minVal {
			minVal = v
		}
	}
	var precision, size uint8
	if math.IsInf(minExp, 0) {
		precision = minPrecision - 1 // All zeros, so 8 decimal places.
	} else {
		precision = uint8(math.Max(0, -1*(math.Floor(minExp)-minPrecision+1)))
	}

	if math.IsInf(maxExp, 0) || maxExp < 1 {
		size = precision + 1 // Size = 'x' + precision
	} else {
		size = uint8(math.Floor(maxExp)) + 1 + precision // Size = 'xxx' + precision
	}
	if precision > 0 {
		size++ // Add a space for a '.'
	}
	if minVal < 0 { // Add space for a '-'
		size++
	}
	return goshp.FloatField(name, size, precision)
}

// Results returns the simulation results.
// Output is in the form of map[variable][row]concentration.
func (d *InMAP) Results(o *Outputter) (map[string][]float64, error) {

	// Prepare output data.
	modelVals := make(map[string]interface{})
	valByRow := make(map[string]interface{})
	output := make(map[string][]float64)
	var nCells int

	// Get the model variables that are to be used in the output.
	for _, name := range o.modelVariables {
		if o.allLayers {
			data := d.toArray(name, -1, o.m)
			modelVals[name] = data
			nCells = len(data)
		} else {
			data := d.toArray(name, 0, o.m)
			modelVals[name] = data
			nCells = len(data)
		}
	}

	// Identify segments of output variable expressions that are surrounded by braces.
	for k, v := range o.outputVariables {
		regx, _ := regexp.Compile("\\{(.*?)\\}")
		matches := regx.FindAllString(v, -1)
		if len(matches) > 0 {
			// For each segment of an expression that is surrounded by braces, evaluate
			// across all grid cells.
			for _, m := range matches {
				expression, err := govaluate.NewEvaluableExpressionWithFunctions(m[1:len(m)-1], o.outputFunctions)
				if err != nil {
					return nil, err
				}
				result, err := expression.Evaluate(modelVals)
				if err != nil {
					return nil, err
				}
				// Replace segments surrounded by braces with corresponding result
				// calculated above.
				o.outputVariables[k] = strings.Replace(o.outputVariables[k], m, strconv.FormatFloat(result.(float64), 'f', -1, 64), 1)
			}
		}
	}
	for k, v := range o.outputVariables {
		expression, err := govaluate.NewEvaluableExpressionWithFunctions(v, o.outputFunctions)
		if err != nil {
			return nil, err
		}
		for i := 0; i < nCells; i++ {
			for name := range modelVals {
				valByRow[name] = modelVals[name].([]float64)[i]
			}
			result, err := expression.Evaluate(valByRow)
			if err != nil {
				return nil, err
			}
			output[k] = append(output[k], result.(float64))
		}
	}
	return output, nil
}

// toArray converts cell data for variable varName into a regular array.
// If layer is less than zero, data for all layers is returned.
func (d *InMAP) toArray(varName string, layer int, m Mechanism) []float64 {
	o := make([]float64, 0, d.cells.len())
	cells := d.cells.array()
	for _, c := range cells {
		c.mutex.RLock()
		if layer >= 0 && c.Layer > layer {
			// The cells should be sorted with the lower layers first, so we
			// should be done here.
			c.mutex.RUnlock()
			return o
		}
		if layer < 0 || c.Layer == layer {
			o = append(o, c.getValue(varName, d.PopIndices, d.mortIndices, m))
		}
		c.mutex.RUnlock()
	}
	return o
}

// Get the value in the current cell of the specified variable, where popIndices
// are array indices of each population type.
func (c *Cell) getValue(varName string, popIndices, mortIndices map[string]int, m Mechanism) float64 {
	v, err := m.Value(c, varName)
	if err == nil {
		return v
	}
	if i, ok := popIndices[varName]; ok { // Population
		return c.PopData[i]

	} else if polConv, ok := baselinePolLabels[varName]; ok { // Baseline concentrations
		var o float64
		for i, ii := range polConv.index {
			o += c.CBaseline[ii] * polConv.conversion[i]
		}
		return o

	} else if i, ok := mortIndices[varName]; ok { // Mortality rate
		return c.MortData[i]

	} // Everything else
	v2 := reflect.ValueOf(c).Elem()
	if _, ok := v2.Type().FieldByName(varName); !ok {
		panic(fmt.Errorf("inmap: missing variable %v", varName))
	}
	val := v2.FieldByName(varName)
	switch val.Type().Kind() {
	case reflect.Float64:
		return val.Float()
	case reflect.Int:
		return float64(val.Int()) // convert integer fields to floats here for consistency.
	default:
		panic(fmt.Errorf("unsupported field type %v", val.Type().Kind()))
	}
}

// getUnits returns the units of a model variable.
func (d *InMAP) getUnits(varName string, m Mechanism) string {
	u, err := m.Units(varName)
	if err == nil {
		return u
	}
	if _, ok := baselinePolLabels[varName]; ok { // Concentrations
		return "μg/m³"
	} else if _, ok := d.PopIndices[varName]; ok { // Population
		return "people/grid cell"
	} else if _, ok := d.mortIndices[varName]; ok { // Mortality Rate
		return "deaths/100,000"
	} else if _, ok := d.PopIndices[strings.Replace(varName, " deaths", "", 1)]; ok {
		// Mortalities
		return "deaths/grid cell"
	}
	// Everything else
	t := reflect.TypeOf(*(*d.cells)[0].Cell)
	ftype, ok := t.FieldByName(varName)
	if ok {
		return ftype.Tag.Get("units")
	}
	panic(fmt.Sprintf("Unknown variable %v.", varName))
}

// OutputOptions returns the options for output variable names and their
// descriptions.
func (d *InMAP) OutputOptions(m Mechanism) (names []string, descriptions []string, units []string) {
	// Model pollutant concentrations
	for _, pol := range m.Species() {
		names = append(names, pol)
	}
	for _, n := range names {
		if strings.Contains(n, "Emissions") {
			descriptions = append(descriptions, n)
		} else {
			descriptions = append(descriptions, n+" Concentration")
		}
	}

	// Baseline pollutant concentrations
	var tempBaseline []string
	for pol := range baselinePolLabels {
		tempBaseline = append(tempBaseline, pol)
	}
	sort.Strings(tempBaseline)
	names = append(names, tempBaseline...)
	for _, n := range tempBaseline {
		descriptions = append(descriptions, n+"Concentration")
	}

	// Population
	var tempPop []string
	for pop := range d.PopIndices {
		tempPop = append(tempPop, pop)
	}
	sort.Strings(tempPop)
	names = append(names, tempPop...)
	for _, n := range tempPop {
		descriptions = append(descriptions, n+"Population")
	}

	// Mortality Rates
	var tempMort []string
	for mort := range d.mortIndices {
		tempMort = append(tempMort, mort)
	}
	sort.Strings(tempMort)
	names = append(names, tempMort...)
	for _, n := range tempMort {
		descriptions = append(descriptions, strings.Replace(n, "Mort", "", 1)+"MortalityRate")
	}

	// Eveything else
	t := reflect.TypeOf(*(*d.cells)[0].Cell)
	var tempNames []string
	var tempDescriptions []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := f.Name
		desc := f.Tag.Get("desc")
		if desc != "" {
			tempDescriptions = append(tempDescriptions, desc)
			tempNames = append(tempNames, v)
		}
	}
	names = append(names, tempNames...)
	descriptions = append(descriptions, tempDescriptions...)

	units = make([]string, len(names))
	for i, n := range names {
		units[i] = d.getUnits(n, m)
	}
	return
}
