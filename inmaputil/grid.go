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
)

// Grid creates and saves a new variable resolution grid.
func Grid(cfg *ConfigData) error {
	// Start a function to receive and print log messages.
	msgLog := make(chan string)
	go func() {
		for msg := range msgLog {
			log.Println(msg)
		}
	}()

	ctmData, err := getCTMData(cfg)
	if err != nil {
		return err
	}

	msgLog <- "Loading population and mortality rate data"

	pop, popIndices, mr, err := cfg.VarGrid.LoadPopMort()
	if err != nil {
		return err
	}

	w, err := os.Create(cfg.VariableGridData)
	if err != nil {
		return fmt.Errorf("problem creating file to store variable grid data in: %v", err)
	}

	msgLog <- "Creating grid"

	mutator, err := inmap.PopulationMutator(&cfg.VarGrid, popIndices)
	if err != nil {
		return err
	}
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			cfg.VarGrid.RegularGrid(ctmData, pop, popIndices, mr, nil),
			cfg.VarGrid.MutateGrid(mutator, ctmData, pop, mr, nil, msgLog),
			inmap.Save(w),
		},
	}
	if err := d.Init(); err != nil {
		return err
	}
	msgLog <- fmt.Sprintf("Grid successfully created at %s", cfg.VariableGridData)
	return nil
}
