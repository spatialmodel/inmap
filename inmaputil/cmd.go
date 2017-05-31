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
              layers specifies a ist of vertical layer numbers to
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
