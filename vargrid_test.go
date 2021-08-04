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
	"flag"
	"io"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
)

func TestVarGridCreate(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr, mortIndices := VarGridTestData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	var m Mech
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	d.TestCellAlignment1(t)
}

func (d *InMAP) TestCellAlignment1(t *testing.T) {
	cells := d.cells.array()

	// Cell 0
	_, inWestBoundary := d.westBoundary.index(cells[0].west.array()[0])
	if len(cells[0].west.array()) != 1 || !inWestBoundary {
		t.Error("Incorrect alignment cell 0 West")
	}
	_, inSouthBoundary := d.southBoundary.index(cells[0].south.array()[0])
	if len(cells[0].south.array()) != 1 || !inSouthBoundary {
		t.Error("Incorrect alignment cell 0 South")
	}
	if len(cells[0].north.array()) != 1 || cells[0].north.array()[0] != cells[1] {
		t.Error("Incorrect alignment cell 0 North")
	}
	if len(cells[0].east.array()) != 1 || cells[0].east.array()[0] != cells[3] {
		t.Error("Incorrect alignment cell 0 East")
	}
	if len(cells[0].above.array()) != 1 || cells[0].above.array()[0] != cells[10] {
		t.Error("Incorrect alignment cell 0 Above")
	}
	if len(cells[0].below.array()) != 1 || cells[0].below.array()[0] != cells[0] {
		t.Error("Incorrect alignment cell 0 Below")
	}
	if len(cells[0].groundLevel.array()) != 1 || cells[0].groundLevel.array()[0] != cells[0] {
		t.Error("Incorrect alignment cell 0 GroundLevel")
	}

	// Cell 1
	_, inWestBoundary = d.westBoundary.index(cells[1].west.array()[0])
	if len(cells[1].west.array()) != 1 || !inWestBoundary {
		t.Error("Incorrect alignment cell 1 West")
	}
	if len(cells[1].south.array()) != 1 || cells[1].south.array()[0] != cells[0] {
		t.Error("Incorrect alignment cell 1 South")
	}
	if len(cells[1].north.array()) != 1 || cells[1].north.array()[0] != cells[2] {
		t.Error("Incorrect alignment cell 1 North")
	}
	if len(cells[1].east.array()) != 1 || cells[1].east.array()[0] != cells[4] {
		t.Error("Incorrect alignment cell 1 East")
	}
	if len(cells[1].above.array()) != 1 || cells[1].above.array()[0] != cells[10] {
		t.Error("Incorrect alignment cell 1 Above")
	}
	if len(cells[1].below.array()) != 1 || cells[1].below.array()[0] != cells[1] {
		t.Error("Incorrect alignment cell 1 Below")
	}
	if len(cells[1].groundLevel.array()) != 1 || cells[1].groundLevel.array()[0] != cells[1] {
		t.Error("Incorrect alignment cell 1 GroundLevel")
	}

	// Cell 2
	_, inWestBoundary = d.westBoundary.index(cells[2].west.array()[0])
	if len(cells[2].west.array()) != 1 || !inWestBoundary {
		t.Error("Incorrect alignment cell 2 West")
	}
	if len(cells[2].south.array()) != 2 || cells[2].south.array()[0] != cells[1] ||
		cells[2].south.array()[1] != cells[4] {
		t.Error("Incorrect alignment cell 2 South")
	}
	if len(cells[2].north.array()) != 1 || cells[2].north.array()[0] != cells[5] {
		t.Error("Incorrect alignment cell 2 North")
	}
	if len(cells[2].east.array()) != 1 || cells[2].east.array()[0] != cells[7] {
		t.Error("Incorrect alignment cell 2 East")
	}
	if len(cells[2].above.array()) != 1 || cells[2].above.array()[0] != cells[10] {
		t.Error("Incorrect alignment cell 2 Above")
	}
	if len(cells[2].below.array()) != 1 || cells[2].below.array()[0] != cells[2] {
		t.Error("Incorrect alignment cell 2 Below")
	}
	if len(cells[2].groundLevel.array()) != 1 || cells[2].groundLevel.array()[0] != cells[2] {
		t.Error("Incorrect alignment cell 2 GroundLevel")
	}

	// Cell 3
	if len(cells[3].west.array()) != 1 || cells[3].west.array()[0] != cells[0] {
		t.Error("Incorrect alignment cell 3 West")
	}
	_, inSouthBoundary = d.southBoundary.index(cells[3].south.array()[0])
	if len(cells[3].south.array()) != 1 || !inSouthBoundary {
		t.Error("Incorrect alignment cell 3 South")
	}
	if len(cells[3].north.array()) != 1 || cells[3].north.array()[0] != cells[4] {
		t.Error("Incorrect alignment cell 3 North")
	}
	if len(cells[3].east.array()) != 1 || cells[3].east.array()[0] != cells[6] {
		t.Error("Incorrect alignment cell 3 East")
	}
	if len(cells[3].above.array()) != 1 || cells[3].above.array()[0] != cells[10] {
		t.Error("Incorrect alignment cell 3 Above")
	}
	if len(cells[3].below.array()) != 1 || cells[3].below.array()[0] != cells[3] {
		t.Error("Incorrect alignment cell 3 Below")
	}
	if len(cells[3].groundLevel.array()) != 1 || cells[3].groundLevel.array()[0] != cells[3] {
		t.Error("Incorrect alignment cell 3 GroundLevel")
	}

	// Cell 4
	if len(cells[4].west.array()) != 1 || cells[4].west.array()[0] != cells[1] {
		t.Error("Incorrect alignment cell 4 West")
	}
	if len(cells[4].south.array()) != 1 || cells[4].south.array()[0] != cells[3] {
		t.Error("Incorrect alignment cell 4 South")
	}
	if len(cells[4].north.array()) != 1 || cells[4].north.array()[0] != cells[2] {
		t.Error("Incorrect alignment cell 4 North")
	}
	if len(cells[4].east.array()) != 1 || cells[4].east.array()[0] != cells[6] {
		t.Error("Incorrect alignment cell 4 East")
	}
	if len(cells[4].above.array()) != 1 || cells[4].above.array()[0] != cells[10] {
		t.Error("Incorrect alignment cell 4 Above")
	}
	if len(cells[4].below.array()) != 1 || cells[4].below.array()[0] != cells[4] {
		t.Error("Incorrect alignment cell 4 Below")
	}
	if len(cells[4].groundLevel.array()) != 1 || cells[4].groundLevel.array()[0] != cells[4] {
		t.Error("Incorrect alignment cell 4 GroundLevel")
	}

	// Cell 5
	_, inWestBoundary = d.westBoundary.index(cells[5].west.array()[0])
	if len(cells[5].west.array()) != 1 || !inWestBoundary {
		t.Error("Incorrect alignment cell 5 West")
	}
	if len(cells[5].south.array()) != 2 || cells[5].south.array()[0] != cells[2] ||
		cells[5].south.array()[1] != cells[7] {
		t.Error("Incorrect alignment cell 5 South")
	}
	_, inNorthBoundary := d.northBoundary.index(cells[5].north.array()[0])
	if len(cells[5].north.array()) != 1 || !inNorthBoundary {
		t.Error("Incorrect alignment cell 5 North")
	}
	if len(cells[5].east.array()) != 1 || cells[5].east.array()[0] != cells[9] {
		t.Error("Incorrect alignment cell 5 East")
	}
	if len(cells[5].above.array()) != 1 || cells[5].above.array()[0] != cells[11] {
		t.Error("Incorrect alignment cell 5 Above")
	}
	if len(cells[5].below.array()) != 1 || cells[5].below.array()[0] != cells[5] {
		t.Error("Incorrect alignment cell 5 Below")
	}
	if len(cells[5].groundLevel.array()) != 1 || cells[5].groundLevel.array()[0] != cells[5] {
		t.Error("Incorrect alignment cell 5 GroundLevel")
	}

	// Cell 6
	if len(cells[6].west.array()) != 2 || cells[6].west.array()[0] != cells[3] ||
		cells[6].west.array()[1] != cells[4] {
		t.Error("Incorrect alignment cell 6 West")
	}
	_, inSouthBoundary = d.southBoundary.index(cells[6].south.array()[0])
	if len(cells[6].south.array()) != 1 || !inSouthBoundary {
		t.Error("Incorrect alignment cell 6 South")
	}
	if len(cells[6].north.array()) != 1 || cells[6].north.array()[0] != cells[7] {
		t.Error("Incorrect alignment cell 6 North")
	}
	if len(cells[6].east.array()) != 1 || cells[6].east.array()[0] != cells[8] {
		t.Error("Incorrect alignment cell 6 East")
	}
	if len(cells[6].above.array()) != 1 || cells[6].above.array()[0] != cells[10] {
		t.Error("Incorrect alignment cell 6 Above")
	}
	if len(cells[6].below.array()) != 1 || cells[6].below.array()[0] != cells[6] {
		t.Error("Incorrect alignment cell 6 Below")
	}
	if len(cells[6].groundLevel.array()) != 1 || cells[6].groundLevel.array()[0] != cells[6] {
		t.Error("Incorrect alignment cell 6 GroundLevel")
	}

	// Cell 7
	if len(cells[7].west.array()) != 1 || cells[7].west.array()[0] != cells[2] {
		t.Error("Incorrect alignment cell 7 West")
	}
	if len(cells[7].south.array()) != 1 || cells[7].south.array()[0] != cells[6] {
		t.Error("Incorrect alignment cell 7 South")
	}
	if len(cells[7].north.array()) != 1 || cells[7].north.array()[0] != cells[5] {
		t.Error("Incorrect alignment cell 7 North")
	}
	if len(cells[7].east.array()) != 1 || cells[7].east.array()[0] != cells[8] {
		t.Error("Incorrect alignment cell 7 East")
	}
	if len(cells[7].above.array()) != 1 || cells[7].above.array()[0] != cells[10] {
		t.Error("Incorrect alignment cell 7 Above")
	}
	if len(cells[7].below.array()) != 1 || cells[7].below.array()[0] != cells[7] {
		t.Error("Incorrect alignment cell 7 Below")
	}
	if len(cells[7].groundLevel.array()) != 1 || cells[7].groundLevel.array()[0] != cells[7] {
		t.Error("Incorrect alignment cell 7 GroundLevel")
	}

	// Cell 8
	if len(cells[8].west.array()) != 2 || cells[8].west.array()[0] != cells[6] ||
		cells[8].west.array()[1] != cells[7] {
		t.Error("Incorrect alignment cell 8 West")
	}
	_, inSouthBoundary = d.southBoundary.index(cells[8].south.array()[0])
	if len(cells[8].south.array()) != 1 || !inSouthBoundary {
		t.Error("Incorrect alignment cell 8 South")
	}
	if len(cells[8].north.array()) != 1 || cells[8].north.array()[0] != cells[9] {
		t.Error("Incorrect alignment cell 8 North")
	}
	_, inEastBoundary := d.eastBoundary.index(cells[8].east.array()[0])
	if len(cells[8].east.array()) != 1 || !inEastBoundary {
		t.Error("Incorrect alignment cell 8 East")
	}
	if len(cells[8].above.array()) != 1 || cells[8].above.array()[0] != cells[12] {
		t.Error("Incorrect alignment cell 8 Above")
	}
	if len(cells[8].below.array()) != 1 || cells[8].below.array()[0] != cells[8] {
		t.Error("Incorrect alignment cell 8 Below")
	}
	if len(cells[8].groundLevel.array()) != 1 || cells[8].groundLevel.array()[0] != cells[8] {
		t.Error("Incorrect alignment cell 8 GroundLevel")
	}

	// Cell 9
	if len(cells[9].west.array()) != 1 || cells[9].west.array()[0] != cells[5] {
		t.Error("Incorrect alignment cell 9 West")
	}
	if len(cells[9].south.array()) != 1 || cells[9].south.array()[0] != cells[8] {
		t.Error("Incorrect alignment cell 9 South")
	}
	_, inNorthBoundary = d.northBoundary.index(cells[9].north.array()[0])
	if len(cells[9].north.array()) != 1 || !inNorthBoundary {
		t.Error("Incorrect alignment cell 9 North")
	}
	_, inEastBoundary = d.eastBoundary.index(cells[9].east.array()[0])
	if len(cells[9].east.array()) != 1 || !inEastBoundary {
		t.Error("Incorrect alignment cell 9 East")
	}
	if len(cells[9].above.array()) != 1 || cells[9].above.array()[0] != cells[13] {
		t.Error("Incorrect alignment cell 9 Above")
	}
	if len(cells[9].below.array()) != 1 || cells[9].below.array()[0] != cells[9] {
		t.Error("Incorrect alignment cell 9 Below")
	}
	if len(cells[9].groundLevel.array()) != 1 || cells[9].groundLevel.array()[0] != cells[9] {
		t.Error("Incorrect alignment cell 0 GroundLevel")
	}

	// Cell 10
	_, inWestBoundary = d.westBoundary.index(cells[10].west.array()[0])
	if len(cells[10].west.array()) != 1 || !inWestBoundary {
		t.Error("Incorrect alignment cell 10 West")
	}
	_, inSouthBoundary = d.southBoundary.index(cells[10].south.array()[0])
	if len(cells[10].south.array()) != 1 || !inSouthBoundary {
		t.Error("Incorrect alignment cell 10 South")
	}
	if len(cells[10].north.array()) != 1 || cells[10].north.array()[0] != cells[11] {
		t.Error("Incorrect alignment cell 10 North")
	}
	if len(cells[10].east.array()) != 1 || cells[10].east.array()[0] != cells[12] {
		t.Error("Incorrect alignment cell 10 East")
	}
	if len(cells[10].above.array()) != 1 || cells[10].above.array()[0] != cells[14] {
		t.Error("Incorrect alignment cell 10 Above")
	}

	if len(cells[10].below.array()) != 7 || cells[10].below.array()[0] != cells[0] ||
		cells[10].below.array()[1] != cells[1] ||
		cells[10].below.array()[2] != cells[2] ||
		cells[10].below.array()[3] != cells[3] ||
		cells[10].below.array()[4] != cells[4] ||
		cells[10].below.array()[5] != cells[6] ||
		cells[10].below.array()[6] != cells[7] {
		t.Error("Incorrect alignment cell 10 Below")
	}

	if len(cells[10].groundLevel.array()) != 7 || cells[10].groundLevel.array()[0] != cells[0] ||
		cells[10].groundLevel.array()[1] != cells[1] ||
		cells[10].groundLevel.array()[2] != cells[2] ||
		cells[10].groundLevel.array()[3] != cells[3] ||
		cells[10].groundLevel.array()[4] != cells[4] ||
		cells[10].groundLevel.array()[5] != cells[6] ||
		cells[10].groundLevel.array()[6] != cells[7] {
		t.Error("Incorrect alignment cell 10 GroundLevel")
	}

	// Cell 11
	_, inWestBoundary = d.westBoundary.index(cells[11].west.array()[0])
	if len(cells[11].west.array()) != 1 || !inWestBoundary {
		t.Error("Incorrect alignment cell 11 West")
	}
	if len(cells[11].south.array()) != 1 || cells[11].south.array()[0] != cells[10] {
		t.Error("Incorrect alignment cell 11 South")
	}
	_, inNorthBoundary = d.northBoundary.index(cells[11].north.array()[0])
	if len(cells[11].north.array()) != 1 || !inNorthBoundary {
		t.Error("Incorrect alignment cell 11 North")
	}
	if len(cells[11].east.array()) != 1 || cells[11].east.array()[0] != cells[13] {
		t.Error("Incorrect alignment cell 11 East")
	}
	if len(cells[11].above.array()) != 1 || cells[11].above.array()[0] != cells[15] {
		t.Error("Incorrect alignment cell 11 Above")
	}
	if len(cells[11].below.array()) != 1 || cells[11].below.array()[0] != cells[5] {
		t.Error("Incorrect alignment cell 11 Below")
	}
	if len(cells[11].groundLevel.array()) != 1 || cells[11].groundLevel.array()[0] != cells[5] {
		t.Error("Incorrect alignment cell 11 GroundLevel")
	}

	// Cell 12
	if len(cells[12].west.array()) != 1 || cells[12].west.array()[0] != cells[10] {
		t.Error("Incorrect alignment cell 12 West")
	}
	_, inSouthBoundary = d.southBoundary.index(cells[12].south.array()[0])
	if len(cells[12].south.array()) != 1 || !inSouthBoundary {
		t.Error("Incorrect alignment cell 12 South")
	}
	if len(cells[12].north.array()) != 1 || cells[12].north.array()[0] != cells[13] {
		t.Error("Incorrect alignment cell 12 North")
	}
	_, inEastBoundary = d.eastBoundary.index(cells[12].east.array()[0])
	if len(cells[12].east.array()) != 1 || !inEastBoundary {
		t.Error("Incorrect alignment cell 12 East")
	}
	if len(cells[12].above.array()) != 1 || cells[12].above.array()[0] != cells[16] {
		t.Error("Incorrect alignment cell 12 Above")
	}
	if len(cells[12].below.array()) != 1 || cells[12].below.array()[0] != cells[8] {
		t.Error("Incorrect alignment cell 12 Below")
	}
	if len(cells[12].groundLevel.array()) != 1 || cells[12].groundLevel.array()[0] != cells[8] {
		t.Error("Incorrect alignment cell 12 GroundLevel")
	}

	// Cell 13
	if len(cells[13].west.array()) != 1 || cells[13].west.array()[0] != cells[11] {
		t.Error("Incorrect alignment cell 13 West")
	}
	if len(cells[13].south.array()) != 1 || cells[13].south.array()[0] != cells[12] {
		t.Error("Incorrect alignment cell 13 South")
	}
	_, inNorthBoundary = d.northBoundary.index(cells[13].north.array()[0])
	if len(cells[13].north.array()) != 1 || !inNorthBoundary {
		t.Error("Incorrect alignment cell 13 North")
	}
	_, inEastBoundary = d.eastBoundary.index(cells[13].east.array()[0])
	if len(cells[13].east.array()) != 1 || !inEastBoundary {
		t.Error("Incorrect alignment cell 13 East")
	}
	if len(cells[13].above.array()) != 1 || cells[13].above.array()[0] != cells[17] {
		t.Error("Incorrect alignment cell 13 Above")
	}
	if len(cells[13].below.array()) != 1 || cells[13].below.array()[0] != cells[9] {
		t.Error("Incorrect alignment cell 13 Below")
	}
	if len(cells[13].groundLevel.array()) != 1 || cells[13].groundLevel.array()[0] != cells[9] {
		t.Error("Incorrect alignment cell 13 GroundLevel")
	}

	// Skip to the top layer
	// Cell 42
	_, inWestBoundary = d.westBoundary.index(cells[42].west.array()[0])
	if len(cells[42].west.array()) != 1 || !inWestBoundary {
		t.Error("Incorrect alignment cell 42 West")
	}
	_, inSouthBoundary = d.southBoundary.index(cells[42].south.array()[0])
	if len(cells[42].south.array()) != 1 || !inSouthBoundary {
		t.Error("Incorrect alignment cell 42 South")
	}
	if len(cells[42].north.array()) != 1 || cells[42].north.array()[0] != cells[43] {
		t.Error("Incorrect alignment cell 42 North")
	}
	if len(cells[42].east.array()) != 1 || cells[42].east.array()[0] != cells[44] {
		t.Error("Incorrect alignment cell 42 East")
	}
	_, inTopBoundary := d.topBoundary.index(cells[42].above.array()[0])
	if len(cells[42].above.array()) != 1 || !inTopBoundary {
		t.Error("Incorrect alignment cell 42 Above")
	}
	if len(cells[42].below.array()) != 1 || cells[42].below.array()[0] != cells[38] {
		t.Error("Incorrect alignment cell 42 Below")
	}
	if len(cells[42].groundLevel.array()) != 7 || cells[42].groundLevel.array()[0] != cells[0] ||
		cells[42].groundLevel.array()[1] != cells[1] ||
		cells[42].groundLevel.array()[2] != cells[2] ||
		cells[42].groundLevel.array()[3] != cells[3] ||
		cells[42].groundLevel.array()[4] != cells[4] ||
		cells[42].groundLevel.array()[5] != cells[6] ||
		cells[42].groundLevel.array()[6] != cells[7] {
		t.Error("Incorrect alignment cell 42 GroundLevel")
	}

	// Cell 43
	_, inWestBoundary = d.westBoundary.index(cells[43].west.array()[0])
	if len(cells[43].west.array()) != 1 || !inWestBoundary {
		t.Error("Incorrect alignment cell 43 West")
	}
	if len(cells[43].south.array()) != 1 || cells[43].south.array()[0] != cells[42] {
		t.Error("Incorrect alignment cell 43 South")
	}
	_, inNorthBoundary = d.northBoundary.index(cells[43].north.array()[0])
	if len(cells[43].north.array()) != 1 || !inNorthBoundary {
		t.Error("Incorrect alignment cell 43 North")
	}
	if len(cells[43].east.array()) != 1 || cells[43].east.array()[0] != cells[45] {
		t.Error("Incorrect alignment cell 43 East")
	}
	_, inTopBoundary = d.topBoundary.index(cells[43].above.array()[0])
	if len(cells[43].above.array()) != 1 || !inTopBoundary {
		t.Error("Incorrect alignment cell 43 Above")
	}
	if len(cells[43].below.array()) != 1 || cells[43].below.array()[0] != cells[39] {
		t.Error("Incorrect alignment cell 43 Below")
	}
	if len(cells[43].groundLevel.array()) != 1 || cells[43].groundLevel.array()[0] != cells[5] {
		t.Error("Incorrect alignment cell 43 GroundLevel")
	}

	// Cell 44
	if len(cells[44].west.array()) != 1 || cells[44].west.array()[0] != cells[42] {
		t.Error("Incorrect alignment cell 44 West")
	}
	_, inSouthBoundary = d.southBoundary.index(cells[44].south.array()[0])
	if len(cells[44].south.array()) != 1 || !inSouthBoundary {
		t.Error("Incorrect alignment cell 44 South")
	}
	if len(cells[44].north.array()) != 1 || cells[44].north.array()[0] != cells[45] {
		t.Error("Incorrect alignment cell 44 North")
	}
	_, inEastBoundary = d.eastBoundary.index(cells[44].east.array()[0])
	if len(cells[44].east.array()) != 1 || !inEastBoundary {
		t.Error("Incorrect alignment cell 44 East")
	}
	_, inTopBoundary = d.topBoundary.index(cells[44].above.array()[0])
	if len(cells[44].above.array()) != 1 || !inTopBoundary {
		t.Error("Incorrect alignment cell 44 Above")
	}
	if len(cells[44].below.array()) != 1 || cells[44].below.array()[0] != cells[40] {
		t.Error("Incorrect alignment cell 44 Below")
	}
	if len(cells[44].groundLevel.array()) != 1 || cells[44].groundLevel.array()[0] != cells[8] {
		t.Error("Incorrect alignment cell 44 GroundLevel")
	}

	// Cell 45
	if len(cells[45].west.array()) != 1 || cells[45].west.array()[0] != cells[43] {
		t.Error("Incorrect alignment cell 45 West")
	}
	if len(cells[45].south.array()) != 1 || cells[45].south.array()[0] != cells[44] {
		t.Error("Incorrect alignment cell 45 South")
	}
	_, inNorthBoundary = d.northBoundary.index(cells[45].north.array()[0])
	if len(cells[45].north.array()) != 1 || !inNorthBoundary {
		t.Error("Incorrect alignment cell 45 North")
	}
	_, inEastBoundary = d.eastBoundary.index(cells[45].east.array()[0])
	if len(cells[45].east.array()) != 1 || !inEastBoundary {
		t.Error("Incorrect alignment cell 45 East")
	}
	_, inTopBoundary = d.topBoundary.index(cells[45].above.array()[0])
	if len(cells[45].above.array()) != 1 || !inTopBoundary {
		t.Error("Incorrect alignment cell 45 Above")
	}
	if len(cells[45].below.array()) != 1 || cells[45].below.array()[0] != cells[41] {
		t.Error("Incorrect alignment cell 45 Below")
	}
	if len(cells[45].groundLevel.array()) != 1 || cells[45].groundLevel.array()[0] != cells[9] {
		t.Error("Incorrect alignment cell 45 GroundLevel")
	}
}

