package inmap

import (
	"encoding/gob"
	"fmt"
	"io"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
)

func init() {
	gob.Register(geom.Polygon{})
}

// Save returns a function that saves the data in d to a gob file
// (format description at https://golang.org/pkg/encoding/gob/).
func Save(w io.Writer) DomainManipulator {
	return func(d *InMAP) error {
		e := gob.NewEncoder(w)

		if err := e.Encode(d.Cells); err != nil {
			return fmt.Errorf("inmap.InMAP.Save: %v", err)
		}
		return nil
	}
}

// Load returns a function that loads the data from a previously Saved file
// into an InMAP object.
func Load(r io.Reader, config *VarGridConfig, emis *Emissions) DomainManipulator {
	return func(d *InMAP) error {
		dec := gob.NewDecoder(r)
		var cells []*Cell
		if err := dec.Decode(&cells); err != nil {
			return fmt.Errorf("inmap.InMAP.Load: %v", err)
		}
		d.initFromCells(cells, emis, config)
		return nil
	}
}

func (d *InMAP) initFromCells(cells []*Cell, emis *Emissions, config *VarGridConfig) {
	// Create a list of array indices for each population type.
	d.popIndices = make(map[string]int)
	for i, p := range config.CensusPopColumns {
		d.popIndices[p] = i
	}
	d.index = rtree.NewTree(25, 50)
	d.AddCells(cells...)
	d.sort()
	// Add emissions to new cells.
	if emis != nil {
		for _, c := range d.Cells {
			c.setEmissionsFlux(emis) // This needs to be called after setNeighbors.
			if c.Layer > d.nlayers-1 {
				d.nlayers = c.Layer + 1
			}
		}
	}
}
