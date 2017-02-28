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
)

var (
	configFile = flag.String("config", "none", "Path to configuration file")
)

const (
	inDateFormat = "20060102"
	tolerance    = 1.e-10 // tolerance for comparing floats
)

// ConfigInfo holds the configuration information for the program run.
type ConfigInfo struct {
	Wrfout           string  // Location of WRF output files. [DATE] is a wild card for the simulation date.
	OutputDir        string  // Directory to put the output files in
	OutputFilePrefix string  // name for output files
	StartDate        string  // Format = "YYYYMMDD"
	EndDate          string  // Format = "YYYYMMDD"
	CtmGridXo        float64 // lower left of Chemical Transport Model (CTM) grid, x
	CtmGridYo        float64 // lower left of grid, y
	CtmGridDx        float64 // m
	CtmGridDy        float64 // m
	CtmGridNx        int
	CtmGridNy        int
	GridProj         string // projection info for CTM grid; Proj4 format
}

func main() {

	flag.Parse()
	if *configFile == "" {
		log.Println("Please specify configuration file as in " +
			"`wrf2inmap -config=configFile.json`")
		os.Exit(1)
	}
	cfg := ReadConfigFile(*configFile)
	w, err := NewWRFChem(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := Preprocess(w, cfg); err != nil {
		log.Fatal(err)
	}
}

// ReadConfigFile Reads and parses a json configuration file.
func ReadConfigFile(filename string) *ConfigInfo {
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

	config := new(ConfigInfo)
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		fmt.Printf(
			"There has been an error parsing the configuration file.\n"+
				"Please ensure that the file is in valid JSON format\n"+
				"(you can check for errors at http://jsonlint.com/)\n"+
				"and try again!\n\n%v\n\n", err.Error())
		os.Exit(1)
	}

	config.OutputDir = os.ExpandEnv(config.OutputDir)
	config.OutputFilePrefix = os.ExpandEnv(config.OutputFilePrefix)
	config.Wrfout = os.ExpandEnv(config.Wrfout)
	config.StartDate = os.ExpandEnv(config.StartDate)
	config.EndDate = os.ExpandEnv(config.EndDate)
	config.GridProj = os.ExpandEnv(config.GridProj)

	err = os.MkdirAll(config.OutputDir, os.ModePerm)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
	return config
}
