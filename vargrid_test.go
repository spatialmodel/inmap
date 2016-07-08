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

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	d.testCellAlignment1(t)
}

func (d *InMAP) testCellAlignment1(t *testing.T) {
	for _, c := range d.cells {
		sortCells(c.west)
		sortCells(c.east)
		sortCells(c.north)
		sortCells(c.south)
		sortCells(c.above)
		sortCells(c.below)
		sortCells(c.groundLevel)
	}

	// Cell 0
	if len(d.cells[0].west) != 1 || d.cells[0].west[0] != d.westBoundary[0] {
		t.Error("Incorrect alignment cell 0 West")
	}
	if len(d.cells[0].south) != 1 || d.cells[0].south[0] != d.southBoundary[0] {
		t.Error("Incorrect alignment cell 0 South")
	}
	if len(d.cells[0].north) != 1 || d.cells[0].north[0] != d.cells[1] {
		t.Error("Incorrect alignment cell 0 North")
	}
	if len(d.cells[0].east) != 1 || d.cells[0].east[0] != d.cells[3] {
		t.Error("Incorrect alignment cell 0 East")
	}
	if len(d.cells[0].above) != 1 || d.cells[0].above[0] != d.cells[10] {
		t.Error("Incorrect alignment cell 0 Above")
	}
	if len(d.cells[0].below) != 1 || d.cells[0].below[0] != d.cells[0] {
		t.Error("Incorrect alignment cell 0 Below")
	}
	if len(d.cells[0].groundLevel) != 1 || d.cells[0].groundLevel[0] != d.cells[0] {
		t.Error("Incorrect alignment cell 0 GroundLevel")
	}

	// Cell 1
	if len(d.cells[1].west) != 1 || d.cells[1].west[0] != d.westBoundary[1] {
		t.Error("Incorrect alignment cell 1 West")
	}
	if len(d.cells[1].south) != 1 || d.cells[1].south[0] != d.cells[0] {
		t.Error("Incorrect alignment cell 1 South")
	}
	if len(d.cells[1].north) != 1 || d.cells[1].north[0] != d.cells[2] {
		t.Error("Incorrect alignment cell 1 North")
	}
	if len(d.cells[1].east) != 1 || d.cells[1].east[0] != d.cells[4] {
		t.Error("Incorrect alignment cell 1 East")
	}
	if len(d.cells[1].above) != 1 || d.cells[1].above[0] != d.cells[10] {
		t.Error("Incorrect alignment cell 1 Above")
	}
	if len(d.cells[1].below) != 1 || d.cells[1].below[0] != d.cells[1] {
		t.Error("Incorrect alignment cell 1 Below")
	}
	if len(d.cells[1].groundLevel) != 1 || d.cells[1].groundLevel[0] != d.cells[1] {
		t.Error("Incorrect alignment cell 1 GroundLevel")
	}

	// Cell 2
	if len(d.cells[2].west) != 1 || d.cells[2].west[0] != d.westBoundary[2] {
		t.Error("Incorrect alignment cell 2 West")
	}
	if len(d.cells[2].south) != 2 || d.cells[2].south[0] != d.cells[1] ||
		d.cells[2].south[1] != d.cells[4] {
		t.Error("Incorrect alignment cell 2 South")
	}
	if len(d.cells[2].north) != 1 || d.cells[2].north[0] != d.cells[5] {
		t.Error("Incorrect alignment cell 2 North")
	}
	if len(d.cells[2].east) != 1 || d.cells[2].east[0] != d.cells[7] {
		t.Error("Incorrect alignment cell 2 East")
	}
	if len(d.cells[2].above) != 1 || d.cells[2].above[0] != d.cells[10] {
		t.Error("Incorrect alignment cell 2 Above")
	}
	if len(d.cells[2].below) != 1 || d.cells[2].below[0] != d.cells[2] {
		t.Error("Incorrect alignment cell 2 Below")
	}
	if len(d.cells[2].groundLevel) != 1 || d.cells[2].groundLevel[0] != d.cells[2] {
		t.Error("Incorrect alignment cell 2 GroundLevel")
	}

	// Cell 3
	if len(d.cells[3].west) != 1 || d.cells[3].west[0] != d.cells[0] {
		t.Error("Incorrect alignment cell 3 West")
	}
	if len(d.cells[3].south) != 1 || d.cells[3].south[0] != d.southBoundary[1] {
		t.Error("Incorrect alignment cell 3 South")
	}
	if len(d.cells[3].north) != 1 || d.cells[3].north[0] != d.cells[4] {
		t.Error("Incorrect alignment cell 3 North")
	}
	if len(d.cells[3].east) != 1 || d.cells[3].east[0] != d.cells[6] {
		t.Error("Incorrect alignment cell 3 East")
	}
	if len(d.cells[3].above) != 1 || d.cells[3].above[0] != d.cells[10] {
		t.Error("Incorrect alignment cell 3 Above")
	}
	if len(d.cells[3].below) != 1 || d.cells[3].below[0] != d.cells[3] {
		t.Error("Incorrect alignment cell 3 Below")
	}
	if len(d.cells[3].groundLevel) != 1 || d.cells[3].groundLevel[0] != d.cells[3] {
		t.Error("Incorrect alignment cell 3 GroundLevel")
	}

	// Cell 4
	if len(d.cells[4].west) != 1 || d.cells[4].west[0] != d.cells[1] {
		t.Error("Incorrect alignment cell 4 West")
	}
	if len(d.cells[4].south) != 1 || d.cells[4].south[0] != d.cells[3] {
		t.Error("Incorrect alignment cell 4 South")
	}
	if len(d.cells[4].north) != 1 || d.cells[4].north[0] != d.cells[2] {
		t.Error("Incorrect alignment cell 4 North")
	}
	if len(d.cells[4].east) != 1 || d.cells[4].east[0] != d.cells[6] {
		t.Error("Incorrect alignment cell 4 East")
	}
	if len(d.cells[4].above) != 1 || d.cells[4].above[0] != d.cells[10] {
		t.Error("Incorrect alignment cell 4 Above")
	}
	if len(d.cells[4].below) != 1 || d.cells[4].below[0] != d.cells[4] {
		t.Error("Incorrect alignment cell 4 Below")
	}
	if len(d.cells[4].groundLevel) != 1 || d.cells[4].groundLevel[0] != d.cells[4] {
		t.Error("Incorrect alignment cell 4 GroundLevel")
	}

	// Cell 5
	if len(d.cells[5].west) != 1 || d.cells[5].west[0] != d.westBoundary[3] {
		t.Error("Incorrect alignment cell 5 West")
	}
	if len(d.cells[5].south) != 2 || d.cells[5].south[0] != d.cells[2] ||
		d.cells[5].south[1] != d.cells[7] {
		t.Error("Incorrect alignment cell 5 South")
	}
	if len(d.cells[5].north) != 1 || d.cells[5].north[0] != d.northBoundary[0] {
		t.Error("Incorrect alignment cell 5 North")
	}
	if len(d.cells[5].east) != 1 || d.cells[5].east[0] != d.cells[9] {
		t.Error("Incorrect alignment cell 5 East")
	}
	if len(d.cells[5].above) != 1 || d.cells[5].above[0] != d.cells[11] {
		t.Error("Incorrect alignment cell 5 Above")
	}
	if len(d.cells[5].below) != 1 || d.cells[5].below[0] != d.cells[5] {
		t.Error("Incorrect alignment cell 5 Below")
	}
	if len(d.cells[5].groundLevel) != 1 || d.cells[5].groundLevel[0] != d.cells[5] {
		t.Error("Incorrect alignment cell 5 GroundLevel")
	}

	// Cell 6
	if len(d.cells[6].west) != 2 || d.cells[6].west[0] != d.cells[3] ||
		d.cells[6].west[1] != d.cells[4] {
		t.Error("Incorrect alignment cell 6 West")
	}
	if len(d.cells[6].south) != 1 || d.cells[6].south[0] != d.southBoundary[2] {
		t.Error("Incorrect alignment cell 6 South")
	}
	if len(d.cells[6].north) != 1 || d.cells[6].north[0] != d.cells[7] {
		t.Error("Incorrect alignment cell 6 North")
	}
	if len(d.cells[6].east) != 1 || d.cells[6].east[0] != d.cells[8] {
		t.Error("Incorrect alignment cell 6 East")
	}
	if len(d.cells[6].above) != 1 || d.cells[6].above[0] != d.cells[10] {
		t.Error("Incorrect alignment cell 6 Above")
	}
	if len(d.cells[6].below) != 1 || d.cells[6].below[0] != d.cells[6] {
		t.Error("Incorrect alignment cell 6 Below")
	}
	if len(d.cells[6].groundLevel) != 1 || d.cells[6].groundLevel[0] != d.cells[6] {
		t.Error("Incorrect alignment cell 6 GroundLevel")
	}

	// Cell 7
	if len(d.cells[7].west) != 1 || d.cells[7].west[0] != d.cells[2] {
		t.Error("Incorrect alignment cell 7 West")
	}
	if len(d.cells[7].south) != 1 || d.cells[7].south[0] != d.cells[6] {
		t.Error("Incorrect alignment cell 7 South")
	}
	if len(d.cells[7].north) != 1 || d.cells[7].north[0] != d.cells[5] {
		t.Error("Incorrect alignment cell 7 North")
	}
	if len(d.cells[7].east) != 1 || d.cells[7].east[0] != d.cells[8] {
		t.Error("Incorrect alignment cell 7 East")
	}
	if len(d.cells[7].above) != 1 || d.cells[7].above[0] != d.cells[10] {
		t.Error("Incorrect alignment cell 7 Above")
	}
	if len(d.cells[7].below) != 1 || d.cells[7].below[0] != d.cells[7] {
		t.Error("Incorrect alignment cell 7 Below")
	}
	if len(d.cells[7].groundLevel) != 1 || d.cells[7].groundLevel[0] != d.cells[7] {
		t.Error("Incorrect alignment cell 7 GroundLevel")
	}

	// Cell 8
	if len(d.cells[8].west) != 2 || d.cells[8].west[0] != d.cells[6] ||
		d.cells[8].west[1] != d.cells[7] {
		t.Error("Incorrect alignment cell 8 West")
	}
	if len(d.cells[8].south) != 1 || d.cells[8].south[0] != d.southBoundary[3] {
		t.Error("Incorrect alignment cell 8 South")
	}
	if len(d.cells[8].north) != 1 || d.cells[8].north[0] != d.cells[9] {
		t.Error("Incorrect alignment cell 8 North")
	}
	if len(d.cells[8].east) != 1 || d.cells[8].east[0] != d.eastBoundary[0] {
		t.Error("Incorrect alignment cell 8 East")
	}
	if len(d.cells[8].above) != 1 || d.cells[8].above[0] != d.cells[12] {
		t.Error("Incorrect alignment cell 8 Above")
	}
	if len(d.cells[8].below) != 1 || d.cells[8].below[0] != d.cells[8] {
		t.Error("Incorrect alignment cell 8 Below")
	}
	if len(d.cells[8].groundLevel) != 1 || d.cells[8].groundLevel[0] != d.cells[8] {
		t.Error("Incorrect alignment cell 8 GroundLevel")
	}

	// Cell 9
	if len(d.cells[9].west) != 1 || d.cells[9].west[0] != d.cells[5] {
		t.Error("Incorrect alignment cell 9 West")
	}
	if len(d.cells[9].south) != 1 || d.cells[9].south[0] != d.cells[8] {
		t.Error("Incorrect alignment cell 9 South")
	}
	if len(d.cells[9].north) != 1 || d.cells[9].north[0] != d.northBoundary[1] {
		t.Error("Incorrect alignment cell 9 North")
	}
	if len(d.cells[9].east) != 1 || d.cells[9].east[0] != d.eastBoundary[1] {
		t.Error("Incorrect alignment cell 9 East")
	}
	if len(d.cells[9].above) != 1 || d.cells[9].above[0] != d.cells[13] {
		t.Error("Incorrect alignment cell 9 Above")
	}
	if len(d.cells[9].below) != 1 || d.cells[9].below[0] != d.cells[9] {
		t.Error("Incorrect alignment cell 9 Below")
	}
	if len(d.cells[9].groundLevel) != 1 || d.cells[9].groundLevel[0] != d.cells[9] {
		t.Error("Incorrect alignment cell 0 GroundLevel")
	}

	// Cell 10
	if len(d.cells[10].west) != 1 || d.cells[10].west[0] != d.westBoundary[4] {
		t.Error("Incorrect alignment cell 10 West")
	}
	if len(d.cells[10].south) != 1 || d.cells[10].south[0] != d.southBoundary[4] {
		t.Error("Incorrect alignment cell 10 South")
	}
	if len(d.cells[10].north) != 1 || d.cells[10].north[0] != d.cells[11] {
		t.Error("Incorrect alignment cell 10 North")
	}
	if len(d.cells[10].east) != 1 || d.cells[10].east[0] != d.cells[12] {
		t.Error("Incorrect alignment cell 10 East")
	}
	if len(d.cells[10].above) != 1 || d.cells[10].above[0] != d.cells[14] {
		t.Error("Incorrect alignment cell 10 Above")
	}
	sortCells(d.cells[10].below)
	if len(d.cells[10].below) != 7 || d.cells[10].below[0] != d.cells[0] ||
		d.cells[10].below[1] != d.cells[1] ||
		d.cells[10].below[2] != d.cells[2] ||
		d.cells[10].below[3] != d.cells[3] ||
		d.cells[10].below[4] != d.cells[4] ||
		d.cells[10].below[5] != d.cells[6] ||
		d.cells[10].below[6] != d.cells[7] {
		t.Error("Incorrect alignment cell 10 Below")
	}
	sortCells(d.cells[10].groundLevel)
	if len(d.cells[10].groundLevel) != 7 || d.cells[10].groundLevel[0] != d.cells[0] ||
		d.cells[10].groundLevel[1] != d.cells[1] ||
		d.cells[10].groundLevel[2] != d.cells[2] ||
		d.cells[10].groundLevel[3] != d.cells[3] ||
		d.cells[10].groundLevel[4] != d.cells[4] ||
		d.cells[10].groundLevel[5] != d.cells[6] ||
		d.cells[10].groundLevel[6] != d.cells[7] {
		t.Error("Incorrect alignment cell 10 GroundLevel")
	}

	// Cell 11
	if len(d.cells[11].west) != 1 || d.cells[11].west[0] != d.westBoundary[5] {
		t.Error("Incorrect alignment cell 11 West")
	}
	if len(d.cells[11].south) != 1 || d.cells[11].south[0] != d.cells[10] {
		t.Error("Incorrect alignment cell 11 South")
	}
	if len(d.cells[11].north) != 1 || d.cells[11].north[0] != d.northBoundary[2] {
		t.Error("Incorrect alignment cell 11 North")
	}
	if len(d.cells[11].east) != 1 || d.cells[11].east[0] != d.cells[13] {
		t.Error("Incorrect alignment cell 11 East")
	}
	if len(d.cells[11].above) != 1 || d.cells[11].above[0] != d.cells[15] {
		t.Error("Incorrect alignment cell 11 Above")
	}
	if len(d.cells[11].below) != 1 || d.cells[11].below[0] != d.cells[5] {
		t.Error("Incorrect alignment cell 11 Below")
	}
	if len(d.cells[11].groundLevel) != 1 || d.cells[11].groundLevel[0] != d.cells[5] {
		t.Error("Incorrect alignment cell 11 GroundLevel")
	}

	// Cell 12
	if len(d.cells[12].west) != 1 || d.cells[12].west[0] != d.cells[10] {
		t.Error("Incorrect alignment cell 12 West")
	}
	if len(d.cells[12].south) != 1 || d.cells[12].south[0] != d.southBoundary[5] {
		t.Error("Incorrect alignment cell 12 South")
	}
	if len(d.cells[12].north) != 1 || d.cells[12].north[0] != d.cells[13] {
		t.Error("Incorrect alignment cell 12 North")
	}
	if len(d.cells[12].east) != 1 || d.cells[12].east[0] != d.eastBoundary[2] {
		t.Error("Incorrect alignment cell 12 East")
	}
	if len(d.cells[12].above) != 1 || d.cells[12].above[0] != d.cells[16] {
		t.Error("Incorrect alignment cell 12 Above")
	}
	if len(d.cells[12].below) != 1 || d.cells[12].below[0] != d.cells[8] {
		t.Error("Incorrect alignment cell 12 Below")
	}
	if len(d.cells[12].groundLevel) != 1 || d.cells[12].groundLevel[0] != d.cells[8] {
		t.Error("Incorrect alignment cell 12 GroundLevel")
	}

	// Cell 13
	if len(d.cells[13].west) != 1 || d.cells[13].west[0] != d.cells[11] {
		t.Error("Incorrect alignment cell 13 West")
	}
	if len(d.cells[13].south) != 1 || d.cells[13].south[0] != d.cells[12] {
		t.Error("Incorrect alignment cell 13 South")
	}
	if len(d.cells[13].north) != 1 || d.cells[13].north[0] != d.northBoundary[3] {
		t.Error("Incorrect alignment cell 13 North")
	}
	if len(d.cells[13].east) != 1 || d.cells[13].east[0] != d.eastBoundary[3] {
		t.Error("Incorrect alignment cell 13 East")
	}
	if len(d.cells[13].above) != 1 || d.cells[13].above[0] != d.cells[17] {
		t.Error("Incorrect alignment cell 13 Above")
	}
	if len(d.cells[13].below) != 1 || d.cells[13].below[0] != d.cells[9] {
		t.Error("Incorrect alignment cell 13 Below")
	}
	if len(d.cells[13].groundLevel) != 1 || d.cells[13].groundLevel[0] != d.cells[9] {
		t.Error("Incorrect alignment cell 13 GroundLevel")
	}

	// Skip to the top layer
	// Cell 42
	if len(d.cells[42].west) != 1 || d.cells[42].west[0] != d.westBoundary[20] {
		t.Error("Incorrect alignment cell 42 West")
	}
	if len(d.cells[42].south) != 1 || d.cells[42].south[0] != d.southBoundary[20] {
		t.Error("Incorrect alignment cell 42 South")
	}
	if len(d.cells[42].north) != 1 || d.cells[42].north[0] != d.cells[43] {
		t.Error("Incorrect alignment cell 42 North")
	}
	if len(d.cells[42].east) != 1 || d.cells[42].east[0] != d.cells[44] {
		t.Error("Incorrect alignment cell 42 East")
	}
	if len(d.cells[42].above) != 1 || d.cells[42].above[0] != d.topBoundary[0] {
		t.Error("Incorrect alignment cell 42 Above")
	}
	if len(d.cells[42].below) != 1 || d.cells[42].below[0] != d.cells[38] {
		t.Error("Incorrect alignment cell 42 Below")
	}
	sortCells(d.cells[42].groundLevel)
	if len(d.cells[42].groundLevel) != 7 || d.cells[42].groundLevel[0] != d.cells[0] ||
		d.cells[42].groundLevel[1] != d.cells[1] ||
		d.cells[42].groundLevel[2] != d.cells[2] ||
		d.cells[42].groundLevel[3] != d.cells[3] ||
		d.cells[42].groundLevel[4] != d.cells[4] ||
		d.cells[42].groundLevel[5] != d.cells[6] ||
		d.cells[42].groundLevel[6] != d.cells[7] {
		t.Error("Incorrect alignment cell 42 GroundLevel")
	}

	// Cell 43
	if len(d.cells[43].west) != 1 || d.cells[43].west[0] != d.westBoundary[21] {
		t.Error("Incorrect alignment cell 43 West")
	}
	if len(d.cells[43].south) != 1 || d.cells[43].south[0] != d.cells[42] {
		t.Error("Incorrect alignment cell 43 South")
	}
	if len(d.cells[43].north) != 1 || d.cells[43].north[0] != d.northBoundary[18] {
		t.Error("Incorrect alignment cell 43 North")
	}
	if len(d.cells[43].east) != 1 || d.cells[43].east[0] != d.cells[45] {
		t.Error("Incorrect alignment cell 43 East")
	}
	if len(d.cells[43].above) != 1 || d.cells[43].above[0] != d.topBoundary[1] {
		t.Error("Incorrect alignment cell 43 Above")
	}
	if len(d.cells[43].below) != 1 || d.cells[43].below[0] != d.cells[39] {
		t.Error("Incorrect alignment cell 43 Below")
	}
	if len(d.cells[43].groundLevel) != 1 || d.cells[43].groundLevel[0] != d.cells[5] {
		t.Error("Incorrect alignment cell 43 GroundLevel")
	}

	// Cell 44
	if len(d.cells[44].west) != 1 || d.cells[44].west[0] != d.cells[42] {
		t.Error("Incorrect alignment cell 44 West")
	}
	if len(d.cells[44].south) != 1 || d.cells[44].south[0] != d.southBoundary[21] {
		t.Error("Incorrect alignment cell 44 South")
	}
	if len(d.cells[44].north) != 1 || d.cells[44].north[0] != d.cells[45] {
		t.Error("Incorrect alignment cell 44 North")
	}
	if len(d.cells[44].east) != 1 || d.cells[44].east[0] != d.eastBoundary[18] {
		t.Error("Incorrect alignment cell 44 East")
	}
	if len(d.cells[44].above) != 1 || d.cells[44].above[0] != d.topBoundary[2] {
		t.Error("Incorrect alignment cell 44 Above")
	}
	if len(d.cells[44].below) != 1 || d.cells[44].below[0] != d.cells[40] {
		t.Error("Incorrect alignment cell 44 Below")
	}
	if len(d.cells[44].groundLevel) != 1 || d.cells[44].groundLevel[0] != d.cells[8] {
		t.Error("Incorrect alignment cell 44 GroundLevel")
	}

	// Cell 45
	if len(d.cells[45].west) != 1 || d.cells[45].west[0] != d.cells[43] {
		t.Error("Incorrect alignment cell 45 West")
	}
	if len(d.cells[45].south) != 1 || d.cells[45].south[0] != d.cells[44] {
		t.Error("Incorrect alignment cell 45 South")
	}
	if len(d.cells[45].north) != 1 || d.cells[45].north[0] != d.northBoundary[19] {
		t.Error("Incorrect alignment cell 45 North")
	}
	if len(d.cells[45].east) != 1 || d.cells[45].east[0] != d.eastBoundary[19] {
		t.Error("Incorrect alignment cell 45 East")
	}
	if len(d.cells[45].above) != 1 || d.cells[45].above[0] != d.topBoundary[3] {
		t.Error("Incorrect alignment cell 45 Above")
	}
	if len(d.cells[45].below) != 1 || d.cells[45].below[0] != d.cells[41] {
		t.Error("Incorrect alignment cell 45 Below")
	}
	if len(d.cells[45].groundLevel) != 1 || d.cells[45].groundLevel[0] != d.cells[9] {
		t.Error("Incorrect alignment cell 45 GroundLevel")
	}
}

