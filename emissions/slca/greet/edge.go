package greet

import (
	"fmt"

	"github.com/spatialmodel/inmap/emissions/slca"
)

// An Edge connects two vertices in a pathway or one vertex in a pathway
// to a pathway output.
type Edge struct {
	// The OutputVertexID references a vertex in the current Pathway.
	// It is the upstream vertex.
	OutputVertexID VertexID `xml:"output-vertex,attr"`

	// Output ID references an output. The actual location of the output
	// should be in a process or another pathway.
	OutputID slca.OutputID `xml:"output-id,attr"`

	// InputVertexID either references a vertex in the current pathway
	// (the downstream vertex), or it matches the ID of one of the outputs
	// from the pathway. If it matches a pathway output, then it means
	// that this edge is to a pathway output rather than another vertex.
	InputVertexID VertexID `xml:"input-vertex,attr"`

	// InputID either matches one of the inputs in the process associated
	// with InputVertexID, or it matches one of the pathway output IDs.
	// If it matches a pathway output, then it means
	// that this edge is to a pathway output rather than another vertex.
	InputID InputID `xml:"input-id,attr"`
}

// GetInputVertex gets the input vertex for this edge.
func (e *Edge) GetInputVertex(db *DB) *Vertex {
	for _, p := range db.Data.Pathways {
		for _, v := range p.Vertices {
			if v.ID == e.InputVertexID {
				return v
			}
		}
	}
	panic(fmt.Sprintf("Did not find input vertex for edge %#v", e))
}

// GetOutputVertex gets the output vertex for this edge.
func (e *Edge) GetOutputVertex(db *DB) *Vertex {
	for _, p := range db.Data.Pathways {
		for _, v := range p.Vertices {
			if v.ID == e.OutputVertexID {
				return v
			}
		}
	}
	panic(fmt.Sprintf("Did not find output vertex for edge %#v", e))
}
