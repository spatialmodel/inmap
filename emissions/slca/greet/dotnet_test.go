package greet

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/unit"
)

var (
	greetTests []greetTest
)

func initDB() *DB {
	runtime.GOMAXPROCS(runtime.NumCPU())

	f1, err := os.Open("default.greet")
	//f1, err := os.Open("C:/Users/Chris/Documents/Greet/Data/defaultxxx.greet")
	if err != nil {
		panic(err)
	}
	db := Load(f1)
	f1.Close()

	DebugLevel = 0

	var greetTestss map[string][]greetTest
	f, err := os.Open("datafor_test.json")
	handle(err)
	d := json.NewDecoder(f)
	handle(d.Decode(&greetTestss))
	greetTests = greetTestss["greetTests"]

	return db
}

func convertUnits(units string) string {
	var unitConversions = map[string]string{
		"joule":     "kg m^2 s^-2",
		"joules":    "kg m^2 s^-2",
		"J":         "kg m^2 s^-2",
		"kilogram":  "kg",
		"kilograms": "kg",
		"kg":        "kg",
		"cu_meters": "m^3",
		"m^3":       "m^3",
	}
	if out, ok := unitConversions[units]; ok {
		return out
	}
	return units
}

type greetTest struct {
	Pathway, Process                                   string
	OutputID                                           slca.OutputID
	WTPEmis, WTPResources, OnSiteEmis, OnSiteResources map[string]value
	FunctionalUnit                                     string
	I                                                  string
}

type value struct {
	Val   float64
	Units string
}

const mainOutput slca.OutputID = "dddddddd-dddd-dddd-dddd-dddddddddddd"

func (gt greetTest) calcWTP(db *DB) (results *slca.OnsiteResultsNoSubprocess, run bool) {
	if gt.OutputID != mainOutput {
		// Only run test for main output of pathway.
		return nil, false
	}
	pathway, err := db.GetPathwayMixOrVehicleFromName(gt.Pathway)
	handle(err)
	amount := getFunctionalUnit(gt.FunctionalUnit)
	wtpResults := slca.SolveGraph(pathway, amount, &slca.DB{LCADB: db})
	//f, err := os.Create("corn_emissionsOnly.csv")
	//handle(err)
	//w := csv.NewWriter(f)
	//w.WriteAll(wtpResults.Table())
	//w.Flush()
	//f.Close()
	results = wtpResults.Sum()
	return results, true
}

func (gt greetTest) calcOnsite(db *DB) (results *slca.OnsiteResultsNoSubprocess, run bool) {
	pathway, err := db.GetPathwayMixOrVehicleFromName(gt.Pathway)
	handle(err)
	amount := getFunctionalUnit(gt.FunctionalUnit)
	switch pathway.(type) {
	case *Pathway:
	case *Mix:
		return nil, false // no onsite emissions from a mix.
	default:
		panic("unknown type")
	}
	path := pathway.(*Pathway)

	var p slca.Process
	var output slca.Output
	if gt.OutputID == mainOutput {
		o := path.GetMainOutput(db)
		v := path.VertexForMainOutput()
		p, _ = v.GetProcess(path, o.GetResource(db).(*Resource), db)
		output = p.GetOutput(o.GetResource(db).(*Resource), db)
	} else {
		foundOutput := false
		for _, proc := range db.Data.StationaryProcesses {
			for _, o := range proc.Outputs {
				if o.GetID() == gt.OutputID {
					output = o
					p = proc
					foundOutput = true
				}
			}
			if proc.Coproducts != nil {
				for _, cp := range proc.Coproducts.Coprods {
					if cp.GetID() == gt.OutputID {
						output = cp
						p = proc
						foundOutput = true
					}
				}
			}
		}
		for _, proc := range db.Data.TransportationProcesses {
			for _, o := range proc.Outputs {
				if o.GetID() == gt.OutputID {
					output = o
					p = proc
					foundOutput = true
				}
			}
		}
		if !foundOutput {
			panic("Couldn't find output")
		}
	}

	switch p.(type) {
	case *TransportationProcess:
		//continue
	case *StationaryProcess:
		//continue
		pp := p.(*StationaryProcess)
		if pp.Coproducts != nil {
			displacement := false
			for _, cp := range pp.Coproducts.Coprods {
				if cp.Method == "displacement" {
					// GREET.net considers displaced emissions as onsite emissions,
					// which we do not, so we can't make a valid comparison.
					displacement = true
				}
			}
			if displacement {
				return nil, false
			}
		}
	}
	resource := output.GetResource(db)
	amount = resource.ConvertToDefaultUnits(amount, db)
	res := p.OnsiteResults(pathway.(*Pathway), output, db).ScaleCopy(amount)
	// Since this is the final stage, add in the end use of the product
	// to match the GREET.net calculation.
	if !output.(OutputLike).IsCoproduct() {
		// Only do this if the output is not a coproduct.
		// TODO: This is necessary to match the GREET.net results, but I'm
		// not sure why.
		res.AddResource(noSubprocess, resource, amount, db)
	}
	return res.FlattenSubprocess(), true
}

