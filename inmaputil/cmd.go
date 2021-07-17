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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ctessum/gobra"
	"github.com/lnashier/viper"
	"github.com/skratchdot/open-golang/open"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Cfg holds configuration information.
type Cfg struct {
	*viper.Viper

	// inputFiles holds the names of the configuration options that are input
	// files.
	inputFiles []string

	// outputFiles holds the names of the configuration options that are output
	// files.
	outputFiles []string

	Root, versionCmd, runCmd, preprocCmd, combineCmd, steadyCmd, gridCmd    *cobra.Command
	srCmd, srPredictCmd, srStartCmd, srSaveCmd, srCleanCmd                  *cobra.Command
	cloudCmd, cloudStartCmd, cloudStatusCmd, cloudOutputCmd, cloudDeleteCmd *cobra.Command
}

// InputFiles returns the names of the configuration options that are input
// files.
func (cfg *Cfg) InputFiles() []string { return cfg.inputFiles }

// OutputFiles returns the names of the configuration options that are output
// files.
func (cfg *Cfg) OutputFiles() []string { return cfg.outputFiles }

var options []struct {
	name, usage, shorthand string
	defaultVal             interface{}
	flagsets               []*pflag.FlagSet
	isInputFile            bool // Does the option represent an input file name?
	isOutputFile           bool // Does the option represent an output file name?
}

