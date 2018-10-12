package greet

import (
	"fmt"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/unit"
)

// Input is a holder for the GREET Input data type.
type Input struct {
	ID InputID `xml:"id,attr"`

	// The Source of the input. Should either be Previous, Mix, Well, or Pathway.
	Source string `xml:"source,attr"`

	// The mix that is the source of this input if Source = "Mix"
	Mix ModelID `xml:"mix,attr"`

	// The pathway that is the source of this input if Source = "Pathway".
	Pathway ModelID `xml:"pathway,attr"`

	// Is this considered the main input?
	ConsideredAsMain bool `xml:"considered_as_main,attr"`

	// Whether requirements and resource use are counted in this input. If "False",
	// only count emissions.
	AccountedInEnergyBalance string `xml:"accounted_in_energy_balance,attr"`

	// The resource that is input.
	Ref ResourceID `xml:"ref,attr"`

	// The share of this input. Used in input groups.
	Share Param `xml:"share"`

	// Technologies that use this input as a fuel source or operate on it.
	TechnologyShares []*TechnologyShare `xml:"technology"`

	// The amount of this input. Used in non-grouped inputs.
	AmountYears []*ValueYear `xml:"amount>year"`

	// EmissionRatios specify fractions of the total mass of the input that
	// are emitted as different gases.
	EmissionRatios []EmissionRatio `xml:"emission_ratio"`
}

// An EmissionRatio specifies the fraction of the total mass of an input that
// is emitted as a certain gas.
type EmissionRatio struct {
	GasID GasID `xml:"gas_id,attr"`
	Rate  Param `xml:"rate"`
}

// GetEmissionRatios returns the emission ratios associated with this input.
func (in *Input) GetEmissionRatios(db *DB) ([]*Gas, []*unit.Unit) {
	var gases []*Gas
	var rates []*unit.Unit
	for _, er := range in.EmissionRatios {
		gases = append(gases, db.getGasFromID(er.GasID))
		rates = append(rates, er.Rate.process(db))
	}
	return gases, rates
}

// GetResource gets the resource associated with this input.
func (in *Input) GetResource(db *DB) *Resource {
	for _, r := range db.Data.Resources {
		if r.ID == in.Ref {
			return r
		}
	}
	panic(fmt.Sprintf("No resource found for Input %#v.", in))
}

// GetAmount gets the amount of this input.
func (in *Input) GetAmount(db *DB) *unit.Unit {
	return db.InterpolateValue(in.AmountYears)
}

// GetAmountDefaultUnits gets the amount of this input in the default units for the input resource.
func (in *Input) GetAmountDefaultUnits(db *DB) *unit.Unit {
	r := in.GetResource(db)
	amt := db.InterpolateValue(in.AmountYears)
	return r.ConvertToDefaultUnits(amt, db)
}

// GetShare returns the share of this input (usually as part of an input group).
func (in *Input) GetShare(db *DB) *unit.Unit {
	if len(in.Share.ValueYears) == 0 {
		panic(fmt.Sprintf("No share for Input %#v.", in))
	}
	return db.InterpolateValue(in.Share.ValueYears)
}

// GetSource gets the source of the current input. It return the upstream
// process and the upstream pathway.
func (in *Input) GetSource(path *Pathway, db *DB) (slca.Process, *Pathway) {
	switch in.Source {
	case "Previous":
		for _, e := range path.Edges {
			if e.InputID == in.ID {
				proc, outPath := e.GetOutputVertex(db).GetProcess(path,
					in.GetResource(db), db)
				return proc, outPath
			}
		}
	case "Mix":
		return db.GetMix(in.Mix), &mixPathway
	case "Well":
		return nil, nil
	case "Pathway":
		for _, p := range db.Data.Pathways {
			if p.ID == in.Pathway {
				return p.GetOutputProcess(in.GetResource(db), db), p
			}
		}
	}
	panic(fmt.Errorf("couldn't find source for Input id=%s; Pathway: %v",
		in.ID, path.ID))
}

// EmissionsAndResourceUse calculates emissions caused and resources used by
// this input.
func (in *Input) EmissionsAndResourceUse(r *slca.OnsiteResults, proc slca.Process,
	path *Pathway, noncombustion subprocess, db *DB) {

	amount := in.GetAmount(db)
	in.emisResUsingAmount(amount, r, proc, path, noncombustion, db)
}

// emisResUsingAmount calculates emissions and resource use using a supplied
// input amount.
func (in *Input) emisResUsingAmount(amount *unit.Unit, r *slca.OnsiteResults,
	proc slca.Process, path *Pathway, nonCombustion subprocess, db *DB) {

	// Calculate resource use.
	ir := in.GetResource(db)
	amountDefault := ir.ConvertToDefaultUnits(amount, db)

	if in.AccountedInEnergyBalance != "False" {
		// Calculate upstream requirements.
		upProc, upPath := in.GetSource(path, db)
		// If it's an internal product, we don't count the energy use.
		r.AddResource(noSubprocess, ir, amountDefault, db)

		if upProc != nil && upPath != nil {
			// If one of the above is nil, the source is a well.
			upOut := upProc.GetOutput(ir, db)
			r.AddRequirement(upProc, upPath, upOut, amountDefault, db)
		}
	}

	// Calculation (combustion) emissions from technologies.
	for _, ts := range in.TechnologyShares {
		tech := ts.GetTechnology(db)
		resource := tech.GetInputResource(db)
		energyamt := resource.ConvertToEnergy(amount, db)
		gases, emissions := tech.GetEmissions(db)
		share := ts.GetShare(db)
		for i, g := range gases {
			e := emissions[i]
			if e.Dimensions().Matches(unit.Kilogram) {
				// Apparently, for Inputs in an input group, sometimes emissions are in units
				// of mass instead of mass per energy, so we check before multiplying them by
				// the total energy.
				r.AddEmission(tech, g, unit.Mul(e, share))
			} else {
				r.AddEmission(tech, g, unit.Mul(e, energyamt, share))
			}
		}
	}

	// Add non-combustion emissions from the input.
	gases, rates := in.GetEmissionRatios(db)
	for i, g := range gases {
		rate := rates[i]
		r.AddEmission(nonCombustion, g, unit.Mul(rate, ir.ConvertToMass(amount, db)))
	}
}

type subprocess struct {
	name string
	scc  slca.SCC
}

func (sp subprocess) GetName() string  { return sp.name }
func (sp subprocess) GetSCC() slca.SCC { return sp.scc }

var noSubprocess = subprocess{name: "--", scc: "--"}
