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

package inmaputil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ctessum/geom/proj"
	"github.com/spatialmodel/inmap"
)

// ConfigData holds information about an InMAP configuration.
type ConfigData struct {
	// VarGrid provides information for specifying the variable resolution grid.
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
	// Acceptable values are 'tons/year', 'kg/year', 'ug/s', and 'μg/s'.
	EmissionUnits string

	// OutputFile is the path to the desired output shapefile location. It can
	// include environment variables.
	OutputFile string

	// LogFile is the path to the desired logfile location. It can include
	// environment variables. If LogFile is left blank, the logfile will be saved in
	// the same location as the OutputFile.
	LogFile string

	// If OutputAllLayers is true, output data for all model layers. If false, only output
	// the lowest layer.
	OutputAllLayers bool

	// OutputVariables specifies which model variables should be included in the
	// output file. It can include environment variables.
	OutputVariables map[string]string

	// NumIterations is the number of iterations to calculate. If < 1, convergence
	// is automatically calculated.
	NumIterations int

	// Port for hosting web page. If HTTPport is `8080`, then the GUI
	// would be viewed by visiting `localhost:8080` in a web browser.
	// If HTTPport is "", then the web server doesn't run.
	HTTPAddress string

	// SR holds information related to source-receptor matrix
	// creation.
	SR struct {
		// LogDir is the directory that log files should be stored in when creating
		// a source-receptor matrix. It can contain environment variables.
		LogDir string

		// OutputFile is the path where the output file is or should be created
		// when creating a source-receptor matrix. It can contain environment variables.
		OutputFile string
	}

	// Preproc holds configuration information for the preprocessor.
	Preproc struct {
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

			// GEOSA3Dyn is the location of the GEOS 3-hour average dynamical
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
	}

	sr *proj.SR
}

// LoadConfig reads and parses an InMAP configuration information.
func LoadConfigFile() (*ConfigData, error) {
	var err error
	// Find and read in the configuration file, if there is one.
	if err = Cfg.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("inmap: problem reading configuration file: %v", err)
	}
	var config ConfigData
	// Load configuration info into our struct.
	if err = Cfg.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("inmap: problem reading configuration file: %v", err)
	}

	// Clean and check configuration.
	for k, v := range config.OutputVariables {
		v = strings.Replace(v, "\r\n", " ", -1)
		v = strings.Replace(v, "\n", " ", -1)
		config.OutputVariables[os.ExpandEnv(k)] = os.ExpandEnv(v)
	}

	config.InMAPData = os.ExpandEnv(config.InMAPData)
	config.VariableGridData = os.ExpandEnv(config.VariableGridData)
	config.OutputFile = os.ExpandEnv(config.OutputFile)
	config.LogFile = os.ExpandEnv(config.LogFile)
	config.VarGrid.CensusFile = os.ExpandEnv(config.VarGrid.CensusFile)
	config.VarGrid.MortalityRateFile = os.ExpandEnv(config.VarGrid.MortalityRateFile)
	config.SR.OutputFile = os.ExpandEnv(config.SR.OutputFile)
	config.SR.LogDir = os.ExpandEnv(config.SR.LogDir)

	for i := 0; i < len(config.EmissionsShapefiles); i++ {
		config.EmissionsShapefiles[i] =
			os.ExpandEnv(config.EmissionsShapefiles[i])
	}

	if config.OutputFile == "" {
		return nil, fmt.Errorf("you need to specify an output file in the " +
			"configuration file(for example: " +
			"\"OutputFile\":\"output.shp\"")
	}

	if config.LogFile == "" {
		config.LogFile = strings.TrimSuffix(config.OutputFile, filepath.Ext(config.OutputFile)) + ".log"
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
			"the OutputVariables configuration and try again.")
	}

	if _, ok := map[string]struct{}{
		"tons/year": {},
		"kg/year":   {},
		"ug/s":      {},
		"μg/s":      {}}[config.EmissionUnits]; !ok {
		return nil, fmt.Errorf("the EmissionUnits variable in the configuration file "+
			"needs to be set to either tons/year, kg/year, ug/s, or μg/s, but is currently set to `%s`",
			config.EmissionUnits)
	}

	outdir := filepath.Dir(config.OutputFile)
	err = os.MkdirAll(outdir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("problem creating output directory: %v", err)
	}
	return &config, nil
}
