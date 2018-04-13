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

import "github.com/ctessum/unit"

// Guid is a globally unique ID
type Guid string

// LCADB specifies the methods that an LCA database must have to
// work with this framework.
type LCADB interface {
	// EndUses returns the potential end Pathways and their names.
	EndUses() ([]Pathway, []string)

	// EndUseFromID returns an end use Pathway when given its ID.
	EndUseFromID(ID string) (Pathway, error)

	// SpatialSCCs returns a list of all SCC codes used by processes in the
	// receiver.
	SpatialSCCs() []SCC

	// GetYear returns the analysis year.
	GetYear() float64
}

// Gas represents a type of emission.
type Gas interface {
	// GetName returns the name of this gas.
	GetName() string

	// GetID returns the ID of this gas.
	GetID() string
}

// Resource represents a material that can be input to or output from
// a process.
type Resource interface {
	// GetName returns the name of this resource.
	GetName() string

	// GetID returns the ID of this resource.
	GetID() string

	// ConvertToDefaultUnits converts the speceified amount to the
	// default units of the receiver.
	ConvertToDefaultUnits(amount *unit.Unit, db LCADB) *unit.Unit
}

// Process represents a process in a life cycle.
type Process interface {
	// Type returns the ProcessType of the reciever.
	Type() ProcessType

	// SpatialRef returns the spatial reference information associated
	// with the receiver.
	SpatialRef() *SpatialRef

	// GetName returns the name of the receiver.
	GetName() string

	// GetIDStr returns the ID of the receiver in string format.
	GetIDStr() string

	GetOutput(Resource, LCADB) Output
	GetMainOutput(LCADB) Output

	// OnsiteResults returns the onsite emissions and input resource
	// requirements per unit of the specified Output, as part
	// of the specified Pathway.
	OnsiteResults(Pathway, Output, LCADB) *OnsiteResults
}

// ProcessType specifies a type of a Process.
type ProcessType int

const (
	// Stationary specifies a process is one that is at
	// a fixed location or locations.
	Stationary ProcessType = iota + 1

	// Transportation specifies a process is one that needs to be routed from a
	// source to a destination.
	Transportation

	// Vehicle specifies an end-use vehicle.
	Vehicle

	// NoSpatial represents a process that does not have any spatial information.
	NoSpatial
)

// Pathway represents a series of processes in a life cycle.
type Pathway interface {
	// GetName gets the name of this Pathway.
	GetName() string

	// GetIDStr returns the ID of the receiver in string format.
	GetIDStr() string

	// MainProcessAndOutput returns the process that outputs the main
	// output of the receiver, and also returns that output.
	MainProcessAndOutput(LCADB) (Process, Output)
}

// OutputID holds the ID code for an output
type OutputID Guid

// Output represents an output from a process or pathway.
type Output interface {
	// GetResource returns the Resource that is output by the receiver.
	GetResource(LCADB) Resource

	// GetID returns the ID of the receiver.
	GetID() OutputID
}

// SubProcess gives information about the specific source
// of emissions or resource use within a process.
type SubProcess interface {
	GetName() string
	GetSCC() SCC
}

// SCC is an EPA source classification code
type SCC string
