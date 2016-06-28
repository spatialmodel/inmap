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

package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/ctessum/geom/proj"
	"github.com/spatialmodel/inmap"
)

// ConfigData holds information about an InMAP configuration.
type ConfigData struct {

	// VarGrid provides information for specifying the variable resolution
	// grid.
	VarGrid inmap.VarGridConfig

	// InMAPData is the path to location of baseline meteorology and pollutant data.
	// The path can include environment variables.
	InMAPData string

	// VariableGridData is the path to the location of the variable-resolution gridded
	// InMAP data, or the location where it should be created if it doesn't already
	// exist. The path can include environment variables.
	VariableGridData string

	// EmissionsShapefiles are the paths to any emissions shapefiles.
	// Can be elevated or ground level; elevated files need to have columns
	// labeled "height", "diam", "temp", and "velocity" containing stack
	// information in units of m, m, K, and m/s, respectively.
	// Emissions will be allocated from the geometries in the shape file
	// to the InMAP computational grid, but the mapping projection of the
	// shapefile must be the same as the projection InMAP uses.
	// Can include environment variables.
	EmissionsShapefiles []string

	// EmissionUnits gives the units that the input emissions are in.
	// Acceptable values are 'tons/year' and 'kg/year'.
	EmissionUnits string

	// Path to desired output shapefile location. Can include environment variables.
	OutputFile string

	// If OutputAllLayers is true, output data for all model layers. If false, only output
	// the lowest layer.
	OutputAllLayers bool

	// OutputVariables specifies which model variables should be included in the
	// output file.
	// Can include environment variables.
	OutputVariables []string

	// NumIterations is the number of iterations to calculate. If < 1, convergence
	// is automatically calculated.
	NumIterations int

	// Port for hosting web page. If HTTPport is `8080`, then the GUI
	// would be viewed by visiting `localhost:8080` in a web browser.
	// If HTTPport is "", then the web server doesn't run.
	HTTPAddress string

	// SRLogDir is the directory that log files should be stored in when creating
	// a source-receptor matrix. It can contain environment variables.
	SRLogDir string

	// SROutputFile is the path where the output file is or should be created
	// when creating a source-receptor matrix. It can contain environment variables.
	SROutputFile string

	sr *proj.SR
}

// ReadConfigFile reads and parses a TOML configuration file.
func ReadConfigFile(filename string) (config *ConfigData, err error) {
	// Open the configuration file
	var (
		file  *os.File
		bytes []byte
	)
	file, err = os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("the configuration file you have specified, %v, does not "+
			"appear to exist. Please check the file name and location and "+
			"try again.\n", filename)
	}
	reader := bufio.NewReader(file)
	bytes, err = ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("problem reading configuration file: %v", err)
	}

	config = new(ConfigData)
	_, err = toml.Decode(string(bytes), config)
	if err != nil {
		return nil, fmt.Errorf(
			"there has been an error parsing the configuration file: %v\n", err)
	}

	for i, v := range config.OutputVariables {
		config.OutputVariables[i] = os.ExpandEnv(v)
	}

	config.InMAPData = os.ExpandEnv(config.InMAPData)
	config.VariableGridData = os.ExpandEnv(config.VariableGridData)
	config.OutputFile = os.ExpandEnv(config.OutputFile)
	config.VarGrid.CensusFile = os.ExpandEnv(config.VarGrid.CensusFile)
	config.VarGrid.MortalityRateFile = os.ExpandEnv(config.VarGrid.MortalityRateFile)
	config.SROutputFile = os.ExpandEnv(config.SROutputFile)
	config.SRLogDir = os.ExpandEnv(config.SRLogDir)

	for i := 0; i < len(config.EmissionsShapefiles); i++ {
		config.EmissionsShapefiles[i] =
			os.ExpandEnv(config.EmissionsShapefiles[i])
	}

	if config.OutputFile == "" {
		return nil, fmt.Errorf("you need to specify an output file in the " +
			"configuration file(for example: " +
			"\"OutputFile\":\"output.shp\"")
	}

	if config.VarGrid.GridProj == "" {
		return nil, fmt.Errorf("you need to specify the InMAP grid projection in the " +
			"'GridProj' configuration variable.")
	}
	config.sr, err = proj.Parse(config.VarGrid.GridProj)
	if err != nil {
		return nil, fmt.Errorf("the following error occured while parsing the InMAP grid"+
			"projection (the InMAPProj variable): %v", err)
	}

	if len(config.OutputVariables) == 0 {
		return nil, fmt.Errorf("there are no variables specified for output. Please fill in " +
			"the OutputVariables section of the configuration file and try again.")
	}

	if config.EmissionUnits != "tons/year" && config.EmissionUnits != "kg/year" {
		return nil, fmt.Errorf("the EmissionUnits variable in the configuration file "+
			"needs to be set to either tons/year or kg/year, but is currently set to `%s`",
			config.EmissionUnits)
	}

	outdir := filepath.Dir(config.OutputFile)
	err = os.MkdirAll(outdir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("problem creating output directory: %v", err)
	}
	return
}
