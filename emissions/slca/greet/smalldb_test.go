package greet

import (
	"os"
	"testing"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/unit"
)

func initTestDB() *DB {
	f1, err := os.Open("testdb.greet")
	if err != nil {
		panic(err)
	}
	testDB := Load(f1)
	f1.Close()
	return testDB
}

func TestOceanTanker(t *testing.T) {
	var (
		// Mode Properties
		distance               = unit.New(1.e6, unit.Meter)
		AverageSpeed           = unit.New(10., unit.MeterPerSecond)
		LoadFactor             = unit.New(1., unit.Dimless) // fraction
		TypicalFuelConsumption = unit.Div(unit.New(50., unit.Kilogram),
			unit.New(1, unit.Joule)) // kg/J
		TypicalHP    = unit.New(1.e4, unit.Watt)
		HPPerPayload = unit.Div(unit.New(80, unit.Watt),
			unit.New(1, unit.Kilogram)) // W / kg
		Payload = unit.New(1.e4, unit.Kilogram)

		// Resource Properties
		density = unit.New(1.e3, unit.KilogramPerMeter3) // kg/m3
		lhv     = unit.Div(unit.New(1, unit.Joule),
			unit.New(1, unit.Meter3)) // J/m3

		// Technology Parameters
		EF = unit.Div(unit.New(1, unit.Kilogram), unit.New(1, unit.Joule))

		GREETEITo = 4.05e-1 // J / m / kg
		//EIFrom = 4.05e-1
		GREETEmissions = 8.1e5   // kg
		GREETRes       = 1620.e3 // J

		tolerance = 1.e-6
	)

	// horsepower: hp = 9070[hp] + 0.101 [hp/ton] * Payload[ton]
	hp := unit.Add(TypicalHP, unit.Mul(HPPerPayload, Payload))
	handle(hp.Check(unit.Watt))

	// energy consumption: ec = ( 14.42 [g/Wh] / LoadFactor + 178.47 [g/Wh] ) * lhv / density
	c1 := unit.Div(unit.New(14.42/1000, unit.Kilogram),
		unit.New(1, unit.Watt), unit.New(3600, unit.Second)) // g / Wh
	//c2 := unit.Div(unit.New(178.47/1000, unit.Kilogram),
	//	unit.New(1, unit.Watt), unit.New(3600, unit.Second)) // g / Wh
	ec := unit.Mul(
		unit.Add(unit.Div(c1, LoadFactor), TypicalFuelConsumption),
		unit.Div(lhv, density))

	ei := unit.Div(
		unit.Mul(ec, hp, LoadFactor),
		unit.Mul(Payload, AverageSpeed))
	handle(ei.Check(unit.MeterPerSecond2))

	energyUse := unit.Mul(ei, distance)

	// Multiply by 2 to account for the back haul.
	emissions := unit.Mul(energyUse, EF, unit.New(2, unit.Dimless))

	if !similar(ei.Value(), GREETEITo, tolerance) {
		t.Errorf("Hand calculated energy intensity (%g) does not equal %g.",
			ei, GREETEITo)
	}
	if !similar(emissions.Value(), GREETEmissions, tolerance) {
		t.Errorf("Hand calculated emissions (%g) do not equal %g.",
			emissions, GREETEmissions)
	}
	// See if the model calculation matches the hand calculation and the GREET
	// calculation.
	emis, progRes, _ := runTestDB("Test Pathway Tanker Trans")
	if !similar(emis, GREETEmissions, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			emis, GREETEmissions)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) do not equal %g.",
			progRes, GREETRes)
	}
}

func TestTruck(t *testing.T) {
	const (
		GREETEmis = 5.0000e+003 // kg
		GREETRes  = 10.e3       // J
		tolerance = 1.e-6
	)
	progEmis, progRes, _ := runTestDB("Truck Transportation Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) do not equal %g.",
			progRes, GREETRes)
	}
}

func TestPipeline(t *testing.T) {
	const (
		GREETEmis = 3.6000e+006 // kg
		GREETRes  = 7200.e3     // J
		tolerance = 1.e-6
	)
	progEmis, progRes, _ := runTestDB("Pipeline Transportation Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) do not equal %g.",
			progRes, GREETRes)
	}
}

func TestRail(t *testing.T) {
	const (
		GREETEmis = 2.0000e+008 // kg
		GREETRes  = 400.e6      // J
		tolerance = 1.e-6
	)
	progEmis, progRes, _ := runTestDB("Rail Transportation Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) do not equal %g.",
			progRes, GREETRes)
	}
}

func TestMultimode(t *testing.T) {
	const (
		GREETEmis = 2.0324e6  // kg
		GREETRes  = 4064.86e3 // J
		tolerance = 1.e-4
	)
	progEmis, progRes, _ := runTestDB("Multimode Transportation Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) do not equal %g.",
			progRes, GREETRes)
	}
}

