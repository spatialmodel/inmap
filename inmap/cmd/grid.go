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
	"log"
	"os"

	"github.com/spatialmodel/inmap"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(gridCmd)
}

// gridCmd is a command that creates and saves a new variable resolution grid.
var gridCmd = &cobra.Command{
	Use:   "grid",
	Short: "Create a variable resolution grid",
	Long: `grid creates and saves a variable resolution grid as specified by the
	information in the configuration file. The saved data can then be loaded
	for future InMAP simulations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return labelErr(Grid())
	},
	DisableAutoGenTag: true,
}

// Grid creates and saves a new variable resolution grid.
func Grid() error {

	// Start a function to receive and print log messages.
	msgLog := make(chan string)
	go func() {
		for msg := range msgLog {
			log.Println(msg)
		}
	}()

	ctmData, err := getCTMData()
	if err != nil {
		return err
	}

	log.Println("Loading population and mortality rate data")

	pop, popIndices, mr, err := Config.VarGrid.LoadPopMort()
	if err != nil {
		return err
	}

	w, err := os.Create(Config.VariableGridData)
	if err != nil {
		return fmt.Errorf("problem creating file to store variable grid data in: %v", err)
	}

	log.Println("Creating grid")

	mutator, err := inmap.PopulationMutator(&Config.VarGrid, popIndices)
	if err != nil {
		return err
	}
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			Config.VarGrid.RegularGrid(ctmData, pop, popIndices, mr, nil),
			Config.VarGrid.MutateGrid(mutator, ctmData, pop, mr, nil, msgLog),
			inmap.Save(w),
		},
	}
	if err := d.Init(); err != nil {
		return err
	}
	log.Printf("Grid successfully created at %s", Config.VariableGridData)
	return nil
}