func InitializeConfig() *Cfg {

	cfg := &Cfg{
		Viper: viper.New(),
	}

	// Root is the main command.
	cfg.Root = &cobra.Command{
		Use:   "inmap",
		Short: "A reduced-form air quality model.",
		Long: `InMAP is a reduced-form air quality model for fine particulate matter (PM2.5).
Use the subcommands specified below to access the model functionality.
Additional information is available at http://inmap.spatialmodel.com.

Refer to the subcommand documentation for configuration options and default settings.
Configuration can be changed by using a configuration file (and providing the
path to the file using the --config flag), by using command-line arguments,
or by setting environment variables in the format 'INMAP_var' where 'var' is the
name of the variable to be set. Many configuration variables are additionally
allowed to contain environment variables within them.
Refer to https://github.com/spf13/viper for additional configuration information.`,
		DisableAutoGenTag: true,
		// Tell the Root command to run this function every time it is run.
		PersistentPreRunE: func(*cobra.Command, []string) error {
			return setConfig(cfg)
		},
	}

	cfg.versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Long:  "version prints the version number of this version of InMAP.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("InMAP v%s\n", inmap.Version)
			cmd.Printf("InMAP v%s\n", inmap.Version)
		},
		DisableAutoGenTag: true,
	}

	cfg.runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the model.",
		Long: `run runs an InMAP simulation. Use the subcommands specified below to
choose a run mode. (Currently 'steady' is the only available run mode.)`,
		DisableAutoGenTag: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := setConfig(cfg); err != nil {
				return err
			}
			outputFile, err := checkOutputFile(cfg.GetString("OutputFile"))
			if err != nil {
				return err
			}
			cfg.Set("LogFile", checkLogFile(cfg.GetString("LogFile"), outputFile))
			return nil
		},
	}

	// steadyCmd is a command that runs a steady-state simulation.
	cfg.steadyCmd = &cobra.Command{
		Use:   "steady",
		Short: "Run InMAP in steady-state mode.",
		Long: `steady runs InMAP in steady-state mode to calculate annual average
concentrations with no temporal variability.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outChan := outChan()

			vgc, err := VarGridConfig(cfg.Viper)
			if err != nil {
				return err
			}
			outputFile, err := checkOutputFile(cfg.GetString("OutputFile"))
			if err != nil {
				return err
			}
			outputVars, err := checkOutputVars(GetStringMapString("OutputVariables", cfg.Viper))
			if err != nil {
				return err
			}
			emisUnits, err := checkEmissionUnits(cfg.GetString("EmissionUnits"))
			if err != nil {
				return err
			}

			shapeFiles := removeShpSupportFiles(expandStringSlice(cfg.GetStringSlice("EmissionsShapefiles")))
			// This goes over each shapeFile and downloads it if necessary.
			for i := range shapeFiles {
				shapeFiles[i] = maybeDownload(context.TODO(), shapeFiles[i], outChan)
			}

			mask, err := parseMask(maybeDownload(context.Background(), cfg.GetString("EmissionMaskGeoJSON"), outChan))
			if err != nil {
				return err
			}

			inventoryConfig, spatialConfig, err := aeputilConfig(cfg.Viper)
			if err != nil {
				return err
			}

			return Run(
				cmd,
				cfg.GetString("LogFile"),
				outputFile,
				cfg.GetBool("OutputAllLayers"),
				outputVars,
				emisUnits,
				shapeFiles, mask,
				vgc,
				inventoryConfig,
				spatialConfig,
				maybeDownload(context.TODO(), os.ExpandEnv(cfg.GetString("InMAPData")), outChan),
				maybeDownload(context.TODO(), os.ExpandEnv(cfg.GetString("VariableGridData")), outChan),
				cfg.GetInt("NumIterations"),
				!cfg.GetBool("static"), cfg.GetBool("creategrid"), DefaultScienceFuncs, nil, nil, nil,
				simplechem.Mechanism{})
		},
		DisableAutoGenTag: true,
	}

	// gridCmd is a command that creates and saves a new variable resolution grid.
	cfg.gridCmd = &cobra.Command{
		Use:   "grid",
		Short: "Create a variable resolution grid",
		Long: `grid creates and saves a variable resolution grid as specified by the
information in the configuration file. The saved data can then be loaded
for future InMAP simulations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outChan := outChan()

			vgc, err := VarGridConfig(cfg.Viper)
			if err != nil {
				return err
			}
			return Grid(
				maybeDownload(context.TODO(), os.ExpandEnv(cfg.GetString("InMAPData")), outChan),
				maybeDownload(context.TODO(), os.ExpandEnv(cfg.GetString("VariableGridData")), outChan),
				vgc)
		},
		DisableAutoGenTag: true,
	}

	cfg.preprocCmd = &cobra.Command{
		Use:   "preproc",
		Short: "Preprocess CTM output",
		Long: `preproc preprocesses chemical transport model
output as specified by information in the configuration
file and saves the result for use in future InMAP simulations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outChan := outChan()
			ctx := context.TODO()

			return Preproc(
				os.ExpandEnv(cfg.GetString("Preproc.StartDate")),
				os.ExpandEnv(cfg.GetString("Preproc.EndDate")),
				os.ExpandEnv(cfg.GetString("Preproc.CTMType")),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("Preproc.WRFChem.WRFOut")), outChan),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("Preproc.GEOSChem.GEOSA1")), outChan),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("Preproc.GEOSChem.GEOSA3Cld")), outChan),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("Preproc.GEOSChem.GEOSA3Dyn")), outChan),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("Preproc.GEOSChem.GEOSI3")), outChan),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("Preproc.GEOSChem.GEOSA3MstE")), outChan),
				os.ExpandEnv(cfg.GetString("Preproc.GEOSChem.GEOSApBp")),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("Preproc.GEOSChem.GEOSChem")), outChan),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("Preproc.GEOSChem.OlsonLandMap")), outChan),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("InMAPData")), outChan),
				cfg.GetFloat64("Preproc.CtmGridXo"),
				cfg.GetFloat64("Preproc.CtmGridYo"),
				cfg.GetFloat64("Preproc.CtmGridDx"),
				cfg.GetFloat64("Preproc.CtmGridDy"),
				cfg.GetBool("Preproc.GEOSChem.Dash"),
				cfg.GetString("Preproc.GEOSChem.ChemRecordInterval"),
				cfg.GetString("Preproc.GEOSChem.ChemFileInterval"),
				cfg.GetBool("Preproc.GEOSChem.NoChemHourIndex"),
			)
		},
		DisableAutoGenTag: true,
	}

	cfg.combineCmd = &cobra.Command{
		Use:   "combine",
		Short: "Combine preprocessed CTM output from nested grids",
		Long: `combine combines preprocessed chemical transport model
output from multiple nested grids into a single InMAP input file.
It should be run after independently preprocessing the output of
each nested grid.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			files := cfg.GetStringSlice("preprocessed_inputs")
			data := make([]*inmap.CTMData, len(files))
			for i, file := range files {
				f, err := os.Open(os.ExpandEnv(file))
				if err != nil {
					return fmt.Errorf("opening preprocessed input file: %w", err)
				}
				cfg := &inmap.VarGridConfig{}
				data[i], err = cfg.LoadCTMData(f)
				if err != nil {
					return fmt.Errorf("loading preprocessed input file: %w", err)
				}
			}
			combined, err := inmap.CombineCTMData(data...)
			if err != nil {
				return fmt.Errorf("combining preprocessed input files: %w", err)
			}
			f, err := os.Create(os.ExpandEnv(cfg.GetString("output_file")))
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			return combined.Write(f)
		},
		DisableAutoGenTag: true,
	}

	cfg.srCmd = &cobra.Command{
		Use:               "sr",
		Short:             "Interact with an SR matrix.",
		DisableAutoGenTag: true,
	}

	cfg.srStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start simulations to create an SR matrix",
		Long: `start starts the InMAP simulations necessary to create
a source-receptor matrix.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			vgc, err := VarGridConfig(cfg.Viper)
			if err != nil {
				return err
			}
			layers, err := intSliceFromString(cfg.GetString("layers"))
			if err != nil {
				return fmt.Errorf("inmap: reading SR 'layers': %v", err)
			}
			c, err := NewCloudClient(cfg)
			if err != nil {
				return err
			}
			ctx := context.TODO()
			return StartSR(
				ctx,
				cfg.GetString("job_name"),
				cfg.GetStringSlice("cmds"),
				int32(cfg.GetInt("memory_gb")),
				os.ExpandEnv(cfg.GetString("VariableGridData")),
				vgc,
				cfg.GetInt("begin"),
				cfg.GetInt("end"),
				layers,
				c,
				cfg,
			)
		},
		DisableAutoGenTag: true,
	}

	cfg.srSaveCmd = &cobra.Command{
		Use:   "save",
		Short: "Save simulation results to create an SR matrix",
		Long:  `save saves the results of InMAP simulations created using 'start'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outChan := outChan()

			vgc, err := VarGridConfig(cfg.Viper)
			if err != nil {
				return err
			}
			layers, err := intSliceFromString(cfg.GetString("layers"))
			if err != nil {
				return fmt.Errorf("inmap: reading SR 'layers': %v", err)
			}
			c, err := NewCloudClient(cfg)
			if err != nil {
				return err
			}
			ctx := context.TODO()
			return SaveSR(
				ctx,
				cfg.GetString("job_name"),
				os.ExpandEnv(cfg.GetString("SR.OutputFile")),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("VariableGridData")), outChan),
				vgc,
				cfg.GetInt("begin"),
				cfg.GetInt("end"),
				layers,
				c,
			)
		},
		DisableAutoGenTag: true,
	}

	cfg.srCleanCmd = &cobra.Command{
		Use:   "clean",
		Short: "clean cleans up temporary simulation output",
		Long:  `save cleans up the InMAP simulations created using 'start'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outChan := outChan()

			vgc, err := VarGridConfig(cfg.Viper)
			if err != nil {
				return err
			}
			layers, err := intSliceFromString(cfg.GetString("layers"))
			if err != nil {
				return fmt.Errorf("inmap: reading SR 'layers': %v", err)
			}
			c, err := NewCloudClient(cfg)
			if err != nil {
				return err
			}
			ctx := context.TODO()
			return CleanSR(
				ctx,
				cfg.GetString("job_name"),
				maybeDownload(ctx, os.ExpandEnv(cfg.GetString("VariableGridData")), outChan),
				vgc,
				cfg.GetInt("begin"),
				cfg.GetInt("end"),
				layers,
				c,
			)
		},
		DisableAutoGenTag: true,
	}

	// cloudCmd is a command that interfaces with the Kubernetes client in the
	// `cloud` subpackage.
	cfg.cloudCmd = &cobra.Command{
		Use:               "cloud",
		Short:             "Interact with a Kubernetes cluster.",
		DisableAutoGenTag: true,
	}

	// cloudStartCmd starts a cloud job.
	cfg.cloudStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start a job on a Kubernetes cluster.",
		Long: "Start a job on a Kubernetes cluster. Of the flags available to this command, " +
			"'cmds', 'storage_gb', and 'memory_gb' relate to the creation of the job." +
			" All other flags and configuation file information are used to configure the remote simulation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := NewCloudClient(cfg)
			if err != nil {
				return err
			}
			ctx := context.Background()
			return CloudJobStart(ctx, c, cfg)
		},
		DisableAutoGenTag: true,
	}

	// cloudStatusCmd checks the status of a cloud job.
	cfg.cloudStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check the status of a job on a Kubernetes cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := NewCloudClient(cfg)
			if err != nil {
				return err
			}
			ctx := context.Background()
			status, err := CloudJobStatus(ctx, c, cfg)
			if err != nil {
				return err
			}
			fmt.Println(status.Status)
			if status.Message != "" {
				fmt.Println(status.Message)
			}
			return nil
		},
		DisableAutoGenTag: true,
	}

	// cloudOutputCmd retrieves and saves the output of a cloud job.
	cfg.cloudOutputCmd = &cobra.Command{
		Use:   "output",
		Short: "Retrieve and save the output of a job on a Kubernetes cluster.",
		Long:  `The files will be saved in 'current_dir/job_name', where current_dir is the directory the command is run in.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := NewCloudClient(cfg)
			if err != nil {
				return err
			}
			ctx := context.Background()
			return CloudJobOutput(ctx, c, cfg)
		},
		DisableAutoGenTag: true,
	}

	// cloudDeleteCmd deletes a cloud job.
	cfg.cloudDeleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a cloud job.",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := NewCloudClient(cfg)
			if err != nil {
				return err
			}
			ctx := context.Background()
			return CloudJobDelete(ctx, cfg.GetString("job_name"), c)
		},
		DisableAutoGenTag: true,
	}

	// srPredictCmd is a command that makes predictions using the SR matrix.
	cfg.srPredictCmd = &cobra.Command{
		Use:   "srpredict",
		Short: "Predict concentrations",
		Long: `predict uses the SR matrix specified in the configuration file
field SR.OutputFile to predict concentrations resulting
from the emissions specified in the EmissionsShapefiles field in the configuration
file, outputting the results in the shapefile specified in OutputFile field.
of the configuration file. The EmissionUnits field in the configuration
file specifies the units of the emissions. The OutputVariables configuration
variable specifies the information to be output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outChan := outChan()

			vgc, err := VarGridConfig(cfg.Viper)
			if err != nil {
				return err
			}
			outputFile, err := checkOutputFile(cfg.GetString("OutputFile"))
			if err != nil {
				return err
			}
			outputVars, err := checkOutputVars(GetStringMapString("OutputVariables", cfg.Viper))
			if err != nil {
				return err
			}
			emisUnits, err := checkEmissionUnits(cfg.GetString("EmissionUnits"))
			if err != nil {
				return err
			}

			shapeFiles := expandStringSlice(cfg.GetStringSlice("EmissionsShapefiles"))
			// This goes over each shapeFile and downloads it.
			for i := range shapeFiles {
				shapeFiles[i] = maybeDownload(context.TODO(), shapeFiles[i], outChan)
			}

			mask, err := parseMask(cfg.GetString("EmissionMaskGeoJSON"))
			if err != nil {
				return err
			}

			return SRPredict(
				emisUnits,
				os.ExpandEnv(cfg.GetString("SR.OutputFile")),
				outputFile,
				outputVars,
				shapeFiles,
				mask,
				vgc,
			)
		},
		DisableAutoGenTag: true,
	}

	// Link the commands together.
	cfg.Root.AddCommand(cfg.versionCmd)
	cfg.Root.AddCommand(cfg.runCmd)
	cfg.runCmd.AddCommand(cfg.steadyCmd)
	cfg.Root.AddCommand(cfg.gridCmd)
	cfg.Root.AddCommand(cfg.preprocCmd)
	cfg.Root.AddCommand(cfg.srCmd)
	cfg.srCmd.AddCommand(cfg.srStartCmd, cfg.srSaveCmd, cfg.srCleanCmd)
	cfg.Root.AddCommand(cfg.srPredictCmd)
	cfg.Root.AddCommand(cfg.cloudCmd)
	cfg.cloudCmd.AddCommand(cfg.cloudStartCmd, cfg.cloudStatusCmd, cfg.cloudOutputCmd, cfg.cloudDeleteCmd)
	cfg.preprocCmd.AddCommand(cfg.combineCmd)

	// Options are the configuration options available to InMAP.
	options = []struct {
		name, usage, shorthand string
		defaultVal             interface{}
		flagsets               []*pflag.FlagSet
		isInputFile            bool // Does the option represent an input file name?
		isOutputFile           bool // Does the option represent an output file name?
	}{
		{
			name:        "config",
			usage:       `config specifies the configuration file location.`,
			defaultVal:  "",
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.Root.PersistentFlags()},
		},
		{
			name: "static",
			usage: `static specifies whether to run with a static grid that is determined before the simulation starts. If false, the simulation runs with a dynamic grid that changes resolution depending on spatial gradients in population density and concentration.
`,
			shorthand:  "s",
			defaultVal: false,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "creategrid",
			usage: `creategrid specifies whether to create the variable-resolution grid as specified in the configuration file before starting the simulation instead of reading it from a file. If --static is false, then this flag will also be automatically set to false.
`,
			defaultVal: false,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "layers",
			usage: `layers specifies a list of vertical layer numbers to be included in the SR matrix.
`,
			defaultVal: []int{0, 2, 4, 6},
			flagsets:   []*pflag.FlagSet{cfg.srCmd.PersistentFlags()},
		},
		{
			name: "begin",
			usage: `begin specifies the beginning grid index (inclusive) for SR matrix generation.
`,
			defaultVal: 0,
			flagsets:   []*pflag.FlagSet{cfg.srCmd.PersistentFlags()},
		},
		{
			name: "end",
			usage: `end specifies the ending grid index (exclusive) for SR matrix generation. The default is -1 which represents the last row.
`,
			defaultVal: -1,
			flagsets:   []*pflag.FlagSet{cfg.srCmd.PersistentFlags()},
		},
		{
			name: "VarGrid.VariableGridXo",
			usage: `VarGrid.VariableGridXo specifies the X coordinate of the lower-left corner of the InMAP grid.
`,
			defaultVal: -4000.0,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name:       "VarGrid.VariableGridYo",
			usage:      `VarGrid.VariableGridYo specifies the Y coordinate of the lower-left corner of the InMAP grid.`,
			defaultVal: -4000.0,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "VarGrid.VariableGridDx",
			usage: `VarGrid.VariableGridDx specifies the X edge lengths of grid cells in the outermost nest, in the units of the grid model spatial projection--typically meters or degrees latitude and longitude.
`,
			defaultVal: 4000.0,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "VarGrid.VariableGridDy",
			usage: `VarGrid.VariableGridDy specifies the Y edge lengths of grid cells in the outermost nest, in the units of the grid model spatial projection--typically meters or degrees latitude and longitude.
`,
			defaultVal: 4000.0,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name:       "VarGrid.Xnests",
			usage:      `Xnests specifies nesting multiples in the X direction.`,
			defaultVal: []int{2, 2, 2},
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name:       "VarGrid.Ynests",
			usage:      `Ynests specifies nesting multiples in the Y direction.`,
			defaultVal: []int{2, 2, 2},
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name:       "VarGrid.GridProj",
			usage:      `GridProj gives projection info for the CTM grid in Proj4 or WKT format.`,
			defaultVal: "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1",
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags(), cfg.srPredictCmd.Flags()},
		},
		{
			name: "VarGrid.HiResLayers",
			usage: `HiResLayers is the number of layers, starting at ground level, to do nesting in. Layers above this will have all grid cells in the lowest spatial resolution. This option is only used with static grids.
`,
			defaultVal: 1,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "VarGrid.PopDensityThreshold",
			usage: `PopDensityThreshold is a limit for people per unit area in a grid cell in units of people / m². If the population density in a grid cell is above this level, the cell in question is a candidate for splitting into smaller cells. This option is only used with static grids.
`,
			defaultVal: 0.0055,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "VarGrid.PopThreshold",
			usage: `PopThreshold is a limit for the total number of people in a grid cell. If the total population in a grid cell is above this level, the cell in question is a candidate for splitting into smaller cells. This option is only used with static grids.
`,
			defaultVal: 40000.0,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "VarGrid.PopConcThreshold",
			usage: `PopConcThreshold is the limit for Σ(|ΔConcentration|)*combinedVolume*|ΔPopulation| / {Σ(|totalMass|)*totalPopulation}. See the documentation for PopConcMutator for more information. This option is only used with dynamic grids.
`,
			defaultVal: 0.000000001,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "VarGrid.CensusFile",
			usage: `VarGrid.CensusFile is the path to the shapefile or COARDs-compliant NetCDF file holding population information.
`,
			defaultVal:  "${INMAP_ROOT_DIR}/cmd/inmap/testdata/testPopulation.shp",
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "VarGrid.CensusPopColumns",
			usage: `VarGrid.CensusPopColumns is a list of the data fields in CensusFile that should be included as population estimates in the model. They can be population of different demographics or for different population scenarios.
`,
			defaultVal: []string{"TotalPop", "WhiteNoLat", "Black", "Native", "Asian", "Latino"},
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "VarGrid.PopGridColumn",
			usage: `VarGrid.PopGridColumn is the name of the field in CensusFile that contains the data that should be compared to PopThreshold and PopDensityThreshold when determining if a grid cell should be split. It should be one of the fields in CensusPopColumns.
`,
			defaultVal: "TotalPop",
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "VarGrid.MortalityRateFile",
			usage: `VarGrid.MortalityRateFile is the path to the shapefile containing baseline mortality rate data.
`,
			defaultVal:  "${INMAP_ROOT_DIR}/cmd/inmap/testdata/testMortalityRate.shp",
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "VarGrid.MortalityRateColumns",
			usage: `VarGrid.MortalityRateColumns gives names of fields in MortalityRateFile that contain baseline mortality rates (as keys) in units of deaths per year per 100,000 people. The values specify the population group that should be used with each mortality rate for population-weighted averaging.
`,
			defaultVal: map[string]string{
				"AllCause":   "TotalPop",
				"WhNoLMort":  "WhiteNoLat",
				"BlackMort":  "Black",
				"NativeMort": "Native",
				"AsianMort":  "Asian",
				"LatinoMort": "Latino",
			},
			flagsets: []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "InMAPData",
			usage: `InMAPData is the path to location of baseline meteorology and pollutant data. The path can include environment variables.
`,
			defaultVal:  "${INMAP_ROOT_DIR}/cmd/inmap/testdata/testInMAPInputData.ncf",
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.srStartCmd.Flags(), cfg.preprocCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "VariableGridData",
			usage: `VariableGridData is the path to the location of the variable-resolution gridded InMAP data, or the location where it should be created if it doesn't already exist. The path can include environment variables.
`,
			defaultVal:  "${INMAP_ROOT_DIR}/cmd/inmap/testdata/inmapVarGrid.gob",
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags(), cfg.srStartCmd.PersistentFlags()},
		},
		{
			name: "EmissionsShapefiles",
			usage: `EmissionsShapefiles are the paths to any emissions shapefiles. Can be elevated or ground level; elevated files need to have columns labeled "height", "diam", "temp", and "velocity" containing stack information in units of m, m, K, and m/s, respectively. Emissions will be allocated from the geometries in the shape file to the InMAP computational grid, but the mapping projection of the shapefile must be the same as the projection InMAP uses. Can include environment variables.
`,
			defaultVal:  []string{"${INMAP_ROOT_DIR}/cmd/inmap/testdata/testEmis.shp"},
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.srPredictCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name:        "EmissionMaskGeoJSON",
			usage:       `EmissionMaskGeoJSON is an optional file containing a GeoJSON-formatted polygon string that specifies the area outside of which emissions will be ignored. The mask is assumed to  use the same spatial reference as VarGrid.GridProj. Example="{\"type\": \"Polygon\",\"coordinates\": [ [ [-4000, -4000], [4000, -4000], [4000, 4000], [-4000, 4000] ] ] }"`,
			defaultVal:  "",
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.srPredictCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "EmissionUnits",
			usage: `EmissionUnits gives the units that the input emissions are in. Acceptable values are 'tons/year', 'kg/year', 'ug/s', and 'μg/s'.
`,
			defaultVal: "tons/year",
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.srPredictCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "OutputFile",
			usage: `OutputFile is the path to the desired output shapefile location. It can include environment variables.
`,
			defaultVal:   "inmap_output.shp",
			isOutputFile: true,
			flagsets:     []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.srPredictCmd.Flags()},
		},
		{
			name: "LogFile",
			usage: `LogFile is the path to the desired logfile location. It can include environment variables. If LogFile is left blank, the logfile will be saved in the same location as the OutputFile.
`,
			defaultVal:   "",
			isOutputFile: true,
			flagsets:     []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.gridCmd.Flags()},
		},
		{
			name: "OutputAllLayers",
			usage: `If OutputAllLayers is true, output data for all model layers. If false, only output the lowest layer.
`,
			defaultVal: false,
			flagsets:   []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "OutputVariables",
			usage: `OutputVariables specifies which model variables should be included in the output file. It can include environment variables.
`,
			defaultVal: map[string]string{
				"TotalPM25": "PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA",
				"TotalPopD": "(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * AllCause / 100000",
			},
			flagsets: []*pflag.FlagSet{cfg.runCmd.PersistentFlags(), cfg.cloudStartCmd.Flags(), cfg.srPredictCmd.Flags()},
		},
		{
			name: "NumIterations",
			usage: `NumIterations is the number of iterations to calculate. If < 1, convergence is automatically calculated.
`,
			defaultVal: 0,
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name: "aep.InventoryConfig.NEIFiles",
			usage: `NEIFiles lists National Emissions Inventory emissions files. The file names can include environment variables. The format is map[sector name][list of files].
`,
			defaultVal:  map[string][]string{},
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.InventoryConfig.COARDSFiles",
			usage: `COARDSFiles lists COARDS-compliant NetCDF emission files (NetCDF 4 and greater not supported). Information regarding the COARDS NetCDF conventions are available here: https://ferret.pmel.noaa.gov/Ferret/documentation/coards-netcdf-conventions. The file names can include environment variables. The format is map[sector name][list of files]. For COARDS files, the sector name will also be used as the SCC code.
`,
			defaultVal:  map[string][]string{},
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.InventoryConfig.COARDSYear",
			usage: `COARDSYear specifies the year of emissions for COARDS emissions files. COARDS emissions are assumed to be in units of mass of emissions per year. The year will not be used for NEI emissions files.
`,
			defaultVal: 0,
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name:       "aep.InventoryConfig.InputUnits",
			usage:      `InputUnits specifies the units of input data. Acceptable values are 'tons', 'tonnes', 'kg', 'g', and 'lbs'. This value will be used for AEP emissions only, not for shapefiles.`,
			defaultVal: "no_default",
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.SrgSpecSMOKE",
			usage: `SrgSpecSMOKE gives the location of the SMOKE-format surrogate specification file, if any. It is used for assigning spatial locations to emissions records.
`,
			defaultVal:  "",
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.SrgSpecOSM",
			usage: `SrgSpecOSM gives the location of the OSM-format surrogate specification file, if any. It is used for assigning spatial locations to emissions records.
`,
			defaultVal:  "",
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.PostGISURL",
			usage: `PostGISURL specifies the URL to use to connect to a PostGIS database
with the OpenStreetMap data loaded. The URL should be in the format:
postgres://username:password@hostname:port/databasename".

The OpenStreetMap data can be loaded into the database using the
osm2pgsql program, for example with the command:
osm2pgsql -l --hstore-all --hstore-add-index --database=databasename --host=hostname --port=port --username=username --create planet_latest.osm.pbf

The -l and --hstore-all flags for the osm2pgsql command are both necessary,
and the PostGIS database should have the "hstore" extension installed before
loading the data.`,
			defaultVal: "",
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.SrgShapefileDirectory",
			usage: `SrgShapefileDirectory gives the location of the directory holding the shapefiles used for creating spatial surrogates. It is used for assigning spatial locations to emissions records. It is only used when SrgSpecType == "SMOKE".
`,
			defaultVal: "no_default",
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.GridRef",
			usage: `GridRef specifies the locations of the spatial surrogate gridding reference files used for processing emissions. It is used for assigning spatial locations to emissions records.
`,
			defaultVal:  []string{"no_default"},
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.SCCExactMatch",
			usage: `SCCExactMatch specifies whether SCC codes must match exactly when processing emissions.
`,
			defaultVal: true,
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.SpatialConfig.InputSR",
			usage: `InputSR specifies the input emissions spatial reference in Proj4 format.
`,
			defaultVal: "+proj=longlat",
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.SpatialConfig.SpatialCache",
			usage: `SpatialCache specifies the location for storing spatial emissions data for quick access. If this is left empty, no cache will be used.
`,
			defaultVal: "",
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name:       "aep.SpatialConfig.SrgDataCache",
			usage:      `SrgDataCache specifies the location for caching spatial surrogate input data. If it is empty, the input surrogate data will be stored in SpatialCache.`,
			defaultVal: "",
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.SpatialConfig.MaxCacheEntries",
			usage: `MaxCacheEntries specifies the maximum number of emissions and concentrations surrogates to hold in a memory cache. Larger numbers can result in faster processing but increased memory usage.
`,
			defaultVal: 10,
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "aep.SpatialConfig.GridName",
			usage: `GridName specifies a name for the grid which is used in the names of intermediate and output files. Changes to the geometry of the grid must be accompanied by either a a change in GridName or the deletion of all the files in the SpatialCache directory.
`,
			defaultVal: "inmap",
			flagsets:   []*pflag.FlagSet{cfg.steadyCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "SR.OutputFile",
			usage: `SR.OutputFile is the path where the output file is or should be created when creating a source-receptor matrix. It can contain environment variables.
`,
			defaultVal:   "${INMAP_ROOT_DIR}/cmd/inmap/testdata/output_${InMAPRunType}.shp",
			isOutputFile: false,
			isInputFile:  false,
			flagsets:     []*pflag.FlagSet{cfg.srSaveCmd.Flags(), cfg.srPredictCmd.Flags(), cfg.cloudStartCmd.Flags()},
		},
		{
			name: "Preproc.CTMType",
			usage: `Preproc.CTMType specifies what type of chemical transport model we are going to be reading data from. Valid options are "GEOS-Chem" and "WRF-Chem".
`,
			defaultVal: "WRF-Chem",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.WRFChem.WRFOut",
			usage: `Preproc.WRFChem.WRFOut is the location of WRF-Chem output files. [DATE] should be used as a wild card for the simulation date.
`,
			defaultVal: "${INMAP_ROOT_DIR}/cmd/inmap/testdata/preproc/wrfout_d01_[DATE]",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSA1",
			usage: `Preproc.GEOSChem.GEOSA1 is the location of the GEOS 1-hour time average files. [DATE] should be used as a wild card for the simulation date.
`,
			defaultVal: "${INMAP_ROOT_DIR}/cmd/inmap/testdata/preproc/GEOSFP.[DATE].A1.2x25.nc",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSA3Cld",
			usage: `Preproc.GEOSChem.GEOSA3Cld is the location of the GEOS 3-hour average cloud parameter files. [DATE] should be used as a wild card for the simulation date.
`,
			defaultVal: "${INMAP_ROOT_DIR}/cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3cld.2x25.nc",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSA3Dyn",
			usage: `Preproc.GEOSChem.GEOSA3Dyn is the location of the GEOS 3-hour average dynamical parameter files. [DATE] should be used as a wild card for the simulation date.
`,
			defaultVal: "${INMAP_ROOT_DIR}/cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3dyn.2x25.nc",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSI3",
			usage: `Preproc.GEOSChem.GEOSI3 is the location of the GEOS 3-hour instantaneous parameter files. [DATE] should be used as a wild card for the simulation date.
`,
			defaultVal: "${INMAP_ROOT_DIR}/cmd/inmap/testdata/preproc/GEOSFP.[DATE].I3.2x25.nc",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSA3MstE",
			usage: `Preproc.GEOSChem.GEOSA3MstE is the location of the GEOS 3-hour average moist parameters on level edges files. [DATE] should be used as a wild card for the simulation date.
`,
			defaultVal: "${INMAP_ROOT_DIR}/cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3mstE.2x25.nc",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSApBp",
			usage: `Preproc.GEOSChem.GEOSApBp is the location of the constant GEOS pressure level variable file. It is optional; if it is not specified the Ap and Bp information will be extracted from the GEOSChem files.
`,
			defaultVal: "",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.GEOSChem",
			usage: `Preproc.GEOSChem.GEOSChem is the location of GEOS-Chem output files. [DATE] should be used as a wild card for the simulation date.
`,
			defaultVal: "${INMAP_ROOT_DIR}/cmd/inmap/testdata/preproc/gc_output.[DATE].nc",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.ChemFileInterval",
			usage: `Preproc.GEOSChem.ChemFileInterval specifies the time duration represented by each GEOS-Chem output file. E.g. "3h" for 3 hours.
`,
			defaultVal: "3h",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.ChemRecordInterval",
			usage: `Preproc.GEOSChem.ChemRecordInterval specifies the time duration represented by each GEOS-Chem output record. E.g. "3h" for 3 hours.
`,
			defaultVal: "3h",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.NoChemHourIndex",
			usage: `If Preproc.GEOSChem.NoChemHourIndex is true, the GEOS-Chem output files will be assumed to not contain a time dimension.
`,
			defaultVal: false,
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.OlsonLandMap",
			usage: `Preproc.GEOSChem.OlsonLandMap is the location of the GEOS-Chem Olson land use map file, which is described here: http://wiki.seas.harvard.edu/geos-chem/index.php/Olson_land_map.
`,
			defaultVal:  "${INMAP_ROOT_DIR}/cmd/inmap/testdata/preproc/geoschem-new/Olson_2001_Land_Map.025x025.generic.nc",
			isInputFile: true,
			flagsets:    []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.GEOSChem.Dash",
			usage: `Preproc.GEOSChem.Dash indicates whether GEOS-Chem chemical variable names should be assumed to be in the form 'IJ-AVG-S__xxx' vs. the form 'IJ_AVG_S__xxx'.
`,
			defaultVal: false,
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.StartDate",
			usage: `Preproc.StartDate is the date of the beginning of the simulation. Format = "YYYYMMDD".
`,
			defaultVal: "No Default",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.EndDate",
			usage: `Preproc.EndDate is the date of the end of the simulation. Format = "YYYYMMDD".
`,
			defaultVal: "No Default",
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name: "Preproc.CtmGridXo",
			usage: `Preproc.CtmGridXo is the lower left of Chemical Transport Model (CTM) grid, x
`,
			defaultVal: 0.0,
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name:       "Preproc.CtmGridYo",
			usage:      `Preproc.CtmGridYo is the lower left of grid, y`,
			defaultVal: 0.0,
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name:       "Preproc.CtmGridDx",
			usage:      `Preproc.CtmGridDx is the grid cell length in x direction [m]`,
			defaultVal: 1000.0,
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name:       "Preproc.CtmGridDy",
			usage:      `Preproc.CtmGridDy is the grid cell length in y direction [m]`,
			defaultVal: 1000.0,
			flagsets:   []*pflag.FlagSet{cfg.preprocCmd.Flags()},
		},
		{
			name:       "job_name",
			usage:      `job_name specifies the name of a cloud job`,
			defaultVal: "test_job",
			flagsets:   []*pflag.FlagSet{cfg.cloudCmd.PersistentFlags(), cfg.srCmd.PersistentFlags()},
		},
		{
			name:       "addr",
			usage:      `addr specifies the URL to connect to for running cloud jobs`,
			defaultVal: "inmap.run:443",
			flagsets:   []*pflag.FlagSet{cfg.cloudCmd.PersistentFlags(), cfg.srCmd.PersistentFlags()},
		},
		{
			name:       "cmds",
			usage:      `cmds specifies the inmap subcommands to run.`,
			defaultVal: []string{"run", "steady"},
			flagsets:   []*pflag.FlagSet{cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name:       "memory_gb",
			usage:      `memory_gb specifies the gigabytes of RAM memory required for this job.`,
			defaultVal: 20,
			flagsets:   []*pflag.FlagSet{cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name:       "version",
			usage:      `version specifies the version of the InMAP Docker container to use, such as "latest" or "v1.7.2".`,
			defaultVal: "latest",
			flagsets:   []*pflag.FlagSet{cfg.cloudStartCmd.Flags(), cfg.srStartCmd.Flags()},
		},
		{
			name:       "preprocessed_inputs",
			usage:      `preprocessed_inputs is a list of preprocessed input files to be combined.`,
			defaultVal: []string{},
			flagsets:   []*pflag.FlagSet{cfg.combineCmd.Flags()},
		},
		{
			name: "output_file",
			usage: `output_file is the location where the combined output file should be written.
`,
			defaultVal: "inmapdata_combined.ncf",
			flagsets:   []*pflag.FlagSet{cfg.combineCmd.Flags()},
		},
	}

	// Set the prefix for configuration environment variables.
	cfg.SetEnvPrefix("INMAP")

	for _, option := range options {
		if option.isInputFile {
			cfg.inputFiles = append(cfg.inputFiles, option.name)
		}
		if option.isOutputFile {
			cfg.outputFiles = append(cfg.outputFiles, option.name)
		}
		for i, set := range option.flagsets {
			if i != 0 { // We don't want to create the same flag twice.
				set.AddFlag(option.flagsets[0].Lookup(option.name))
				continue
			}
			switch option.defaultVal.(type) {
			case string:
				if option.shorthand == "" {
					set.String(option.name, option.defaultVal.(string), option.usage)
				} else {
					set.StringP(option.name, option.shorthand, option.defaultVal.(string), option.usage)
				}
			case []string:
				if option.shorthand == "" {
					set.StringSlice(option.name, option.defaultVal.([]string), option.usage)
				} else {
					set.StringSliceP(option.name, option.shorthand, option.defaultVal.([]string), option.usage)
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
			case map[string]string, map[string][]string:
				b := bytes.NewBuffer(nil)
				e := json.NewEncoder(b)
				e.Encode(option.defaultVal)
				s := string(b.Bytes())
				if option.shorthand == "" {
					set.String(option.name, s, option.usage)
				} else {
					set.StringP(option.name, option.shorthand, s, option.usage)
				}
			default:
				panic(fmt.Errorf("invalid argument type: %T", option.defaultVal))
			}
			cfg.BindPFlag(option.name, set.Lookup(option.name))
		}
	}
	return cfg
}

func intSliceFromString(s string) ([]int, error) {
	s = strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
	sSlice := strings.Split(s, ",")
	o := make([]int, len(sSlice))
	for i, s := range sSlice {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		o[i] = int(v)
	}
	return o, nil
}

// outChan returns a channel printing to standard output.
func outChan() chan string {
	outChan := make(chan string)
	go func() {
		for {
			msg := <-outChan
			fmt.Printf(msg)
		}
	}()
	return outChan
}

// setConfig finds and reads in the configuration file, if there is one.
func setConfig(cfg *Cfg) error {
	if cfgpath := cfg.GetString("config"); cfgpath != "" {
		cfg.SetConfigFile(cfgpath)
		if err := cfg.ReadInConfig(); err != nil {
			return fmt.Errorf("inmap: problem reading configuration file: %v", err)
		}
	}
	return nil
}

// StartWebServer starts the web server.
func (cfg *Cfg) StartWebServer() {
	setConfig(cfg) // Ignore any errors for now.

	http.HandleFunc("/setConfig", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		configFile := r.Form["config"][0]
		cfg.Root.Flags().Set("config", configFile)
		err := setConfig(cfg)
		if err != nil {
			http.Error(w, err.Error(), 204)
			return
		}
		config := make(map[string]interface{})
		for _, option := range options {
			config[option.name] = cfg.Get(option.name)
		}
		e := json.NewEncoder(w)
		if err := e.Encode(config); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	})

	log.Println("Loading front-end...")

	for _, cmd := range []*cobra.Command{cfg.Root, cfg.versionCmd, cfg.runCmd, cfg.steadyCmd,
		cfg.gridCmd, cfg.preprocCmd, cfg.srCmd, cfg.srPredictCmd} {
		cmd.SilenceUsage = true // We don't want the usage messages in the GUI.
	}

	const address = "localhost:7171"
	const tmpl = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>InMap</title>
	<style>
		html, body {padding: 0; margin: 2% 0; font-family: sans-serif;}
		.container { max-width: 700px; margin: 0 auto; padding: 10px; }
		div[id^="gobra-"] blockquote { border-left: 3px solid #bbb; margin: .3em; color: #333; padding-left: 5px; font-size: 75%; }
		div[id^="gobra-"] code { font-weight: bold; }
		div[id^="gobra-"] input { font-family: monospace; margin-left: .2em; width: 50%; outline:none; }
		.red-border{ border: 1px solid #c35; }
		.green-border{ border: 1px solid #3c5; }
		.blue-border{ border: 1px solid #35c; }
	</style>
</head>
<body>
<div class="container">
	<h1>InMap</h1>
	<p>Configure the simulation below.</p>
	<p>
		Color key: black=default;
		<font color="red">red</font>=error;
		<font color="green">green</font>=value from config file;
		<font color="blue">blue</font>=user entered
	</p>
	<div>
		{{.}}
	</div>
	<footer>
		© 2018 InMAP Authors
	</footer>
</div>

<script>
// If the configuration file is changed, send the new file path
// to the server and update fields

let allFlags = [...document.querySelectorAll('[data-name]')];
allFlags.forEach(x => {
	let inputField = x.children[0];
	inputField.addEventListener("input", e => {
		inputField.classList.remove("green-border");
		inputField.classList.add("blue-border");
	})
})

let configInput = allFlags.filter(x => x.dataset.name == "config")[0].children[0];
configInput.addEventListener("input", e => {
	fetch("http://` + address + `/setConfig?config="+configInput.value)
		.then( res => {
			if (res.status !== 200) {
				if (res.status == 204) {
					configInput.classList.remove("blue-border");
					configInput.classList.remove("green-border");
					configInput.classList.add("red-border");
				} else {
					console.log("Error fetching /setConfig: ", response.text());
				}
			} else {
				res.json().then( data => {
					configInput.classList.remove("red-border");
					for (let key in data)
						for(let f of allFlags)
							if (f.dataset.name == key) {
								let input = f.children[0];
								var newValue = JSON.stringify(data[key]).replace(/^"+|"+$/g,'');
								if (input.value != newValue) {
									input.value = newValue
									input.classList.remove("blue-border");
									input.classList.add("green-border");
								}
							}
				})
			}
		})
		.catch( err => {
			console.log("Error fetching /setConfig", err)
		})
})
let configFileInput = allFlags.filter(x => x.dataset.name == "config")[0].children[1];
configFileInput.addEventListener("change", e => {
	let formData = new FormData();
	let flagName = configFileInput.parentElement.dataset.name;
	formData.append("data", configFileInput.files[0]);

	fetch("/upload", {
		method: "POST",
		body: formData
	})
	.catch(err => {
		alert("Failed uploading: " + err + "\n");
	})
	.then(res => res.json())
	.then(res => {
		configInput.value = res.path;
		configInput.disabled = false;
		configFileInput.value = '';
		var event = document.createEvent('Event');
		event.initEvent('input', true, true);
		configInput.dispatchEvent(event);
	})
	.catch(err => {
		return Promise.reject("Failed processing file: " + err + "\n");
	})
})
</script>
</body>
</html>`

	output := template.Must(template.New("").Parse(tmpl))
	server := gobra.Server{Root: cfg.Root, ServerAddress: address, AllowCORS: false, HTML: output}
	server.MakeFlagUploadable(cfg.inputFiles...)
	log.Println("Server starting... ")
	open.Run("http://" + address)
	fmt.Println("If not opened automatically, please visit http://localhost:7171")
	server.Start()
}
