package greet

import (
	"fmt"

	"github.com/ctessum/unit"
	"github.com/spatialmodel/inmap/emissions/slca"
)

// Mode is a holder for information about a transportation mode.
type Mode struct {
	Name         string `xml:"name,attr"`
	Type         string `xml:"type,attr"`
	ID           ModeID `xml:"id,attr"`
	AverageSpeed Param  `xml:"average_speed"` // [length/time]

	// Load factor is the percentage of installed power that is used for the trip
	LoadFactorFrom Param `xml:"load_factor_from"` // [Dimless]
	LoadFactorTo   Param `xml:"load_factor_to"`   // [Dimless]

	FuelEconomyFrom Param `xml:"fuel_economy_from"`
	FuelEconomyTo   Param `xml:"fuel_economy_to"`

	// TypicalFC is the mass of fuel consumption per unit of output work
	// (Brake specific fuel consumption (BSFC)).
	TypicalFC Param `xml:"typical_fc"` // [mass/energy]

	// TypicalHP is the amount of power required when the vehicle is unloaded.
	TypicalHP Param `xml:"typical_hp"` // [power]

	// HPFactor is the amount of additional power required per unit mass of payload.
	HPFactor Param `xml:"hp_factor"` // [power/mass]

	// EnergyIntensity is energy used per unit distance per unit payload mass. It is only reported
	// directly for the rail mode.
	EnergyIntensity Param `xml:"ei"` // [energy/distance/mass]

	// BSFCAdjustment defines how (brake-specific) fuel consumption varies with
	// varying load factor.
	BSFCAdjustment Param `xml:"bsfc_adjustment"`

	FuelShares []*FuelShare `xml:"fuel_shares>share"`
	Payloads   []*Payload   `xml:"payload>material_transported"`

	// Energy intensity for pipelines
	EnergyIntensities []*EnergyIntensity `xml:"energy_intensity>material_transported"`
}

// CalculateEnergyIntensity returns the energy used per unit distance per unit
// payload mass for this mode,
// for both the outbound and inbound legs of the journey.
func (m *Mode) CalculateEnergyIntensity(materialTransported *Resource,
	fuel *Fuel, db *DB) (from, to *unit.Unit) {

	switch m.Type {
	case "1": // Ocean tanker or barge
		speed := m.GetAverageSpeed(db) // m/s
		loadFactorFrom := m.GetLoadFactorFrom(db)
		loadFactorTo := m.GetLoadFactorTo(db)
		typicalFC := m.GetTypicalFC(db)                  // Fuel consumption [kg/J]
		fcAdj := m.GetBSFCAdjustment(db)                 // Fuel consumption adjustment factor [kg/J]
		typicalPower := m.GetTypicalHP(db)               // W
		powerPerPayload := m.GetHPFactor(db)             // W/kg
		payload := m.GetPayload(materialTransported, db) // kg

		// horsepower: hp = 9070[hp] + 0.101 [hp/ton] * Payload[ton]
		power := unit.Add(typicalPower, unit.Mul(powerPerPayload, payload))
		handle(power.Check(unit.Watt))

		// Calculate fuel energy needed per unit of work energy output.
		fuelRes := fuel.GetFuel(db)
		hv := fuelRes.GetHeatingValueVolume(db) // J/kg
		density := fuelRes.GetDensity(db)       // kg/m3

		// energy consumption: ec = ( fcAdj / LoadFactor + typicalFC ) * lhv / density
		ecTo := unit.Mul(
			unit.Add(unit.Div(fcAdj, loadFactorTo), typicalFC),
			unit.Div(hv, density))
		handle(ecTo.Check(unit.Dimless))

		ecFrom := unit.Mul(
			unit.Add(unit.Div(fcAdj, loadFactorFrom), typicalFC),
			unit.Div(hv, density))
		handle(ecFrom.Check(unit.Dimless))

		to = unit.Div(
			unit.Mul(ecTo, power, loadFactorTo),
			unit.Mul(payload, speed))
		handle(to.Check(unit.MeterPerSecond2))

		from = unit.Div(
			unit.Mul(ecFrom, power, loadFactorFrom),
			unit.Mul(payload, speed))
		handle(to.Check(unit.MeterPerSecond2))

		return
	case "2": // truck
		payload := m.GetPayload(materialTransported, db) // kg
		fuelEconomyFrom := m.GetFuelEconomyFrom(db)      // m/m3
		fuelEconomyTo := m.GetFuelEconomyTo(db)          // m/m3
		fuelRes := fuel.GetFuel(db)
		hv := fuelRes.GetHeatingValueVolume(db)   // J/m3
		to = unit.Div(hv, fuelEconomyTo, payload) // J / m / kg
		from = unit.Div(hv, fuelEconomyFrom, payload)
		return
	case "3": // pipeline
		to = m.GetPipelineEnergyIntensity(materialTransported, db)
		from = unit.New(0, joulesPerMPerKg)
		return
	case "4": // rail
		to = m.GetRailEnergyIntensity(db)
		// Rail never has an empty back haul.
		from = unit.New(0, joulesPerMPerKg)
		return
	case "5": // "connector" or "magicMove"
		to = unit.New(0, joulesPerMPerKg)
		from = to
		return
	default:
		panic(fmt.Errorf("Unknown mode type in %#v.", m))
	}
}

