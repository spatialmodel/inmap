package greet

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/ctessum/unit"
)

// ValueYear is a holder for a value that applies to a specific year.
type ValueYear struct {
	Value Expression `xml:"value,attr"`
	Year  string     `xml:"year,attr"`
}

// GetYear returns the year.
func (v *ValueYear) GetYear() float64 {
	yy, err := strconv.ParseFloat(v.Year, 64)
	handle(err)
	return yy
}

// GetValue returns the value.
func (v *ValueYear) GetValue(db *DB) *unit.Unit {
	if v.Value == "" {
		panic(fmt.Sprintf("Year %#v doesn't contain a value.", v))
	}
	return db.evalExpr(v.Value)
}

// InterpolateValue returns the value associated with the year set in the
// GREET database, interpolating if necessary.
func (db *DB) InterpolateValue(y []*ValueYear) *unit.Unit {
	yy := make([]getYearer, len(y))
	for i, yyy := range y {
		yy[i] = yyy
	}
	dbYear := db.GetYear()
	v1, v2, frac1, frac2 := db.interpolateYear(yy, dbYear)
	val1 := v1.(*ValueYear).GetValue(db)
	val2 := v2.(*ValueYear).GetValue(db)
	return unit.Add(unit.Mul(val1, frac1), unit.Mul(val2, frac2))
}

// InterpolateValueWithLag returns the value associated with
// the year set in the GREET database minus the specified lag (in years),
// interpolating if necessary.
func (db *DB) InterpolateValueWithLag(y []*ValueYear, lag float64) *unit.Unit {
	yy := make([]getYearer, len(y))
	for i, yyy := range y {
		yy[i] = yyy
	}
	dbYear := db.GetYear()
	v1, v2, frac1, frac2 := db.interpolateYear(yy, dbYear)
	val1 := v1.(*ValueYear).GetValue(db)
	val2 := v2.(*ValueYear).GetValue(db)
	return unit.Add(unit.Mul(val1, frac1), unit.Mul(val2, frac2))
}

type getYearer interface {
	GetYear() float64
}

type getYearerSorter []getYearer

func (s getYearerSorter) Len() int           { return len(s) }
func (s getYearerSorter) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s getYearerSorter) Less(i, j int) bool { return s[i].GetYear() < s[j].GetYear() }

// interpolateYear takes a slice of getYearers and returns the two data members
// on either side of the desired value and the fraction of each of them that should
// be used.
func (db *DB) interpolateYear(data []getYearer, dbYear float64) (
	before, after getYearer, frac1, frac2 *unit.Unit) {
	sort.Sort(getYearerSorter(data))
	years := make([]float64, len(data))
	for i, y := range data {
		years[i] = y.GetYear()
	}
	if len(years) < 1 {
		panic("interpolateYear: no data")
	} else if len(years) == 1 || dbYear <= years[0] {
		before = data[0]
		after = data[0]
		frac1 = unit.New(1, unit.Dimless)
		frac2 = unit.New(0, unit.Dimless)
		return
	} else if l := len(years) - 1; dbYear >= years[l] {
		before = data[l]
		after = data[l]
		frac1 = unit.New(1, unit.Dimless)
		frac2 = unit.New(0, unit.Dimless)
		return
	}
	for i, y1 := range years {
		if y2 := years[i+1]; y2 >= dbYear {
			before = data[i]
			after = data[i+1]
			frac2 = unit.New((dbYear-y1)/(y2-y1), unit.Dimless)
			frac1 = unit.Sub(unit.New(1, unit.Dimless), frac2)
			return
		}
	}
	// This should never happen.
	panic(fmt.Sprintf("couldn't find year %v in %#v", dbYear, years))
}
