/*
Copyright (C) 2013-2014 Regents of the University of Minnesota.
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

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/ctessum/geomconv"
	"github.com/ctessum/geomop"
	"github.com/ctessum/inmap/lib.inmap"
	"github.com/ctessum/shapefile"
	"github.com/patrick-higgins/rtreego"
	"github.com/twpayne/gogeom/geom"
	"github.com/twpayne/gogeom/geom/encoding/geojson"
)

var configFile *string = flag.String("config", "none", "Path to configuration file")

const version = "0.1.0"

type configData struct {
	// Path to location of baseline meteorology and pollutant data,
	// where [layer] is a stand-in for the model layer number. The files
	// should be in Gob format (http://golang.org/pkg/encoding/gob/).
	// Can include environment variables.
	InMAPdataTemplate string

	NumLayers     int // Number of vertical layers to use in the model
	NumProcessors int // Number of processors to use for calculations

	// Paths to emissions shapefiles.
	// Can be elevated or ground level; elevated files need to have columns
	// labeled "height", "diam", "temp", and "velocity" containing stack
	// information in units of m, m, K, and m/s, respectively.
	// Emissions will be allocated from the geometries in the shape file
	// to the InMAP computational grid, but the mapping projection of the
	// shapefile must be the same as the projection InMAP uses.
	// Can include environment variables.
	EmissionsShapefiles []string

	// Path to desired output file location, where [layer] is a stand-in
	// for the model layer number. Can include environment variables.
	OutputTemplate string

	// If true, output data for all model layers. If false, only output
	// the lowest layer.
	OutputAllLayers bool

	// Number of iterations to calculate. If < 1, convergence
	// is automatically calculated.
	NumIterations int

	// Port for hosting web page. If HTTPport is `8080`, then the GUI
	// would be viewed by visiting `localhost:8080` in a web browser.
	// If HTTPport is "", then the web server doesn't run.
	HTTPport string
}

func main() {
	flag.Parse()
	if *configFile == "" {
		fmt.Println("Need to specify configuration file as in " +
			"`inmap -config=configFile.json`")
		os.Exit(1)
	}
	config := ReadConfigFile(*configFile)

	fmt.Println("\n",
		"------------------------------------------------\n",
		"                    Welcome!\n",
		"  (In)tervention (M)odel for (A)ir (P)ollution  \n",
		"                Version "+version+"             \n",
		"               Copyright 2013-2014              \n",
		"     Regents of the University of Minnesota     \n",
		"------------------------------------------------\n")

	runtime.GOMAXPROCS(config.NumProcessors)

	fmt.Println("Reading input data...")
	d := inmap.InitInMAPdata(config.InMAPdataTemplate,
		config.NumLayers, config.NumIterations, config.HTTPport)

	emissions := make(map[string][]float64)
	for _, pol := range inmap.EmisNames {
		if _, ok := emissions[pol]; !ok {
			emissions[pol] = make([]float64, len(d.Data))
		}
	}

	// Add in emissions shapefiles
	// Load emissions into rtree for fast searching
	emisTree := rtreego.NewTree(25, 50)
	for _, fname := range config.EmissionsShapefiles {
		fmt.Println("Loading emissions shapefile:\n", fname)
		fname = strings.Replace(fname, ".shp", "", -1)
		f1, err := os.Open(fname + ".shp")
		if err != nil {
			panic(err)
		}
		shp, err := shapefile.OpenShapefile(f1)
		if err != nil {
			panic(err)
		}
		f2, err := os.Open(fname + ".dbf")
		if err != nil {
			panic(err)
		}
		dbf, err := shapefile.OpenDBFFile(f2)
		if err != nil {
			panic(err)
		}
		for i := 0; i < int(dbf.DBFFileHeader.NumRecords); i++ {
			sRec, err := shp.NextRecord()
			if err != nil {
				panic(err)
			}
			fields, err := dbf.NextRecord()
			if err != nil {
				panic(err)
			}
			e := new(emisRecord)
			e.emis = make([]float64, len(inmap.EmisNames))
			e.g = sRec.Geometry
			for ii, pol := range inmap.EmisNames {
				// Input units = tons/year; output units = μg/s
				const massConv = 907184740000.       // μg per short ton
				const timeConv = 3600. * 8760.       // seconds per year
				const emisConv = massConv / timeConv // convert tons/year to μg/s
				if iii, ok := dbf.FieldIndicies[pol]; ok {
					switch fields[iii].(type) {
					case float64:
						e.emis[ii] += fields[iii].(float64) * emisConv
					case int:
						e.emis[ii] += float64(fields[iii].(int)) * emisConv
					}
				}
			}
			if iii, ok := dbf.FieldIndicies["height"]; ok {
				e.height = fields[iii].(float64) // stack height [m]
			}
			if iii, ok := dbf.FieldIndicies["diam"]; ok {
				e.diam = fields[iii].(float64) // stack diameter [m]
			}
			if iii, ok := dbf.FieldIndicies["temp"]; ok {
				e.temp = fields[iii].(float64) // stack temperature [K]
			}
			if iii, ok := dbf.FieldIndicies["velocity"]; ok {
				e.velocity = fields[iii].(float64) // stack velocity [m/s]
			}
			e.bounds, err = geomconv.GeomToRect(e.g)
			if err != nil {
				panic(err)
			}
			emisTree.Insert(e)
		}
		f1.Close()
		f2.Close()
	}

	fmt.Println("Allocating emissions to grid cells...")
	// allocate emissions to appropriate grid cells
	for i := d.LayerStart[0]; i < d.LayerEnd[0]; i++ {
		cell := d.Data[i]
		bounds, err := geomconv.GeomToRect(cell.Geom)
		if err != nil {
			panic(err)
		}
		for _, eTemp := range emisTree.SearchIntersect(bounds) {
			e := eTemp.(*emisRecord)
			var intersection geom.T
			switch e.g.(type) {
			case geom.Point:
				in, err := geomop.Within(e.g, cell.Geom)
				if err != nil {
					panic(err)
				}
				if in {
					intersection = e.g
				} else {
					continue
				}
			default:
				intersection, err = geomop.Construct(e.g, cell.Geom,
					geomop.INTERSECTION)
				if err != nil {
					panic(err)
				}
			}
			if intersection == nil {
				continue
			}
			var weightFactor float64 // fraction of geometry in grid cell
			switch e.g.(type) {
			case geom.Polygon, geom.MultiPolygon:
				weightFactor = geomop.Area(intersection) / geomop.Area(e.g)
			case geom.LineString, geom.MultiLineString:
				weightFactor = geomop.Length(intersection) / geomop.Length(e.g)
			case geom.Point:
				weightFactor = 1.
			default:
				panic(geomop.UnsupportedGeometryError{intersection})
			}
			var plumeRow int
			if e.height > 0. { // calculate plume rise
				plumeRow, err = d.CalcPlumeRise(
					e.height, e.diam, e.temp, e.velocity, i)
				if err != nil {
					panic(err)
				}
			} else {
				plumeRow = i
			}
			for j, val := range e.emis {
				emissions[inmap.EmisNames[j]][plumeRow] += val * weightFactor
			}
		}
	}

	for pol, arr := range emissions {
		sum := 0.
		for _, val := range arr {
			sum += val
		}
		fmt.Printf("%v, %g ug/s\n", pol, sum)
	}

	// Run model
	finalConc := d.Run(emissions, config.OutputAllLayers)

	writeOutput(finalConc, d, config.OutputTemplate, config.OutputAllLayers)

	fmt.Println("\nIntake fraction results:")
	breathingRate := 15. // [m³/day]
	iF := d.IntakeFraction(breathingRate)
	// Write iF to stdout
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	popList := make([]string, 0)
	for _, m := range iF {
		for p := range m {
			popList = append(popList, p)
		}
		break
	}
	sort.Strings(popList)
	fmt.Fprintln(w, strings.Join(append([]string{"pol"}, popList...), "\t"))
	for pol, m := range iF {
		temp := make([]string, len(popList))
		for i, pop := range popList {
			temp[i] = fmt.Sprintf("%.3g", m[pop])
		}
		fmt.Fprintln(w, strings.Join(append([]string{pol}, temp...), "\t"))
	}
	w.Flush()

	fmt.Println("\n",
		"------------------------------------\n",
		"           InMAP Completed!\n",
		"------------------------------------\n")
}

type emisRecord struct {
	g        geom.T
	emis     []float64
	bounds   *rtreego.Rect
	height   float64 // stack height [m]
	diam     float64 // stack diameter [m]
	temp     float64 // stack temperature [K]
	velocity float64 // stack velocity [m/s]
}

func (e emisRecord) Bounds() *rtreego.Rect {
	return e.bounds
}

type JsonHolder struct {
	Type       string
	Geometry   *geojson.Geometry
	Properties map[string]float64
}
type JsonHolderHolder struct {
	Proj4, Type string
	Features    []*JsonHolder
}

// write data out to GeoJSON
func writeOutput(finalConc map[string][][]float64, d *inmap.InMAPdata,
	outFileTemplate string, writeAllLayers bool) {
	var err error
	// Initialize data holder
	outData := make([]*JsonHolderHolder, d.Nlayers)
	row := 0
	var nlayers int
	if writeAllLayers {
		nlayers = d.Nlayers
	} else {
		nlayers = 1
	}
	for k := 0; k < nlayers; k++ {
		outData[k] = new(JsonHolderHolder)
		outData[k].Type = "FeatureCollection"
		outData[k].Features = make([]*JsonHolder, d.LayerEnd[k]-d.LayerStart[k])
		for i := 0; i < len(outData[k].Features); i++ {
			x := new(JsonHolder)
			x.Type = "Feature"
			x.Properties = make(map[string]float64)
			x.Geometry, err = geojson.ToGeoJSON(d.Data[row].Geom)
			if err != nil {
				panic(err)
			}
			outData[k].Features[i] = x
			row++
		}
	}
	for pol, polData := range finalConc {
		for k, layerData := range polData {
			for i, conc := range layerData {
				outData[k].Features[i].Properties[pol] = conc
			}
		}
	}
	for k := 0; k < nlayers; k++ {
		filename := strings.Replace(outFileTemplate, "[layer]",
			fmt.Sprintf("%v", k), -1)
		f, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		e := json.NewEncoder(f)
		if err := e.Encode(outData[k]); err != nil {
			panic(err)
		}
		f.Close()
	}
}

func s2i(s string) int {
	i, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		panic(err)
	}
	return int(i)
}
func s2f(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return f
}

// Reads and parse a json configuration file.
// See below for the required variables.
func ReadConfigFile(filename string) (config *configData) {
	// Open the configuration file
	var (
		file  *os.File
		bytes []byte
		err   error
	)
	file, err = os.Open(filename)
	if err != nil {
		fmt.Printf("The configuration file you have specified, %v, does not "+
			"appear to exist. Please check the file name and location and "+
			"try again.\n", filename)
		os.Exit(1)
	}
	reader := bufio.NewReader(file)
	bytes, err = ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	config = new(configData)
	err = json.Unmarshal(bytes, config)
	if err != nil {
		fmt.Printf(
			"There has been an error parsing the configuration file.\n"+
				"Please ensure that the file is in valid JSON format\n"+
				"(you can check for errors at http://jsonlint.com/)\n"+
				"and try again!\n\n%v\n\n", err.Error())
		os.Exit(1)
	}

	config.InMAPdataTemplate = os.ExpandEnv(config.InMAPdataTemplate)
	config.OutputTemplate = os.ExpandEnv(config.OutputTemplate)

	for i := 0; i < len(config.EmissionsShapefiles); i++ {
		config.EmissionsShapefiles[i] =
			os.ExpandEnv(config.EmissionsShapefiles[i])
	}

	if config.OutputTemplate == "" {
		fmt.Println("You need to specify an output template in the " +
			"configuration file(for example: " +
			"\"OutputTemplate\":\"output_[layer].geojson\"")
		os.Exit(1)
	}

	outdir := filepath.Dir(config.OutputTemplate)
	err = os.MkdirAll(outdir, os.ModePerm)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return
}
