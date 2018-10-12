package greet

import (
	"fmt"

	"github.com/ctessum/unit"
)

// Step is a holder for a step in a transportation process.
type Step struct {
	DestRef      LocationID  `xml:"dest_ref,attr"`
	OriginRef    LocationID  `xml:"origin_ref,attr"`
	Ref          ModeID      `xml:"ref,attr"`
	Distance     Param       `xml:"distance"`
	Share        Param       `xml:"share"`
	FuelShareRef FuelShareID `xml:"fuel_share_ref,attr"`
	BackHaul     bool        `xml:"back_haul,attr"` // Is an empty backhaul required?
	ID           string      `xml:"id,attr"`
	NLoss        *NLoss      `xml:"nloss"`
}

// Locations is a holder for transportation source and
// destination locations.
type Locations struct {
	LocationGroups []LocationGroup `xml:"groups>group"`
	Locations      []*Location     `xml:"location"`
}

// Location holds information about a transportation
// source or destination location.
type Location struct {
	Name       string                `xml:"name,attr"`
	Picture    string                `xml:"picture,attr"`
	ID         LocationID            `xml:"id,attr"`
	Notes      string                `xml:"notes,attr"`
	Membership []*LocationMembership `xml:"membership"`
}

// LocationGroup holds information about how different locations
// can be combined into groups.
type LocationGroup struct {
	Name  string          `xml:"name,attr"`
	ID    LocationGroupID `xml:"id,attr"`
	Notes string          `xml:"notes,attr"`
}

// LocationMembership gives information about which group a
// location belongs to.
type LocationMembership struct {
	GroupID LocationGroupID `xml:"group_id,attr"`
}

// GetDistance returns the distance traveled by this step.
func (s *Step) GetDistance(db *DB) *unit.Unit {
	return db.InterpolateValue(s.Distance.ValueYears)
}

// GetShare gets the share of the product that is transported by this step.
func (s *Step) GetShare(db *DB) *unit.Unit {
	return db.InterpolateValue(s.Share.ValueYears)
}

// GetNLoss returns the fraction of product that is lost during this step.
func (s *Step) GetNLoss(db *DB) *unit.Unit {
	if s.NLoss == nil {
		return unit.New(0, unit.Dimless)
	}
	return db.evalExpr(s.NLoss.Rate)
}

// GetLossAmount calculates the loss of the transported resource during this
// step.
func (s *Step) GetLossAmount(res *Resource, amountTransported *unit.Unit,
	db *DB) *unit.Unit {
	loss := s.GetNLoss(db)
	amtTrans := res.ConvertToMass(amountTransported, db)
	return unit.Mul(loss, amtTrans)
}

// GetLossEmissions calculates emissions through evaporation of a product (or other losses).
func (s *Step) GetLossEmissions(res *Resource, amountTransported *unit.Unit, db *DB) (
	[]*Gas, []*unit.Unit) {

	loss := s.GetNLoss(db)
	if loss.Value() == 0. {
		return nil, nil
	}
	amtTrans := res.ConvertToMass(amountTransported, db)
	gases, shares := res.GetEvaporationShares(db)
	amounts := make([]*unit.Unit, len(shares))
	for i, sh := range shares {
		amounts[i] = unit.Mul(loss, amtTrans, sh)
	}
	return gases, amounts
}

// GetModeAndFuelShare gets the mode and fuel share associated with this step
func (s *Step) GetModeAndFuelShare(db *DB) (*Mode, *FuelShare) {
	for _, m := range db.Data.Modes {
		if m.ID == s.Ref {
			for _, fs := range m.FuelShares {
				if fs.ID == s.FuelShareRef {
					return m, fs
				}
			}
		}
	}
	panic(fmt.Sprintf("Couldn't find mode and/or fuel share for %#v.", s))
}

// NLoss is a holder for product loss fraction during a transportation step.
type NLoss struct {
	Rate       Expression `xml:"rate,attr"`
	Dependency string     `xml:"dependency,attr"`
}