func getFunctionalUnit(fUnit string) *unit.Unit {
	var amount *unit.Unit
	switch fUnit {
	case "kilograms", "kg":
		amount = unit.New(1, unit.Kilogram)
	case "joules", "J":
		amount = unit.New(1., unit.Dimensions{
			unit.MassDim:   1,
			unit.LengthDim: 2,
			unit.TimeDim:   -2})
	case "cu_meters", "m^3":
		amount = unit.New(1., unit.Dimensions{
			unit.LengthDim: 3})
	case "$":
		amount = unit.New(1, dollars)
	default:
		panic("Unknown unit: " + fUnit)
	}
	return amount
}

func TestDotNetOnsite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	dotNetTest("onsite", 0.001, t)
}

func TestDotNetWTP(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	dotNetTest("WTP", 0.01, t)
}

const showPassedTests = false

// Passed 27726 out of 29956 tests (92.6%)

func dotNetTest(testType string, tolerance float64, t *testing.T) {
	db := initDB()
	var testsTotal, testsPassed int
	for _, data := range greetTests[0:10] {
		if data.Process == "RNG Production from Landfill Gas" {
			// This is a strange process that has a lot of inputs and an output that
			// sum to zero resource use, causing the two models to disagree owing
			// to floating point math errors. Therefore, we ignore it.
			continue
		}
		log.Printf("Testing %s: %s; %s; %s", data.I, data.Pathway,
			data.Process, testType)
		var results *slca.OnsiteResultsNoSubprocess
		var run bool // Whether to run the test.
		switch testType {
		case "WTP":
			results, run = data.calcWTP(db)
		case "onsite":
			results, run = data.calcOnsite(db)
		default:
			panic("Invalid test type " + testType)
		}
		if !run {
			continue
		}

		// Check emissions
		var resVal float64
		var resUnits string
		var emis map[string]value
		if testType == "onsite" {
			emis = data.OnSiteEmis
		} else {
			emis = data.WTPEmis
		}
		for gas, v := range emis {
			// TODO: Implement CO2 biogenic calculations.
			if gas == "CO2_Biogenic" { // && testType == "onsite" {
				// Because there is no way to correctly calculate CO2_Biogenic
				// emissions without calculating the whole life cycle, we ignore
				// them here.
				continue
			}
			foundGas := false
			for g, resV := range results.Emissions {
				if g.GetName() == gas {
					resVal = resV.Value()
					resUnits = resV.Dimensions().String()
					foundGas = true
				}
			}
			if !foundGas {
				t.Errorf("%v FAIL: %s; %s %s results don't contain emissions of '%s'",
					data.I, data.Pathway, data.Process, testType, gas)
				testsTotal++
				continue
			}
			passFail := "Pass"
			if math.Abs(resVal-v.Val) > math.Abs(v.Val)*tolerance ||
				resUnits != convertUnits(v.Units) {
				t.Fail()
				passFail = "FAIL"
			}
			testsTotal++
			if passFail == "Pass" {
				testsPassed++
				if !showPassedTests {
					continue
				}
			}
			t.Logf("%v %s: %s; %s; %s '%s' emissions are '%.5g %s' "+
				"and should equal '%.5g %s'  (%.3g%%)",
				data.I, passFail, data.Pathway, data.Process, testType, gas, resVal, resUnits,
				v.Val, v.Units,
				2*math.Abs(resVal-v.Val)/(resVal+v.Val)*100)
		}

		// Check resource use
		resVal = 0
		resUnits = ""
		var res map[string]value
		if testType == "onsite" {
			res = data.OnSiteResources
		} else {
			res = data.WTPResources
		}
		for resource, v := range res {
			foundResource := false
			for rI, resV := range results.Resources {
				r := rI.(*Resource)
				if r.GetName() == resource {
					if (v.Units == "joules" || v.Units == "J") &&
						resV.Dimensions().String() == "kg" {
						// convert from mass to energy format
						resV = unit.Mul(resV, r.GetHeatingValueMass(db))
					} else if (v.Units == "cu_meters" || v.Units == "m^3") &&
						resV.Dimensions().String() == "kg" {
						// convert from mass to volume format
						resV = unit.Div(resV, r.GetDensity(db))
					}
					resVal = resV.Value()
					resUnits = resV.Dimensions().String()
					foundResource = true
				}
			}
			if !foundResource {
				t.Errorf("%v FAIL: %s; %s %s results don't contain resource use for '%s'",
					data.I, data.Pathway, data.Process, testType, resource)
				testsTotal++
				continue
			}
			passFail := "Pass"
			if math.Abs(resVal-v.Val) > math.Abs(v.Val)*tolerance ||
				resUnits != convertUnits(v.Units) {
				t.Fail()
				passFail = "FAIL"
			}
			testsTotal++
			if passFail == "Pass" {
				testsPassed++
				if !showPassedTests {
					continue
				}
			}
			t.Logf("%v %s: %s; %s %s '%s' use is '%.5g %s' "+
				"and should equal '%.5g %s'  (%.3g%%)",
				data.I, passFail, data.Pathway, data.Process, testType, resource,
				resVal, resUnits, v.Val, v.Units,
				2*math.Abs(resVal-v.Val)/(resVal+v.Val)*100)
		}
	}
	t.Logf("Passed %d out of %d tests (%.3g%%).", testsPassed, testsTotal,
		float64(testsPassed)/float64(testsTotal)*100)
}

