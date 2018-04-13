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
	"log"
	"math"
	"os"
	"sort"

	"github.com/ctessum/unit"
)

// Results is a holder for life cycle process requirements, emissions,
// and resource use.
// It allows the determination of which downstream processes are causing
// the resource use and emissions of the upstream process.
type Results struct {
	Nodes map[string]*ResultNode
	Edges []*ResultEdge

	db LCADB

	// edgeHealthTotals is a cache for health results for individual edges.
	// Using a cache here greatly speeds up the sorting of the edges.
	edgeHealthTotals map[*ResultEdge]map[SubProcess]map[string]map[string]float64
}

// ResultNode is a holder for information about a single pathway/process/output
// combination.
type ResultNode struct {
	ID      string
	Process Process
	Pathway Pathway
	Output  Output

	// requiredBy holds the amounts of this node required by other nodes.
	// Data structure is map[downstream node]amount
	requiredBy map[*ResultNode]*unit.Unit

	// sum of process requirements from the previous iteration.
	oldReqSum *unit.Unit
}

// addRequiredBy add the amount of this node required by node n.
func (rn *ResultNode) addRequiredBy(n *ResultNode, amount *unit.Unit) {
	if _, ok := rn.requiredBy[n]; ok {
		rn.requiredBy[n].Add(amount)
	} else {
		rn.requiredBy[n] = amount.Clone()
	}
}

// sumRequiredBy sums the amount of this node that is required
// by other nodes.
func (rn *ResultNode) sumRequiredBy() *unit.Unit {
	var sum *unit.Unit
	for _, v := range rn.requiredBy {
		if sum == nil {
			sum = v.Clone()
		} else {
			sum.Add(v)
		}
	}
	return sum
}

// ResultEdge is a holder for information about the relation between two Nodes
// and the emissions created and resources used by the From process to supply
// the To process.
type ResultEdge struct {
	FromID, ToID, ID string
	FromResults      *OnsiteResults
}

// NewResults initializes a new instance of results.
func NewResults(db LCADB) *Results {
	return &Results{
		Nodes:            make(map[string]*ResultNode),
		db:               db,
		edgeHealthTotals: make(map[*ResultEdge]map[SubProcess]map[string]map[string]float64),
	}
}

// GetNode checks whether the Results already have a node for the given
// process, pathway, and output. If the node exists, it is returned, otherwise a new node
// is created, added to the results, and returned.
func (r *Results) GetNode(proc Process, path Pathway, o Output) *ResultNode {

	for _, n := range r.Nodes {
		if n.Process == proc && n.Pathway == path && n.Output == o {
			return n
		}
	}
	n := &ResultNode{
		Process: proc,
		Pathway: path,
		Output:  o,
		ID:      proc.GetIDStr() + path.GetIDStr() + string(o.GetID()),
	}
	n.requiredBy = make(map[*ResultNode]*unit.Unit)
	r.Nodes[n.ID] = n
	return n
}

// GetEdge returns the edge connecting from and to if it exist, otherwise
// a new edge is created, added to the results, and returned.
func (r *Results) GetEdge(from, to *ResultNode) *ResultEdge {
	for _, e := range r.Edges {
		if e.FromID == from.ID && e.ToID == to.ID {
			return e
		}
	}
	e := &ResultEdge{
		ID:          from.ID + to.ID,
		FromID:      from.ID,
		ToID:        to.ID,
		FromResults: NewOnsiteResults(r.db),
	}
	r.Edges = append(r.Edges, e)
	return e
}

// GetFromNode gets the upstream node associated with this edge.
func (r *Results) GetFromNode(e *ResultEdge) *ResultNode {
	return r.Nodes[e.FromID]
}

// GetToNode gets the downstream node associated with this edge.
func (r *Results) GetToNode(e *ResultEdge) *ResultNode {
	return r.Nodes[e.ToID]
}

// SumFor sums all of the results for the node.
func (r *Results) SumFor(n *ResultNode) *OnsiteResults {
	o := NewOnsiteResults(r.db)

	for _, e := range r.Edges {
		if e.FromID == n.ID {
			o.Add(e.FromResults)
		}
	}
	return o
}

func (r *Results) Len() int      { return len(r.Edges) }
func (r *Results) Swap(i, j int) { r.Edges[i], r.Edges[j] = r.Edges[j], r.Edges[i] }
func (r *Results) Less(i, j int) bool {
	ifrom := r.GetFromNode(r.Edges[i])
	ito := r.GetToNode(r.Edges[i])
	jfrom := r.GetFromNode(r.Edges[j])
	jto := r.GetToNode(r.Edges[j])
	if ifrom.Pathway.GetName() != jfrom.Pathway.GetName() {
		return ifrom.Pathway.GetName() < jfrom.Pathway.GetName()
	}
	if ifrom.Process.GetName() != jfrom.Process.GetName() {
		return ifrom.Process.GetName() != jfrom.Process.GetName()
	}
	if ito.Pathway.GetName() != jto.Pathway.GetName() {
		return ito.Pathway.GetName() < jto.Pathway.GetName()
	}
	return ito.Process.GetName() < jto.Process.GetName()
}

// String prints the results in sorted format, so they are the same every time.
func (r *Results) String() string {
	s := ""
	sort.Sort(r)
	for _, e := range r.Edges {
		from := r.GetFromNode(e)
		to := r.GetToNode(e)
		s += fmt.Sprintf("-----(%s, %s) --> (%s, %s)-----\n%s",
			from.Pathway.GetName(), from.Process.GetName(),
			to.Pathway.GetName(), to.Process.GetName(),
			e.FromResults.String())
	}
	return s
}

