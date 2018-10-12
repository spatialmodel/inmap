package greet

import (
	"fmt"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/unit"
)

// Technology is a holder for the GREET technology type. Refer to the GREET
// documentation for more information.
type Technology struct {
	ID            TechnologyID    `xml:"id,attr"`
	Name          string          `xml:"name,attr"`
	InputRef      ResourceID      `xml:"inputRef,attr"`
	OutputRef     ResourceID      `xml:"outputRef,attr"`
	MassTransfer  string          `xml:"massTransfer,attr"`
	BaseTech      string          `xml:"basetech,attr"`
	EmissionsYear []*EmissionYear `xml:"year"`

	SCC slca.SCC
}

// GetEmissions returns the emissions associated with this technology
func (t *Technology) GetEmissions(db *DB) ([]*Gas, []*unit.Unit) {
	return db.interpolateEmissions(t.EmissionsYear, t.GetInputResource(db))
}

// GetInputResource gets the resource used as an input for this technology
func (t *Technology) GetInputResource(db *DB) *Resource {
	if t.InputRef == ResourceID("-1") {
		panic(fmt.Errorf("No resource for %#v.", t))
	}
	return db.GetResource(t.InputRef, t)
}

// GetOutputResource gets the resource output by this technology
func (t *Technology) GetOutputResource(db *DB) *Resource {
	if t.OutputRef == ResourceID("-1") {
		panic(fmt.Errorf("No resource for %#v.", t))
	}
	return db.GetResource(t.OutputRef, t)
}

// GetID returns the ID of this Technology.
func (t *Technology) GetID() VertexID {
	return VertexID("Technology" + t.ID)
}

// GetName returns the name of this technology.
func (t *Technology) GetName() string {
	return t.Name
}

// GetSCC returns the SCC code associated with this technology.
func (t *Technology) GetSCC() slca.SCC {
	return t.SCC
}

// TechnologyShare holds information about the fraction of work done by
// a technology.
type TechnologyShare struct {
	Ref              TechnologyID `xml:"ref,attr"`
	Share            Param        `xml:"share"`
	AccountInBalance bool         `xml:"account_in_balance,attr"`
}

// GetTechnology returns the Technology associated with this TechnologyShare
func (ts *TechnologyShare) GetTechnology(db *DB) *Technology {
	for _, t := range db.Data.Technologies {
		if ts.Ref == t.ID {
			return t
		}
	}
	panic(fmt.Sprintf("Couldn't find technology for %#v.", ts))
}

// GetShare returns the share associated with this TechnologyShare
func (ts *TechnologyShare) GetShare(db *DB) *unit.Unit {
	return db.InterpolateValue(ts.Share.ValueYears)
}
