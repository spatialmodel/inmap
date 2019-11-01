package greet

import (
	"fmt"
	"sync"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/unit"
)

// TransportationProcess is a holder for transportation process data
// from the GREET database.
type TransportationProcess struct {
	sync.RWMutex            `xml:"-"`
	ID                      ModelID      `xml:"id,attr"`
	Name                    string       `xml:"name,attr"`
	IOCarbonMap             *IOCarbonMap `xml:"io-carbon-map"`
	PreferredFunctionalUnit *Unit        `xml:"prefered_functional_unit"`
	Outputs                 []*Output    `xml:"output"`
	Inputs                  []*Input     `xml:"input"`
	MoistureStr             Expression   `xml:"moisture,attr"`
	Steps                   []*Step      `xml:"step"`
	Notes                   string       `xml:"notes,attr"`

	results map[*Pathway]*slca.OnsiteResults
}

// SpatialRef returns the spatial reference for the receiver.
// TODO: Need to implement something for this.
func (p *TransportationProcess) SpatialRef(aqm string) *slca.SpatialRef {
	return &slca.SpatialRef{Type: slca.Transportation, AQM: aqm}
}

// moisture gets the moisture in the item being transported. Units = [unitless]
func (p *TransportationProcess) moisture(db *DB) *unit.Unit {
	m := db.evalExpr(p.MoistureStr)
	handle(m.Check(unit.Dimless))
	return m
}

// moistureAdjust adjusts the amount of material transported to account for The
// fact that moisture must also be transported.
func (p *TransportationProcess) moistureAdjust(amount *unit.Unit, db *DB) *unit.Unit {
	return unit.Div(amount, unit.Sub(unit.New(1, unit.Dimless), p.moisture(db)))
}

// GetID returns the ID of this process.
func (p *TransportationProcess) GetID() ModelID {
	return p.ID
}

// GetIDStr returns the ID of this process in string format.
func (p *TransportationProcess) GetIDStr() string {
	return "Trans" + string(p.ID)
}

// GetOutput gets the output of this process that outputs the given resource.
// Transportation processes should only have one output, so it panics if there
// is more or less than one output or if the output does not output the
// given resource.
func (p *TransportationProcess) GetOutput(res slca.Resource, db slca.LCADB) slca.Output {
	r := res.(*Resource)
	if len(p.Outputs) != 1 {
		panic(fmt.Errorf("TransportationProcess %s (id=%v) has %d outputs and "+
			"should have 1", p.Name, p.ID, len(p.Outputs)))
	}
	o := p.Outputs[0]
	if o.GetResource(db).(*Resource).IsCompatible(r) {
		return o
	}
	panic(fmt.Errorf("TransportationProcess %s (id=%v) doesn't output "+
		"resource %s (id=%v).", p.Name, p.ID, r.Name, r.ID))
}

// GetMainOutput gets the main output of this process. It panics if there is more than one main output.
func (p *TransportationProcess) GetMainOutput(db slca.LCADB) slca.Output {
	if len(p.Outputs) != 1 {
		panic(fmt.Errorf("Incorrect number of outputs for %#v.", p))
	}
	return p.Outputs[0]
}

// GetInput gets the single input to this process. It panics if there is more than one input.
func (p *TransportationProcess) GetInput() *Input {
	if len(p.Inputs) != 1 {
		panic(fmt.Errorf("Incorrect number of inputs for %#v.", p))
	}
	return p.Inputs[0]
}

// GetName gets the name of this process.
func (p *TransportationProcess) GetName() string {
	return p.Name
}

// Type returns the process type.
func (p *TransportationProcess) Type() slca.ProcessType {
	return slca.Transportation
}

// DefaultNonCombustionSCC is used for everything we don't have an SCC for.
const DefaultNonCombustionSCC slca.SCC = "0028888801" // TODO: Perhaps figure out a way to do better.

