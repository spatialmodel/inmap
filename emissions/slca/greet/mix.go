package greet

import (
	"fmt"
	"sync"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/unit"
)

// Mix is a holder for GREET data about pathway mixes.
type Mix struct {
	sync.RWMutex
	ID                      ModelID        `xml:"id,attr"`
	Name                    string         `xml:"name,attr"`
	UseDefaultValues        bool           `xml:"use_default_values,attr"`
	ShareType               string         `xml:"share_type,attr"` // "energy" or "mass"
	CreatedResource         ResourceID     `xml:"created_resource,attr"`
	PathwayRefs             []*MixPathway  `xml:"pathway"`
	MixRefs                 []*MixResource `xml:"resource"`
	PreferredFunctionalUnit *Unit          `xml:"prefered_functional_unit"`
	Outputs                 []*Output      `xml:"output"`

	results *slca.OnsiteResults
}

var mixPathway = Pathway{Name: "Mix", ID: "Mix"}

// MainProcessAndOutput returns the process that outputs the main
// output of the receiver, and also returns that output.
func (m *Mix) MainProcessAndOutput(db slca.LCADB) (slca.Process, slca.Output) {
	output := m.GetMainOutput(db.(*DB))
	return m, output
}

// GetOutputProcess returns the receiver.
func (m *Mix) GetOutputProcess(_ *Resource, _ slca.LCADB) slca.Process {
	return m
}

// GetOutput returns the output of this mix that outputs the requested
// resource.
func (m *Mix) GetOutput(r slca.Resource, db slca.LCADB) slca.Output {
	o := m.GetMainOutput(db)
	if !r.(*Resource).IsCompatible(o.GetResource(db).(*Resource)) {
		panic(fmt.Errorf("Created resource in %#v doesn't match requested resource %#v.", m, r))
	}
	return o
}

// GetMainOutput returns the main output from this mix.
func (m *Mix) GetMainOutput(db slca.LCADB) slca.Output {
	if len(m.Outputs) != 1 {
		panic(fmt.Errorf("Incorrect number of outputs for %#v.", m))
	}
	// Fill in the output information that is missing for mixes.
	m.Outputs[0].ResourceID = m.CreatedResource
	amountyear := &ValueYear{
		Value: expression{
			val:   "1",
			units: m.ShareType,
		}.encode(),
		Year: "0",
	}
	m.Outputs[0].AmountYears = []*ValueYear{amountyear}
	return m.Outputs[0]
}

// GetIDStr returns the mix ID in string form.
func (m *Mix) GetIDStr() string {
	return "Mix" + string(m.ID)
}

// GetID returns the mix ID
func (m *Mix) GetID() ModelID {
	return m.ID
}

// GetName returns the mix name.
func (m *Mix) GetName() string {
	return m.Name
}

// Type returns the type of the process.
func (m *Mix) Type() slca.ProcessType {
	return slca.NoSpatial
}

// MixPathway is a holder for the corresponding GREET
// datatype. It defines the pathways that are upstream of
// the current mix.
type MixPathway struct {
	Ref      string        `xml:"ref,attr"`
	OutputID slca.OutputID `xml:"output,attr"`
	Shares   []*ValueYear  `xml:"shares>year"`
	Notes    string        `xml:"notes,attr"`
}

// MixResource is a holder for the corresponding GREET
// datatype. It defines the mixes that are upstream of the
// current mix.
type MixResource struct {
	Mix    ModelID      `xml:"mix,attr"`
	Shares []*ValueYear `xml:"shares>year"`
	Notes  string       `xml:"notes,attr"`
}

// OnsiteResults is required for a mix to fulfill the slca.Process interface,
// but pathways don't directly create any emissions.
// The returned value is a pointer and is cached for future use, so be sure to
// clone the result before modifying it.
func (m *Mix) OnsiteResults(_ slca.Pathway, o slca.Output, lcadb slca.LCADB) *slca.OnsiteResults {
	db := lcadb.(*DB)
	output := o.(*Output)
	m.checkOutput(output)

	m.Lock()
	defer m.Unlock()
	// Return saved results if they have already been calculated.
	if m.results != nil {
		return m.results
	}
	r := slca.NewOnsiteResults(db)

	var outputAmt *unit.Unit
	outputResource := output.GetResource(db)
	switch m.ShareType {
	case "energy":
		outputAmt = outputResource.(*Resource).ConvertToEnergy(output.GetAmount(db), db)
	case "mass":
		outputAmt = outputResource.(*Resource).ConvertToMass(output.GetAmount(db), db)
	case "volume":
		outputAmt = outputResource.(*Resource).ConvertToVolume(output.GetAmount(db), db)
	default:
		panic(fmt.Errorf("In mix %v, unknown share type %v.", m.ID, m.ShareType))
	}
	for _, mp := range m.PathwayRefs {
		foundOutput := false
		for _, path := range db.Data.Pathways {
			if o := path.GetMainOutput(db); mp.OutputID == o.(*Output).ID {
				share := db.InterpolateValue(mp.Shares)
				v := unit.Mul(share, outputAmt)
				resource := o.GetResource(db)
				proc := path.GetOutput(resource.(*Resource), db).(OutputLike).GetProcess(path, db)
				procOut := proc.GetOutput(resource, db)
				r.AddRequirement(proc, path, procOut, v, db)
				foundOutput = true
			}
		}
		if !foundOutput {
			panic(fmt.Sprintf("Couldn't find output into mix pathway %#v.\n", mp))
		}
	}
	for _, mr := range m.MixRefs {
		foundMix := false
		for _, mix := range db.Data.Mixes {
			if mix.ID == mr.Mix {
				o := mix.GetMainOutput(db)
				share := db.InterpolateValue(mr.Shares)
				v := unit.Mul(share, outputAmt)
				r.AddRequirement(mix, &mixPathway, o, v, db)
				foundMix = true
			}
		}
		if !foundMix {
			panic(fmt.Sprintf("Couldn't find mix output into mix resource %#v.\n", mr))
		}
	}

	// Divide the results by the output amount to get results per unit output.
	r.Div(output.GetResource(db).ConvertToDefaultUnits(output.GetAmount(db), db))

	// Save results for future use.
	m.results = r
	return r
}

// checkOutput makes sure output o is part of mix m.
func (m *Mix) checkOutput(o *Output) {
	for _, oo := range m.Outputs {
		if oo.ID == o.ID {
			return
		}
	}
	panic(fmt.Errorf("output %v is not in process %v", o.ID, m.ID))
}

// GetMix returns the Mix with the given ID.
func (db *DB) GetMix(ID ModelID) *Mix {
	for _, m := range db.Data.Mixes {
		if m.ID == ID {
			return m
		}
	}
	panic(fmt.Errorf("greet: no mix ID '%s' in database", ID))
}

// SpatialRef returns this spatial reference for this mix (which is NoSpatial).
func (m *Mix) SpatialRef(aqm string) *slca.SpatialRef {
	return &slca.SpatialRef{NoSpatial: true, Type: slca.NoSpatial, AQM: aqm}
}
