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
	"fmt"

	"github.com/spf13/cobra"
)

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

	steadyCmd.PersistentFlags().BoolVarP(&dynamic, "dynamic", "d", false,
		"Run with a dynamic grid that changes resolution depending on spatial "+
			"gradients in population density and concentration.")
	steadyCmd.PersistentFlags().BoolVar(&createGrid, "creategrid", false,
		"Create the variable-resolution grid as specified in the configuration file"+
			" before starting the simulation instead of reading it from a file. "+
			"If --dynamic is set to true, then this flag will also be automatically set to true.")

}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the model.",
	Long: "run runs an InMAP simulation. Use the subcommands specified below to " +
		" choose a run mode. (Currently 'steady' is the only avaible run mode.)",
}

// steadyCmd is a command that runs a steady-state simulation.
var steadyCmd = &cobra.Command{
	Use:   "steady",
	Short: "Run InMAP in steady-state mode.",
	Long: "steady runs InMAP in steady-state mode to calculate annual average " +
		"concentrations with no temporal variability.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return labelErr(Run(dynamic, createGrid))
	},
}

func labelErr(err error) error {
	if err != nil {
		return fmt.Errorf("ERROR: %v", err)
	}
	return nil
}
