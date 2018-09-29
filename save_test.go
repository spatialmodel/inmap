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

package inmap_test

import (
	"bytes"
	"testing"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
)

func TestSaveLoad(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})

	cfg, ctmdata, pop, popIndices, mr, mortIndices := inmap.VarGridTestData()
	emis := inmap.NewEmissions()

	mutator, err := inmap.PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	var m simplechem.Mechanism
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
			inmap.Save(buf),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	d2 := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			inmap.Load(buf, cfg, nil, m),
		},
	}
	if err := d2.Init(); err != nil {
		t.Error(err)
	}

	d2.TestCellAlignment1(t)
	d2.TestCellAlignment2(t)
}
