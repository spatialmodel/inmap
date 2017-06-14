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

package inmaputil

import (
	"fmt"
	"strconv"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/sr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const year = "2017"

var Cfg *viper.Viper

type option struct {
	name, usage, shorthand string
	defaultVal             interface{}
	commands               []*cobra.Command
	flagsets               []*pflag.FlagSet
}

// Options are the configuration options available to InMAP.
var Options []option

func init() {
	Options = []option{
		{
			name: "config",
			usage: `
              config specifies the configuration file location.`,
			defaultVal: "",
			commands:   []*cobra.Command{Root},
			flagsets:   []*pflag.FlagSet{Root.PersistentFlags()},
		},
		{
			name: "static",
			usage: `
              static specifies whether to run with a static grid that
              is determined before the simulation starts. If false, the
              simulation runs with a dynamic grid that changes resolution
              depending on spatial gradients in population density and
              concentration.`,
			shorthand:  "s",
			defaultVal: false,
			commands:   []*cobra.Command{steadyCmd},
			flagsets:   []*pflag.FlagSet{steadyCmd.PersistentFlags()},
		},
		{
			name: "creategrid",
			usage: `
              creategrid specifies whether to create the
              variable-resolution grid as specified in the configuration file before starting
              the simulation instead of reading it from a file. If --static is false, then
              this flag will also be automatically set to false.`,
			defaultVal: false,
			commands:   []*cobra.Command{steadyCmd},
			flagsets:   []*pflag.FlagSet{steadyCmd.PersistentFlags()},
		},
		{
			name: "layers",
			usage: `
              layers specifies a list of vertical layer numbers to
              be included in the SR matrix.`,
			defaultVal: []int{0, 2, 4, 6},
			commands:   []*cobra.Command{srCmd},
			flagsets:   []*pflag.FlagSet{srCmd.Flags()},
		},
		{
			name: "begin",
			usage: `
              begin specifies the beginning grid index (inclusive) for SR
              matrix generation.`,
			defaultVal: 0,
			commands:   []*cobra.Command{srCmd},
			flagsets:   []*pflag.FlagSet{srCmd.Flags()},
		},
		{
			name: "end",
			usage: `
              end specifies the ending grid index (exclusive) for SR matrix
              generation. The default is -1 which represents the last row.`,
			defaultVal: -1,
			commands:   []*cobra.Command{srCmd},
			flagsets:   []*pflag.FlagSet{srCmd.Flags()},
		},
		{
			name: "rpcport",
			usage: `
              rpcport specifies the port to be used for RPC communication
              when using distributed computing.`,
			defaultVal: "6060",
			commands:   []*cobra.Command{srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.VariableGridXo",
			usage: `
              Vargrid.VariableGridXo specifies the X coordinate of the
              lower-left corner of the InMAP grid.`,
			defaultVal: -2736000.0,
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.VariableGridYo",
			usage: `
              Vargrid.VariableGridYo specifies the Y coordinate of the
              lower-left corner of the InMAP grid.`,
			defaultVal: -2088000.0,
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.VariableGridDx",
			usage: `
              Vargrid.VariableGridDx specifies the X edge lengths of grid
              cells in the outermost nest, in the units of the grid model
              spatial projection--typically meters or degrees latitude
              and longitude.`,
			defaultVal: 48000.0,
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.VariableGridDy",
			usage: `
              Vargrid.VariableGridDy specifies the Y edge lengths of grid
              cells in the outermost nest, in the units of the grid model
              spatial projection--typically meters or degrees latitude
              and longitude.`,
			defaultVal: 48000.0,
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.Xnests",
			usage: `
							Xnests specifies nesting multiples in the X direction.`,
			defaultVal: []int{111, 2, 2, 3, 2, 2},
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.Ynests",
			usage: `
							Ynests specifies nesting multiples in the Y direction.`,
			defaultVal: []int{84, 2, 2, 3, 2, 2},
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.HiResLayers",
			usage: `
							HiResLayers is the number of layers, starting at ground level, to do
							nesting in. Layers above this will have all grid cells in the lowest
							spatial resolution.`,
			defaultVal: 8,
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.GridProj",
			usage: `
							GridProj gives projection info for the CTM grid in Proj4 or WKT format.`,
			defaultVal: "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.PopDensityThreshold",
			usage: `
							PopDensityThreshold is a limit for people per unit area in a grid cell
							(units will typically be either people / m^2 or people / degree^2,
							depending on the spatial projection of the model grid). If
							the population density in a grid cell is above this level, the cell in question
							is a candidate for splitting into smaller cells.`,
			defaultVal: 0.0055,
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.PopThreshold",
			usage: `
							PopThreshold is a limit for the total number of people in a grid cell.
							If the total population in a grid cell is above this level, the cell in question
							is a candidate for splitting into smaller cells.`,
			defaultVal: 40000.0,
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Vargrid.PopConcThreshold",
			usage: `
							PopConcThreshold is the limit for
							Σ(|ΔConcentration|)*combinedVolume*|ΔPopulation| / {Σ(|totalMass|)*totalPopulation}.
							See the documentation for PopConcMutator for more information.`,
			defaultVal: 0.000000001,
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "InMAPData",
			usage: `
			        InMAPData is the path to location of baseline meteorology and pollutant data.
			        The path can include environment variables.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/testInMAPInputData.ncf",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "VariableGridData",
			usage: `
			        VariableGridData is the path to the location of the variable-resolution gridded
			        InMAP data, or the location where it should be created if it doesn't already
			        exist. The path can include environment variables.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/inmapVarGrid.gob",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "EmissionsShapefiles",
			usage: `
							EmissionsShapefiles are the paths to any emissions shapefiles.
						  Can be elevated or ground level; elevated files need to have columns
							labeled "height", "diam", "temp", and "velocity" containing stack
							information in units of m, m, K, and m/s, respectively.
							Emissions will be allocated from the geometries in the shape file
							to the InMAP computational grid, but the mapping projection of the
							shapefile must be the same as the projection InMAP uses.
							Can include environment variables.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/testEmis.shp",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "EmissionUnits",
			usage: `
			        EmissionUnits gives the units that the input emissions are in.
							Acceptable values are 'tons/year', 'kg/year', 'ug/s', and 'μg/s'.`,
			defaultVal:"tons/year",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "OutputFile",
			usage: `
			        OutputFile is the path to the desired output shapefile location. It can
							include environment variables.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/output_${InMAPRunType}.shp",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "LogFile",
			usage: `
			        LogFile is the path to the desired logfile location. It can include
							environment variables. If LogFile is left blank, the logfile will be saved in
							the same location as the OutputFile.`,
			defaultVal:"",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "OutputAllLayers",
			usage: `
			        If OutputAllLayers is true, output data for all model layers. If false, only output
							the lowest layer.`,
			defaultVal:false,
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "OutputVariables",
			usage: `
			        OutputVariables specifies which model variables should be included in the
							output file. It can include environment variables.`,
			defaultVal:"",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "NumIterations",
			usage: `
			        NumIterations is the number of iterations to calculate. If < 1, convergence
							is automatically calculated.`,
			defaultVal:0,
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "HTTPAddress",
			usage: `
			        Port for hosting web page. If HTTPport is `8080`, then the GUI
			 			  would be viewed by visiting "localhost:8080" in a web browser.
						  If HTTPport is "", then the web server doesn't run.`,
			defaultVal:":8080",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "SR.LogDir",
			usage: `
			        LogDir is the directory that log files should be stored in when creating
							a source-receptor matrix. It can contain environment variables.`,
			defaultVal:"log",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "SR.OutputFile",
			usage: `
			        SR.OutputFile is the path where the output file is or should be created
		 					when creating a source-receptor matrix. It can contain environment variables.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/output_${InMAPRunType}.shp",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.CTMType",
			usage: `
			        Preproc.CTMType specifies what type of chemical transport
							model we are going to be reading data from. Valid
							options are "GEOS-Chem" and "WRF-Chem".`,
			defaultVal:"WRF-Chem",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.WRFChem.WRFOut",
			usage: `
			        Preproc.WRFChem.WRFOut is the location of WRF-Chem output files.
							[DATE] should be used as a wild card for the simulation date.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/preproc/wrfout_d01_[DATE]",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSA1",
			usage: `
			        Preproc.GEOSChem.GEOSA1 is the location of the GEOS 1-hour time average files.
							[DATE] should be used as a wild card for the simulation date.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/preproc/GEOSFP.[DATE].A1.2x25.nc",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSA3Cld",
			usage: `
			        Preproc.GEOSChem.GEOSA3Cld is the location of the GEOS 3-hour average cloud
							parameter files. [DATE] should be used as a wild card for
							the simulation date.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/preproc/GEOSFP.[DATE].A3cld.2x25.nc",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSA3Dyn",
			usage: `
							Preproc.GEOSChem.GEOSA3Dyn is the location of the GEOS 3-hour average dynamical
							parameter files. [DATE] should be used as a wild card for
							the simulation date.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/preproc/GEOSFP.[DATE].A3dyn.2x25.nc",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSI3",
			usage: `
							Preproc.GEOSChem.GEOSI3 is the location of the GEOS 3-hour instantaneous parameter
							files. [DATE] should be used as a wild card for
							the simulation date.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/preproc/GEOSFP.[DATE].I3.2x25.nc",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSA3MstE",
			usage: `
							Preproc.GEOSChem.GEOSA3MstE is the location of the GEOS 3-hour average moist parameters
							on level edges files. [DATE] should be used as a wild card for
							the simulation date.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/preproc/GEOSFP.[DATE].A3mstE.2x25.nc",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSChem",
			usage: `
							Preproc.GEOSChem.GEOSChem is the location of GEOS-Chem output files.
							[DATE] should be used as a wild card for the simulation date.`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/preproc/gc_output.[DATE].nc",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.VegTypeGlobal",
			usage: `
							Preproc.GEOSChem.VegTypeGlobal is the location of the GEOS-Chem vegtype.global file,
							which is described here:
							http://wiki.seas.harvard.edu/geos-chem/index.php/Olson_land_map#Structure_of_the_vegtype.global_file`,
			defaultVal:"${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/preproc/vegtype.global.txt",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.StartDate",
			usage: `
							Preproc.StartDate is the date of the beginning of the simulation.
							Format = "YYYYMMDD".`,
			defaultVal:"",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.EndDate",
			usage: `
							Preproc.EndDate is the date of the end of the simulation.
							Format = "YYYYMMDD".`,
			defaultVal:"",
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.CtmGridXo",
			usage: `
							Preproc.CtmGridXo is the lower left of Chemical Transport Model (CTM) grid, x`,
			defaultVal:math.NaN(),
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.CtmGridYo",
			usage: `
							Preproc.CtmGridYo is the lower left of grid, y`,
			defaultVal:math.NaN(),
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.CtmGridDx",
			usage: `
							Preproc.CtmGridDx is the grid cell length in x direction [m]`,
			defaultVal:math.NaN(),
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},
		{
			name: "Preproc.CtmGridDy",
			usage: `
							Preproc.CtmGridDy is the grid cell length in y direction [m]`,
			defaultVal:math.NaN(),
			commands:   []*cobra.Command{runCmd, gridCmd, srCmd, workerCmd},
			flagsets:   []*pflag.FlagSet{runCmd.PersistentFlags(), gridCmd.Flags(), srCmd.Flags(), workerCmd.Flags()},
		},

	}

	Cfg = viper.New()

	// Set up the configuration environment.
	Cfg.SetConfigName("inmap") // Set the default configuration file name.
	// Set a directory to search for the configuration file in.
	Cfg.AddConfigPath(".")
	// Set the prefix for configuration environment variables.
	Cfg.SetEnvPrefix("INMAP")

	for _, option := range Options {
		if Cfg.IsSet(option.name) {
			// Don't set the option if it is already set somewhere else,
			// such as in a test.
			continue
		}
		for _, set := range option.flagsets {
			switch option.defaultVal.(type) {
			case string:
				if option.shorthand == "" {
					set.String(option.name, option.defaultVal.(string), option.usage)
				} else {
					set.StringP(option.name, option.shorthand, option.defaultVal.(string), option.usage)
				}
			case bool:
				if option.shorthand == "" {
					set.Bool(option.name, option.defaultVal.(bool), option.usage)
				} else {
					set.BoolP(option.name, option.shorthand, option.defaultVal.(bool), option.usage)
				}
			case int:
				if option.shorthand == "" {
					set.Int(option.name, option.defaultVal.(int), option.usage)
				} else {
					set.IntP(option.name, option.shorthand, option.defaultVal.(int), option.usage)
				}
			case []int:
				if option.shorthand == "" {
					set.IntSlice(option.name, option.defaultVal.([]int), option.usage)
				} else {
					set.IntSliceP(option.name, option.shorthand, option.defaultVal.([]int), option.usage)
				}
			case float64:
				if option.shorthand == "" {
					set.Float64(option.name, option.defaultVal.(float64), option.usage)
				} else {
					set.Float64P(option.name, option.shorthand, option.defaultVal.(float64), option.usage)
				}
			default:
				panic("invalid argument type")
			}
			Cfg.BindPFlag(option.name, set.Lookup(option.name))
		}
	}
	// If the user has set a config file path, use that one.
	if cfgpath := Cfg.GetString("config"); cfgpath != "" {
		Cfg.SetConfigFile(cfgpath)
	}
}

// Link the commands together.
func init() {
	Root.AddCommand(versionCmd)
	Root.AddCommand(runCmd)
	runCmd.AddCommand(steadyCmd)
	Root.AddCommand(gridCmd)
	Root.AddCommand(preprocCmd)
	Root.AddCommand(srCmd)
	srCmd.AddCommand(srPredictCmd)
	Root.AddCommand(workerCmd)
}

// Root is the main command.
var Root = &cobra.Command{
	Use:   "inmap",
	Short: "A reduced-form air quality model.",
	Long: `InMAP is a reduced-form air quality model for fine particulate matter (PM2.5).
Use the subcommands specified below to access the model functionality.
Additional information is available at http://inmap.spatialmodel.com.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		fmt.Println(`
	------------------------------------------------
	                    Welcome!
	  (In)tervention (M)odel for (A)ir (P)ollution
	                Version " + inmap.Version + "
	               Copyright 2013-` + year + `
	                the InMAP Authors
	------------------------------------------------`)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		fmt.Println(`
	------------------------------------
	           InMAP Completed!
	------------------------------------`)
	},
	DisableAutoGenTag: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  "version prints the version number of this version of InMAP.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("InMAP v%s\n", inmap.Version)
	},
	DisableAutoGenTag: true,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the model.",
	Long: `run runs an InMAP simulation. Use the subcommands specified below to
choose a run mode. (Currently 'steady' is the only available run mode.)`,
	DisableAutoGenTag: true,
}

// steadyCmd is a command that runs a steady-state simulation.
var steadyCmd = &cobra.Command{
	Use:   "steady",
	Short: "Run InMAP in steady-state mode.",
	Long: `steady runs InMAP in steady-state mode to calculate annual average
concentrations with no temporal variability.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfigFile()
		if err != nil {
			return err
		}
		return Run(cfg, !Cfg.GetBool("static"), Cfg.GetBool("createGrid"), DefaultScienceFuncs, nil, nil, nil)
	},
	DisableAutoGenTag: true,
}

// gridCmd is a command that creates and saves a new variable resolution grid.
var gridCmd = &cobra.Command{
	Use:   "grid",
	Short: "Create a variable resolution grid",
	Long: `grid creates and saves a variable resolution grid as specified by the
information in the configuration file. The saved data can then be loaded
for future InMAP simulations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfigFile()
		if err != nil {
			return err
		}
		return Grid(cfg)
	},
	DisableAutoGenTag: true,
}

var preprocCmd = &cobra.Command{
	Use:   "preproc",
	Short: "Preprocess CTM output",
	Long: `preproc preprocesses chemical transport model
output as specified by information in the configuration
file and saves the result for use in future InMAP simulations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfigFile()
		if err != nil {
			return err
		}
		return Preproc(cfg)
	},
	DisableAutoGenTag: true,
}

