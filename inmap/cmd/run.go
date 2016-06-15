package cmd

import "github.com/spf13/cobra"

var (
	// dynamic specifies whether the simulation should be run with a dynamic
	// (vs. static) resolution grid.
	dynamic bool

	// createGrid specifies whether the variable-resolution grid should be
	// created on-the-fly for static runs rather than reading it from a file.
	// For dynamic gridding, the grid is always created on-the-fly.
	createGrid bool
)

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.AddCommand(steadyCmd)

	runCmd.PersistentFlags().BoolVarP(&dynamic, "dynamic", "d", false,
		"Run with a dynamic grid that changes resolution depending on spatial "+
			"gradients in population density and concentration.")
	runCmd.PersistentFlags().BoolVar(&createGrid, "creategrid", false,
		"Create the variable-resolution grid as specified in the configuration file"+
			"before starting the simulation instead of reading it from a file. "+
			"If --dynamic is set to true, then this flag will also be automatically set to true.")

}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the model",
	Long:  "Run InMAP. Subcommands specify the run mode.",
}

// steadyCmd is a command that runs a steady-state simulation.
var steadyCmd = &cobra.Command{
	Use:   "steady",
	Short: "Run InMAP in steady-state mode.",
	Long: "Run InMAP in steady-state mode to calculate annual average " +
		"concentrations with no temporal variability.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return Run(dynamic, createGrid)
	},
}