func (d *InMAP) TestCellAlignment2(t *testing.T) {
	const testTolerance = 1.e-8
	for _, cell := range *d.cells {
		var westCoverage, eastCoverage, northCoverage, southCoverage float64
		var aboveCoverage, belowCoverage, groundLevelCoverage float64
		for _, w := range *cell.west {
			westCoverage += w.info.coverFrac
			if !w.boundary {
				pass := false
				for _, e := range *w.east {
					if e.Cell == cell.Cell {
						pass = true
						if different(w.info.diff, e.info.diff, testTolerance) {
							t.Errorf("Kxx doesn't match")
						}
						if different(w.info.centerDistance, e.info.centerDistance, testTolerance) {
							t.Errorf("Dx doesn't match")
							break
						}
					}
				}
				if !pass {
					t.Errorf("Failed for Cell %v West", cell)
				}
			}
		}
		for _, e := range *cell.east {
			eastCoverage += e.info.coverFrac
			if !e.boundary {
				pass := false
				for _, w := range *e.west {
					if w.Cell == cell.Cell {
						pass = true
						if different(e.info.diff, w.info.diff, testTolerance) {
							t.Errorf("Kxx doesn't match")
						}
						if different(e.info.centerDistance, w.info.centerDistance, testTolerance) {
							t.Errorf("Dx doesn't match")
						}
						break
					}
				}
				if !pass {
					t.Errorf("Failed for Cell %v East", cell)
				}
			}
		}
		for _, n := range *cell.north {
			northCoverage += n.info.coverFrac
			if !n.boundary {
				pass := false
				for _, s := range *n.south {
					if s.Cell == cell.Cell {
						pass = true
						if different(n.info.diff, s.info.diff, testTolerance) {
							t.Errorf("Kyy doesn't match")
						}
						if different(n.info.centerDistance, s.info.centerDistance, testTolerance) {
							t.Errorf("Dy doesn't match")
						}
						break
					}
				}
				if !pass {
					t.Errorf("Failed for Cell %v  North", cell)
				}
			}
		}
		for _, s := range *cell.south {
			southCoverage += s.info.coverFrac
			if !s.boundary {
				pass := false
				for _, n := range *s.north {
					if n.Cell == cell.Cell {
						pass = true
						if different(s.info.diff, n.info.diff, testTolerance) {
							t.Errorf("Kyy doesn't match")
						}
						if different(s.info.centerDistance, n.info.centerDistance, testTolerance) {
							t.Errorf("Dy doesn't match")
						}
						break
					}
				}
				if !pass {
					t.Errorf("Failed for Cell %v South", cell)
				}
			}
		}
		for _, a := range *cell.above {
			aboveCoverage += a.info.coverFrac
			if !a.boundary {
				pass := false
				for _, b := range *a.below {
					if b.Cell == cell.Cell {
						pass = true
						if different(a.info.diff, b.info.diff, testTolerance) {
							t.Errorf("Kzz doesn't match above (layer=%v, "+
								"KzzAbove=%v, KzzBelow=%v)", cell.Layer,
								b.info.diff, a.info.diff)
						}
						if different(a.info.centerDistance, b.info.centerDistance, testTolerance) {
							t.Errorf("Dz doesn't match")
						}
						break
					}
				}
				if !pass {
					t.Errorf("Failed for Cell %v Above", cell)
				}
			}
		}
		for _, b := range *cell.below {
			belowCoverage += b.info.coverFrac
			pass := false
			if cell.Layer == 0 && b.Cell == cell.Cell {
				pass = true
			} else {
				for _, a := range *b.above {
					if a.Cell == cell.Cell {
						pass = true
						if different(b.info.diff, a.info.diff, testTolerance) {
							t.Errorf("Kzz doesn't match below")
						}
						if different(b.info.centerDistance, a.info.centerDistance, testTolerance) {
							t.Errorf("Dz doesn't match")
						}
						break
					}
				}
			}
			if !pass {
				t.Errorf("Failed for Cell %v  Below", cell)
			}
		}
		// Assume upper cells are never higher resolution than lower cells
		for _, g := range *cell.groundLevel {
			groundLevelCoverage += g.info.coverFrac
			g2 := g
			pass := false
			for {
				if g2.above.len() == 0 {
					pass = false
					break
				}
				if g2.Cell == (*g2.above)[0].Cell {
					pass = false
					break
				}
				if g2.Cell == cell.Cell {
					pass = true
					break
				}
				g2 = (*g2.above)[0]
			}
			if !pass {
				t.Errorf("Failed for Cell %v GroundLevel", cell)
			}
		}
		const tolerance = 1.0e-10
		if different(westCoverage, 1, tolerance) {
			t.Errorf("cell %v, west coverage %g!=1", cell, westCoverage)
		}
		if different(eastCoverage, 1, tolerance) {
			t.Errorf("cell %v, east coverage %g!=1", cell, eastCoverage)
		}
		if different(southCoverage, 1, tolerance) {
			t.Errorf("cell %v, south coverage %g!=1", cell, southCoverage)
		}
		if different(northCoverage, 1, tolerance) {
			t.Errorf("cell %v, north coverage %g!=1", cell, northCoverage)
		}
		if different(belowCoverage, 1, tolerance) {
			t.Errorf("cell %v, below coverage %g!=1", cell, belowCoverage)
		}
		if different(aboveCoverage, 1, tolerance) {
			t.Errorf("cell %v, above coverage %g!=1", cell, aboveCoverage)
		}
		if different(groundLevelCoverage, 1, tolerance) {
			t.Errorf("cell %v, groundLevel coverage %g!=1", cell, groundLevelCoverage)
		}
	}
}

