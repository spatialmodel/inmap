/*
Copyright © 2017 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.*/

package aep

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ctessum/unit"
)

// emissionsPeriod specifies a rate of emissions of a certain pollutant over
// a given period.
type emissionsPeriod struct {
	begin, end time.Time
	rate       float64
	Pollutant  // the pollutant being emitted.
}

// Pollutant holds information about a pollutant.
type Pollutant struct {
	// Name is the name of the pollutant.
	Name string

	// Prefix holds information about the pollutant prefix, such as BRK, TIR, etc.
	Prefix string
}

func (p Pollutant) String() string {
	if p.Prefix == "" {
		return p.Name
	}
	return p.Prefix + "__" + p.Name
}

func (e emissionsPeriod) String() string {
	return fmt.Sprintf("%s: %v -- %v: %g", e.Pollutant, e.begin, e.end, e.rate)
}

// Emissions holds information about the rate of emissions from a source of
// different pollutants and potentially at different times.
type Emissions struct {
	e     []*emissionsPeriod
	units map[Pollutant]unit.Dimensions
}

func (e Emissions) String() string {
	s := ""
	for i, ee := range e.e {
		if i != 0 {
			s += "\n"
		}
		s += fmt.Sprintf("%s %s", ee.String(), e.units[ee.Pollutant].String())
	}
	return s
}

// Clone returns a copy of the receiver.
func (e *Emissions) Clone() *Emissions {
	o := Emissions{
		e:     make([]*emissionsPeriod, len(e.e)),
		units: make(map[Pollutant]unit.Dimensions),
	}
	for i, p := range e.e {
		p2 := *p
		o.e[i] = &p2
	}
	for p, d := range e.units {
		o.units[p] = d
	}
	return &o
}

// Add adds emissions beginning and ending at times begin and end, respectively,
// of pollutant 'pollutant' with prefix 'polPrefix' (e.g., BRK, TIR, etc),
// and of total amount 'amount'. Emissions are expected to be in units of g/s.
func (e *Emissions) Add(begin, end time.Time, pollutant, polPrefix string, rate *unit.Unit) {
	p := Pollutant{Name: pollutant, Prefix: polPrefix}
	dims := rate.Dimensions()

	if e.units == nil {
		e.units = make(map[Pollutant]unit.Dimensions)
	}
	if d, ok := e.units[p]; ok {
		if !d.Matches(dims) {
			panic(fmt.Errorf("aep.Emissions.Add: incompatible units '%s' and '%s'", d, dims))
		}
	} else {
		e.units[p] = dims
	}

	e.e = append(e.e, &emissionsPeriod{
		begin:     begin,
		end:       end,
		Pollutant: p,
		rate:      rate.Value(),
	})
}

var secPerYear, secPerDay *unit.Unit

func init() {
	secPerDay = unit.New(86400., unit.Second)
	secPerYear = unit.New(31536000., unit.Second)
}

// convert emissions to [mass]/[time] when given mass of emissions/year (ann)
// and/or mass of emissions per day (avd) and a function to convert to SI units (inputConv).
func parseEmisRateAnnual(ann, avd string, inputConv func(float64) *unit.Unit) (*unit.Unit, error) {
	annf, err := stringToFloat(ann)
	if err != nil {
		return nil, err
	}
	avdf, err := stringToFloat(avd)
	if err != nil {
		return nil, err
	}
	// if annual emissions not present, fill with average day
	if annf <= 0. {
		if avdf >= 0. {
			return unit.Div(inputConv(avdf), secPerDay), nil
		}
	}
	return unit.Div(inputConv(annf), secPerYear), nil
}

// convert emissions to [mass]/[time] when given mass of emissions/time period,
// the number of seconds in that period, and
// a function to convert to SI units (inputConv).
func parseEmisRate(emis string, secPerPeriod *unit.Unit, inputConv func(float64) *unit.Unit) (*unit.Unit, error) {
	v, err := stringToFloat(emis)
	if err != nil {
		return nil, err
	}
	return unit.Div(inputConv(v), secPerPeriod), nil
}

