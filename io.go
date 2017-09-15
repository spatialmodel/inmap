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
	"regexp"
	"sort"
	"strings"

	"bitbucket.org/ctessum/aqhealth"
	"github.com/Knetic/govaluate"
	"github.com/ctessum/aep"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/op"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
	goshp "github.com/jonas-p/go-shp"
	"gonum.org/v1/gonum/floats"
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
	data      *rtree.Rtree
	dataSlice []*EmisRecord
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
	e.dataSlice = append(e.dataSlice, er)
}

// EmisRecords returns all EmisRecords stored in the
// receiver.
func (e *Emissions) EmisRecords() []*EmisRecord { return e.dataSlice }

// ReadEmissionShapefiles returns the emissions data in the specified shapefiles,
// and converts them to the spatial reference gridSR. Input units are specified
// by units; options are tons/year, kg/year, ug/s, and μg/s. Output units = μg/s.
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
	case "ug/s", "μg/s":
		// Input units = μg/s; output units = μg/s
		emisConv = 1
	default:
		return nil, fmt.Errorf("inmap: invalid emissions units '%s'", units)
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
			for pRec, v := range e {
				var found bool
				for _, p := range VOC {
					if pRec.Name == p.Name {
						found = true
						er.VOC += checkDim(v) * frac * kgPerYearToUgPerS
					}
				}
				for _, p := range NOx {
					if pRec.Name == p.Name {
						found = true
						er.NOx += checkDim(e[p]) * frac * kgPerYearToUgPerS
					}
				}
				for _, p := range NH3 {
					if pRec.Name == p.Name {
						found = true
						er.NH3 += checkDim(e[p]) * frac * kgPerYearToUgPerS
					}
				}
				for _, p := range SOx {
					if pRec.Name == p.Name {
						found = true
						er.SOx += checkDim(e[p]) * frac * kgPerYearToUgPerS
					}
				}
				for _, p := range PM25 {
					if pRec.Name == p.Name {
						found = true
						er.PM25 += checkDim(e[p]) * frac * kgPerYearToUgPerS
					}
				}
				if !found {
					return nil, fmt.Errorf("inmap: no match for pollutant '%s'", pRec.Name)
				}
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
}

// NewOutputter initializes a new Outputter holder and adds a set of default
// output functions. Default functions include:
//
// 'exp(x)' which applies the exponetional function e^x.
//
// 'loglogRR(PM 2.5 Concentration)' which calculates relative risk (or risk ratio)
// associated with a given change in PM2.5 concentration, assumung a log-log
// dose response (almost a linear relationship).
//
// 'coxHazard(Relative Risk, Population, Mortality Rate)' which calculates a
// deaths estimate based on the relative risk associated with PM 2.5 changes,
// population, and the baseline mortality rate (deaths per 100,000 people per year).
//
// 'sum(x)' which sums a variable across all grid cells.
func NewOutputter(fileName string, allLayers bool, outputVariables map[string]string, outputFunctions map[string]govaluate.ExpressionFunction) (*Outputter, error) {
	defaultOutputFuncs := map[string]govaluate.ExpressionFunction{
		"exp": func(arg ...interface{}) (interface{}, error) {
			if len(arg) != 1 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'exp', but needs 1", len(arg))
			}
			return (float64)(math.Exp(arg[0].(float64))), nil
		},
		"loglogRR": func(arg ...interface{}) (interface{}, error) {
			if len(arg) != 1 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'loglogRR', but needs 1", len(arg))
			}
			return (float64)(aqhealth.RRpm25Linear(arg[0].(float64))), nil
		},
		"coxHazard": func(args ...interface{}) (interface{}, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'coxHazard', but needs 3", len(args))
			}
			return (float64)((args[0].(float64) - 1) * args[1].(float64) * args[2].(float64) / 100000), nil
		},
		"sum": func(arg ...interface{}) (interface{}, error) {
			if len(arg) != 1 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'sum', but needs 1", len(arg))
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
	}

	for _, val := range o.outputVariables {
		regx, _ := regexp.Compile("\\{(.*?)\\}")
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
func (d *InMAP) checkModelVars(g ...string) error {
	outputOps, _, _ := d.OutputOptions()
	mapOutputOps := make(map[string]uint8)
	for _, n := range outputOps {
		mapOutputOps[n] = 0
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

// CheckOutputVars ensures the output variables can be calculated.
func (o *Outputter) CheckOutputVars() DomainManipulator {
	return func(d *InMAP) error {
		if err := d.checkModelVars(o.modelVariables...); err != nil {
			return err
		} else if err := checkOutputNames(o.outputVariables); err != nil {
			return err
		} else {
			return nil
		}
	}
}

func (o *Outputter) Output() DomainManipulator {
	return func(d *InMAP) error {
		// Projection definition. This may need to be changed for a different
		// spatial domain.
		// TODO: Make this settable by the user, or at least check to make sure it
		// matches the InMAPProj configuration variable.
		const proj4 = `PROJCS["Lambert_Conformal_Conic",GEOGCS["GCS_unnamed ellipse",DATUM["D_unknown",SPHEROID["Unknown",6370997,0]],PRIMEM["Greenwich",0],UNIT["Degree",0.017453292519943295]],PROJECTION["Lambert_Conformal_Conic"],PARAMETER["standard_parallel_1",33],PARAMETER["standard_parallel_2",45],PARAMETER["latitude_of_origin",40],PARAMETER["central_meridian",-97],PARAMETER["false_easting",0],PARAMETER["false_northing",0],UNIT["Meter",1]]`

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
			fields[i] = goshp.FloatField(v, 14, 8)
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
		fmt.Fprint(f, proj4)
		f.Close()

		return nil
	}
}