func TestGetGeometry(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr, mortIndices := VarGridTestData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	var m Mech
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	g0 := d.GetGeometry(0, true)
	g5 := d.GetGeometry(5, true)

	want0 := []geom.Polygonal{geom.Polygon{geom.Path{geom.Point{X: -1.0803243503695702e+07, Y: 4.860686654254725e+06}, geom.Point{X: -1.0801930279663565e+07, Y: 4.860687250907495e+06}, geom.Point{X: -1.0801930791146653e+07, Y: 4.862000560256863e+06}, geom.Point{X: -1.080324418567307e+07, Y: 4.861999963449156e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.080324418567307e+07, Y: 4.861999963449156e+06}, geom.Point{X: -1.0801930791146653e+07, Y: 4.862000560256863e+06}, geom.Point{X: -1.0801931302762568e+07, Y: 4.8633140401226785e+06}, geom.Point{X: -1.0803244867827544e+07, Y: 4.863313443159976e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.0803244867827544e+07, Y: 4.863313443159976e+06}, geom.Point{X: -1.080061773756471e+07, Y: 4.86331446652465e+06}, geom.Point{X: -1.0800618419985117e+07, Y: 4.865941938204324e+06}, geom.Point{X: -1.0803246232668078e+07, Y: 4.865940914307926e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.0801930279663565e+07, Y: 4.860687250907495e+06}, geom.Point{X: -1.0800617055498654e+07, Y: 4.86068767708809e+06}, geom.Point{X: -1.0800617396487407e+07, Y: 4.862000986548126e+06}, geom.Point{X: -1.0801930791146653e+07, Y: 4.862000560256863e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.0801930791146653e+07, Y: 4.862000560256863e+06}, geom.Point{X: -1.0800617396487407e+07, Y: 4.862000986548126e+06}, geom.Point{X: -1.080061773756471e+07, Y: 4.86331446652465e+06}, geom.Point{X: -1.0801931302762568e+07, Y: 4.8633140401226785e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.0803246232668078e+07, Y: 4.865940914307926e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.871199271364988e+06}, geom.Point{X: -1.0803248964477425e+07, Y: 4.87119790475015e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.0800617055498654e+07, Y: 4.86068767708809e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.860688018032593e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.863314807646255e+06}, geom.Point{X: -1.080061773756471e+07, Y: 4.86331446652465e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.080061773756471e+07, Y: 4.86331446652465e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.863314807646255e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06}, geom.Point{X: -1.0800618419985117e+07, Y: 4.865941938204324e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.0797990606947536e+07, Y: 4.860688018032593e+06}, geom.Point{X: -1.0792737710199371e+07, Y: 4.860686654254725e+06}, geom.Point{X: -1.0792734981226994e+07, Y: 4.865940914307926e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06}, geom.Point{X: -1.0792734981226994e+07, Y: 4.865940914307926e+06}, geom.Point{X: -1.0792732249417646e+07, Y: 4.87119790475015e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.871199271364988e+06}}}}
	want5 := []geom.Polygonal{geom.Polygon{geom.Path{geom.Point{X: -1.0803243503695702e+07, Y: 4.860686654254725e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.860688018032593e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06}, geom.Point{X: -1.0803246232668078e+07, Y: 4.865940914307926e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.0803246232668078e+07, Y: 4.865940914307926e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.871199271364988e+06}, geom.Point{X: -1.0803248964477425e+07, Y: 4.87119790475015e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.0797990606947536e+07, Y: 4.860688018032593e+06}, geom.Point{X: -1.0792737710199371e+07, Y: 4.860686654254725e+06}, geom.Point{X: -1.0792734981226994e+07, Y: 4.865940914307926e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06}}}, geom.Polygon{geom.Path{geom.Point{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06}, geom.Point{X: -1.0792734981226994e+07, Y: 4.865940914307926e+06}, geom.Point{X: -1.0792732249417646e+07, Y: 4.87119790475015e+06}, geom.Point{X: -1.0797990606947536e+07, Y: 4.871199271364988e+06}}}}
	if !reflect.DeepEqual(g0, want0) {
		t.Errorf("layer 0 not matching")
	}
	if !reflect.DeepEqual(g5, want5) {
		t.Errorf("layer 5 not matching")
	}
}

