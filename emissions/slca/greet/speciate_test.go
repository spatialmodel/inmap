package greet

import (
	"log"
	"math"
	"testing"

	"github.com/ctessum/unit"
	"github.com/spatialmodel/inmap/emissions/slca"
)

func TestSpeciate(t *testing.T) {
	if testing.Short() {
		return
	}
	lcadb, slcadb := initCSTDB()

	for ii, pathway := range lcadb.Data.Pathways {
		if ii > 0 {
			break
		}
		log.Printf("speciate testing %s", pathway.Name)

		pathOutput := pathway.GetMainOutput(lcadb)
		p := pathway.GetOutputProcess(pathOutput.GetResource(lcadb).(*Resource), lcadb)
		outputAmount := p.GetMainOutput(lcadb).(OutputLike).GetAmount(lcadb)
		// The functional unit is 1 of whatever the output units are.
		functionalUnit := unit.New(1, outputAmount.Dimensions())

		wtpResults := slca.SolveGraph(pathway, functionalUnit, &slca.DB{LCADB: lcadb})

		specResults, err := slcadb.Speciate(wtpResults)
		if err != nil {
			t.Fatal(err)
		}
		sum := specResults.Sum()
		var foundGas bool
		const want = 7.482365532278292e-12
		for g, v := range sum.Emissions {
			if g.GetName() == "2-hexenes" {
				foundGas = true
				have := v.Value()
				if 2*math.Abs(want-have)/(want+have) > 1.e-3 {
					t.Errorf("want %g, have %g", want, have)
				}
			}
		}
		if !foundGas {
			t.Error("missing pollutant")
		}
	}
}
