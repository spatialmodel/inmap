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

package cmd

import (
	"fmt"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/inmaputil"
	"github.com/spatialmodel/inmap/sr"
	"github.com/spf13/cobra"
)

const year = "2017"

// These variables specify confuration flags.
var (
	// configFile specifies the location of the configuration file.
	configFile string

	// static specifies whether the simulation should be run with a static
	// (vs. dynamic) resolution grid.
	static bool

	// createGrid specifies whether the variable-resolution grid should be
	// created on-the-fly for static runs rather than reading it from a file.
	// For dynamic gridding, the grid is always created on-the-fly.
	createGrid bool

	// layers specifies a list of vertical layer numbers to be included
	// in an SR matrix.
	layers []int

	// begin specifies the starting grid index for SR matrix creation.
	begin int

	// end specifies the ending grid index for SR matrix creation.
	end int
)

func init() {
	// Link the commands together.
	Root.AddCommand(versionCmd)
	Root.AddCommand(runCmd)
	runCmd.AddCommand(steadyCmd)
	Root.AddCommand(gridCmd)
	Root.AddCommand(preprocCmd)
	Root.AddCommand(srCmd)
	srCmd.AddCommand(srPredictCmd)
	Root.AddCommand(workerCmd)

	// Create the configuration flags.
	Root.PersistentFlags().StringVar(&configFile, "config", "./inmap.toml", "configuration file location")

	steadyCmd.PersistentFlags().BoolVarP(&static, "static", "s", false,
		"Run with a static grid that is determined before the simulation starts. "+
			"If false, run with a dynamic grid that changes resolution depending on spatial "+
			"gradients in population density and concentration.")
	steadyCmd.PersistentFlags().BoolVar(&createGrid, "creategrid", false,
		"Create the variable-resolution grid as specified in the configuration file"+
			" before starting the simulation instead of reading it from a file. "+
			"If --static is false, then this flag will also be automatically set to false.")

	srCmd.Flags().IntSliceVar(&layers, "layers", []int{0, 2, 4, 6},
		"List of layer numbers to create matrices for.")
	srCmd.Flags().IntVar(&begin, "begin", 0, "Beginning row index.")
	srCmd.Flags().IntVar(&end, "end", -1, "End row index. Default is -1 (the last row).")

	srCmd.Flags().StringVar(&sr.RPCPort, "rpcport", "6060",
		"Set the port to be used for RPC communication.")
	workerCmd.Flags().StringVar(&sr.RPCPort, "rpcport", "6060",
		"Set the port to be used for RPC communication.")
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
		cfg, err := inmaputil.ReadConfigFile(configFile)
		if err != nil {
			return err
		}
		return inmaputil.Run(cfg, !static, createGrid, inmaputil.DefaultScienceFuncs, nil, nil, nil)
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
		cfg, err := inmaputil.ReadConfigFile(configFile)
		if err != nil {
			return err
		}
		return inmaputil.Grid(cfg)
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
		cfg, err := inmaputil.ReadConfigFile(configFile)
		if err != nil {
			return err
		}
		return inmaputil.Preproc(cfg)
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
		cfg, err := inmaputil.ReadConfigFile(configFile)
		if err != nil {
			return err
		}
		return inmaputil.RunSR(cfg, configFile, begin, end, layers)
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
		cfg, err := inmaputil.ReadConfigFile(configFile)
		if err != nil {
			return err
		}
		worker, err := inmaputil.NewWorker(cfg)
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
		cfg, err := inmaputil.ReadConfigFile(configFile)
		if err != nil {
			return err
		}
		return inmaputil.SRPredict(cfg)
	},
	DisableAutoGenTag: true,
}
