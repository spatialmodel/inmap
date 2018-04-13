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

package slca

import (
	"fmt"
	"math"
	"sort"

	"github.com/ctessum/unit"
)

// OnsiteResults is a holder for emissions, resource use, and requirements
// for a single process.
type OnsiteResults struct {
	Emissions    map[SubProcess]map[Gas]*unit.Unit
	Resources    map[SubProcess]map[Resource]*unit.Unit
	Requirements map[Process]map[Pathway]map[Output]*unit.Unit

	db LCADB
}

// NewOnsiteResults initializes a new instance of OnsiteResults.
func NewOnsiteResults(db LCADB) *OnsiteResults {
	r := new(OnsiteResults)
	r.Emissions = make(map[SubProcess]map[Gas]*unit.Unit)
	r.Resources = make(map[SubProcess]map[Resource]*unit.Unit)
	r.Requirements = make(map[Process]map[Pathway]map[Output]*unit.Unit)
	r.db = db
	return r
}

// OnsiteResultsNoSubprocess holds onsite results without
// any subprocess information.
type OnsiteResultsNoSubprocess struct {
	Emissions map[Gas]*unit.Unit
	Resources map[Resource]*unit.Unit
}

// getNamer is an interface for objects with a name.
type getNamer interface {
	GetName() string
}

// getNamerSorter is a list of GetNamers that fulfils the requirements for
// sort.Interface.
type getNamerSorter []getNamer

func (n getNamerSorter) Len() int           { return len(n) }
func (n getNamerSorter) Less(i, j int) bool { return n[i].GetName() < n[j].GetName() }
func (n getNamerSorter) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

func (or *OnsiteResults) sortOrder() (subProcesses, gases, resources getNamerSorter) {
	sMap := make(map[SubProcess]int)
	gasMap := make(map[Gas]int)
	resourceMap := make(map[Resource]int)

	for s, gases := range or.Emissions {
		sMap[s] = 1
		for g := range gases {
			gasMap[g] = 1
		}
	}
	for s, res := range or.Resources {
		sMap[s] = 1
		for r := range res {
			resourceMap[r] = 1
		}
	}

	for s := range sMap {
		subProcesses = append(subProcesses, s)
	}
	sort.Sort(subProcesses)
	for g := range gasMap {
		gases = append(gases, g)
	}
	sort.Sort(gases)
	for r := range resourceMap {
		resources = append(resources, r)
	}
	sort.Sort(resources)

	return
}

// SortOrder returns the gases emitted and resources used by the sub processes
// in the receiver in alphabetical order.
func (or *OnsiteResults) SortOrder() (subProcesses []SubProcess, gases []Gas, resources []Resource) {
	sp, gg, rr := or.sortOrder()
	for _, s := range sp {
		subProcesses = append(subProcesses, s.(SubProcess))
	}
	for _, g := range gg {
		gases = append(gases, g.(Gas))
	}
	for _, r := range rr {
		resources = append(resources, r.(Resource))
	}
	return
}

// SortOrder returns the gases emitted and resources used
// in the receiver in alphabetical order.
func (r *OnsiteResultsNoSubprocess) SortOrder() (gases []Gas, resources []Resource) {
	total := subProcess("Total")
	or := OnsiteResults{
		Emissions: map[SubProcess]map[Gas]*unit.Unit{total: r.Emissions},
		Resources: map[SubProcess]map[Resource]*unit.Unit{total: r.Resources},
	}
	_, gases, resources = or.SortOrder()
	return
}

