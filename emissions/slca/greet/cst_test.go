package greet

import (
	"log"
	"math"
	"os"
	"testing"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/BurntSushi/toml"
	"github.com/ctessum/unit"
)

// Set up directory location for configuration file.
func init() {
	os.Setenv("INMAP_ROOT_DIR", "../../../")
}

func initCSTDB() (*DB, *slca.DB) {
	f1, err := os.Open("default.greet")
	if err != nil {
		panic(err)
	}
	f2, err := os.Open("../eieio/data/test_config.toml")
	if err != nil {
		panic(err)
	}
	f3, err := os.Open("scc/GREET to SCC.csv")
	if err != nil {
		panic(err)
	}

	f4, err := os.Open("scc/GREET vehicle SCC.csv")
	if err != nil {
		panic(err)
	}

	f5, err := os.Open("scc/GREET technology SCC.csv")
	if err != nil {
		panic(err)
	}

	lcadb := Load(f1)
	if err = lcadb.AddSCCs(f3, f4, f5); err != nil {
		panic(err)
	}

	cstConfig := new(slca.CSTConfig)
	if _, err := toml.DecodeReader(f2, cstConfig); err != nil {
		panic(err)
	}
	if err := cstConfig.Setup(); err != nil {
		panic(err)
	}

	slcadb := &slca.DB{
		LCADB:     lcadb,
		CSTConfig: cstConfig,
	}

	f1.Close()
	f2.Close()
	f3.Close()
	f4.Close()
	f5.Close()

	return lcadb, slcadb
}

func TestSpatial(t *testing.T) {
	if testing.Short() {
		return
	}
	const (
		tolerance = 1.e-3
		pol       = "PM2.5"
	)
	lcadb, slcadb := initCSTDB()
	var polgas *Gas
	for _, g := range lcadb.Data.Gases {
		if g.Name == pol {
			polgas = g
		}
	}
	for ii, pathway := range lcadb.Data.Pathways {
		if ii > 0 {
			break
		}
		log.Printf("spatial testing %s", pathway.Name)

		pathOutput := pathway.GetMainOutput(lcadb)
		p := pathway.GetOutputProcess(pathOutput.GetResource(lcadb).(*Resource), lcadb)
		outputAmount := p.GetMainOutput(lcadb).(OutputLike).GetAmount(lcadb)
		// The functional unit is 1 of whatever the output units are.
		functionalUnit := unit.New(1, outputAmount.Dimensions())

		wtpResults := slca.SolveGraph(pathway, functionalUnit, &slca.DB{LCADB: lcadb})
		sum := wtpResults.Sum()
		if _, ok := sum.Emissions[polgas]; !ok {
			continue
		}
		resultsSum := sum.Emissions[polgas].Value()

		spatialResults := slca.NewSpatialResults(wtpResults, slcadb)
		emis, err := spatialResults.Emissions()
		handle(err)
		gridSum := emis[polgas].Sum()
		passFail := "Pass"
		if math.Abs(gridSum-resultsSum) > tolerance*math.Abs(resultsSum) {
			passFail = "FAIL"
			t.Fail()
		}
		t.Logf("%s: %s sum of spatial data equals %v and should equal %v",
			passFail, pathway.Name, gridSum, resultsSum)

		conc, err := spatialResults.Concentrations()
		handle(err)
		if conc == nil {
			t.Errorf("FAIL: %s concentration is nil", pathway.Name)
		}
		concSum := conc["PrimaryPM25"].Sum()
		expectedConcSum := 4.6073141635999126e-09
		if math.Abs(concSum-expectedConcSum) > tolerance*math.Abs(expectedConcSum) {
			passFail = "FAIL"
			t.Fail()
		}
		t.Logf("%s: %s sum of concentrations equals %v and should equal %v",
			passFail, pathway.Name, concSum, expectedConcSum)

		health, err := spatialResults.Health("NasariACS")
		handle(err)
		h := health["TotalPop"]["PrimaryPM25"]
		healthSum := h.Sum()
		expectedHealthSum := 2.019143577308587e-09
		if math.Abs(healthSum-expectedHealthSum) > tolerance*math.Abs(expectedHealthSum) {
			passFail = "FAIL"
			t.Fail()
		}
		t.Logf("%s: %s sum of health data equals %v and should equal %v",
			passFail, pathway.Name, healthSum, expectedHealthSum)
	}
}