func TestReadWriteCTMData(t *testing.T) {
	cfg, ctmdata := CreateTestCTMData()

	f, err := os.Create(TestCTMDataFile)
	if err != nil {
		t.Fatal(err)
	}

	if err = ctmdata.Write(f); err != nil {
		t.Fatal(err)
	}

	f.Close()
	f, err = os.Open(TestCTMDataFile)
	if err != nil {
		t.Fatal(err)
	}

	ctmdata2, err := cfg.LoadCTMData(f)
	if err != nil {
		t.Fatal(err)
	}
	const tolerance = 1.0e-10
	compareCTMData(ctmdata, ctmdata2, tolerance, t)
	f.Close()
	os.Remove(TestCTMDataFile)
}

func TestCombineCTMData(t *testing.T) {
	flag.Parse()

	const (
		tolerance      = 1.0e-6
		outerNestName  = "cmd/inmap/testdata/inmapData_combine_outerNest.ncf"
		innerNestName  = "cmd/inmap/testdata/inmapData_combine_innerNest.ncf"
		goldenFileName = "cmd/inmap/testdata/inmapData_combine_golden.ncf"
	)

	if regenGoldenFiles {
		gc, err := NewGEOSChem(
			"cmd/inmap/testdata/preproc/geoschem-new/MERRA2.[DATE].A1.2x25.nc3",
			"cmd/inmap/testdata/preproc/geoschem-new/MERRA2.[DATE].A3cld.2x25.nc3",
			"cmd/inmap/testdata/preproc/geoschem-new/MERRA2.[DATE].A3dyn.2x25.nc3",
			"cmd/inmap/testdata/preproc/geoschem-new/MERRA2.[DATE].I3.2x25.nc3",
			"cmd/inmap/testdata/preproc/geoschem-new/MERRA2.[DATE].A3mstE.2x25.nc3",
			"cmd/inmap/testdata/preproc/geoschem-new/GEOSFP.ApBp.nc",
			"cmd/inmap/testdata/preproc/geoschem-new/ts.[DATE].nc",
			"cmd/inmap/testdata/preproc/geoschem-new/Olson_2001_Land_Map.025x025.generic.nc",
			"20160102",
			"20160103",
			false,
			"3h",
			"24h",
			false,
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}
		outerNest, err := Preprocess(gc, -2.5, 50, 2.5, 2)
		if err != nil {
			t.Fatal(err)
		}

		innerNest, err := Preprocess(gc, 0, 52, 1.25, 1)
		if err != nil {
			t.Fatal(err)
		}

		if err := regenGoldenFile(outerNest, outerNestName); err != nil {
			t.Errorf("regenerating golden file: %v", err)
		}
		if err := regenGoldenFile(innerNest, innerNestName); err != nil {
			t.Errorf("regenerating golden file: %v", err)
		}

		combined, err := CombineCTMData(outerNest, innerNest)
		if err != nil {
			t.Fatal(err)
		}
		if err := regenGoldenFile(combined, goldenFileName); err != nil {
			t.Errorf("regenerating golden file: %v", err)
		}
	}

	cfg := VarGridConfig{}
	f, err := os.Open(outerNestName)
	if err != nil {
		t.Fatal(err)
	}
	outerNest, err := cfg.LoadCTMData(f)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	f, err = os.Open(innerNestName)
	if err != nil {
		t.Fatal(err)
	}
	innerNest, err := cfg.LoadCTMData(f)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	combined, err := CombineCTMData(outerNest, innerNest)
	if err != nil {
		t.Fatal(err)
	}

	f, err = os.Open(goldenFileName)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	goldenData, err := cfg.LoadCTMData(f)
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}
	compareCTMData(goldenData, combined, tolerance, t)
}