func (or *OnsiteResults) String() string {
	subProcesses, gases, resources := or.SortOrder()
	s := ""
	if len(or.Emissions) > 0 {
		s += "Emissions:\n"
	}
	for _, sp := range subProcesses {
		for _, g := range gases {
			e, ok := or.Emissions[sp][g]
			if ok {
				s += fmt.Sprintf("\t%s: \t%s: %g %s\n", sp.GetName(), g.GetName(),
					e.Value(), e.Dimensions().String())
			}
		}
	}
	if len(or.Resources) > 0 {
		s += "Resources:\n"
	}
	for _, sp := range subProcesses {
		for _, r := range resources {
			v, ok := or.Resources[sp][r]
			if ok {
				s += fmt.Sprintf("\t%s: \t%s: %g %s\n", sp.GetName(), r.GetName(),
					v.Value(), v.Dimensions().String())
			}
		}
	}
	if len(or.Requirements) > 0 {
		s += "Requirements:\n"
	}
	for proc, pov := range or.Requirements {
		for path, ov := range pov {
			for o, v := range ov {
				s += fmt.Sprintf("\t%s, %s; %s: %g %s\n",
					path.GetName(), proc.GetName(), o.GetResource(or.db).GetName(),
					v.Value(), v.Dimensions().String())
			}
		}
	}

	if s == "" {
		s += "Nothing happens\n"
	}
	return s
}

func (or *OnsiteResultsNoSubprocess) String() string {
	gases, resources := or.SortOrder()
	s := ""
	if len(or.Emissions) > 0 {
		s += "Emissions:\n"
	}
	for _, g := range gases {
		e, ok := or.Emissions[g]
		if ok {
			s += fmt.Sprintf("\t%s: %g %v\n", g.GetName(), e.Value(), e.Dimensions())
		}
	}
	if len(or.Resources) > 0 {
		s += "Resources:\n"
	}
	for _, r := range resources {
		v, ok := or.Resources[r]
		if ok {
			s += fmt.Sprintf("\t%s: %g %s\n", r.GetName(), v.Value(), v.Dimensions())
		}
	}
	if s == "" {
		s += "Nothing happens\n"
	}
	return s
}

// Add adds values from another OnsiteResults to this OnsiteResults.
func (or *OnsiteResults) Add(r *OnsiteResults) {
	for sp, gg := range r.Emissions {
		if _, ok := or.Emissions[sp]; !ok {
			or.Emissions[sp] = make(map[Gas]*unit.Unit)
		}
		for g, v := range gg {
			if _, ok := or.Emissions[sp][g]; !ok {
				or.Emissions[sp][g] = v.Clone()
			} else {
				or.Emissions[sp][g].Add(v)
			}
		}
	}
	for sp, rr := range r.Resources {
		if _, ok := or.Resources[sp]; !ok {
			or.Resources[sp] = make(map[Resource]*unit.Unit)
		}
		for res, v := range rr {
			if _, ok := or.Resources[sp][res]; !ok {
				or.Resources[sp][res] = v.Clone()
			} else {
				or.Resources[sp][res].Add(v)
			}
		}
	}
	for proc, pov := range r.Requirements {
		if _, ok := or.Requirements[proc]; !ok {
			or.Requirements[proc] = make(map[Pathway]map[Output]*unit.Unit)
		}
		for path, ov := range pov {
			if _, ok := or.Requirements[proc][path]; !ok {
				or.Requirements[proc][path] = make(map[Output]*unit.Unit)
			}
			for o, v := range ov {
				if _, ok := or.Requirements[proc][path][o]; ok {
					or.Requirements[proc][path][o].Add(v)
				} else {
					or.Requirements[proc][path][o] = v.Clone()
				}
			}
		}
	}
}

// Clone returns a copy of or.
func (or *OnsiteResults) Clone() *OnsiteResults {
	o := NewOnsiteResults(or.db)
	o.Add(or)
	return o
}