// DropPols removes the pollutants that are not in polsToKeep
// and returns the total emissions removed, in units of [mass].
// If polsToKeep is nil, all pollutants are kept.
func (e *Emissions) DropPols(polsToKeep Speciation) map[Pollutant]*unit.Unit {
	if polsToKeep == nil {
		return nil
	}
	droppedTotals := make(map[Pollutant]*unit.Unit)
	var iToDelete []int
	unitsToDelete := make(map[Pollutant]empty)
	for i, em := range e.e {
		if _, ok := polsToKeep[em.Pollutant.Name]; !ok {
			iToDelete = append(iToDelete, i)
			units, ok := e.units[em.Pollutant]
			if !ok {
				panic(fmt.Errorf("aep: missing units for pollutant %s", em.Pollutant))
			}
			v := unit.Mul(unit.New(em.rate, units),
				unit.New(em.end.Sub(em.begin).Seconds(), unit.Second))
			if _, ok := droppedTotals[em.Pollutant]; !ok {
				droppedTotals[em.Pollutant] = v
			} else {
				droppedTotals[em.Pollutant].Add(v)
			}
			unitsToDelete[em.Pollutant] = empty{}
		}
	}
	for p := range unitsToDelete {
		delete(e.units, p)
	}
	numDeleted := 0
	for _, i := range iToDelete {
		e.deleteItem(i - numDeleted)
		numDeleted++
	}
	return droppedTotals
}

func (e *Emissions) deleteItem(index int) {
	e.e, e.e[len(e.e)-1] = append(e.e[:index], e.e[index+1:]...), &emissionsPeriod{}
}

// Totals returns the total emissions in units of [mass].
func (e *Emissions) Totals() map[Pollutant]*unit.Unit {
	totals := make(map[Pollutant]*unit.Unit)
	for _, em := range e.e {
		v := unit.Mul(unit.New(em.rate, e.units[em.Pollutant]),
			unit.New(em.end.Sub(em.begin).Seconds(), unit.Second))
		if _, ok := totals[em.Pollutant]; !ok {
			totals[em.Pollutant] = v
		} else {
			totals[em.Pollutant].Add(v)
		}
	}
	return totals
}

// timeBetween returns true if t is between t1 and t2
func timeBetween(t, t1, t2 time.Time) bool {
	return t.After(t1) && t2.After(t)
}

// PeriodTotals returns the total emissions from this emissions source between
// the times begin and end.
func (e *Emissions) PeriodTotals(begin, end time.Time) map[Pollutant]*unit.Unit {
	if begin.After(end) {
		panic(fmt.Errorf("begin (%v) is after end (%v)", begin, end))
	}

	totals := make(map[Pollutant]*unit.Unit)
	for _, em := range e.e {
		var emisBegin, emisEnd time.Time

		if !(em.end.After(begin) && end.After(em.begin)) {
			// Skip emissions that don't overlap with our period.
			continue
		}
		if timeBetween(begin, em.begin, em.end) {
			emisBegin = begin
		} else {
			emisBegin = em.begin
		}
		if timeBetween(end, em.begin, em.end) {
			emisEnd = end
		} else {
			emisEnd = em.end
		}

		v := unit.Mul(unit.New(em.rate, e.units[em.Pollutant]),
			unit.New(emisEnd.Sub(emisBegin).Seconds(), unit.Second))
		if _, ok := totals[em.Pollutant]; !ok {
			totals[em.Pollutant] = v
		} else {
			totals[em.Pollutant].Add(v)
		}
	}
	return totals
}

// GetEmissions returns the emissions associated with this record
func (e *Emissions) GetEmissions() *Emissions {
	return e
}

// CombineEmissions combines emissions from r2 into this record.
func (e *Emissions) CombineEmissions(r2 Record) {
	e.combine(*r2.GetEmissions())
}

func (e *Emissions) combine(e2 Emissions) {
	e.e = append(e.e, e2.e...)
	for pol, u := range e2.units {
		if uu, ok := e.units[pol]; ok {
			if !reflect.DeepEqual(uu, u) {
				panic(fmt.Errorf("units don't match: %v != %v", uu, u))
			}
		} else {
			e.units[pol] = u
		}
	}
}

// Scale scales the emissions in the receiver according to f, which
// is a function that gives emissions multipliers by Pollutant.
func (e *Emissions) Scale(f func(Pollutant) (float64, error)) error {
	for _, ee := range e.e {
		s, err := f(ee.Pollutant)
		if err != nil {
			return err
		}
		ee.rate *= s
	}
	return nil
}
