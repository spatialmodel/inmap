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
	"bytes"
	"testing"
)

func TestSaveLoad(t *testing.T) {

	buf := bytes.NewBuffer([]byte{})

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := NewEmissions()

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
			cfg.MutateGrid(PopulationMutator(cfg, popIndices), ctmdata, pop, mr, emis),
			Save(buf),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	d2 := &InMAP{
		InitFuncs: []DomainManipulator{
			Load(buf, cfg, nil),
		},
	}
	if err := d2.Init(); err != nil {
		t.Error(err)
	}

	d2.testCellAlignment1(t)
	d2.testCellAlignment2(t)
}
