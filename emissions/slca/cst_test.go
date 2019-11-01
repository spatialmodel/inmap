package slca_test

import (
	"log"
	"math"
	"os"
	"testing"

	"github.com/spatialmodel/inmap/emissions/slca"
	"github.com/spatialmodel/inmap/emissions/slca/greet"

	"github.com/BurntSushi/toml"
	"github.com/ctessum/geom"
	"github.com/ctessum/unit"
)

// Set up directory location for configuration file.
func init() {
	os.Setenv("INMAP_ROOT_DIR", "../../")
}

func initCSTDB() (*greet.DB, *slca.DB) {
	f1, err := os.Open("greet/testdb.greet")
	if err != nil {
		panic(err)
	}
	f2, err := os.Open("testdata/test_config.toml")
	if err != nil {
		panic(err)
	}
	f3, err := os.Open("greet/scc/GREET to SCC.csv")
	if err != nil {
		panic(err)
	}

	f4, err := os.Open("greet/scc/GREET vehicle SCC.csv")
	if err != nil {
		panic(err)
	}

	f5, err := os.Open("greet/scc/GREET technology SCC_test.csv")
	if err != nil {
		panic(err)
	}

	lcadb := greet.Load(f1)
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
	cstConfig.DefaultFIPS = "36119"

	// Transform grid cells so they line up with spatial surrogates.
	cstConfig.SpatialConfig.GridCells = []geom.Polygonal{
		geom.Polygon{{{X: 1.916e+06, Y: 346000}, {X: 1.917e+06, Y: 346000}, {X: 1.917e+06, Y: 347000}, {X: 1.916e+06, Y: 347000}, {X: 1.916e+06, Y: 346000}}},
		geom.Polygon{{{X: 1.916e+06, Y: 347000}, {X: 1.917e+06, Y: 347000}, {X: 1.917e+06, Y: 348000}, {X: 1.916e+06, Y: 348000}, {X: 1.916e+06, Y: 347000}}},
		geom.Polygon{{{X: 1.916e+06, Y: 348000}, {X: 1.918e+06, Y: 348000}, {X: 1.918e+06, Y: 350000}, {X: 1.916e+06, Y: 350000}, {X: 1.916e+06, Y: 348000}}},
		geom.Polygon{{{X: 1.917e+06, Y: 346000}, {X: 1.918e+06, Y: 346000}, {X: 1.918e+06, Y: 347000}, {X: 1.917e+06, Y: 347000}, {X: 1.917e+06, Y: 346000}}},
		geom.Polygon{{{X: 1.917e+06, Y: 347000}, {X: 1.918e+06, Y: 347000}, {X: 1.918e+06, Y: 348000}, {X: 1.917e+06, Y: 348000}, {X: 1.917e+06, Y: 347000}}},
		geom.Polygon{{{X: 1.916e+06, Y: 350000}, {X: 1.92e+06, Y: 350000}, {X: 1.92e+06, Y: 354000}, {X: 1.916e+06, Y: 354000}, {X: 1.916e+06, Y: 350000}}},
		geom.Polygon{{{X: 1.918e+06, Y: 346000}, {X: 1.92e+06, Y: 346000}, {X: 1.92e+06, Y: 348000}, {X: 1.918e+06, Y: 348000}, {X: 1.918e+06, Y: 346000}}},
		geom.Polygon{{{X: 1.918e+06, Y: 348000}, {X: 1.92e+06, Y: 348000}, {X: 1.92e+06, Y: 350000}, {X: 1.918e+06, Y: 350000}, {X: 1.918e+06, Y: 348000}}},
		geom.Polygon{{{X: 1.92e+06, Y: 346000}, {X: 1.924e+06, Y: 346000}, {X: 1.924e+06, Y: 350000}, {X: 1.92e+06, Y: 350000}, {X: 1.92e+06, Y: 346000}}},
		geom.Polygon{{{X: 1.92e+06, Y: 350000}, {X: 1.924e+06, Y: 350000}, {X: 1.924e+06, Y: 354000}, {X: 1.92e+06, Y: 354000}, {X: 1.92e+06, Y: 350000}}},
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
	const (
		tolerance = 1.e-3
		pol       = "PM2.5"
		aqm       = "isrm"
	)
	lcadb, slcadb := initCSTDB()
	var polgas *greet.Gas
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
		p := pathway.GetOutputProcess(pathOutput.GetResource(lcadb).(*greet.Resource), lcadb)
		outputAmount := p.GetMainOutput(lcadb).(greet.OutputLike).GetAmount(lcadb)
		// The functional unit is 1 of whatever the output units are.
		functionalUnit := unit.New(1, outputAmount.Dimensions())

		wtpResults := slca.SolveGraph(pathway, functionalUnit, &slca.DB{LCADB: lcadb})
		sum := wtpResults.Sum()
		if _, ok := sum.Emissions[polgas]; !ok {
			continue
		}
		resultsSum := sum.Emissions[polgas].Value()

		spatialResults := slca.NewSpatialResults(wtpResults, slcadb)
		emis, err := spatialResults.Emissions(aqm)
		if err != nil {
			t.Fatal(err)
		}
		gridSum := emis[polgas].Sum()
		passFail := "Pass"
		if math.Abs(gridSum-resultsSum) > tolerance*math.Abs(resultsSum) {
			passFail = "FAIL"
			t.Fail()
		}
		t.Logf("%s: %s sum of spatial data equals %v and should equal %v",
			passFail, pathway.Name, gridSum, resultsSum)

		conc, err := spatialResults.Concentrations(aqm)
		t.Fatal(err)
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

		health, err := spatialResults.Health("NasariACS", aqm)
		t.Fatal(err)
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