// GetPayload returns the payload for this mode carrying the specified resource.
func (m *Mode) GetPayload(r *Resource, db *DB) *unit.Unit {
	for _, p := range m.Payloads {
		if p.Ref == r.ID {
			return db.evalExpr(p.Payload)
		}
	}
	panic(fmt.Errorf("Couldn't find payload for %#v in mode %s.", r, m.Name))
}

// GetPipelineEnergyIntensity gets the energy intensity for the mode in transporting the given resource.
// It only works for pipelines.
func (m *Mode) GetPipelineEnergyIntensity(r *Resource, db *DB) *unit.Unit {
	for _, e := range m.EnergyIntensities {
		if e.Ref == r.ID {
			return db.InterpolateValue(e.EnergyIntensity.ValueYears)
		}
	}
	// If we can't find the EI, just match it to liquid or solid transport
	for _, e := range m.EnergyIntensities {
		if e.Name == r.State {
			return db.InterpolateValue(e.EnergyIntensity.ValueYears)
		}
	}
	panic(fmt.Errorf("Couldn't find energy intensity for %#v in mode %s.", r, m.Name))
}

// GetAverageSpeed returns the average speed for this mode.
func (m *Mode) GetAverageSpeed(db *DB) *unit.Unit {
	return db.InterpolateValue(m.AverageSpeed.ValueYears)
}

// GetLoadFactorFrom returns the load factor in the from or inbound direction.
func (m *Mode) GetLoadFactorFrom(db *DB) *unit.Unit {
	return db.InterpolateValue(m.LoadFactorFrom.ValueYears)
}

// GetLoadFactorTo returns the load factor in the to or outbound direction.
func (m *Mode) GetLoadFactorTo(db *DB) *unit.Unit {
	return db.InterpolateValue(m.LoadFactorTo.ValueYears)
}

// GetFuelEconomyFrom returns the fuel economy in the from or inbound direction.
func (m *Mode) GetFuelEconomyFrom(db *DB) *unit.Unit {
	return db.InterpolateValue(m.FuelEconomyFrom.ValueYears)
}

// GetFuelEconomyTo returns the fuel economy in the to or outbound direction.
func (m *Mode) GetFuelEconomyTo(db *DB) *unit.Unit {
	return db.InterpolateValue(m.FuelEconomyTo.ValueYears)
}

// GetTypicalFC returns the typical fuel consumption.
func (m *Mode) GetTypicalFC(db *DB) *unit.Unit {
	return db.InterpolateValue(m.TypicalFC.ValueYears)
}

// GetBSFCAdjustment returns the brake-specific fuel consumption adjustment
// factor.
func (m *Mode) GetBSFCAdjustment(db *DB) *unit.Unit {
	return db.InterpolateValue(m.BSFCAdjustment.ValueYears)
}

// GetTypicalHP returns the typical power consumption.
func (m *Mode) GetTypicalHP(db *DB) *unit.Unit {
	return db.InterpolateValue(m.TypicalHP.ValueYears)
}

// GetHPFactor gets the power adjustment factor.
func (m *Mode) GetHPFactor(db *DB) *unit.Unit {
	return db.InterpolateValue(m.HPFactor.ValueYears)
}

// GetRailEnergyIntensity returns the energy intensity for the rail mode.
func (m *Mode) GetRailEnergyIntensity(db *DB) *unit.Unit {
	return db.InterpolateValue(m.EnergyIntensity.ValueYears)
}

