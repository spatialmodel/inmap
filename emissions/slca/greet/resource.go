package greet

import (
	"fmt"
	"strings"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/unit"
)

// ResourceGroup is a group of resources.
type ResourceGroup struct {
	ID        string `xml:"id,attr"`
	Name      string `xml:"name"`
	IncludeIn string `xml:"include_in"`
}

// Resource is a holder for the GREET model Resource type. Refer to the GREET
// documentation for more information.
type Resource struct {
	HiddenAsMain    bool                `xml:"hidden_as_main,attr"`
	CanBePrimary    bool                `xml:"can_be_primary,attr"`
	MarketValue     Expression          `xml:"market_value,attr"`
	Density         Expression          `xml:"density,attr"`
	HeatingValueHHV Expression          `xml:"heating_value_hhv,attr"`
	HeatingValueLHV Expression          `xml:"heating_value_lhv,attr"`
	Temperature     string              `xml:"temperature,attr"`
	Pressure        string              `xml:"pressure,attr"`
	ID              ResourceID          `xml:"id,attr"`
	Name            string              `xml:"name,attr"`
	CRatio          Expression          `xml:"c_ratio,attr"`
	SRatioDefault   Expression          `xml:"s_ratio,attr"`
	SRatioYear      []*ValueYear        `xml:"s_ratio>year"`
	State           string              `xml:"state,attr"`
	Family          string              `xml:"family,attr"`
	NickName        []string            `xml:"nick_name"`
	Membership      []*Membership       `xml:"membership"`
	Compatibility   []*Compatibility    `xml:"compatibility"`
	Evaporation     []*EvaporationShare `xml:"evaporation>gas"`
}

// EvaporationShare gives the fraction of a given resource that evaporates as
// a given gas.
type EvaporationShare struct {
	Ref   GasID      `xml:"ref,attr"`
	Share Expression `xml:"share,attr"`
}

// GetEvaporationShares returns the gas that this resource evaporates to,
// and the fraction that is evaporated.
func (r *Resource) GetEvaporationShares(db *DB) ([]*Gas, []*unit.Unit) {
	// TODO: The commented code below gets evaporation shares from compatiable
	// resources if the current resource doesn't have any defined. This seems
	// like a good idea but the GREET.net model doesn't do this. For example,
	// E85 doesn't have any evaporation shares, but it should have VOC evaporation,
	// which it would using the scheme mentioned here.
	//if len(r.Evaporation) == 0 {
	//	for _, c := range r.Compatibility {
	//		rr := db.GetResource(c.MatID, r)
	//		if len(rr.Evaporation) != 0 {
	//			return rr.GetEvaporationShares(db)
	//		}
	//	}
	//}
	var gases []*Gas
	var shares []*unit.Unit
	for _, e := range r.Evaporation {
		gases = append(gases, db.getGasFromID(e.Ref))
		shares = append(shares, db.evalExpr(e.Share))
	}
	return gases, shares
}

// GetID gets the ID of this Resource.
func (r *Resource) GetID() string {
	return "Resource" + string(r.ID)
}

// GetName gets the name of this resource.
func (r *Resource) GetName() string {
	return r.Name
}

// Membership gives a group that a resource is a member of.
type Membership struct {
	GroupID string `xml:"group_id"`
}

// Compatibility gives another resource this resource is compatible with.
type Compatibility struct {
	MatID ResourceID `xml:"mat_id,attr"`
}

// IsCompatible returns true if two resources are compatible.
func (r *Resource) IsCompatible(other *Resource) bool {
	if r.ID == other.ID {
		return true
	}
	if strings.Index(r.Name, other.Name) >= 0 ||
		strings.Index(other.Name, r.Name) >= 0 {
		return true
	}
	for _, c := range r.Compatibility {
		if other.ID == c.MatID {
			return true
		}
	}
	for _, c := range other.Compatibility {
		if r.ID == c.MatID {
			return true
		}
	}
	return false
}

// GetDensity returns the density of this resource.
func (r *Resource) GetDensity(db *DB) *unit.Unit {
	return db.evalExpr(r.Density)
}