// ScaleCopy returns a copy of or, multiplying the values in or by factor.
func (or *OnsiteResults) ScaleCopy(factor *unit.Unit) *OnsiteResults {
	r := NewOnsiteResults(or.db)

	for sp, gg := range or.Emissions {
		if _, ok := r.Emissions[sp]; !ok {
			r.Emissions[sp] = make(map[Gas]*unit.Unit)
		}
		for g, v := range gg {
			if _, ok := r.Emissions[sp][g]; !ok {
				r.Emissions[sp][g] = unit.Mul(v, factor)
			} else {
				r.Emissions[sp][g].Add(unit.Mul(v, factor))
			}
		}
	}
	for sp, rr := range or.Resources {
		if _, ok := r.Resources[sp]; !ok {
			r.Resources[sp] = make(map[Resource]*unit.Unit)
		}
		for res, v := range rr {
			if _, ok := r.Resources[sp][res]; !ok {
				r.Resources[sp][res] = unit.Mul(v, factor)
			} else {
				r.Resources[sp][res].Add(unit.Mul(v, factor))
			}
		}
	}
	for proc, pov := range or.Requirements {
		if _, ok := r.Requirements[proc]; !ok {
			r.Requirements[proc] = make(map[Pathway]map[Output]*unit.Unit)
		}
		for path, ov := range pov {
			if _, ok := r.Requirements[proc][path]; !ok {
				r.Requirements[proc][path] = make(map[Output]*unit.Unit)
			}
			for o, v := range ov {
				if _, ok := r.Requirements[proc][path][o]; ok {
					r.Requirements[proc][path][o].Add(unit.Mul(v, factor))
				} else {
					r.Requirements[proc][path][o] = unit.Mul(v, factor)
				}
			}
		}
	}
	return r
}

// AddResource adds an amount of a resource to the results.
func (or *OnsiteResults) AddResource(sp SubProcess, r Resource, val *unit.Unit, db LCADB) {
	if _, ok := or.Resources[sp]; !ok {
		or.Resources[sp] = make(map[Resource]*unit.Unit)
	}
	if _, ok := or.Resources[sp][r]; !ok {
		or.Resources[sp][r] = r.ConvertToDefaultUnits(val, db).Clone()
	} else {
		or.Resources[sp][r].Add(r.ConvertToDefaultUnits(val, db))
	}
}

// SubResource subtracts an amount of a resource from the results.
func (or *OnsiteResults) SubResource(sp SubProcess, r Resource, val *unit.Unit, db LCADB) {
	if _, ok := or.Resources[sp]; !ok {
		or.Resources[sp] = make(map[Resource]*unit.Unit)
	}
	if _, ok := or.Resources[sp][r]; !ok {
		or.Resources[sp][r] = unit.Negate(val).Clone()
	} else {
		or.Resources[sp][r].Sub(val)
	}
}

// AddEmission adds an amount of a resource to the results.
func (or *OnsiteResults) AddEmission(sp SubProcess, g Gas, val *unit.Unit) {
	if err := val.Check(unit.Kilogram); err != nil {
		panic(err)
	}
	if _, ok := or.Emissions[sp]; !ok {
		or.Emissions[sp] = make(map[Gas]*unit.Unit)
	}
	if _, ok := or.Emissions[sp][g]; !ok {
		or.Emissions[sp][g] = val.Clone()
	} else {
		or.Emissions[sp][g].Add(val)
	}
}

// SubEmission subtracts an amount of a resource from the results.
func (or *OnsiteResults) SubEmission(sp SubProcess, g Gas, val *unit.Unit) {
	if err := val.Check(unit.Kilogram); err != nil {
		panic(err)
	}
	if _, ok := or.Emissions[sp]; !ok {
		or.Emissions[sp] = make(map[Gas]*unit.Unit)
	}
	if _, ok := or.Emissions[sp][g]; !ok {
		or.Emissions[sp][g] = unit.Negate(val).Clone()
	} else {
		or.Emissions[sp][g].Sub(val)
	}
}

