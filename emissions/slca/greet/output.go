package greet

import (
	"fmt"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/unit"
)

// OutputLike is an interface that allows the use of outputs and coproducts
// together.
type OutputLike interface {
	IsCoproduct() bool
	GetName(*DB) string
	GetAmount(*DB) *unit.Unit
	GetAmountBeforeLoss(*DB) *unit.Unit
	GetResource(slca.LCADB) slca.Resource
	GetProcess(*Pathway, *DB) slca.Process
	GetID() slca.OutputID
	CalcAllocationAmount(outputUnits unit.Dimensions,
		AllocationMethod string, db *DB) *unit.Unit
	GetLossEmissions(*DB) ([]*Gas, []*unit.Unit)
}

// Output is a holder for the type and amount of a resource that is output from
// a process or pathway.
type Output struct {
	ID          slca.OutputID `xml:"id,attr"`
	ResourceID  ResourceID    `xml:"resource,attr"`
	Ref         ResourceID    `xml:"ref,attr"`
	AmountYears []*ValueYear  `xml:"amount>year"`
	NLoss       *NLoss        `xml:"nloss"`
}

// GetName returns the name out the resource output by this output.
func (o *Output) GetName(db *DB) string {
	return o.GetResource(db).GetName()
}

// IsCoproduct returns whether this output is a coproduct.
// It is an output, not a coproduct, the the answer is always
// false.
func (o *Output) IsCoproduct() bool {
	return false
}

// GetAmount calculates the amount of this output after accounting for losses.
func (o *Output) GetAmount(db *DB) *unit.Unit {
	val := db.InterpolateValue(o.AmountYears)
	return unit.Sub(val, unit.Mul(val, o.GetNLoss(db)))
}

// GetAmountBeforeLoss calculates the amount of this output without
// accounting for losses.
func (o *Output) GetAmountBeforeLoss(db *DB) *unit.Unit {
	return db.InterpolateValue(o.AmountYears)
}

// GetLossEmissions calculates emissions through evaporation of a product (or other losses).
func (o *Output) GetLossEmissions(db *DB) ([]*Gas, []*unit.Unit) {
	loss := o.GetNLoss(db)
	if loss.Value() == 0. {
		return nil, nil
	}
	res := o.GetResource(db).(*Resource)
	gases, shares := res.GetEvaporationShares(db)
	amounts := make([]*unit.Unit, len(shares))
	outputAmount := db.InterpolateValue(o.AmountYears)
	for i, s := range shares {
		amounts[i] = unit.Mul(res.ConvertToMass(outputAmount, db), loss, s)
	}
	return gases, amounts
}

// GetResource returns the resource associated with this output.
func (o *Output) GetResource(db slca.LCADB) slca.Resource {
	for _, r := range db.(*DB).Data.Resources {
		if r.ID == o.ResourceID || r.ID == o.Ref {
			return r
		}
	}
	panic(fmt.Sprintf("Couldn't find resource for Output %#v", o))
}

// GetNLoss returns the loss rate associated with this output.
func (o *Output) GetNLoss(db *DB) *unit.Unit {
	if o.NLoss == nil {
		return unit.New(0, unit.Dimless)
	}
	return db.evalExpr(o.NLoss.Rate)
}

// GetID returns the ID of this output.
func (o *Output) GetID() slca.OutputID {
	return o.ID
}

// Coproducts is a holder for data about the coproducts of a process and
// how to deal with them.
type Coproducts struct {
	// Either "Mass", "Energy", "Market", "Volume" or ""
	AllocationMethod string `xml:"allocation_method,attr"`

	Coprods []*Coproduct `xml:"coproduct"`
}

// Coproduct is a holder for information about a single coproduct.
type Coproduct struct {
	ID                   slca.OutputID `xml:"id,attr"`
	Ref                  ResourceID    `xml:"ref,attr"`
	AmountYears          []*ValueYear  `xml:"amount>year"`
	Method               string        `xml:"method,attr"` // either "allocation" or "displacement"
	ConventionalProducts []*Product    `xml:"conventional_products>product"`
}

// GetID returns the ID of this coproduct.
func (cp *Coproduct) GetID() slca.OutputID {
	return cp.ID
}

// Displacement calculates the amounts of different processes that are displaced by this process.
// The results are returned as negative numbers.
func (cps *Coproducts) Displacement(r *slca.OnsiteResults, db *DB) {
	for _, cp := range cps.Coprods {
		if cp.Method == "displacement" {
			for _, p := range cp.ConventionalProducts {
				resource := p.GetResource(db)
				displacedProcess, displacedPath := p.GetProcess(resource, db)
				dispO := displacedProcess.GetOutput(resource, db)
				displacedReq := unit.Mul(cp.GetAmount(db), p.GetRatio(db))
				displacedReq.Negate()
				r.AddRequirement(displacedProcess, displacedPath, dispO, displacedReq, db)
			}
		}
	}
}

// GetResource returns the resource associated with this coproduct.
func (cp *Coproduct) GetResource(db slca.LCADB) slca.Resource {
	for _, r := range db.(*DB).Data.Resources {
		if r.ID == cp.Ref {
			return r
		}
	}
	panic(fmt.Sprintf("Couldn't find resource for coproduct %#v.", cp))
}

