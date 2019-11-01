package greet

import (
	"fmt"
	"sync"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/unit"
)

// StationaryProcess is a holder for stationary process data
// from the GREET database.
type StationaryProcess struct {
	sync.RWMutex            `xml:"-"`
	ID                      ModelID          `xml:"id,attr"`
	Name                    string           `xml:"name,attr"`
	IOCarbonMap             *IOCarbonMap     `xml:"io-carbon-map"`
	PreferredFunctionalUnit *Unit            `xml:"prefered_functional_unit"`
	Outputs                 []*Output        `xml:"output"`
	Inputs                  []*Input         `xml:"input"`
	InputGroups             []*InputGroup    `xml:"group"`
	Coproducts              *Coproducts      `xml:"coproducts"`
	OtherEmissions          []*OtherEmission `xml:"other_emissions>emission"`
	Notes                   string           `xml:"notes,attr"`

	results map[*Pathway]map[OutputLike]*slca.OnsiteResults

	// Spatial reference information for this process
	SpatialReference *slca.SpatialRef

	// NoncombustionSCC is used for speciating non-combustion emissions.
	NoncombustionSCC slca.SCC
}

// SpatialRef returns the spatial reference for the receiver.
func (p *StationaryProcess) SpatialRef(aqm string) *slca.SpatialRef {
	r := *p.SpatialReference
	r.AQM = aqm
	return &r
}

// GetOutput gets the output (or coproduct) of this process that outputs the
// given resource. It assumes that there is only one output or coproduct for
// each resource.
// It panics if none of the outputs output the given resource.
func (p *StationaryProcess) GetOutput(rI slca.Resource, db slca.LCADB) slca.Output {
	r := rI.(*Resource)
	for _, o := range p.Outputs {
		if o.GetResource(db).(*Resource).IsCompatible(r) {
			return o
		}
	}
	if p.Coproducts != nil {
		for _, cp := range p.Coproducts.Coprods {
			// Allocated coproducts are treated as outputs, but displaced
			// coproducts are not.
			if cp.Method == "allocation" && cp.GetResource(db).(*Resource).IsCompatible(r) {
				return cp
			}
		}
	}
	panic(fmt.Errorf("Process %s doesn't output resource %v.", p.Name, r.ID))
}

// GetMainOutput gets the main output of this process. It panics if there is more than one main output.
func (p *StationaryProcess) GetMainOutput(db slca.LCADB) slca.Output {
	if len(p.Outputs) != 1 {
		panic(fmt.Errorf("Incorrect number of outputs for %#v.", p))
	}
	return p.Outputs[0]
}

// GetID returns the ID of this process.
func (p *StationaryProcess) GetID() ModelID {
	return p.ID
}

// GetIDStr returns the ID of this process in string format.
func (p *StationaryProcess) GetIDStr() string {
	return "Stationary" + string(p.ID)
}

// GetName returns the name of this process.
func (p *StationaryProcess) GetName() string {
	return p.Name
}

// Type returns the process type.
func (p *StationaryProcess) Type() slca.ProcessType {
	return slca.Stationary
}

// IOCarbonMap is a holder for information about biogenic and fossil carbon
type IOCarbonMap struct {
	Outputs []*IOCarbonOutput `xml:"output"`
}

// IOCarbonOutput is a holder for information about biogenic and fossil carbon
type IOCarbonOutput struct {
	ID     slca.OutputID  `xml:"id,attr"`
	Inputs *IOCarbonInput `xml:"input"`
}

// IOCarbonInput is a holder for the fraction biogenic vs. fossil carbon.
type IOCarbonInput struct {
	ID    InputID `xml:"id,attr"`
	Ratio float64 `xml:"ratio,attr"`
}

// InputGroup is a holder for a group of inputs sharing a common efficiency
// or adding up to a single amount.
type InputGroup struct {
	Type        string       `xml:"type,attr"` // This should be "efficiency" or "amount"
	Efficiency  []*ValueYear `xml:"efficiency>year"`
	Amount      []*ValueYear `xml:"amount>year"`
	InputShares []*Input     `xml:"shares>input"`
	Inputs      []*Input     `xml:"input"`
}

// value gets the value (either amount or efficiency) associated with this input group
// For efficiency, the value is the output amount (without accounting for
// any losses) divided by the efficiency.
func (ig *InputGroup) value(output OutputLike, db *DB) *unit.Unit {
	if ig.Type == "efficiency" {
		eff := db.InterpolateValue(ig.Efficiency)
		outAmt := output.GetAmountBeforeLoss(db)
		if len(ig.Inputs) == 0 {
			// If there aren't any amount inputs in the efficiency group, the group
			// amount is the output amount divided by the efficiency.
			return unit.Div(outAmt, eff)
		}
		// If there are amount inputs in the efficiency group, the group amount is
		// the the output amount divided by the efficiency minus the sum of the
		// input amounts.
		var inAmt *unit.Unit
		for _, i := range ig.Inputs {
			inAmt = unit.Add(inAmt, i.GetAmount(db))
		}
		return unit.Sub(unit.Div(outAmt, eff), inAmt)
	} else if ig.Type == "amount" {
		return db.InterpolateValue(ig.Amount)
	}
	panic("unsupported type `" + ig.Type + "`")
}

