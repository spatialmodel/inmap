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
	"sort"

	"github.com/spatialmodel/epi"
	"github.com/spatialmodel/inmap"

	"golang.org/x/net/context"

	"bitbucket.org/ctessum/sparse"
)

// SpatialResults is a wrapper for Results that can do spatial calculations.
type SpatialResults struct {
	*Results
	db *DB
}

// NewSpatialResults returns a new SpatialResults variable.
func NewSpatialResults(res *Results, db *DB) *SpatialResults {
	r := SpatialResults{
		Results: res,
		db:      db,
	}
	return &r
}

// Emissions gets the total spatially explicit emissions caused by life cycle sr.
func (sr *SpatialResults) Emissions() (map[Gas]*sparse.SparseArray, error) {
	o := make(map[Gas]*sparse.SparseArray)
	for _, e := range sr.Edges {
		emis, err := sr.EdgeEmissions(e)
		if err != nil {
			return nil, err
		}

		for _, spData := range emis {
			for pol, data := range spData {
				if o[pol] == nil {
					o[pol] = data.Copy()
				} else {
					o[pol].AddSparse(data)
				}
			}
		}
	}
	return o, nil
}

// ResourceUse gets the total spatially explicit resource
// use caused by life cycle sr.
func (sr *SpatialResults) ResourceUse() (map[Resource]*sparse.SparseArray, error) {
	o := make(map[Resource]*sparse.SparseArray)
	for _, e := range sr.Edges {
		res, err := sr.EdgeResourceUse(e)
		if err != nil {
			return nil, err
		}

		for _, spData := range res {
			for pol, data := range spData {
				if o[pol] == nil {
					o[pol] = data.Copy()
				} else {
					o[pol].AddSparse(data)
				}
			}
		}
	}
	return o, nil
}

// Concentrations gets the total change in concentrations of PM2.5 and its
// subspecies caused by life cycle sr.
func (sr *SpatialResults) Concentrations() (map[string]*sparse.DenseArray, error) {
	o := make(map[string]*sparse.DenseArray)
	for _, e := range sr.Edges {
		conc, err := sr.EdgeConcentrations(e)
		if err != nil {
			return nil, err
		}

		for _, spData := range conc {
			for pol, data := range spData {
				if _, ok := o[pol]; !ok {
					o[pol] = data.Copy()
				} else {
					o[pol].AddDense(data)
				}
			}
		}
	}
	return o, nil
}

// Health gets the total PM2.5 health impacts caused by life cycle sr.
// The format of the output is map[population][pollutant]impacts.
// HR specifies the function used to calculate the hazard ratio.
func (sr *SpatialResults) Health(HR epi.HRer) (map[string]map[string]*sparse.DenseArray, error) {
	o := make(map[string]map[string]*sparse.DenseArray)
	for _, e := range sr.Edges {
		health, err := sr.EdgeHealth(e, HR)
		if err != nil {
			return nil, err
		}

		for _, spData := range health {
			for pop, data := range spData {
				if _, ok := o[pop]; !ok {
					o[pop] = make(map[string]*sparse.DenseArray)
				}
				for pol, d2 := range data {
					if o[pop][pol] == nil {
						o[pop][pol] = d2.Copy()
					} else {
						o[pop][pol].AddDense(d2)
					}
				}
			}
		}
	}
	return o, nil
}

// EdgeEmissions returns the spatialized emissions associated with edge e.
func (sr *SpatialResults) EdgeEmissions(e *ResultEdge) (map[SubProcess]map[Gas]*sparse.SparseArray, error) {
	spatialSrg, err := sr.spatialize(e)
	if err != nil {
		return nil, err
	}
	o := make(map[SubProcess]map[Gas]*sparse.SparseArray)
	for sp, ee := range e.FromResults.Emissions {
		o[sp] = make(map[Gas]*sparse.SparseArray)
		for gas, emis := range ee {
			o[sp][gas], err = sr.db.CSTConfig.scaleFlattenSrg(spatialSrg, PM25, emis.Value())
			if err != nil {
				return nil, err
			}
		}
	}
	return o, nil
}

// EdgeResourceUse returns the spatialized resource use associated with edge e.
func (sr *SpatialResults) EdgeResourceUse(e *ResultEdge) (map[SubProcess]map[Resource]*sparse.SparseArray, error) {
	spatialSrg, err := sr.spatialize(e)
	if err != nil {
		return nil, err
	}
	o := make(map[SubProcess]map[Resource]*sparse.SparseArray)
	for sp, rr := range e.FromResults.Resources {
		o[sp] = make(map[Resource]*sparse.SparseArray)
		for res, val := range rr {
			o[sp][res], err = sr.db.CSTConfig.scaleFlattenSrg(spatialSrg, PM25, val.Value())
			if err != nil {
				return nil, err
			}
		}
	}
	return o, nil
}

