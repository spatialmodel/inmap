package greet

import (
	"fmt"
	"strconv"

	"github.com/ctessum/unit"
)

// Emission is a holder for emissions data
// in the GREET database.
type Emission struct {
	Ref        GasID      `xml:"ref,attr"`
	Factor     Expression `xml:"factor,attr"`
	Amount     Expression `xml:"amount,attr"`
	Calculated bool       `xml:"calculated,attr"`
}

// GetGas returns the gas associated with this emission
func (e *Emission) GetGas(db *DB) *Gas {
	return db.gasFromID(e.Ref)
}

// GetEmissionFactor returns the emissions factor for emission corresponding to
// index i in emissions.
// fuel is used to calculate some emissions factors based on fuel properties.
// It should be safe to pass a nil value for fuel for non-combustion emissions.
func GetEmissionFactor(emissions []*Emission, i int, fuel *Resource,
	db *DB) *unit.Unit {
	e := emissions[i]
	if !e.Calculated {
		if e.Factor != "" {
			return db.evalExpr(e.Factor)
		} else if e.Amount != "" {
			return db.evalExpr(e.Amount)
		} else {
			panic(fmt.Errorf("missing emission factor"))
		}
	}
	// Emissions need to be calculated.
	switch e.Ref {
	case "6": // SOx
		return unit.Div(fuel.GetSRatio(db), fuel.GetHeatingValueMass(db),
			e.GetGas(db).GetSRatio(db))
	case "9": // CO2
		// Carbon balancing from GREET.net model description page 3.
		v1 := unit.Div(fuel.GetCRatio(db), fuel.GetHeatingValueMass(db))
		var otherEmis = map[string]string{"VOC": "", "CO": "", "CH4": ""}
		for ii, ee := range emissions {
			if _, ok := otherEmis[ee.GetGas(db).Name]; ok {
				fac := GetEmissionFactor(emissions, ii, fuel, db)
				cr := ee.GetGas(db).GetCRatio(db)
				v1 = unit.Sub(v1, unit.Mul(fac, cr))
			}
		}
		return unit.Div(v1, e.GetGas(db).GetCRatio(db))
	default:
		panic(fmt.Sprintf("calculated emissions not implemented for gas %v in %#v. ",
			e.Ref, e))
	}
}

// EmissionYear is a holder for emissions information
// for a specific year in the GREET database.
type EmissionYear struct {
	Year      string      `xml:"value,attr"`
	Emissions []*Emission `xml:"emission"`
}

// GetYear converts the year of this emission from
// string to float format.
func (e *EmissionYear) GetYear() float64 {
	yy, err := strconv.ParseFloat(e.Year, 64)
	handle(err)
	return yy
}

// interpolateEmissions takes an array of emissions years and returns
// the emissions for analysis year that is set in db. The emissions must
// be sorted by year and have all of the same gases in the same order.
func (db *DB) interpolateEmissions(y []*EmissionYear, fuel *Resource) (
	[]*Gas, []*unit.Unit) {
	var gases []*Gas
	var amounts []*unit.Unit
	if len(y) < 1 {
		return nil, nil // no data to interpolate
	}
	yy := make([]getYearer, len(y))
	for i, yyy := range y {
		yy[i] = yyy
	}
	dbYear := db.GetYear()
	v1, v2, frac1, frac2 := db.interpolateYear(yy, dbYear)
	val1 := v1.(*EmissionYear)
	val2 := v2.(*EmissionYear)
	for j := range val1.Emissions {
		if val1.Emissions[j].Ref != val2.Emissions[j].Ref {
			panic("emissions don't match")
		}
		e1 := GetEmissionFactor(val1.Emissions, j, fuel, db)
		e2 := GetEmissionFactor(val2.Emissions, j, fuel, db)
		yEmis := unit.Add(unit.Mul(e1, frac1), unit.Mul(e2, frac2))
		gases = append(gases, val1.Emissions[j].GetGas(db))
		amounts = append(amounts, yEmis)
	}
	return gases, amounts
}

// Gas is a holder for emissions types in the GREET database.
type Gas struct {
	Ref                    string        `xml:"ref,attr"`
	Share                  string        `xml:"share,attr"`
	Name                   string        `xml:"name,attr"`
	ID                     GasID         `xml:"id,attr"`
	CRatio                 Expression    `xml:"c_ratio,attr"` // mass ratio
	SRatio                 Expression    `xml:"s_ratio,attr"` // mass ratio
	GlobalWarmingPotential string        `xml:"global_warming_potential,attr"`
	Membership             []*Membership `xml:"membership"`
}

// GasGroup is a holder for gas grouping information
// in the GREET database.
type GasGroup struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// GetSRatio returns the fraction sulfur in this gas.
func (g *Gas) GetSRatio(db *DB) *unit.Unit {
	return db.evalExpr(g.SRatio)
}

// GetCRatio returns the fraction carbon in this gas.
func (g *Gas) GetCRatio(db *DB) *unit.Unit {
	return db.evalExpr(g.CRatio)
}

// GetID returns the ID number for this
// gas
func (g *Gas) GetID() string {
	return "Gas" + string(g.ID)
}

// GetName returns the gas name.
func (g *Gas) GetName() string {
	return g.Name
}

// OtherEmission is a holder for "other" emissions from StationaryProcesses.
type OtherEmission struct {
	// TODO: What does MostRecent mean?
	MostRecent string       `xml:"mostRecent,attr"`
	Ref        GasID        `xml:"ref,attr"`
	ValueYears []*ValueYear `xml:"year"`
	Notes      string       `xml:"notes,attr"`
}

// Gas returns the gas associated with this emission
func (e *OtherEmission) Gas(db *DB) *Gas {
	for _, g := range db.Data.Gases {
		if g.ID == e.Ref {
			return g
		}
	}
	panic(fmt.Sprintf("Couldn't find gas for %#v.", e))
}

// Amount returns the amount of emission.
func (e *OtherEmission) Amount(db *DB) *unit.Unit {
	return db.InterpolateValue(e.ValueYears)
}