// getHHV returns the higher heating value for r resource.
func (r *Resource) getHHV(db *DB) *unit.Unit {
	return db.evalExpr(r.HeatingValueHHV)
}

// getLHV returns the lower heating value for this resource.
func (r *Resource) getLHV(db *DB) *unit.Unit {
	return db.evalExpr(r.HeatingValueLHV)
}

// GetMarketValue returns the market value for this resource.
func (r *Resource) GetMarketValue(db *DB) *unit.Unit {
	return db.evalExpr(r.MarketValue)
}

func (r *Resource) getHHVMass(db *DB) *unit.Unit {
	hhv := r.getHHV(db)
	if hhv.Dimensions().Matches(joulesPerM3) {
		return unit.Div(hhv, r.GetDensity(db))
	} else if hhv.Dimensions().Matches(joulesPerKg) {
		return hhv
	} else {
		panic("Unknown HHV type")
	}
}

func (r *Resource) getHHVVolume(db *DB) *unit.Unit {
	hhv := r.getHHV(db)
	if hhv.Dimensions().Matches(joulesPerM3) {
		return hhv
	} else if hhv.Dimensions().Matches(joulesPerKg) {
		return unit.Mul(hhv, r.GetDensity(db))
	} else {
		panic("Unknown HHV type")
	}
}

func (r *Resource) getLHVMass(db *DB) *unit.Unit {
	lhv := r.getLHV(db)
	if lhv.Dimensions().Matches(joulesPerM3) {
		return unit.Div(lhv, r.GetDensity(db))
	} else if lhv.Dimensions().Matches(joulesPerKg) {
		return lhv
	} else {
		panic("Unknown LHV type")
	}
}

func (r *Resource) getLHVVolume(db *DB) *unit.Unit {
	lhv := r.getLHV(db)
	if lhv.Dimensions().Matches(joulesPerM3) {
		return lhv
	} else if lhv.Dimensions().Matches(joulesPerKg) {
		return unit.Mul(lhv, r.GetDensity(db))
	} else {
		panic("Unknown LHV type")
	}
}

// GetHeatingValueMass returns the mass-specific heating value for this resource.
func (r *Resource) GetHeatingValueMass(db *DB) *unit.Unit {
	if db.BasicParameters.LHV == true {
		return r.getLHVMass(db)
	}
	return r.getHHVMass(db)
}

// GetHeatingValueVolume returns the volume-specific heating value for this resource.
func (r *Resource) GetHeatingValueVolume(db *DB) *unit.Unit {
	if db.BasicParameters.LHV == true {
		return r.getLHVVolume(db)
	}
	return r.getHHVVolume(db)
}

// GetSRatio returns the fraction sulfur in this resource.
func (r *Resource) GetSRatio(db *DB) *unit.Unit {
	if len(r.SRatioYear) > 0 {
		return db.InterpolateValue(r.SRatioYear)
	}
	if r.SRatioDefault == "" {
		return unit.New(0, unit.Dimless)
	}
	return db.evalExpr(r.SRatioDefault)
}

// GetCRatio returns the fraction carbon in this resource.
func (r *Resource) GetCRatio(db *DB) *unit.Unit {
	return db.evalExpr(r.CRatio)
}