// CalcAllocationAmount calculates the amount of this output to be used when
// allocating resource use and emissions. If AllocationMethod is "", it allocates
// all emissions to the main output and none to the coproducts.
func (o *Output) CalcAllocationAmount(outputUnit unit.Dimensions, AllocationMethod string,
	db *DB) *unit.Unit {

	res := o.GetResource(db).(*Resource)
	amt := o.GetAmount(db)
	switch AllocationMethod {
	case "Mass":
		return res.ConvertToMass(amt, db)
	case "Energy":
		return res.ConvertToEnergy(amt, db)
	case "Market":
		return res.ConvertToMarketValue(amt, db)
	case "Volume":
		return res.ConvertToVolume(amt, db)
	case "":
		return amt
	default:
		panic("Unknown allocation method: " + AllocationMethod)
	}
}

// CalcAllocationAmount calculates the amount of this coproduct to be used when
// allocating resource use and emissions. If AllocationMethod is "", it allocates
// all emissions to the main output and none to the coproducts.
func (cp *Coproduct) CalcAllocationAmount(outputUnits unit.Dimensions,
	AllocationMethod string, db *DB) *unit.Unit {

	res := cp.GetResource(db).(*Resource)
	amt := cp.GetAmount(db)
	switch AllocationMethod {
	case "Mass":
		return res.ConvertToMass(amt, db)
	case "Energy":
		return res.ConvertToEnergy(amt, db)
	case "Market":
		return res.ConvertToMarketValue(amt, db)
	case "Volume":
		return res.ConvertToVolume(amt, db)
	case "":
		return unit.New(0, outputUnits)
	default:
		panic("Unknown allocation method: " + AllocationMethod)
	}
}

// Product is a holder for information about the product that is being
// displaced by a coproduct.
type Product struct {
	Ref       ResourceID `xml:"ref,attr"`
	PathwayID ModelID    `xml:"pathway,attr"`
	MixID     ModelID    `xml:"mix,attr"`
	RatioStr  Expression `xml:"ratio,attr"`
}

// GetResource returns the resource associated with this product.
func (p *Product) GetResource(db slca.LCADB) *Resource {
	for _, r := range db.(*DB).Data.Resources {
		if r.ID == p.Ref {
			return r
		}
	}
	panic(fmt.Sprintf("Couldn't find resource for coproduct product %#v.", p))
}

// GetRatio returns the displacement ratio for this coproduct.
func (p *Product) GetRatio(db *DB) *unit.Unit {
	return db.evalExpr(p.RatioStr)
}

// GetProcess returns the process and pathway this coproduct is displacing.
func (p *Product) GetProcess(r *Resource, db *DB) (slca.Process, *Pathway) {
	if p.PathwayID != "" {
		for _, path := range db.Data.Pathways {
			if path.ID == p.PathwayID {
				proc := path.GetOutput(r, db).(OutputLike).GetProcess(path, db)
				return proc, path
			}
		}
	}
	if p.MixID != "" {
		for _, mix := range db.Data.Mixes {
			if mix.ID == p.MixID {
				return mix, &mixPathway
			}
		}
	}
	panic(fmt.Sprintf("Couldn't find process for coproduct product %#v.", p))
}

// GetAmount returns the amount of this coproduct that is produced.
func (cp *Coproduct) GetAmount(db *DB) *unit.Unit {
	return db.InterpolateValue(cp.AmountYears)
}

// GetAmountBeforeLoss calculates the amount of this coproduct is produced.
// Because coproducts do not have losses, it gives the same result as
// GetAmount().
func (cp *Coproduct) GetAmountBeforeLoss(db *DB) *unit.Unit {
	return cp.GetAmount(db)
}

// GetName gets the name associated with the resource produced by this
// coproduct.
func (cp *Coproduct) GetName(db *DB) string {
	return cp.GetResource(db).GetName()
}

// IsCoproduct is for implementing the OutputLike interface and is always true.
func (cp *Coproduct) IsCoproduct() bool {
	return true
}

// GetProcess returns the process associated with this output.
func (o *Output) GetProcess(path *Pathway, db *DB) slca.Process {
	// Find the corresponding process model
	for _, proc := range db.Data.StationaryProcesses {
		for _, oo := range proc.Outputs {
			if oo.ID == o.ID {
				return proc
			}
		}
	}
	for _, proc := range db.Data.TransportationProcesses {
		for _, oo := range proc.Outputs {
			if oo.ID == o.ID {
				return proc
			}
		}
	}
	panic(fmt.Sprintf("Output %#v has no process model.\n", o))
}

// GetProcess returns the process associated with this Coproduct.
func (cp *Coproduct) GetProcess(path *Pathway, db *DB) slca.Process {
	// Find the corresponding process model
	for _, proc := range db.Data.StationaryProcesses {
		for _, oo := range proc.Outputs {
			if oo.ID == cp.ID {
				return proc
			}
		}
	}
	for _, proc := range db.Data.TransportationProcesses {
		for _, oo := range proc.Outputs {
			if oo.ID == cp.ID {
				return proc
			}
		}
	}
	panic(fmt.Sprintf("Coproduct %#v has no process model.\n", cp))
}

// GetLossEmissions is required to fulfill the OutputLike interface, but
// coproducts don't have any loss emissions.
func (cp *Coproduct) GetLossEmissions(_ *DB) ([]*Gas, []*unit.Unit) {
	return nil, nil
}
