package inmap

import (
	"bytes"
	"testing"

	"github.com/ctessum/geom/index/rtree"
)

func TestSaveLoad(t *testing.T) {

	buf := bytes.NewBuffer([]byte{})

	cfg, ctmdata, pop, popIndices, mr := VarGridData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

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