func TestMultimode2(t *testing.T) {
	const (
		GREETEmis = 4.86e4 // kg
		GREETRes  = 97.2e3 // J
		tolerance = 1.e-4
	)
	progEmis, progRes, _ := runTestDB("Multimode Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) do not equal %g.",
			progRes, GREETRes)
	}
}

func TestInput(t *testing.T) {
	const (
		GREETEmis = 3.e-3 // kg
		GREETRes  = 6.e-3 // J
		tolerance = 1.e-4
	)
	progEmis, progRes, _ := runTestDB("Input Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) do not equal %g.",
			progRes, GREETRes)
	}
}

func TestAmountGroup(t *testing.T) {
	const (
		GREETEmis = 1.5e-3 // kg
		GREETRes  = 3.e-3  // J
		tolerance = 1.e-4
	)
	progEmis, progRes, _ := runTestDB("Amount Group Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) do not equal %g.",
			progRes, GREETRes)
	}
}

func TestEfficiencyGroup(t *testing.T) {
	const (
		GREETEmis = 1.6667e-3 // kg
		GREETRes  = 3.3333e-3 // J
		tolerance = 1.e-4
	)
	progEmis, progRes, _ := runTestDB("Test Efficiency Group")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) do not equal %g.",
			progRes, GREETRes)
	}
}

func TestLoop(t *testing.T) {
	const (
		GREETEmis = 1.1e-2     // kg
		GREETRes  = 19.9998e-3 // J
		tolerance = 1.e-4
	)
	progEmis, progRes, _ := runTestDB("Loop Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) do not equal %g.",
			progRes, GREETRes)
	}
}

func TestMassAllocation(t *testing.T) {
	const (
		GREETEmis = 1.1429e-3 // kg
		GREETRes  = 1.1429e-3 // J
		tolerance = 1.e-4
	)
	// hand calculation
	const (
		input1       = 1000. // J
		output1      = 750.  // J
		output2      = 150.  // J
		lhv1         = 1.    // J/m3
		lhv2         = 1.2   // J/m3
		density1     = 1000. // kg/m3
		density2     = 1000. // kg/m3
		requirement1 = 1.    // kg
		EF1          = 1.    // kg/J
	)
	req := requirement1 / density1 * lhv1 // J
	output1Mass := output1 / lhv1 * density1
	output2Mass := output2 / lhv2 * density2
	allocFactor := output1Mass / (output1Mass + output2Mass)
	use := req / output1 * (input1*allocFactor - output1)
	allocatedTotal := req + use
	if !similar(allocatedTotal, GREETRes, tolerance) {
		t.Errorf("Hand calculated resource use (%g) does not equal %g.",
			allocatedTotal, GREETRes)
	}
	emissionsTotal := allocatedTotal * EF1
	if !similar(allocatedTotal, GREETRes, tolerance) {
		t.Errorf("Hand calculated emissions use (%g) do not equal %g.",
			emissionsTotal, GREETEmis)
	}
	// model calculation
	progEmis, progRes, _ := runTestDB("Mass Allocation Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	} else {
		t.Logf("Model calculated emissions (%g) equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) does not equal %g.",
			progRes, GREETRes)
	}
}

func TestMarketAllocation(t *testing.T) {
	const (
		GREETEmis = 0.1428571428e-3 // kg
		GREETRes  = 0.1428571428e-3 // J
		tolerance = 1.e-4
	)
	// hand calculation
	const (
		input1       = 1000. // J
		output1      = 750.  // J
		output2      = 150.  // J
		lhv1         = 1.    // J/m3
		lhv2         = 1.2   // J/m3
		value1       = 1.    // $/m3
		value2       = 50    // J/m3
		density1     = 1000. // kg/m3
		density2     = 1000. // kg/m3
		requirement1 = 1.    // kg
	)
	req := requirement1 / density1 * lhv1 // J
	output1Val := output1 / lhv1 * value1
	output2Val := output2 / lhv2 * value2
	allocFactor := output1Val / (output1Val + output2Val)
	use := req / output1 * (input1*allocFactor - output1)
	allocatedTotal := req + use
	if !similar(allocatedTotal, GREETRes, tolerance) {
		t.Errorf("Hand calculated resource use (%g) does not equal %g.",
			allocatedTotal, GREETRes)
	}

	// model calculation
	progEmis, progRes, _ := runTestDB("Market Allocation Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) does not equal %g.",
			progRes, GREETRes)
	}
}

func TestEnergyAllocation(t *testing.T) {
	const (
		GREETEmis = 1.1111e-3 // kg
		GREETRes  = 1.1111e-3 // J
		tolerance = 1.e-4
	)
	// hand calculation
	const (
		input1       = 1000. // J
		output1      = 750.  // J
		output2      = 150.  // J
		lhv1         = 1.    // J/m3
		lhv2         = 1.2   // J/m3
		density1     = 1000. // kg/m3
		density2     = 1000. // kg/m3
		requirement1 = 1.    // kg
	)
	req := requirement1 / density1 * lhv1 // J
	allocFactor := output1 / (output1 + output2)
	use := req / output1 * (input1*allocFactor - output1)
	allocatedTotal := req + use
	if !similar(allocatedTotal, GREETRes, tolerance) {
		t.Errorf("Hand calculated resource use (%g) does not equal %g.",
			allocatedTotal, GREETRes)
	}

	// model calculation
	progEmis, progRes, _ := runTestDB("Energy Allocation Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) does not equal %g.",
			progRes, GREETRes)
	}
}

