package greet

import (
	"fmt"
	"sync"

	"github.com/spatialmodel/inmap/emissions/slca"
)

// Pathway is holder for a pathway in the GREET model. Refer to
// the GREET documentation for more information.
type Pathway struct {
	sync.RWMutex
	ID         ModelID       `xml:"id,attr"`
	Name       string        `xml:"name,attr"`
	Notes      string        `xml:"notes,attr"`
	MainOutput slca.OutputID `xml:"main-output,attr"`
	Vertices   []*Vertex     `xml:"vertex"`
	Edges      []*Edge       `xml:"edge"`
	Outputs    []*Output     `xml:"output"`
}

// PathwayLike is an interface for things that can be treated like a
// pathway.
type PathwayLike interface {
	GetName() string
	GetID() ModelID
	GetMainOutput(slca.LCADB) slca.Output
	GetOutput(slca.Resource, slca.LCADB) slca.Output
	GetOutputProcess(*Resource, slca.LCADB) slca.Process
}

// VertexForMainOutput returns the vertex associated with the
// main output of the pathway.
func (path *Pathway) VertexForMainOutput() *Vertex {
	for _, e := range path.Edges {
		if Guid(e.InputID) == Guid(path.MainOutput) {
			for _, v := range path.Vertices {
				if v.ID == e.OutputVertexID {
					return v
				}
			}
		}
	}
	panic(fmt.Errorf("Couldn't find vertex for main output for %#v.", path))
}

// VertexForOutput returns the vertex associated with the given
// pathway output.
func (path *Pathway) VertexForOutput(o *Output, db *DB) *Vertex {
	for _, e := range path.Edges {
		if Guid(e.InputID) == Guid(o.ID) {
			for _, v := range path.Vertices {
				if v.ID == e.OutputVertexID {
					return v
				}
			}
		}
	}
	panic(fmt.Sprintf("Couldn't find vertex for output %#v\n for %#v.", o, path))
}

// GetMainOutput returns the main output from this pathway.
func (path *Pathway) GetMainOutput(_ slca.LCADB) slca.Output {
	for _, o := range path.Outputs {
		if o.ID == path.MainOutput {
			return o
		}
	}
	panic(fmt.Errorf("Couldn't find main output for %#v.", path))
}

// GetOutput returns the output from this pathway that outputs resource r.
func (path *Pathway) GetOutput(rI slca.Resource, lcadb slca.LCADB) slca.Output {
	db := lcadb.(*DB)
	r := rI.(*Resource)
	for _, o := range path.Outputs {
		if o.GetResource(db).(*Resource).IsCompatible(r) {
			p, _ := path.VertexForOutput(o, db).GetProcess(path, o.GetResource(db).(*Resource), db)
			return p.GetOutput(r, db).(OutputLike)
		}
	}
	panic(fmt.Errorf("Pathway %#v doesn't output resource %#v.", path.Name, r.Name))
}

// GetOutputProcess returns the process that outputs resource r.
// It assumes only one process outputs resource r from this pathway.
func (path *Pathway) GetOutputProcess(r *Resource, lcadb slca.LCADB) slca.Process {
	db := lcadb.(*DB)
	for _, o := range path.Outputs {
		if o.GetResource(db).(*Resource).IsCompatible(r) {
			p, _ := path.VertexForOutput(o, db).GetProcess(path, r, db)
			return p
		}
	}
	panic(fmt.Errorf("Pathway %#v doesn't output resource %#v.", path, r))
}

// GetID gets the pathway ID.
func (path *Pathway) GetID() ModelID {
	return path.ID
}

// GetIDStr gets the pathway ID in string format
func (path *Pathway) GetIDStr() string {
	return "Pathway" + string(path.ID)
}

// GetName gets the name of this pathway.
func (path *Pathway) GetName() string {
	return path.Name
}

// Type returns the type: "Pathway".
func (path *Pathway) Type() string {
	return "Pathway"
}

var dummyProc = &StationaryProcess{Name: "Dummy process", ID: "Dummy Process"}

// MainProcessAndOutput returns the process that outputs the main
// output of the receiver, and also returns that output.
func (path *Pathway) MainProcessAndOutput(lcadb slca.LCADB) (slca.Process, slca.Output) {
	db := lcadb.(*DB)
	pathOutput := path.GetMainOutput(db)
	v := path.VertexForMainOutput()
	p, _ := v.GetProcess(path, pathOutput.GetResource(db).(*Resource), db)
	output := p.GetMainOutput(db)
	return p, output
}
