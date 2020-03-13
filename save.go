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

package inmap

import (
	"encoding/gob"
	"fmt"
	"io"
	"sort"

	"github.com/ctessum/geom"
)

func init() {
	gob.Register(geom.Polygon{})
	gob.Register(geom.MultiPolygon{})
	gob.Register(&geom.Bounds{})
	gob.Register(geom.Point{})
	gob.Register(geom.MultiPoint{})
	gob.Register(geom.LineString{})
	gob.Register(geom.MultiLineString{})
}

type versionCells struct {
	// DataVersion holds the variable grid data version of the software
	// that saved this data, if any, and should match the VarGridDataVersion
	// global variable.
	DataVersion string
	Cells       []*Cell
}

// Save returns a function that saves the data in d to a gob file
// (format description at https://golang.org/pkg/encoding/gob/).
func Save(w io.Writer) DomainManipulator {
	return func(d *InMAP) error {

		if d.cells.len() == 0 {
			return fmt.Errorf("inmap.InMAP.Save: no grid cells to save")
		}

		// Set the data version so it can be checked when the data is loaded.
		data := versionCells{
			DataVersion: VarGridDataVersion,
			Cells:       d.cells.array(),
		}

		e := gob.NewEncoder(w)

		if err := e.Encode(data); err != nil {
			return fmt.Errorf("inmap.InMAP.Save: %v", err)
		}
		return nil
	}
}

// Load returns a function that loads the data from a previously Saved file
// into an InMAP object.
func Load(r io.Reader, config *VarGridConfig, emis *Emissions, m Mechanism) DomainManipulator {
	return func(d *InMAP) error {
		dec := gob.NewDecoder(r)
		var data versionCells
		if err := dec.Decode(&data); err != nil {
			return fmt.Errorf("inmap.InMAP.Load: %v", err)
		}
		if err := d.initFromCells(data.Cells, emis, config, m); err != nil {
			return err
		}
		if data.DataVersion != VarGridDataVersion {
			return fmt.Errorf("InMAP variable grid data version %s is not compatible with "+
				"the required version %s", data.DataVersion, VarGridDataVersion)
		}
		return nil
	}
}

func (d *InMAP) initFromCells(cells []*Cell, emis *Emissions, config *VarGridConfig, m Mechanism) error {
	d.init()
	// Create a list of array indices for each population type.
	d.PopIndices = make(map[string]int)
	for i, p := range config.CensusPopColumns {
		d.PopIndices[p] = i
	}
	d.mortIndices = make(map[string]int)
	mortRateColumns := make([]string, len(config.MortalityRateColumns))
	i := 0
	for m := range config.MortalityRateColumns {
		mortRateColumns[i] = m
		i++
	}
	sort.Strings(mortRateColumns)
	for i, m := range mortRateColumns {
		d.mortIndices[m] = i
	}
	for _, c := range cells {
		d.InsertCell(c, m)
	}

	// Add emissions to new cells.
	// This needs to be called after setNeighbors.
	if err := d.SetEmissionsFlux(emis, m); err != nil {
		return err
	}
	return nil
}