func (sr *SpatialResults) spatialize(e *ResultEdge) ([]*inmap.EmisRecord, error) {
	ctx := context.TODO()
	req, err := newRequestPayload(sr, e)
	if err != nil {
		return nil, err
	}
	return sr.db.CSTConfig.spatialSurrogate(ctx, req.spatialRef)
}

const totalPM25 = "TotalPM2_5"

// EdgeConcentrations gets the change in concentrations of PM2.5 and its
// subspecies caused by the emissions in e.
func (sr *SpatialResults) EdgeConcentrations(e *ResultEdge) (map[SubProcess]map[string]*sparse.DenseArray, error) {
	ctx := context.TODO()

	concs := make(map[SubProcess]map[string]*sparse.DenseArray)

	req, err := newRequestPayload(sr, e)
	if err != nil {
		return nil, err
	}

	r := sr.db.CSTConfig.concRequestCache.NewRequest(ctx, req, req.key)
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	// The inmapSurrogate contains PM2.5 impacts of 1kg/year of
	// emissions from this edge.
	inmapSurrogate := result.(map[string][]float64)

	for sp, spData := range req.edge.FromResults.Emissions {
		if _, ok := concs[sp]; !ok {
			concs[sp] = make(map[string]*sparse.DenseArray)
		}
		for gas, emis := range spData {
			// Calculate total PM2.5 subspecies impacts from this record.
			pol, ok := sr.db.CSTConfig.PolTrans[gas.GetName()]
			if !ok {
				return nil, fmt.Errorf("in slca.edgeConcentrations: no InMAP variable for %v ('none' is a valid option)", gas.GetName())
			} else if pol == "none" {
				continue // skip this pol if it doesn't belong in InMAP.
			}
			if u := emis.Dimensions().String(); u != kg {
				return nil, fmt.Errorf("in slca.edgeConcentrations: emissions units expected to be kg but are %v", u)
			}
			d, ok := inmapSurrogate[pol]
			if !ok {
				return nil, fmt.Errorf("in slca.EdgeConcentrations: no InMAP surrogate for %s", pol)
			}
			if _, ok := concs[sp][pol]; !ok {
				concs[sp][pol] = sparse.ZerosDense(len(d))
			}
			dd := concs[sp][pol]
			for i, v := range d {
				// Add the InMAP surrogate value for 1kg of this pollutant times
				// the total emissions in this record to get the contribution to the
				// total concentration of this pollutant.
				dd.Elements[i] += v * emis.Value()
			}
			concs[sp][pol] = dd
		}
	}

	// Calculate total PM2.5 concentrations.
	for sp, spData := range concs {
		var cPM *sparse.DenseArray
		for _, c := range spData {
			if cPM == nil {
				cPM = c.Copy()
			} else {
				cPM.AddDense(c)
			}
		}
		if cPM != nil {
			concs[sp][totalPM25] = cPM
		}
	}
	return concs, nil
}

