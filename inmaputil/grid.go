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
	"log"
	"os"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
)

// Grid creates and saves a new variable resolution grid.
//
// InMAPData is the path to location of baseline meteorology and pollutant data.
// The path can include environment variables.
//
// VariableGridData is the path to the location where the variable-resolution gridded
// InMAP data should be created.
//
// VarGrid provides information for specifying the variable resolution grid.
func Grid(InMAPData, VariableGridData string, VarGrid *inmap.VarGridConfig) error {
	// Start a function to receive and print log messages.
	msgLog := make(chan string)
	go func() {
		for msg := range msgLog {
			log.Println(msg)
		}
	}()

	ctmData, err := getCTMData(InMAPData, VarGrid)
	if err != nil {
		return err
	}

	msgLog <- "Loading population and mortality rate data"

	pop, popIndices, mr, mortIndices, err := VarGrid.LoadPopMort()
	if err != nil {
		return err
	}

	w, err := os.Create(VariableGridData)
	if err != nil {
		return fmt.Errorf("problem creating file to store variable grid data in: %v", err)
	}

	msgLog <- "Creating grid"

	mutator, err := inmap.PopulationMutator(VarGrid, popIndices)
	if err != nil {
		return err
	}
	var m simplechem.Mechanism
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			VarGrid.RegularGrid(ctmData, pop, popIndices, mr, mortIndices, nil, m),
			VarGrid.MutateGrid(mutator, ctmData, pop, mr, nil, m, msgLog),
			inmap.Save(w),
		},
	}
	if err := d.Init(); err != nil {
		return err
	}
	msgLog <- fmt.Sprintf("Grid successfully created at %s", VariableGridData)
	return nil
}