func different(a, b, tolerance float64) bool {
	if 2*math.Abs(a-b)/math.Abs(a+b) > tolerance || math.IsNaN(a) || math.IsNaN(b) {
		return true
	}
	return false
}

func TestLoadPopulationCOARDS(t *testing.T) {
	cfg, _ := CreateTestCTMData()
	cfg.CensusFile = "cmd/inmap/testdata/havana_ppp_2020.ncf"
	cfg.CensusPopColumns = cfg.CensusPopColumns[0:1]
	sr, err := proj.Parse(cfg.GridProj)
	if err != nil {
		t.Fatal(err)
	}
	cfg.VariableGridXo = 1528594
	cfg.VariableGridYo = -1771972
	cfg.VariableGridDx = 1546419 - 1528594
	cfg.VariableGridDy = -1748640 - -1771972
	cfg.Xnests = []int{1}
	cfg.Ynests = []int{1}
	data, index, err := cfg.loadPopulation(sr, cfg.bounds())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(index, map[string]int{"TotalPop": 0}) {
		t.Errorf("invalid index %v", index)
	}
	var popSum float64
	min, max := math.Inf(1), math.Inf(-1)
	popGen := data(cfg.bounds())
	for {
		pop, err := popGen()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		if pop == nil {
			continue
		}
		v := pop.PopData[0]
		popSum += v
		min = math.Min(min, v)
		max = math.Max(max, v)
	}

	const (
		wantMin = 6.7580990791321
		wantMax = 84.202423095703
		wantSum = 351202.19796419144
	)
	if different(min, wantMin, 1.0e-8) {
		t.Errorf("minimum: %g != %g", min, wantMin)
	}
	if different(max, wantMax, 1.0e-8) {
		t.Errorf("maximum: %g != %g", max, wantMax)
	}
	if different(popSum, wantSum, 1.0e-8) {
		t.Errorf("sum: %g != %g", popSum, wantSum)
	}
}
