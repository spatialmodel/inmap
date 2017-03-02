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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
)

var (
	configFile = flag.String("config", "none", "Path to configuration file")
)

const (
	inDateFormat = "20060102"
)

// ConfigInfo holds the configuration information for the program run.
type ConfigInfo struct {
	// CTMType specifies what type of chemical transport
	// model we are going to be reading data from. Valid
	// options are "GEOS-Chem" and "WRF-Chem".
	CTMType string

	WRFChem struct {
		// WRFOut is the location of WRF-Chem output files.
		// [DATE] should be used as a wild card for the simulation date.
		WRFOut string
	}

	GEOSChem struct {
		// GEOSA1 is the location of the GEOS 1-hour time average files.
		// [DATE] should be used as a wild card for the simulation date.
		GEOSA1 string

		// GEOSA3Cld is the location of the GEOS 3-hour average cloud
		// parameter files. [DATE] should be used as a wild card for
		// the simulation date.
		GEOSA3Cld string

		// GEOSA3Cld is the location of the GEOS 3-hour average dynamical
		// parameter files. [DATE] should be used as a wild card for
		// the simulation date.
		GEOSA3Dyn string

		// GEOSI3 is the location of the GEOS 3-hour instantaneous parameter
		// files. [DATE] should be used as a wild card for
		// the simulation date.
		GEOSI3 string

		// GEOSA3MstE is the location of the GEOS 3-hour average moist parameters
		// on level edges files. [DATE] should be used as a wild card for
		// the simulation date.
		GEOSA3MstE string

		// GEOSChem is the location of GEOS-Chem output files.
		// [DATE] should be used as a wild card for the simulation date.
		GEOSChem string

		// VegTypeGlobal is the location of the GEOS-Chem vegtype.global file,
		// which is described here:
		// http://wiki.seas.harvard.edu/geos-chem/index.php/Olson_land_map#Structure_of_the_vegtype.global_file
		VegTypeGlobal string
	}

	// OutputFile is the location where the output file should go.
	OutputFile string

	// StartDate is the date of the beginning of the simulation.
	// Format = "YYYYMMDD".
	StartDate string

	// EndDate is the date of the end of the simulation.
	// Format = "YYYYMMDD".
	EndDate string

	CtmGridXo float64 // lower left of Chemical Transport Model (CTM) grid, x
	CtmGridYo float64 // lower left of grid, y

	CtmGridDx float64 // m
	CtmGridDy float64 // m

	GridProj string // projection info for CTM grid; Proj4 format
}

func main() {
	flag.Parse()
	if *configFile == "" {
		log.Fatal("Please specify configuration file as in " +
			"`wrf2inmap -config=configFile.json`")
	}
	cfg, err := ReadConfigFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	var ctm Preprocessor
	switch cfg.CTMType {
	case "GEOS-Chem":
		var err error
		ctm, err = NewGEOSChem(cfg)
		if err != nil {
			log.Fatal(err)
		}
	case "WRF-Chem":
		var err error
		ctm, err = NewWRFChem(cfg)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("inmap preprocessor: the CTMType you specified, '%s', is invalid. Valid options are WRF-Chem and GEOS-Chem", cfg.CTMType)
	}
	if err := Preprocess(ctm, cfg); err != nil {
		log.Fatal(err)
	}
}

// ReadConfigFile Reads and parses a json configuration file.
func ReadConfigFile(filename string) (*ConfigInfo, error) {
	// Open the configuration file
	var (
		file  *os.File
		bytes []byte
		err   error
	)
	file, err = os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("The configuration file you have specified, %v, does not "+
			"appear to exist. Please check the file name and location and "+
			"try again.\n", filename)
	}
	reader := bufio.NewReader(file)
	bytes, err = ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	config := new(ConfigInfo)
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, fmt.Errorf(
			"There has been an error parsing the configuration file.\n"+
				"Please ensure that the file is in valid JSON format\n"+
				"(you can check for errors at http://jsonlint.com/)\n"+
				"and try again!\n\n%v\n\n", err.Error())
	}

	config.OutputFile = os.ExpandEnv(config.OutputFile)
	if _, err := os.Stat(filepath.Dir(config.OutputFile)); err != nil {
		return nil, fmt.Errorf("inmap: preprocessor output directory '%s' does not exist", config.OutputFile)
	}
	config.StartDate = os.ExpandEnv(config.StartDate)
	config.EndDate = os.ExpandEnv(config.EndDate)
	config.GridProj = os.ExpandEnv(config.GridProj)

	switch config.CTMType {
	case "WRF-Chem":
		err := checkPaths(&config.WRFChem)
		if err != nil {
			return nil, err
		}
	case "GEOS-Chem":
		err := checkPaths(&config.GEOSChem)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("inmap preprocessor: the CTMType you specified, '%s', is invalid. Valid options are WRF-Chem and GEOS-Chem", config.CTMType)
	}
	return config, nil
}

// checkPaths makes sure that none of the String
// fields in the given variable are empty and expands
// any environment variables that they contain.
// The given variable must be a pointer to a struct.
func checkPaths(paths interface{}) error {
	v := reflect.ValueOf(paths).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Type().Kind() == reflect.String {
			s := f.String()
			if s == "" {
				name := v.Type().Field(i).Name
				return fmt.Errorf("inmap preprocessor: configuration file field %s is empty", name)
			}
			s = os.ExpandEnv(s)
			f.SetString(s)
		}
	}
	return nil
}