// srCmd is a command that creates an SR matrix.
var srCmd = &cobra.Command{
	Use:   "sr",
	Short: "Create an SR matrix.",
	Long: `sr creates a source-receptor matrix from InMAP simulations.
Simulations will be run on the cluster defined by $PBS_NODEFILE.
If $PBS_NODEFILE doesn't exist, the simulations will run on the
local machine.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfigFile()
		if err != nil {
			return err
		}
		layersStr := Cfg.GetStringSlice("layers")
		layers := make([]int, len(layersStr))
		for i, l := range layersStr {
			li, err := strconv.ParseInt(l, 10, 64)
			if err != nil {
				return err
			}
			layers[i] = int(li)
		}
		return RunSR(cfg, Cfg.GetString("configFile"), Cfg.GetInt("begin"), Cfg.GetInt("end"), layers)
	},
	DisableAutoGenTag: true,
}

// workerCmd is a command that starts a new worker.
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start an InMAP worker.",
	Long: `worker starts an InMAP worker that listens over RPC for simulation requests,
does the simulations, and returns results.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfigFile()
		if err != nil {
			return err
		}
		worker, err := NewWorker(cfg)
		if err != nil {
			return err
		}
		return sr.WorkerListen(worker, sr.RPCPort)
	},
	DisableAutoGenTag: true,
}

// srPredictCmd is a command that makes predictions using the SR matrix.
var srPredictCmd = &cobra.Command{
	Use:   "predict",
	Short: "Predict concentrations",
	Long: `predict uses the SR matrix specified in the configuration file
field SR.OutputFile to predict concentrations resulting
from the emissions specified in the EmissionsShapefiles field in the configuration
file, outputting the results in the shapefile specified in OutputFile field.
of the configuration file. The EmissionUnits field in the configuration
file specifies the units of the emissions. Output units are μg particulate
matter per m³ air.

	 Output variables:
	 PNH4: Particulate ammonium
	 PNO3: Particulate nitrate
	 PSO4: Particulate sulfate
	 SOA: Secondary organic aerosol
	 PrimaryPM25: Primarily emitted PM2.5
	 TotalPM25: The sum of the above components`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfigFile()
		if err != nil {
			return err
		}
		return SRPredict(cfg)
	},
	DisableAutoGenTag: true,
}