func TestGetGeometry(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
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

	if len(ctmdata.data) != len(ctmdata2.data) {
		t.Fatalf("new and old ctmdata have different number of variables (%d vs. %d)",
			len(ctmdata2.data), len(ctmdata.data))
	}
	for name, dd1 := range ctmdata.data {
		if _, ok := ctmdata2.data[name]; !ok {
			t.Errorf("ctmdata2 doesn't have variable %s", name)
			continue
		}
		dd2 := ctmdata2.data[name]
		if !reflect.DeepEqual(dd1.dims, dd2.dims) {
			t.Errorf("%s dims problem: %v != %v", name, dd1.dims, dd2.dims)
		}
		if dd1.description != dd2.description {
			t.Errorf("%s description problem: %s != %s", name, dd1.description, dd2.description)
		}
		if dd1.units != dd2.units {
			t.Errorf("%s units problem: %s != %s", name, dd1.units, dd2.units)
		}
		if !reflect.DeepEqual(dd1.data.Shape, dd2.data.Shape) {
			t.Errorf("%s data shape problem: %v != %v", name, dd1.data.Shape, dd2.data.Shape)
		}
		if !reflect.DeepEqual(dd1.data.Elements, dd2.data.Elements) {
			t.Errorf("%s data problem: %v != %v", name, dd1.data.Elements, dd2.data.Elements)
		}
	}

	f.Close()
	os.Remove(TestCTMDataFile)
}
