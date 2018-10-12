package greet

import (
	"fmt"
	"log"

	"github.com/spatialmodel/inmap/emissions/slca"
)

// Vertex is a holder for the Vertex datatype in the GREET
// database.
type Vertex struct {
	ID      VertexID `xml:"id,attr"`
	ModelID ModelID  `xml:"model-id,attr"`
	Type    string   `xml:"type,attr"` // 0=process; 1=pathway; 2=mix
}

// GetProcess finds the corresponding process model for this vertex. It can be a
// *StationaryProcess, *TransportationProcess, *Pathway, or *Mix.
// If it is a stationary or transportation process, it is assumed to be
// part of the requestingPathway.
func (this *Vertex) GetProcess(requestingPath *Pathway, resource *Resource,
	db *DB) (slca.Process, *Pathway) {
	switch this.Type {
	case "0":
		for _, proc := range db.Data.StationaryProcesses {
			if proc.GetID() == this.ModelID {
				return proc, requestingPath
			}
		}
		for _, proc := range db.Data.TransportationProcesses {
			if proc.GetID() == this.ModelID {
				return proc, requestingPath
			}
		}
	case "1":
		for _, p := range db.Data.Pathways {
			if p.GetID() == this.ModelID {
				return p.GetOutputProcess(resource, db), p
			}
		}
	case "2":
		for _, m := range db.Data.Mixes {
			if m.GetID() == this.ModelID {
				return m, &mixPathway
			}
		}
	default:
		log.Panicf("Invalid type %v for %#v", this.Type, this)
	}
	panic(fmt.Errorf("Vertex %#v has no process model.\n", this))
}