// EdgeHealth gets the spatialized health impacts caused by the emissions in edge e.
// It calculates the number of deaths in each demographic group caused by these
// emissions. The format of the output is map[population][pollutant]effects.
// HR specifies the function used to calculate the hazard ratio.
func (sr *SpatialResults) EdgeHealth(e *ResultEdge, HR epi.HRer) (map[SubProcess]map[string]map[string]*sparse.DenseArray, error) {
	req, err := newRequestPayload(sr, e)
	if err != nil {
		return nil, err
	}

	ctx := context.TODO()
	health := make(map[SubProcess]map[string]map[string]*sparse.DenseArray)
	healthSurrogate, err := sr.db.CSTConfig.HealthSurrogate(ctx, req.spatialRef, HR)
	if err != nil {
		return nil, err
	}

	for sp, spData := range req.edge.FromResults.Emissions {
		health[sp] = make(map[string]map[string]*sparse.DenseArray)
		for gas, emis := range spData {
			// Calculate total health impacts from this record.
			pol, ok := sr.db.CSTConfig.PolTrans[gas.GetName()]
			if !ok {
				return nil, fmt.Errorf("in slca.edgeHealth: no InMAP variable for %v ('none' is a valid option)", gas.GetName())
			} else if pol == "none" {
				continue // skip this pol if it doesn't belong in InMAP.
			}
			if u := emis.Dimensions().String(); u != kg {
				return nil, fmt.Errorf("in slca.edgeHealth: emissions units expected to be kg but are %v", u)
			}
			for pop, hs := range healthSurrogate {
				if _, ok := health[sp][pop]; !ok {
					health[sp][pop] = make(map[string]*sparse.DenseArray)
				}
				d, ok := hs[pol]
				if !ok {
					return nil, fmt.Errorf("in slca.edgeHealth: no health surrogate for %s", pol)
				}
				if _, ok := health[sp][pop][pol]; !ok {
					health[sp][pop][pol] = sparse.ZerosDense(d.Shape...)
				}
				dd := health[sp][pop][pol]
				for i, v := range d.Elements {
					// Add the InMAP surrogate value for 1kg of this pollutant times
					// the total emissions in this record to get the contribution to the
					// total concentration of this pollutant.
					dd.Elements[i] += v * emis.Value()
				}
				health[sp][pop][pol] = dd
			}
		}
		// Calculate total PM2.5 concentrations.
		for pop := range healthSurrogate {
			var hPM *sparse.DenseArray
			for _, h := range health[sp][pop] {
				if hPM == nil {
					hPM = h.Copy()
				} else {
					hPM.AddDense(h)
				}
			}
			if hPM != nil {
				health[sp][pop][totalPM25] = hPM
			}
		}
	}

	return health, nil
}

// EdgeHealthTotals gets the total (non-spatialized) health impacts caused by
// the emissions in edge e.
// It calculates the number of deaths in each demographic group caused by these
// emissions.
// HR specifies the function used to calculate the hazard ratio.
// The format of the output is map[population][pollutant]total effects.
func (sr *SpatialResults) EdgeHealthTotals(e *ResultEdge, HR epi.HRer) (map[SubProcess]map[string]map[string]float64, error) {
	if h, ok := sr.edgeHealthTotals[e]; ok {
		return h, nil // Return the cached value if it exists.
	}
	o := make(map[SubProcess]map[string]map[string]float64)
	health, err := sr.EdgeHealth(e, HR)
	if err != nil {
		return nil, err
	}
	for sp, spData := range health {
		o[sp] = make(map[string]map[string]float64)
		for pop, d := range spData {
			if _, ok := o[sp][pop]; !ok {
				o[sp][pop] = make(map[string]float64)
			}
			for pol, dd := range d {
				o[sp][pop][pol] = dd.Sum()
			}
		}
	}
	sr.edgeHealthTotals[e] = o
	return o, nil
}

// Less is used to implement the sort.Sort interface and returns whether
// the health impacts of edge i are less than the health impacts of edge j.
func (sr spatialResultsSorter) Less(i, j int) bool {
	iHealth, err := sr.EdgeHealthTotals(sr.Edges[i], sr.hr)
	if err != nil {
		panic(err)
	}
	jHealth, err := sr.EdgeHealthTotals(sr.Edges[j], sr.hr)
	if err != nil {
		panic(err)
	}

	healthSum := func(m map[SubProcess]map[string]map[string]float64) (float64, bool) {
		var ok bool
		var sum float64
		for _, spData := range m {
			v, okx := spData[sr.db.CSTConfig.CensusPopColumns[0]][totalPM25]
			if okx {
				ok = true
				sum += v
			}
		}
		return sum, ok
	}

	ih, ihok := healthSum(iHealth)
	jh, jhok := healthSum(jHealth)
	if !ihok && !jhok {
		return sr.Results.Less(i, j)
	}
	if !ihok && jhok {
		if jh == 0 {
			return sr.Results.Less(i, j)
		}
		return jh > 0
	}
	if !jhok && ihok {
		if ih == 0 {
			return sr.Results.Less(i, j)
		}
		return ih < 0
	}
	if x := ih - jh; x != 0 {
		return x < 0
	}
	return sr.Results.Less(i, j)
}

func (sr spatialResultsSorter) Len() int      { return sr.Results.Len() }
func (sr spatialResultsSorter) Swap(i, j int) { sr.Results.Swap(i, j) }

// Sum returns a sum of the (non-spatial) results.
func (sr *SpatialResults) Sum() *OnsiteResultsNoSubprocess { return sr.Results.Sum() }

type getNameSorter []getNamer

func (a getNameSorter) Len() int           { return len(a) }
func (a getNameSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a getNameSorter) Less(i, j int) bool { return a[i].GetName() < a[j].GetName() }