/*
// TestTracePathway performs tests tracing backwards through a pathway,
// stopping when the tests pass. This can be helpful to determine which
// process(es) are causing the errors.
func TestTracePathway(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	const (
		tolerance = 0.01
		resource  = "Bitumen"
		//"Pathway": "Pet Coke from Crude for Use in U.S. Refineries",
		//"Process": "Pet Coke Refining-with pre-defined Crude oil mixes",
		//"OutputID": "01cb54c8-0000-0000-0000-000000000000",
		pathwayName                = "NA NG from Shale and Regular Recovery"
		processName                = ""
		OutputID     slca.OutputID = mainOutput
		maxNestDepth               = 3
	)
	path, err := db.GetPathwayMixOrVehicleFromName(pathwayName)
	handle(err)
	var pathway *Pathway
	var proc slca.Process
	var output slca.Output
	switch path.(type) {
	case *Pathway:
		pathway = path.(*Pathway)
		pathOutput := pathway.GetMainOutput(db)
		if processName == "" {
			v := pathway.VertexForMainOutput()
			proc, _ = v.GetProcess(pathway, pathOutput.GetResource(db).(*Resource), db)
			output = proc.GetMainOutput(db)
		} else {
			for _, v := range pathway.Vertices {
				proc, _ = v.GetProcess(pathway, pathOutput.GetResource(db).(*Resource), db)
				output = proc.GetMainOutput(db)
				// We'll assume that we're always interested in the main output.
				//		if output.GetID() != OutputID {
				//			panic("wrong output")
				//		}
				if proc.GetName() == processName {
					break
				}
			}
		}
	case *Mix:
		pathway = &mixPathway
		proc = path.(*Mix)
		output = path.(*Mix).GetMainOutput(db)
	}

	testTracePathway(t, pathway, pathway, proc, proc, resource,
		output.(OutputLike), output.(OutputLike), tolerance, 0, maxNestDepth)

}

func testTracePathway(t *testing.T, pathway, downPath *Pathway,
	process, downProc slca.Process, resource string,
	output, downOutput OutputLike, tolerance float64, nestDepth, maxNestDepth int) {

	if nestDepth > maxNestDepth {
		return
	}

	testData := getTestData(t, pathway.Name, process.GetName(), output.GetID())

	amount := getFunctionalUnit(testData.FunctionalUnit)
	amount = output.GetResource(db).ConvertToDefaultUnits(amount, db)
	r := slca.NewResults(db)
	r.wtp(process, downProc, pathway, downPath, output, downOutput, amount, 0)
	results := r.Sum()

	keepGoing := checkResourceUse(t, resource, results, testData, pathway.Name,
		process.GetName(), downPath.GetName(), downProc.GetName(), tolerance, nestDepth)
	if !keepGoing {
		return
	}
	onsite := process.EmissionsAndResourceUse(pathway, output,
		results.db).scaleCopy(amount)
	for upProc, pov := range onsite.Requirements {
		for upPath, ov := range pov {
			for upOutput := range ov {
				testTracePathway(t, upPath, pathway, upProc, process, resource,
					upOutput, output, tolerance, nestDepth+1, maxNestDepth)
			}
		}
	}
}
*/