// OnsiteResults calculates the onsite emissions from and resource
// use of this process per unit output, where "output" is the output that is
// required from the process, "path" is the pathway the process is a part of.
// The returned value is a pointer and is cached for future use, so be sure to
// clone the result before modifying it.
func (p *TransportationProcess) OnsiteResults(pathI slca.Pathway, outputI slca.Output, lcadb slca.LCADB) *slca.OnsiteResults {
	db := lcadb.(*DB)
	path := pathI.(*Pathway)
	output := outputI.(OutputLike)
	p.Lock()
	defer p.Unlock()
	if p.results == nil {
		p.results = make(map[*Pathway]*slca.OnsiteResults)
	}
	// Return the saved results if they have already been calculated.
	if rr, ok := p.results[path]; ok {
		return rr
	}
	r := slca.NewOnsiteResults(db)

	p.checkOutput(output)

	input, material, amountTransportedDry := p.getMaterialAmount(db)
	// Add input resource (amount transported)
	r.AddResource(noSubprocess, material, amountTransportedDry, db)

	sourceProc, sourcePath := input.GetSource(path, db)
	sourceO := sourceProc.GetOutput(material, db)
	r.AddRequirement(sourceProc, sourcePath, sourceO, amountTransportedDry, db)

	lossProc := subprocess{
		name: "Losses",
		scc:  DefaultNonCombustionSCC,
	}

	var stepLosses *unit.Unit
	for _, step := range p.Steps {
		distance := step.GetDistance(db) // m
		stepshare := step.GetShare(db)
		amountTransportedWet := material.ConvertToMass(
			p.moistureAdjust(amountTransportedDry, db), db)
		mode, fuelShare := step.GetModeAndFuelShare(db)

		for _, fuel := range fuelShare.Fuels {
			eiFrom, eiTo := mode.CalculateEnergyIntensity(material, fuel, db) // J / kg / m
			fuelRes := fuel.GetFuel(db)
			fuelshare := fuel.GetShare(db)
			energyFrom := unit.Mul(amountTransportedWet, eiFrom, distance,
				stepshare, fuelshare) // J
			handle(energyFrom.Check(unit.Joule))
			energyTo := unit.Mul(amountTransportedWet, eiTo, distance,
				stepshare, fuelshare) // J
			handle(energyTo.Check(unit.Joule))

			techTo := fuel.GetTechTo(db)
			techFrom := fuel.GetTechFrom(db)

			// Add fuel use to
			fuelProc, fuelPath := fuel.GetPathway(db)
			r.AddResource(techTo, fuelRes, energyTo, db)
			sourceO := fuelProc.GetOutput(fuelRes, db)
			r.AddRequirement(fuelProc, fuelPath, sourceO, energyTo, db)

			// Add emissions from fuel use to
			gases, emissions := fuel.calcEmissionsTo(energyTo, db)
			for i, g := range gases {
				r.AddEmission(techTo, g, emissions[i])
			}
			if step.BackHaul {
				// Add fuel use from
				r.AddResource(techFrom, fuelRes, energyFrom, db)
				r.AddRequirement(fuelProc, fuelPath, sourceO, energyFrom, db)

				// Add emissions from fuel use from.
				gases, emissions := fuel.calcEmissionsFrom(energyFrom, db)
				for i, g := range gases {
					r.AddEmission(techFrom, g, emissions[i])
				}
			}
		}
		// Add emissions from losses from this step.
		gases, emissions := step.GetLossEmissions(material, amountTransportedWet, db)
		for i, g := range gases {
			r.AddEmission(lossProc, g, emissions[i])
		}
		// Add in the amount lost
		stepLosses = unit.Add(stepLosses, step.GetLossAmount(material,
			amountTransportedWet, db))
	}
	// Subtract resources that go out (minus losses).
	r.SubResource(noSubprocess, material, unit.Sub(
		material.ConvertToDefaultUnits(output.GetAmount(db), db),
		material.ConvertToDefaultUnits(stepLosses, db)), db)

	// Add emissions from overall losses.
	gases, emissions := output.GetLossEmissions(db)
	for i, g := range gases {
		r.AddEmission(lossProc, g, emissions[i])
	}

	// Divide the results by the output amount to get results per unit output.
	r.Div(output.GetResource(db).ConvertToDefaultUnits(output.GetAmount(db), db))

	// Save results for future use.
	p.results[path] = r
	return r
}

// checkOutput makes sure output o is part of process p.
func (p *TransportationProcess) checkOutput(o OutputLike) {
	for _, oo := range p.Outputs {
		if oo.ID == o.GetID() {
			return
		}
	}
	panic(fmt.Errorf("output %v is not in process %v", o.GetID(), p.ID))
}

// getMaterialAmount returns the material and amount transported by this process,
// and the process and path it is coming from.
func (p *TransportationProcess) getMaterialAmount(db *DB) (
	input *Input, material *Resource, amountTransported *unit.Unit) {

	input = p.GetInput()
	material = input.GetResource(db)
	amountTransported = material.ConvertToDefaultUnits(input.GetAmount(db), db)
	return
}
