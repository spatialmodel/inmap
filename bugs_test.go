package inmap

import (
	"os"
	"testing"

	"github.com/ctessum/geom/index/rtree"
)

func TestGridBug1(t *testing.T) {
	cfg, _, _, _, _, _ := VarGridTestData()
	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}

	cfg.VariableGridXo = -5.0
	cfg.VariableGridYo = -6.0
	cfg.VariableGridDx = 1.0
	cfg.VariableGridDy = 1.0
	cfg.Xnests = []int{10, 3, 2, 3, 2, 2, 2, 2}
	cfg.Ynests = []int{12, 3, 2, 3, 2, 2, 2, 2}
	cfg.GridProj = "+proj=longlat +units=degrees"
	cfg.PopDensityThreshold = 67765500.0
	cfg.PopThreshold = 10000.0
	cfg.CensusFile = "cmd/inmap/testdata/bug_data/grid_bug_1/pop.shp"
	cfg.CensusPopColumns = []string{"TotalPop"}
	cfg.PopGridColumn = "TotalPop"
	cfg.MortalityRateFile = "cmd/inmap/testdata/testMortalityRate.shp"

	pop, popIndices, mr, mortIndices, err := cfg.LoadPopMort()
	if err != nil {
		t.Fatal(err)
	}

	r, err := os.Open("cmd/inmap/testdata/bug_data/grid_bug_1/test.ncf")
	if err != nil {
		t.Fatal(err)
	}
	ctmdata, err := cfg.LoadCTMData(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()

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
}
