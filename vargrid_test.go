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
	"os"
	"reflect"
	"testing"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
)

func TestVarGridCreate(t *testing.T) {

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	d.testCellAlignment1(t)
}

func (d *InMAP) testCellAlignment1(t *testing.T) {
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

func TestGetGeometry(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	mutator, err := PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, nil),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	g0 := d.GetGeometry(0, true)
	g5 := d.GetGeometry(5, true)

	want0 := []geom.Polygonal{
		geom.Polygon{[]geom.Point{
			{X: -1.0803243503695702e+07, Y: 4.860686654254725e+06},
			{X: -1.0801930279663565e+07, Y: 4.860687250907495e+06},
			{X: -1.0801930791146653e+07, Y: 4.862000560256863e+06},
			{X: -1.080324418567307e+07, Y: 4.861999963449156e+06},
			{X: -1.0803243503695702e+07, Y: 4.860686654254725e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.080324418567307e+07, Y: 4.861999963449156e+06},
			{X: -1.0801930791146653e+07, Y: 4.862000560256863e+06},
			{X: -1.0801931302762568e+07, Y: 4.8633140401226785e+06},
			{X: -1.0803244867827544e+07, Y: 4.863313443159976e+06},
			{X: -1.080324418567307e+07, Y: 4.861999963449156e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.0803244867827544e+07, Y: 4.863313443159976e+06},
			{X: -1.080061773756471e+07, Y: 4.86331446652465e+06},
			{X: -1.0800618419985117e+07, Y: 4.865941938204324e+06},
			{X: -1.0803246232668078e+07, Y: 4.865940914307926e+06},
			{X: -1.0803244867827544e+07, Y: 4.863313443159976e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.0801930279663565e+07, Y: 4.860687250907495e+06},
			{X: -1.0800617055498654e+07, Y: 4.86068767708809e+06},
			{X: -1.0800617396487407e+07, Y: 4.862000986548126e+06},
			{X: -1.0801930791146653e+07, Y: 4.862000560256863e+06},
			{X: -1.0801930279663565e+07, Y: 4.860687250907495e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.0801930791146653e+07, Y: 4.862000560256863e+06},
			{X: -1.0800617396487407e+07, Y: 4.862000986548126e+06},
			{X: -1.080061773756471e+07, Y: 4.86331446652465e+06},
			{X: -1.0801931302762568e+07, Y: 4.8633140401226785e+06},
			{X: -1.0801930791146653e+07, Y: 4.862000560256863e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.0803246232668078e+07, Y: 4.865940914307926e+06},
			{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06},
			{X: -1.0797990606947536e+07, Y: 4.871199271364988e+06},
			{X: -1.0803248964477425e+07, Y: 4.87119790475015e+06},
			{X: -1.0803246232668078e+07, Y: 4.865940914307926e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.0800617055498654e+07, Y: 4.86068767708809e+06},
			{X: -1.0797990606947536e+07, Y: 4.860688018032593e+06},
			{X: -1.0797990606947536e+07, Y: 4.863314807646255e+06},
			{X: -1.080061773756471e+07, Y: 4.86331446652465e+06},
			{X: -1.0800617055498654e+07, Y: 4.86068767708809e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.080061773756471e+07, Y: 4.86331446652465e+06},
			{X: -1.0797990606947536e+07, Y: 4.863314807646255e+06},
			{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06},
			{X: -1.0800618419985117e+07, Y: 4.865941938204324e+06},
			{X: -1.080061773756471e+07, Y: 4.86331446652465e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.0797990606947536e+07, Y: 4.860688018032593e+06},
			{X: -1.0792737710199371e+07, Y: 4.860686654254725e+06},
			{X: -1.0792734981226994e+07, Y: 4.865940914307926e+06},
			{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06},
			{X: -1.0797990606947536e+07, Y: 4.860688018032593e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06},
			{X: -1.0792734981226994e+07, Y: 4.865940914307926e+06},
			{X: -1.0792732249417646e+07, Y: 4.87119790475015e+06},
			{X: -1.0797990606947536e+07, Y: 4.871199271364988e+06},
			{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06}},
		},
	}
	want5 := []geom.Polygonal{
		geom.Polygon{[]geom.Point{
			{X: -1.0803243503695702e+07, Y: 4.860686654254725e+06},
			{X: -1.0797990606947536e+07, Y: 4.860688018032593e+06},
			{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06},
			{X: -1.0803246232668078e+07, Y: 4.865940914307926e+06},
			{X: -1.0803243503695702e+07, Y: 4.860686654254725e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.0803246232668078e+07, Y: 4.865940914307926e+06},
			{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06},
			{X: -1.0797990606947536e+07, Y: 4.871199271364988e+06},
			{X: -1.0803248964477425e+07, Y: 4.87119790475015e+06},
			{X: -1.0803246232668078e+07, Y: 4.865940914307926e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.0797990606947536e+07, Y: 4.860688018032593e+06},
			{X: -1.0792737710199371e+07, Y: 4.860686654254725e+06},
			{X: -1.0792734981226994e+07, Y: 4.865940914307926e+06},
			{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06},
			{X: -1.0797990606947536e+07, Y: 4.860688018032593e+06}},
		},
		geom.Polygon{[]geom.Point{
			{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06},
			{X: -1.0792734981226994e+07, Y: 4.865940914307926e+06},
			{X: -1.0792732249417646e+07, Y: 4.87119790475015e+06},
			{X: -1.0797990606947536e+07, Y: 4.871199271364988e+06},
			{X: -1.0797990606947536e+07, Y: 4.865942279503172e+06}},
		},
	}
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

	if err = ctmdata.Write(f, cfg.ctmGridXo, cfg.ctmGridYo, cfg.ctmGridDx, cfg.ctmGridDy); err != nil {
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
	compareCTMData(ctmdata, ctmdata2, t)
	f.Close()
	os.Remove(TestCTMDataFile)
}

func compareCTMData(ctmdata, ctmdata2 *CTMData, t *testing.T) {
	if len(ctmdata.Data) != len(ctmdata2.Data) {
		t.Fatalf("new and old ctmdata have different number of variables (%d vs. %d)",
			len(ctmdata2.Data), len(ctmdata.Data))
	}
	for name, dd1 := range ctmdata.Data {
		if _, ok := ctmdata2.Data[name]; !ok {
			t.Errorf("ctmdata2 doesn't have variable %s", name)
			continue
		}
		dd2 := ctmdata2.Data[name]
		if !reflect.DeepEqual(dd1.Dims, dd2.Dims) {
			t.Errorf("%s dims problem: %v != %v", name, dd1.Dims, dd2.Dims)
		}
		if dd1.Description != dd2.Description {
			t.Errorf("%s description problem: %s != %s", name, dd1.Description, dd2.Description)
		}
		if dd1.Units != dd2.Units {
			t.Errorf("%s units problem: %s != %s", name, dd1.Units, dd2.Units)
		}
		if !reflect.DeepEqual(dd1.Data.Shape, dd2.Data.Shape) {
			t.Errorf("%s data shape problem: %v != %v", name, dd1.Data.Shape, dd2.Data.Shape)
		}
		if !reflect.DeepEqual(dd1.Data.Elements, dd2.Data.Elements) {
			t.Errorf("%s data problem: %v != %v", name, dd1.Data.Elements, dd2.Data.Elements)
		}
	}
}