func getTestData(t *testing.T, pathwayName, processName string, OutputID slca.OutputID) greetTest {

	var testData greetTest
	found := false
	for _, testData = range greetTests {
		if (testData.Pathway == pathwayName && testData.Process == processName &&
			testData.OutputID == OutputID) ||
			(pathwayName == "Mix" && testData.Pathway == processName) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("couldn't find %s; %s; %s", pathwayName, processName, OutputID)
	}
	return testData
}

func checkResourceUse(t *testing.T, resource string, results *slca.OnsiteResultsNoSubprocess,
	testData greetTest, pathwayName, processName, downPathwayName, downProcessName string,
	tolerance float64, nestDepth int, db *DB) (keepGoing bool) {

	found := false
	for r, v := range results.Resources {
		if r.GetName() == resource {
			found = true
			vv := r.(*Resource).ConvertToEnergy(v, db)
			testV := testData.WTPResources[resource]
			if !similar(vv.Value(), testV.Val, tolerance) {
				str := fmt.Sprintf("%s; %s: %g %s doesn't match %g %s (%.3g%%)", pathwayName,
					processName, vv.Value(), vv.Dimensions().String(),
					testV.Val, testV.Units,
					2*math.Abs(vv.Value()-testV.Val)/(vv.Value()+testV.Val)*100)
				t.Log(str)
				log.Println(str)
				t.Fail()
				keepGoing = true
			} else {
				str := fmt.Sprintf("%s; %s: %g %s matches %g %s (%.3g%%)\n"+
					"Downstream is %s; %s", pathwayName,
					processName, vv.Value(), vv.Dimensions().String(),
					testV.Val, testV.Units,
					2*math.Abs(vv.Value()-testV.Val)/(vv.Value()+testV.Val)*100,
					downPathwayName, downProcessName)
				t.Log(str)
				log.Println(str)
				//if nestDepth > 0 {
				//	t.FailNow()
				//}
			}
		}
	}
	if !found {
		t.Logf("%s couldn't find %s", pathwayName, resource)
	}
	return
}

func similar(a, b, tolerance float64) bool {
	if a == 0 && b == 0 {
		return true
	}
	if 2*math.Abs(a-b)/(a+b) > tolerance {
		return false
	}
	return true
}