func TestVolumeAllocation(t *testing.T) {
	const (
		GREETEmis = 1.1429e-3 // kg
		GREETRes  = 1.1429e-3 // J
		tolerance = 1.e-4
	)
	// hand calculation
	const (
		input1       = 1000. // J
		output1      = 750.  // J
		output2      = 150.  // J
		lhv1         = 1.    // J/m3
		lhv2         = 1.2   // J/m3
		density1     = 1000. // kg/m3
		density2     = 1000. // kg/m3
		requirement1 = 1.    // kg
	)
	req := requirement1 / density1 * lhv1 // J
	output1Volume := output1 / lhv1
	output2Volume := output2 / lhv2
	allocFactor := output1Volume / (output1Volume + output2Volume)
	use := req / output1 * (input1*allocFactor - output1)
	allocatedTotal := req + use
	if !similar(allocatedTotal, GREETRes, tolerance) {
		t.Errorf("Hand calculated resource use (%g) does not equal %g.",
			allocatedTotal, GREETRes)
	}

	// model calculation
	progEmis, progRes, _ := runTestDB("Volume Allocation Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes, GREETRes, tolerance) {
		t.Errorf("Model calculated resource use (%g) does not equal %g.",
			progRes, GREETRes)
	}
}

func TestDisplacement(t *testing.T) {
	const (
		GREETEmis = -0.0020 // kg
		GREETRes1 = 2.0e-3  // J
		GREETRes2 = -0.0008 // J
		tolerance = 1.e-4
	)
	progEmis, progRes1, progRes2 := runTestDB("Displacement Test")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes1, GREETRes1, tolerance) {
		t.Errorf("Model calculated resource 1 use (%g) do not equal %g.",
			progRes1, GREETRes1)
	}
	if !similar(progRes2, GREETRes2, tolerance) {
		t.Errorf("Model calculated resource 2 use (%g) do not equal %g.",
			progRes2, GREETRes2)
	}
}

func TestMixEnergy(t *testing.T) {
	const (
		GREETEmis = 0.75e-3 // kg
		GREETRes1 = 2.5e-3  // J
		tolerance = 1.e-4
	)
	progEmis, progRes1, _ := runTestDB("Test Mix Energy")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes1, GREETRes1, tolerance) {
		t.Errorf("Model calculated resource 1 use (%g) do not equal %g.",
			progRes1, GREETRes1)
	}
}

func TestMixMass(t *testing.T) {
	const (
		GREETEmis = 0.75e-3 // kg
		GREETRes1 = 2.5e-3  // J
		tolerance = 1.e-4
	)
	progEmis, progRes1, _ := runTestDB("Test Mix Mass")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes1, GREETRes1, tolerance) {
		t.Errorf("Model calculated resource 1 use (%g) do not equal %g.",
			progRes1, GREETRes1)
	}
}

func TestMixVolume(t *testing.T) {
	const (
		GREETEmis = 0.75e-3 // kg
		GREETRes1 = 2.5e-3  // J
		tolerance = 1.e-4
	)
	progEmis, progRes1, _ := runTestDB("Test Mix Volume")
	if !similar(progEmis, GREETEmis, tolerance) {
		t.Errorf("Model calculated emissions (%g) do not equal %g.",
			progEmis, GREETEmis)
	}
	if !similar(progRes1, GREETRes1, tolerance) {
		t.Errorf("Model calculated resource 1 use (%g) do not equal %g.",
			progRes1, GREETRes1)
	}
}

// runTestDB calculates emissions for the requested pathway in units of kg.
func runTestDB(pathname string) (float64, float64, float64) {
	testDB := initTestDB()
	path, err := testDB.GetPathwayMixOrVehicleFromName(pathname)
	handle(err)
	r := slca.SolveGraph(path, unit.New(1, unit.Kilogram), &slca.DB{LCADB: testDB})
	sum := r.Sum()
	g, err := testDB.GetGas("Test Gas 1")
	handle(err)
	emis := sum.Emissions[g]
	handle(emis.Check(unit.Kilogram))
	res1 := testDB.GetResourceFromName("Test Resource Liquid")
	resource1 := res1.ConvertToEnergy(sum.Resources[res1], testDB)
	handle(resource1.Check(unit.Joule))
	var resource2 float64
	res2 := testDB.GetResourceFromName("Test Resource Liquid 2")
	if v, ok := sum.Resources[res2]; ok {
		r2 := res2.ConvertToEnergy(v, testDB)
		handle(r2.Check(unit.Joule))
		resource2 = r2.Value()
	}
	return emis.Value(), resource1.Value(), resource2
}