// Table creates a table from the results, suitable for outputting to a
// CSV file.
func (r *Results) Table() ([][]string, error) {
	var out [][]string
	sort.Sort(r)
	totals := r.Sum()

	pols, resources := totals.SortOrder()
	o := []string{"Pathway", "Process", "Subprocess", "Downstream pathway", "Downstream process"}
	for _, p := range pols {
		o = append(o, fmt.Sprintf("%s (%v)", p.GetName(), totals.Emissions[p].Dimensions()))
	}
	for _, r := range resources {
		o = append(o, fmt.Sprintf("%s (%v)", r.GetName(), totals.Resources[r].Dimensions()))
	}
	o = append(o, "SCC")
	out = append(out, o)

	o = []string{"Totals", "", "", "", ""}
	for _, p := range pols {
		o = append(o, fmt.Sprintf("%g", totals.Emissions[p].Value()))
	}
	for _, r := range resources {
		o = append(o, fmt.Sprintf("%g", totals.Resources[r].Value()))
	}
	o = append(o, "--")
	out = append(out, o)

	for _, e := range r.Edges {
		subProcesses, _, _ := e.FromResults.SortOrder()
		for _, sp := range subProcesses {
			from := r.GetFromNode(e)
			to := r.GetToNode(e)
			o := make([]string, 5)
			o[0] = from.Pathway.GetName()
			o[1] = from.Process.GetName()
			o[2] = sp.GetName()
			o[3] = to.Pathway.GetName()
			o[4] = to.Process.GetName()
			for _, p := range pols {
				if _, ok := e.FromResults.Emissions[sp][p.(Gas)]; ok {
					o = append(o, fmt.Sprintf("%g", e.FromResults.Emissions[sp][p.(Gas)].Value()))
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
			o = append(o, string(sp.GetSCC()))
			out = append(out, o)
		}
	}
	return out, nil
}

// subProcess is a placeholder SubProcess.
type subProcess string

func (sp subProcess) GetName() string { return string(sp) }
func (sp subProcess) GetSCC() SCC     { return "0000000000" }

// Sum returns a sum of the results.
func (r Results) Sum() *OnsiteResultsNoSubprocess {
	o := NewOnsiteResults(r.db)
	for _, e := range r.Edges {
		o.Add(e.FromResults)
	}
	return o.FlattenSubprocess()
}

// SolveGraph calculates the life cycle resource use and emissions
// of the specified amount of the specified pathway, using a graph-based
// solving method.
func SolveGraph(path Pathway, amount *unit.Unit, db LCADB) *Results {
	proc, output := path.MainProcessAndOutput(db)
	r := NewResults(db)
	amt := output.GetResource(db).ConvertToDefaultUnits(amount, db)
	r.wtp(proc, endUse(0), path, path, output, output, amt, 0)
	return r
}

// wtp recursively calculates well to process results for the given Process
// and pathway.
func (r *Results) wtp(proc, downProc Process, path, downPath Pathway, output, downOutput Output, amount *unit.Unit, nestDepth int) {
	const (
		tolerance    = 1.e-5 // convergence criteria for iterative loops
		maxNestDepth = 1000
	)

	// disregard zero requirements.
	if amount.Value() == 0. {
		return
	}

	n := r.GetNode(proc, path, output)
	forN := r.GetNode(downProc, downPath, downOutput)
	e := r.GetEdge(n, forN)

	// reqsSum is the total amount output of interest from
	// node n required by downstream processes for
	// this life cycle.
	n.addRequiredBy(forN, amount)
	reqSum := n.sumRequiredBy()
	oldReqSum := n.oldReqSum
	n.oldReqSum = reqSum.Clone()

	// Add onsite results to edge instead of replacing them to avoid
	// overwriting results calculated in a deeper nest.
	onsiteTemp := proc.OnsiteResults(path, output, r.db)
	onsite := onsiteTemp.ScaleCopy(amount)
	e.FromResults.Add(onsite)

	// fmt.Println(nestDepth, "running: ", path.GetName(), proc.GetName(), reqSum, oldReqSum)

	// Check for convergence. This calculation has converged if:
	// a) The calculation has already run at least once, and
	// b) The result of this calculation is within the tolerance of the
	//		   previous iteration.
	if oldReqSum != nil &&
		math.Abs(reqSum.Value()-oldReqSum.Value())/
			math.Abs(oldReqSum.Value()) < tolerance {
		return
	}

	// fmt.Println(nestDepth, "Not converged: ", path.GetName(), proc.GetName(), reqSum, oldReqSum)

	for upProc, pov := range onsite.Requirements {
		for upPath, ov := range pov {
			for upOutput, upAmount := range ov {
				// fmt.Println("----->", nestDepth, upPath.GetName(), upProc.GetName(), upAmount)
				r.wtp(upProc, proc, upPath, path, upOutput, output, upAmount, nestDepth+1)
			}
		}
	}
	// Extra steps for the top level calculation.
	if nestDepth == 0 {
		// Add end-use resource use.
		e := r.GetEdge(forN, forN)
		e.FromResults.AddResource(subProcess("End use"), output.GetResource(r.db), amount, r.db)
		e.FromResults.AddRequirement(proc, path, output, amount, r.db)
	}

	if nestDepth > maxNestDepth {
		log.Println("Exceeded max nest depth")
		os.Exit(1)
	}
}
