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
	"bitbucket.org/ctessum/inmap/lib.inmap"
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/twpayne/gogeom/geom/encoding/geojson"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var configFile *string = flag.String("config", "none", "Path to configuration file")

type configData struct {
	InMAPdataTemplate    string // Path to location of baseline meteorology and pollutant data, where [layer] is a stand-in for the model layer number. The files should be in Gob format (http://golang.org/pkg/encoding/gob/). Can include environment variables.
	NumLayers            int    // Number of vertical layers to use in the model
	NumProcessors        int    // Number of processors to use for calculations
	GroundLevelEmissions string // Path to ground level emissions file. Can include environment variables.
	ElevatedEmissions    string // Path to elevated emissions file. Can include environment variables.
	OutputTemplate       string // Path to desired output file location, where [layer] is a stand-in for the model layer number. Can include environment variables.
	HTTPport             string // Port for hosting web page.
	// If HTTPport is `8080`, then the GUI would be viewed by visiting `localhost:8080` in a web browser.
}

func main() {
	flag.Parse()
	if *configFile == "" {
		fmt.Println("Need to specify configuration file as in " +
			"`aim -config=configFile.json`")
		os.Exit(1)
	}
	config := ReadConfigFile(*configFile)

	fmt.Println("\n",
		"-------------------------------------\n",
		"             Welcome!\n",
		"  (A)irshed (I)ntervention (M)odel\n",
		"   Copyright 2013 Chris Tessum\n",
		"-------------------------------------\n")

	runtime.GOMAXPROCS(config.NumProcessors)

	fmt.Println("Reading input data...")
	d := inmap.InitInMAPdata(config.InMAPdataTemplate,
		config.NumLayers, config.HTTPport)
	fmt.Println("Reading plume rise information...")

	const (
		height   = 75. * 0.3048             // m
		diam     = 11.28 * 0.3048           // m
		temp     = (377.-32)*5./9. + 273.15 // K
		velocity = 61.94 * 1097. / 3600.    // m/hr
	)

	emissions := make(map[string][]float64)
	// Add in ground level emissions
	if config.GroundLevelEmissions != "" {
		groundLevelEmis := getEmissionsCSV(config.GroundLevelEmissions, d)
		for pol, vals := range groundLevelEmis {
			if _, ok := emissions[pol]; !ok {
				emissions[pol] = make([]float64, len(d.Data))
			}
			for i, val := range vals {
				emissions[pol][i] += val
			}
		}
	}

	// Add in elevated emissions
	if config.ElevatedEmissions != "" {
		elevatedEmis := getEmissionsCSV(config.ElevatedEmissions, d)
		// apply plume rise
		for pol, elev := range elevatedEmis {
			if _, ok := emissions[pol]; !ok {
				emissions[pol] = make([]float64, len(d.Data))
			}
			for i, val := range elev {
				if val != 0. {
					plumeRow, err := d.CalcPlumeRise(
						height, diam, temp, velocity, i)
					if err != nil {
						panic(err)
					}
					emissions[pol][plumeRow] += val
				}
			}
		}
	}

	// Run model
	finalConc := d.Run(emissions)

	writeOutput(finalConc, d, config.OutputTemplate)

	fmt.Println("\n",
		"------------------------------------\n",
		"           InMAP Completed!\n",
		"------------------------------------\n")
}

// Get the emissions from a csv file.
// Input units = tons/year; output units = μg/s
func getEmissionsCSV(filename string, d *inmap.InMAPdata) (
	emissions map[string][]float64) {

	const massConv = 907184740000.       // μg per short ton
	const timeConv = 3600. * 8760.       // seconds per year
	const emisConv = massConv / timeConv // convert tons/year to μg/s

	emissions = make(map[string][]float64)
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer f.Close()
	r := csv.NewReader(f)
	vars, err := r.Read() // Pollutant names in the header
	if err != nil {
		if err == io.EOF {
			return
		}
		panic(err)
	}
	for _, Var := range vars {
		if Var == "CO" || Var == "PM10" || Var == "CH4" {
			continue
		}
		emissions[polTrans(Var)] = make([]float64, d.LayerEnd[0]-d.LayerStart[0])
	}
	row := 0
	for {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		for i, Var := range vars {
			if Var == "CO" || Var == "PM10" || Var == "CH4" {
				continue
			}
			emissions[polTrans(Var)][row] = s2f(record[i]) * emisConv
		}
		row++
	}
	return
}

func polTrans(pol string) string {
	switch pol {
	case "PM2.5":
		return "PM2_5"
	default:
		return pol
	}
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
	outFileTemplate string) {
	var err error
	// Initialize data holder
	outData := make([]*JsonHolderHolder, d.Nlayers)
	row := 0
	for k := 0; k < d.Nlayers; k++ {
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
	for k := 0; k < d.Nlayers; k++ {
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
	config.GroundLevelEmissions = os.ExpandEnv(config.GroundLevelEmissions)
	config.ElevatedEmissions = os.ExpandEnv(config.ElevatedEmissions)
	config.OutputTemplate = os.ExpandEnv(config.OutputTemplate)

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
