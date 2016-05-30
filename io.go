package inmap

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/op"
	"github.com/ctessum/geom/proj"
	goshp "github.com/jonas-p/go-shp"
)

// AddEmissionsFlux adds emissions to c.Cf and sets c.Ci equal to c.Cf.
// It should be run once for each timestep,
// and it should not be run in parallel with other CellManipulators.
func AddEmissionsFlux() CellManipulator {
	return func(c *Cell, Dt float64) {
		for i := range polNames {
			c.Cf[i] += c.emisFlux[i] * Dt
			c.Ci[i] = c.Cf[i]
		}
	}
}

// Emissions is a holder for input emissions data.
type Emissions struct {
	data *rtree.Rtree
}

type emisRecord struct {
	geom.Geom
	VOC, NOx, NH3, SOx float64 // emissions [tons/year]
	PM25               float64 `shp:"PM2_5"` // emissions [tons/year]
	Height             float64 // stack height [m]
	Diam               float64 // stack diameter [m]
	Temp               float64 // stack temperature [K]
	Velocity           float64 // stack velocity [m/s]
}

// ReadEmissionShapefiles returns the emissions data in the specified shapefiles,
// and converts them to the spatial reference gridSR. Input units are specified
// by units; options are tons/year and kg/year. Output units = μg/s.
func ReadEmissionShapefiles(gridSR *proj.SR, units string, shapefiles ...string) (*Emissions, error) {

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
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}
	for _, fname := range shapefiles {
		log.Printf("Loading emissions shapefile: %s.", fname)
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
			var e emisRecord
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
			emis.data.Insert(e)
		}
		f.Close()
		if err := f.Error(); err != nil {
			return nil, fmt.Errorf("problem reading emissions shapefile."+
				"\nfile: %s\nerror: %v", fname, err)
		}
	}
	return emis, nil
}

// addEmisFlux calculates emissions flux given emissions array in units of μg/s
// and a scale for molecular mass conversion.
func (c *Cell) addEmisFlux(val float64, scale float64, iPol int) {
	fluxScale := 1. / c.Dx / c.Dy / c.Dz // μg/s /m/m/m = μg/m3/s
	c.emisFlux[iPol] += val * scale * fluxScale
}

// calcIntersection calculates the geometry of any intersection between e and c.
func calcIntersection(e geom.Geom, c *Cell) geom.Geom {
	var intersection geom.Geom
	switch e.(type) {
	case geom.Point:
		if e.(geom.Point).Within(c) {
			intersection = e
		} else {
			return nil
		}
	case geom.Polygonal:
		poly := e.(geom.Polygonal)
		intersection = poly.Intersection(c)
	case geom.Linear:
		var err error
		intersection, err = op.Construct(e, c.Polygonal, op.INTERSECTION)
		if err != nil {
			log.Fatalf("while allocating emissions to grid: %v", err)
		}
	default:
		log.Fatalf("unsupported geometry type: %#v in emissions file", e)
	}
	return intersection
}

// calcWeightFactor calculate the fraction of emissions in e that should be
// allocated to intersection based on the areas of lengths or areas.
func calcWeightFactor(e, intersection geom.Geom) float64 {
	var weightFactor float64 // fraction of geometry in grid cell
	switch e.(type) {
	case geom.Polygonal:
		ep := e.(geom.Polygonal)
		ip := intersection.(geom.Polygonal)
		weightFactor = ip.Area() / ep.Area()
	case geom.Linear:
		el := e.(geom.Linear)
		il := intersection.(geom.Linear)
		weightFactor = il.Length() / el.Length()
	case geom.Point:
		weightFactor = 1.
	default:
		log.Fatalf("unsupported geometry type: %#v", e)
	}
	return weightFactor
}

// setEmissionsFlux sets the emissions flux for c based on the emissions in e.
func (c *Cell) setEmissionsFlux(e *Emissions) {
	c.emisFlux = make([]float64, len(polNames))
	for _, eTemp := range e.data.SearchIntersect(c.Bounds()) {
		e := eTemp.(emisRecord)
		intersection := calcIntersection(e.Geom, c)
		if intersection == nil {
			continue
		}
		weightFactor := calcWeightFactor(e.Geom, intersection)
		if e.Height > 0. {
			// Figure out if this cell is at the right hight for the plume.
			in, err := c.IsPlumeIn(e.Height, e.Diam, e.Temp, e.Velocity)
			if err != nil {
				panic(err)
			}
			if !in {
				continue
			}
		} else if c.Layer != 0 {
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
// outputVariables is a list of the names of the variables to be output.
func Output(fileTemplate string, allLayers bool, outputVariables ...string) DomainManipulator {
	return func(d *InMAPdata) error {

		// Projection definition. This may need to be changed for a different
		// spatial domain.
		// TODO: Make this settable by the user, or at least check to make sure it
		// matches the InMAPProj configuration variable.
		const proj4 = `PROJCS["Lambert_Conformal_Conic",GEOGCS["GCS_unnamed ellipse",DATUM["D_unknown",SPHEROID["Unknown",6370997,0]],PRIMEM["Greenwich",0],UNIT["Degree",0.017453292519943295]],PROJECTION["Lambert_Conformal_Conic"],PARAMETER["standard_parallel_1",33],PARAMETER["standard_parallel_2",45],PARAMETER["latitude_of_origin",40],PARAMETER["central_meridian",-97],PARAMETER["false_easting",0],PARAMETER["false_northing",0],UNIT["Meter",1]]`

		results := d.Results(allLayers, outputVariables...)

		vars := make([]string, 0, len(results))
		for v := range results {
			vars = append(vars, v)
		}
		sort.Strings(vars)
		fields := make([]goshp.Field, len(vars))
		for i, v := range vars {
			fields[i] = goshp.FloatField(v, 14, 8)
		}

		var nlayers int
		if allLayers {
			nlayers = d.nlayers
		} else {
			nlayers = 1
		}
		row := 0
		for k := 0; k < nlayers; k++ {

			filename := strings.Replace(fileTemplate, "[layer]",
				fmt.Sprintf("%v", k), -1)
			// remove extension and replace it with .shp
			extIndex := strings.LastIndex(filename, ".")
			if extIndex == -1 {
				extIndex = len(filename)
			}
			filename = filename[0:extIndex] + ".shp"
			shape, err := shp.NewEncoderFromFields(filename, goshp.POLYGON, fields...)
			if err != nil {
				return fmt.Errorf("error creating output shapefile: %v", err)
			}

			numRowsInLayer := len(results[vars[0]][k])
			for i := 0; i < numRowsInLayer; i++ {
				outFields := make([]interface{}, len(vars))
				for j, v := range vars {
					outFields[j] = results[v][k][i]
				}
				err = shape.EncodeFields(d.Cells[row].Polygonal, outFields...)
				if err != nil {
					return fmt.Errorf("error writing output shapefile: %v", err)
				}
				row++
			}
			shape.Close()

			// Create .prj file
			f, err := os.Create(filename[0:extIndex] + ".prj")
			if err != nil {
				return fmt.Errorf("error creating output prj file: %v", err)
			}
			fmt.Fprint(f, proj4)
			f.Close()
		}

		return nil
	}
}