// FuelShare gives information on the different fuels used by a Mode.
type FuelShare struct {
	Name  string      `xml:"name,attr"`
	ID    FuelShareID `xml:"id,attr"`
	Fuels []*Fuel     `xml:"fuel"`
}

// Fuel specifies a type of fuel used by a Mode.
type Fuel struct {
	FuelRef    ResourceID   `xml:"fuel_ref,attr"`
	Pathway    ModelID      `xml:"pathway,attr"`
	Mix        ModelID      `xml:"mix,attr"`
	ShareStr   Expression   `xml:"share,attr"`
	TechToID   TechnologyID `xml:"tech_to,attr"`
	TechFromID TechnologyID `xml:"tech_from,attr"`
}

// Payload specifies the amount of a given resource that a Mode can carry.
type Payload struct {
	Payload Expression `xml:"payload,attr"`
	Ref     ResourceID `xml:"ref,attr"`
}

// EnergyIntensity specifies the energy intensity of a Mode. Only used for rail.
type EnergyIntensity struct {
	EnergyIntensity Param      `xml:"ei"`
	Ref             ResourceID `xml:"ref,attr"`
	Name            string     `xml:"name,attr"`
}

// GetFuel returns the fuel that is being provided.
func (f *Fuel) GetFuel(db *DB) *Resource {
	for _, r := range db.Data.Resources {
		if r.ID == f.FuelRef {
			return r
		}
	}
	panic(fmt.Sprintf("Couldn't find fuel resource for %#v.", f))
}

// GetPathway returns the pathway or mix that provides the fuel.
func (f *Fuel) GetPathway(db *DB) (slca.Process, *Pathway) {
	for _, p := range db.Data.Pathways {
		if p.ID == f.Pathway {
			proc := p.GetOutput(f.GetFuel(db), db).(OutputLike).GetProcess(p, db)
			return proc, p
		}
	}
	for _, m := range db.Data.Mixes {
		if m.ID == f.Mix {
			return m, &mixPathway
		}
	}
	panic(fmt.Sprintf("Couldn't find pathway for fuel %#v.", f))
}

// GetShare returns the fraction of use of this fuel.
func (f *Fuel) GetShare(db *DB) *unit.Unit {
	return db.evalExpr(f.ShareStr)
}

// GetTechTo returns the technology that is providing outbound transportation
func (f *Fuel) GetTechTo(db *DB) *Technology {
	for _, t := range db.Data.Technologies {
		if t.ID == f.TechToID {
			return t
		}
	}
	panic(fmt.Sprintf("Couldn't find outbound technology for fuel %#v.", f))
}

// GetTechFrom returns the technology that is providing inbound transportation
func (f *Fuel) GetTechFrom(db *DB) *Technology {
	for _, t := range db.Data.Technologies {
		if t.ID == f.TechFromID {
			return t
		}
	}
	panic(fmt.Sprintf("Couldn't find inbound technology for fuel %#v.", f))
}

// calcEmissionsFrom calculates the emissions from this fuel in the inbound
// or from direction. units = grams
func (f *Fuel) calcEmissionsFrom(energyFrom *unit.Unit, db *DB) (
	gases []*Gas, vals []*unit.Unit) {

	techFrom := f.GetTechFrom(db)
	return f.calcEmissions(energyFrom, techFrom, db)
}

// calcEmissionsTo calculates the emissions from this fuel in the outbound
// or to direction. units = grams
func (f *Fuel) calcEmissionsTo(energyTo *unit.Unit, db *DB) (
	gases []*Gas, vals []*unit.Unit) {

	techTo := f.GetTechTo(db)
	return f.calcEmissions(energyTo, techTo, db)
}

// calcEmissions calculate the emissions from this for either the inbound
// or outbound directions. units=grams.
func (f *Fuel) calcEmissions(energy *unit.Unit, tech *Technology, db *DB) (
	gases []*Gas, vals []*unit.Unit) {

	var EFs []*unit.Unit
	gases, EFs = tech.GetEmissions(db)
	vals = make([]*unit.Unit, len(gases))
	for i := range gases {
		vals[i] = unit.Mul(energy, EFs[i])
		handle(vals[i].Check(unit.Kilogram))
	}
	return gases, vals
}
