package greet

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/unit"
)

// DebugLevel sets the amount of debugging output that is written to the
// console. Higher numbers lead to more output.
var DebugLevel = 0

// DB is a holder for the GREET database.
type DB struct {
	Version         string           `xml:"version,attr"`
	BasicParameters *BasicParameters `xml:"basic_parameters"`
	Data            *Data            `xml:"data"`

	// parameters that have been converted from strings into values
	processedParameters map[string]*unit.Unit

	// SpatialSCCs are all the SCC codes used by this database
	// for spatialization. SCC codes that are not
	// used for spatialization are not included here.
	SpatialSCC []slca.SCC
}

// Data is a holder for the LCA data within the GREET
// database.
type Data struct {
	Pathways                []*Pathway               `xml:"pathways>pathway"`
	ResourceGroups          []*ResourceGroup         `xml:"resources>groups>group"`
	Resources               []*Resource              `xml:"resources>resources>resource"`
	TransportationProcesses []*TransportationProcess `xml:"processes>transportation"`
	StationaryProcesses     []*StationaryProcess     `xml:"processes>stationary"`
	Technologies            []*Technology            `xml:"technologies>technology"`
	Modes                   []*Mode                  `xml:"modes>mode"`
	Locations               Locations                `xml:"locations"`
	GasGroups               []*GasGroup              `xml:"gases>groups>group"`
	Gases                   []*Gas                   `xml:"gases>gases>gas"`

	// VehicleTechnologyLag gives the lag in years between the current year
	// and the vehicle model year.
	VehicleTechnologyLag struct {
		Value Expression `xml:"value,attr"`
	} `xml:"vehicles>vehicle_technology_lag"`

	Vehicles    []*Vehicle    `xml:"vehicles>vehicle"`
	Mixes       []*Mix        `xml:"mixes>mix"`
	InputTables []*InputTable `xml:"inputs>input"`
}

// Guid is a globally unique ID
type Guid string

// VertexID holds the ID code for a vertex
type VertexID Guid

// InputID holds the ID code for an input
type InputID Guid

// ModeID holds the ID code for a travel mode
type ModeID Guid

// ModelID holds the ID code for a Process model
// (TransportationProcess or StationaryProcess)
type ModelID string

// ResourceID holds the ID code for a resource
type ResourceID string

// FuelShareID holds the ID code for a FuelShare
type FuelShareID string

// TechnologyID holds the ID code for a technology
type TechnologyID string

// GasID holds the ID code for a gas
type GasID string

// LocationID holds the ID code for a location
type LocationID string

// LocationGroupID holds the ID code for a location group
type LocationGroupID string

// BasicParameters holds the GREET database basic parameters.
type BasicParameters struct {
	YearSelected Expression `xml:"year_selected,attr"`
	LHV          bool       `xml:"lhv,attr"`
}

// GetYear returns the analysis year for this database.
func (db *DB) GetYear() float64 {
	yy := db.evalExpr(db.BasicParameters.YearSelected)
	return yy.Value()
}

// GetPathwayMixOrVehicleFromName returns the pathway, mix, or vehicle with
// the given name.
func (db *DB) GetPathwayMixOrVehicleFromName(name string) (slca.Pathway, error) {
	for _, pathway := range db.Data.Pathways {
		if pathway.Name == name {
			return pathway, nil
		}
	}
	for _, mix := range db.Data.Mixes {
		if mix.Name == name {
			return mix, nil
		}
	}
	for _, vehicle := range db.Data.Vehicles {
		if vehicle.Name == name {
			return vehicle, nil
		}
	}
	return nil, fmt.Errorf("Couldn't find a pathway, mix, or vehicle named %s.", name)
}

// EndUseFromID returns the pathway mix, or vehicle with the given id.
func (db *DB) EndUseFromID(ID string) (slca.Pathway, error) {
	for _, pathway := range db.Data.Pathways {
		if pathway.GetIDStr() == ID {
			return pathway, nil
		}
	}
	for _, mix := range db.Data.Mixes {
		if mix.GetIDStr() == ID {
			return mix, nil
		}
	}
	for _, vehicle := range db.Data.Vehicles {
		if vehicle.GetIDStr() == ID {
			return vehicle, nil
		}
	}
	return nil, fmt.Errorf("Couldn't find a pathway, mix, or vehicle with ID %s.", ID)
}