type spatialResultsSorter struct {
	*SpatialResults
	hr epi.HRer
}

// Table creates a table from the results, suitable for outputting to a
// CSV file.
// HR specifies the function used to calculate the hazard ratio.
func (sr *SpatialResults) Table(HR epi.HRer) ([][]string, error) {
	var out [][]string
	sort.Sort(sort.Reverse(spatialResultsSorter{SpatialResults: sr, hr: HR}))
	nonHealthTotals := sr.Sum()
	health, err := sr.Health(HR)
	if err != nil {
		return nil, err
	}
	healthPolMap := make(map[string]string)
	healthTotals := make(map[string]map[string]float64)
	for pop, d := range health {
		if _, ok := healthTotals[pop]; !ok {
			healthTotals[pop] = make(map[string]float64)
		}
		for pol, dd := range d {
			healthTotals[pop][pol] = dd.Sum()
			healthPolMap[pol] = ""
		}
	}
	healthPols := make([]string, len(healthPolMap))
	i := 0
	for pol := range healthPolMap {
		healthPols[i] = pol
		i++
	}
	sort.Strings(healthPols)

	var pols []getNamer
	for g := range nonHealthTotals.Emissions {
		pols = append(pols, g)
	}
	var resources []getNamer
	for r := range nonHealthTotals.Resources {
		resources = append(resources, r)
	}
	sort.Sort(getNameSorter(pols))
	sort.Sort(getNameSorter(resources))
	o := []string{"Pathway", "Process", "Subprocess", "Downstream pathway", "Downstream process"}
	for _, p := range pols {
		o = append(o, fmt.Sprintf("%s (%v)", p.GetName(),
			nonHealthTotals.Emissions[p.(Gas)].Dimensions()))
	}
	for _, r := range resources {
		o = append(o, fmt.Sprintf("%s (%v)", r.GetName(),
			nonHealthTotals.Resources[r.(Resource)].Dimensions()))
	}
	for _, p := range sr.db.CSTConfig.CensusPopColumns {
		for _, pol := range healthPols {
			o = append(o, fmt.Sprintf("%s %s deaths", p, pol))
		}
	}
	o = append(o, "Spatial surrogate") // spatial data used to spatialize the data.
	out = append(out, o)

	o = []string{"Totals", "", "", "", ""}
	for _, p := range pols {
		o = append(o, fmt.Sprintf("%g", nonHealthTotals.Emissions[p.(Gas)].Value()))
	}
	for _, r := range resources {
		o = append(o, fmt.Sprintf("%g", nonHealthTotals.Resources[r.(Resource)].Value()))
	}
	for _, p := range sr.db.CSTConfig.CensusPopColumns {
		for _, pol := range healthPols {
			o = append(o, fmt.Sprintf("%g", healthTotals[p][pol]))
		}
	}
	o = append(o, "") // empty quotes are a filler for the map names.
	out = append(out, o)

	for _, e := range sr.Edges {
		subProcesses, _, _ := e.FromResults.SortOrder()
		for _, sp := range subProcesses {

			from := sr.Results.GetFromNode(e)
			to := sr.Results.GetToNode(e)
			o := make([]string, 5)
			o[0] = from.Pathway.GetName()
			o[1] = from.Process.GetName()
			o[2] = sp.GetName()
			o[3] = to.Pathway.GetName()
			o[4] = to.Process.GetName()
			for _, p := range pols {
				if _, ok := e.FromResults.Emissions[sp][p.(Gas)]; ok {
					o = append(o, fmt.Sprintf("%g",
						e.FromResults.Emissions[sp][p.(Gas)].Value()))
				} else {
					o = append(o, "")
				}
			}
			for _, r := range resources {
				if _, ok := e.FromResults.Resources[sp][r.(Resource)]; ok {
					o = append(o, fmt.Sprintf("%g",
						e.FromResults.Resources[sp][r.(Resource)].Value()))
				} else {
					o = append(o, "")
				}
			}
			health, err := sr.EdgeHealth(e, HR)
			if err != nil {
				return nil, err
			}
			for _, p := range sr.db.CSTConfig.CensusPopColumns {
				for _, pol := range healthPols {
					if health[sp][p][pol] == nil {
						o = append(o, "")
					} else {
						o = append(o, fmt.Sprintf("%g", health[sp][p][pol].Sum()))
					}
				}
			}
			req, err := newRequestPayload(sr, e)
			if err != nil {
				return nil, err
			}
			o = append(o, req.key)
			out = append(out, o)
		}
	}
	return out, nil
}
