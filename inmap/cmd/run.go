package cmd

import "github.com/spf13/cobra"

var (
	// static specifies whether the simulation should be run with a static
	// (vs. dynamic) resolution grid.
	dynamic bool
)

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.AddCommand(SteadyCmd)

	runCmd.PersistentFlags().BoolVarP(&dynamic, "dynamic", "s", false,
		"Run with a dynamic grid that changes resolution depending on spatial "+
			"gradients in population density and concentration.")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the model",
	Long:  "Run InMAP. Subcommands specify the run mode.",
}

// SteadyCmd is a command that runs a steady-state simulation.
var SteadyCmd = &cobra.Command{
	Use:   "steady",
	Short: "Run InMAP in steady-state mode.",
	Long: "Run InMAP in steady-state mode to calculate annual average " +
		"concentrations with no temporal variability.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return Run(dynamic)
	},
}