// ConvertToDefaultUnits converts the amount of the resource to the default units
// for this resource: energy for resources in in the "energy" state and mass for all other states.
// If the resource doesn't contain enough information to make the conversion, however, the
// original value will be returned.
func (r *Resource) ConvertToDefaultUnits(amt *unit.Unit, dbI slca.LCADB) *unit.Unit {
	d := amt.Dimensions()
	db := dbI.(*DB)
	if d.Matches(dollars) {
		return r.ConvertToDefaultUnits(unit.Div(amt, r.GetMarketValue(db)), db)
	}
	if r.State == "energy" { // Default units are Joules.
		switch {
		case d.Matches(unit.Kilogram):
			if (!db.BasicParameters.LHV && r.HeatingValueHHV == "") ||
				(db.BasicParameters.LHV && r.HeatingValueLHV == "") {
				return amt
			}
			return unit.Mul(amt, r.GetHeatingValueMass(db))
		case d.Matches(unit.Meter3):
			if (!db.BasicParameters.LHV && r.HeatingValueHHV == "") ||
				(db.BasicParameters.LHV && r.HeatingValueLHV == "") {
				return amt
			}
			return unit.Mul(amt, r.GetHeatingValueVolume(db))
		case d.Matches(unit.Joule):
			return amt
		default:
			panic(fmt.Errorf("Unsupported units `%v` for converting %s "+
				"(resource %v) to default units.", d, r.Name, r.ID))
		}
	}
	switch { // Default units are kilograms.
	case d.Matches(unit.Kilogram): // already in mass units
		return amt
	case d.Matches(unit.Meter3):
		if r.Density == "" {
			return amt
		}
		return unit.Mul(amt, r.GetDensity(db))
	case d.Matches(unit.Joule):
		if (!db.BasicParameters.LHV && r.HeatingValueHHV == "") ||
			(db.BasicParameters.LHV && r.HeatingValueLHV == "") {
			return amt
		}
		return unit.Div(amt, r.GetHeatingValueMass(db))
	case d.Matches(unit.Meter):
		return amt
	default:
		panic(fmt.Errorf("Unsupported units `%v` for converting %s "+
			"(resource %v) to default units.", d, r.Name, r.ID))
	}
}

// ConvertToMass converts an amount of this resource to mass units.
func (r *Resource) ConvertToMass(amt *unit.Unit, db *DB) *unit.Unit {
	d := amt.Dimensions()
	switch {
	case d.Matches(unit.Kilogram): // already in mass units
		return amt
	case d.Matches(unit.Meter3):
		return unit.Mul(amt, r.GetDensity(db))
	case d.Matches(unit.Joule):
		return unit.Div(amt, r.GetHeatingValueMass(db))
	default:
		panic(fmt.Errorf("Unsupported units `%v`. for %#v.", d, r))
	}
}

// ConvertToVolume converts an amount of this resource to volume units.
func (r *Resource) ConvertToVolume(amt *unit.Unit, db *DB) *unit.Unit {
	d := amt.Dimensions()
	switch {
	case d.Matches(unit.Kilogram):
		return unit.Div(amt, r.GetDensity(db))
	case d.Matches(unit.Meter3):
		return amt
	case d.Matches(unit.Joule):
		return unit.Div(amt, r.GetHeatingValueVolume(db))
	default:
		panic(fmt.Errorf("Unsupported units `%v`. for %#v.", d, r))
	}
}

// ConvertToEnergy converts an amount of this resource to energy units.
func (r *Resource) ConvertToEnergy(amt *unit.Unit, db *DB) *unit.Unit {
	d := amt.Dimensions()
	switch {
	case d.Matches(unit.Kilogram):
		return unit.Mul(amt, r.GetHeatingValueMass(db))
	case d.Matches(unit.Meter3):
		return unit.Mul(amt, r.GetHeatingValueVolume(db))
	case d.Matches(unit.Joule):
		return amt
	default:
		panic(fmt.Errorf("Unsupported units `%v`. for %#v.", d, r))
	}
}

// ConvertToMarketValue converts an amount of this resource to its market value
func (r *Resource) ConvertToMarketValue(amt *unit.Unit, db *DB) *unit.Unit {
	value := r.GetMarketValue(db)
	vd := value.Dimensions()
	switch {
	case vd.Matches(dollarsPerM3):
		return unit.Mul(r.ConvertToVolume(amt, db), value)
	case vd.Matches(dollarsPerKg):
		return unit.Mul(r.ConvertToMass(amt, db), value)
	case vd.Matches(dollarsPerJoule):
		return unit.Mul(r.ConvertToEnergy(amt, db), value)
	default:
		panic(fmt.Errorf("Unsupported market value `%v`. for %#v.", value, r))
	}
}