// inputRequirement calculates the resource requirements for this input group.
func (ig *InputGroup) inputRequirement(output OutputLike, db *DB) (inp []*Input, amount []*unit.Unit) {
	total := ig.value(output, db)
	for _, i := range ig.InputShares {
		inp = append(inp, i)
		amount = append(amount, unit.Mul(total, i.GetShare(db)))
	}
	for _, i := range ig.Inputs {
		inp = append(inp, i)
		amount = append(amount, i.GetAmount(db))
	}
	return
}

// EmissionsAndResourceUse calculates emissions caused and resources used by
// this input group. The returned value is a pointer and is cached for future
// use, so be sure to clone the result before modifying it.
func (ig *InputGroup) EmissionsAndResourceUse(r *slca.OnsiteResults, proc slca.Process,
	path *Pathway, output OutputLike, noncombustion subprocess, db *DB) {

	inputs, amounts := ig.inputRequirement(output, db)
	for i, input := range inputs {
		amount := amounts[i]
		input.emisResUsingAmount(amount, r, proc, path, noncombustion, db)
	}
}

// OnsiteResults calculates the onsite emissions from and resource
// use of this process per unit output, where "output" is the output that is
// required from the process, "path" is the pathway the process is a part of.
func (p *StationaryProcess) OnsiteResults(pathI slca.Pathway, outputI slca.Output, lcadb slca.LCADB) *slca.OnsiteResults {
	db := lcadb.(*DB)
	output := outputI.(OutputLike)
	path := pathI.(*Pathway)
	p.Lock()
	defer p.Unlock()
	if p.results == nil {
		p.results = make(map[*Pathway]map[OutputLike]*slca.OnsiteResults)
	}
	// Return the saved results if they have already been calculated.
	if rr, ok := p.results[path][output]; ok {
		return rr
	}
	r := slca.NewOnsiteResults(db)

	p.checkOutput(output)

	noncombustion := subprocess{name: "Non-combustion", scc: DefaultNonCombustionSCC} // scc: p.NoncombustionSCC}

	// Add in "other" (typically noncombustion) emissions
	for _, e := range p.OtherEmissions {
		r.AddEmission(noncombustion, e.Gas(db), e.Amount(db))
	}
	// Add resources that come in and associated emissions.
	for _, input := range p.Inputs {
		input.EmissionsAndResourceUse(r, p, path, noncombustion, db)
	}
	for _, ig := range p.InputGroups {
		ig.EmissionsAndResourceUse(r, p, path, output, noncombustion, db)
	}

	// Add in emissions from losses.
	gases, emissions := output.GetLossEmissions(db)
	for i, g := range gases {
		r.AddEmission(subprocess{name: "Losses", scc: p.SpatialReference.SCCs[0]}, g, emissions[i])
	}

	// deal with coproducts before subtracting outputs but
	// after everything else.
	p.dealWithCoproducts(r, output, db)

	outputAmount := output.GetResource(db).ConvertToDefaultUnits(
		output.GetAmount(db), db)

	// Subtract resources that go out. Only subract primary outputs
	// (not coproducts) and only subract when there is a corresponding input
	// of the resource.
	// TODO: It is not clear why only non-coproduct resources get subtracted,
	// but this is necessary to match the GREET.net results.
	if !output.IsCoproduct() {
		ores := output.GetResource(db)
		if _, ok := r.Resources[noSubprocess][ores]; ok {
			r.SubResource(noSubprocess, ores, outputAmount, db)
		}
	}

	// Divide the results by the output amount to get results per unit output.
	r.Div(outputAmount)

	if _, ok := p.results[path]; !ok {
		p.results[path] = make(map[OutputLike]*slca.OnsiteResults)
	}
	// Save results for future use.
	p.results[path][output] = r
	return r
}

// checkOutput makes sure output o is part of process p.
func (p *StationaryProcess) checkOutput(o OutputLike) {
	for _, oo := range p.Outputs {
		if oo.ID == o.GetID() {
			return
		}
	}
	if p.Coproducts != nil {
		for _, cp := range p.Coproducts.Coprods {
			if cp.GetID() == o.GetID() {
				return
			}
		}
	}
	panic(fmt.Errorf("output %v is not in process %v", o.GetID(), p.ID))
}

// dealWithCoproducts calculates deals with coproduct allocation and
// displacement. Allocation occurs before output is subtracted.
func (p *StationaryProcess) dealWithCoproducts(r *slca.OnsiteResults,
	output OutputLike, db *DB) {

	// Subtract coproduct outputs.
	if p.Coproducts != nil {
		// We consider displaced coproducts as upstream here because the amount
		// of this process that is required impacts the amount produced by the
		// displaced process.
		p.Coproducts.Displacement(r, db)

		// Perform the allocation for any coproducts that are allocated.
		oVal := output.CalcAllocationAmount(output.GetAmount(db).Dimensions(),
			p.Coproducts.AllocationMethod, db)
		allocTotal := oVal.Clone()
		for _, o := range p.Outputs {
			if o.GetID() != output.GetID() {
				allocTotal = unit.Add(
					o.CalcAllocationAmount(o.GetAmount(db).Dimensions(),
						p.Coproducts.AllocationMethod, db), allocTotal)
			}
		}
		for _, cp := range p.Coproducts.Coprods {
			if cp.Method == "allocation" {
				if cp.GetID() != output.GetID() {
					allocTotal = unit.Add(
						cp.CalcAllocationAmount(cp.GetAmount(db).Dimensions(),
							p.Coproducts.AllocationMethod, db), allocTotal)
				}
			}
		}
		r.Mul(unit.Div(oVal, allocTotal))
	}
}