// AddRequirement adds a requirement of the current process for another
// process.
func (or *OnsiteResults) AddRequirement(proc Process, path Pathway, o Output, amount *unit.Unit, db LCADB) {
	if proc == nil {
		panic("Error: Process is nil!")
	}
	if path == nil {
		panic("Error: Pathway is nil!")
	}
	if math.IsNaN(amount.Value()) ||
		math.IsInf(amount.Value(), 0) {
		panic(fmt.Errorf("Invalid amount: %v.\n"+
			"Upstream proc: %v, path: %v", amount, proc.GetName(),
			path.GetName()))
	}
	if amount.Value() == 0. {
		// Ignore requirement of amount 0.
		return
	}
	amt := o.GetResource(db).ConvertToDefaultUnits(amount, db)
	if pp, ok := or.Requirements[proc]; ok {
		if ppp, ok := pp[path]; ok {
			if _, ok := ppp[o]; ok {
				ppp[o].Add(amt)
			} else {
				or.Requirements[proc][path][o] = amt.Clone()
			}
		} else {
			or.Requirements[proc][path] = make(map[Output]*unit.Unit)
			or.Requirements[proc][path][o] = amt.Clone()
		}
	} else {
		or.Requirements[proc] = make(map[Pathway]map[Output]*unit.Unit)
		or.Requirements[proc][path] = make(map[Output]*unit.Unit)
		or.Requirements[proc][path][o] = amt.Clone()
	}
}

// Mul multiplies the results by a constant.
func (or *OnsiteResults) Mul(v *unit.Unit) {
	for sp, rr := range or.Resources {
		for r := range rr {
			or.Resources[sp][r].Mul(v)
		}
	}
	for sp, gg := range or.Emissions {
		for g := range gg {
			or.Emissions[sp][g].Mul(v)
		}
	}
	for proc, pov := range or.Requirements {
		for path, ov := range pov {
			for o := range ov {
				or.Requirements[proc][path][o].Mul(v)
			}
		}
	}
}

// Div divides the results by a constant.
func (or *OnsiteResults) Div(v *unit.Unit) {
	for sp, rr := range or.Resources {
		for r, val := range rr {
			or.Resources[sp][r] = unit.Div(val, v)
		}
	}
	for sp, gg := range or.Emissions {
		for g, val := range gg {
			or.Emissions[sp][g] = unit.Div(val, v)
		}
	}
	for proc, pov := range or.Requirements {
		for path, ov := range pov {
			for o := range ov {
				or.Requirements[proc][path][o].Div(v)
			}
		}
	}
}

// sumRequirements returns the sum of the Requirements part
// of these results
func (or *OnsiteResults) sumRequirements(output Output) *unit.Unit {
	var result *unit.Unit
	for _, pov := range or.Requirements {
		for _, ov := range pov {
			for o, v := range ov {
				if o.GetID() == output.GetID() {
					result.Add(v)
				}
			}
		}
	}
	return result
}

// FilterEmissions returns an OnsiteResults object that only contains emissions.
func (or *OnsiteResults) FilterEmissions() *OnsiteResults {
	return &OnsiteResults{Emissions: or.Emissions}
}

// FilterResources returns an OnsiteResults object that only contains resource use.
func (or *OnsiteResults) FilterResources() *OnsiteResults {
	return &OnsiteResults{Resources: or.Resources}
}

// FilterRequirements returns an OnsiteResults object that only contains Requirements.
func (or *OnsiteResults) FilterRequirements() *OnsiteResults {
	return &OnsiteResults{Requirements: or.Requirements}
}

// FlattenSubprocess returns a version of the results with the
// resource use and emissions from all subprocesses combined.
func (or *OnsiteResults) FlattenSubprocess() *OnsiteResultsNoSubprocess {
	totalSP := subProcess("Total")
	oTotal := NewOnsiteResults(or.db)
	for _, e := range or.Emissions {
		for g, v := range e {
			oTotal.AddEmission(totalSP, g, v)
		}
	}
	for _, rr := range or.Resources {
		for res, v := range rr {
			oTotal.AddResource(totalSP, res, v, or.db)
		}
	}
	return &OnsiteResultsNoSubprocess{
		Emissions: oTotal.Emissions[totalSP],
		Resources: oTotal.Resources[totalSP],
	}
}