// GetPathway returns the pathway with the given id.
func (db *DB) GetPathway(ID ModelID) *Pathway {
	for _, pathway := range db.Data.Pathways {
		if pathway.ID == ID {
			return pathway
		}
	}
	panic(fmt.Errorf("Couldn't find a pathway with ID %v.", ID))
}

// EndUses returns information about the pathways, mixes, and
// vehicles in this database.
func (db *DB) EndUses() ([]slca.Pathway, []string) {
	var pathways []slca.Pathway
	var ids []string
	for _, pathway := range db.Data.Pathways {
		pathways = append(pathways, pathway)
		ids = append(ids, string(pathway.GetIDStr()))
	}
	for _, mix := range db.Data.Mixes {
		pathways = append(pathways, mix)
		ids = append(ids, string(mix.GetIDStr()))
	}
	for _, vehicle := range db.Data.Vehicles {
		pathways = append(pathways, vehicle)
		ids = append(ids, string(vehicle.GetIDStr()))
	}
	return pathways, ids
}

// GetResource returns the resource with the specified ID. The requester input is only
// used for debugging if the resource is not found.
func (db *DB) GetResource(id ResourceID, requester interface{}) *Resource {
	for _, r := range db.Data.Resources {
		if r.ID == id {
			return r
		}
	}
	panic(fmt.Errorf("Couldn't find resource for %#v.", requester))
}

// GetResourceFromName returns the resource with the specified name.
func (db *DB) GetResourceFromName(name string) *Resource {
	for _, r := range db.Data.Resources {
		if r.Name == name {
			return r
		}
	}
	panic(fmt.Errorf("Couldn't find resource %s.", name))
}

// Load loads the GREET database from an XML file.
func Load(dbFile io.Reader) *DB {
	db := new(DB)
	d := xml.NewDecoder(dbFile)
	err := d.Decode(db)
	if err != nil {
		panic(err)
	}
	db.processedParameters = make(map[string]*unit.Unit)

	// Add on extra info for vehicles.
	db.Data.Resources = append(db.Data.Resources, drivingResource)

	return db
}

// Write writes out a database to w in indented XML format.
func (db *DB) Write(w io.Writer) (int, error) {
	b, err := xml.MarshalIndent(db, "", "  ")
	if err != nil {
		return 0, err
	}
	return w.Write(b)
}

// GetResultVars returns the gases and resources that
// can be considered as model result types.
func (db *DB) GetResultVars() (ids, names []string) {
	for _, g := range db.Data.Gases {
		ids = append(ids, g.GetID())
		names = append(names, g.Name)
	}
	for _, r := range db.Data.Resources {
		if r.CanBePrimary {
			ids = append(ids, r.GetID())
			names = append(names, r.Name)
		}
	}
	return
}

// GetGas returns the gas in the database with the specified name. It returns
// an error if there is no match.
func (db *DB) GetGas(name string) (*Gas, error) {
	for _, g := range db.Data.Gases {
		if g.Name == name {
			return g, nil
		}
	}
	return nil, fmt.Errorf("could not find gas %s", name)
}

func (db *DB) gasFromID(id GasID) *Gas {
	for _, g := range db.Data.Gases {
		if g.ID == id {
			return g
		}
	}
	panic(fmt.Sprintf("Couldn't find gas with ID %s.", id))
}

// getGasFromID returns the gas in the database with the specified ID. It returns
// an error if there is no match.
func (db *DB) getGasFromID(ID GasID) *Gas {
	for _, g := range db.Data.Gases {
		if g.ID == ID {
			return g
		}
	}
	panic(fmt.Errorf("could not find gas %s", ID))
}

// reset resets the database.
func (db *DB) reset() {
	db.processedParameters = make(map[string]*unit.Unit)
	for _, p := range db.Data.StationaryProcesses {
		p.results = nil
	}
	for _, p := range db.Data.TransportationProcesses {
		p.results = nil
	}
}

// SpatialSCCs returns all of the SCC codes used by this database
// for spatialization.
func (db *DB) SpatialSCCs() []slca.SCC {
	return db.SpatialSCC
}

// handle handles errors.
func handle(err error) {
	if err != nil {
		panic(err)
	}
}

// debug logs debugging statements.
func debug(level int, format string, v ...interface{}) {
	if level <= DebugLevel {
		log.Printf(format, v...)
	}
}
