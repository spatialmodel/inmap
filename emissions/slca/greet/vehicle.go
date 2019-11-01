package greet

import (
	"fmt"
	"sync"

	"github.com/ctessum/unit"

	"github.com/spatialmodel/inmap/emissions/slca"
)

// Vehicle is a holder for information about a vehicle in the GREET
// database.
type Vehicle struct {
	sync.Mutex
	ID    ModelID `xml:"id,attr"`
	Name  string  `xml:"name,attr"`
	Notes string  `xml:"notes,attr"`

	// Modes gives information on the mode the vehicle is operating in.
	// Plug-in hybrid vehicles can operate in multiple modes.
	Modes []struct {
		Name string  `xml:"name,attr"`
		ID   ModelID `xml:"id,attr"`

		VMTShare []*ValueYear `xml:"vmtShare>year"`

		// Plant holds information about a vehicle power plant.
		Plant struct {
			Name  string `xml:"name,attr"`
			ID    Guid   `xml:"id,attr"`
			Notes string `xml:"notes,attr"`

			Fuel struct {
				Resource           ResourceID   `xml:"ref,attr"`
				Pathway            ModelID      `xml:"pathway,attr"`
				Consumption        []*ValueYear `xml:"consumption>year"`
				ChargingEfficiency []*ValueYear `xml:"charging_efficiency>year"`
			} `xml:"fuel"`

			Emissions []struct {
				Gas   GasID        `xml:"id,attr"`
				Value []*ValueYear `xml:"value_ts>year"`
			} `xml:"emission"`
		} `xml:"plant"`
	} `xml:"mode"`

	LifetimeVMT []*ValueYear `xml:"lifetime_vmt>year"`

	// Manufacturing holds information about
	// emissions and resource use from the manufacturing
	// of groups of vehicle components.
	Manufacturing []struct {
		Name string `xml:"name,attr"`

		// Materials holds information on materials
		// used in vehicle manufacturing.
		Materials []struct {
			Resource   ResourceID `xml:"resource_id,attr"`
			Source     ModelID    `xml:"entity_id,attr"`
			SourceType string     `xml:"source_type,attr"`

			// Quantity is the amount needed of this
			// pathway per unit.
			Quantity []*ValueYear `xml:"quantity>year"`

			// Replacements is the number of times each
			// unit will need to be replaced during
			// the vehicle lifetime.
			Replacements []*ValueYear `xml:"replacements>year"`

			// Units is the number of units per vehicle.
			Units []*ValueYear `xml:"units>year"`
		} `xml:"material"`
	} `xml:"manufacturing"`

	NonCombustionEmissions []struct {
		Gas    GasID        `xml:"id,attr"`
		Values []*ValueYear `xml:"year"`
	} `xml:"nonCombustionEmission"`

	// SCC is the SCC code for this vehicle.
	SCC slca.SCC

	results *slca.OnsiteResults
}

// SpatialRef returns the spatial reference for the receiver.
// TODO: Need to implement something for this.
func (v *Vehicle) SpatialRef(aqm string) *slca.SpatialRef {
	return &slca.SpatialRef{Type: slca.Vehicle, AQM: aqm}
}

// GetName returns the name of this vehicle.
func (v *Vehicle) GetName() string {
	return v.Name
}

// GetID returns the vehicle ID
func (v *Vehicle) GetID() ModelID {
	return v.ID
}

// GetIDStr returns the ID of this vehicle in string format
func (v *Vehicle) GetIDStr() string {
	return "Vehicle" + string(v.ID)
}

// Type returns the type of this process.
func (v *Vehicle) Type() slca.ProcessType {
	return slca.Vehicle
}

// GetOutput always returns nil because vehicles have no process outputs.
func (v *Vehicle) GetOutput(_ slca.Resource, _ slca.LCADB) slca.Output {
	return nil
}

// GetMainOutput always returns nil because vehicles have no process outputs.
func (v *Vehicle) GetMainOutput(_ slca.LCADB) slca.Output {
	return nil
}

// GetVehicleFromID finds the vehicle in the database with the matching ID.
func (db *DB) GetVehicleFromID(ID ModelID) *Vehicle {
	for _, v := range db.Data.Vehicles {
		if ID == v.ID {
			return v
		}
	}
	panic(fmt.Errorf("could not find vehicle %v in the database", ID))
}

// GetVehicleFromName finds the vehicle in the database with the matching ID.
func (db *DB) GetVehicleFromName(name string) *Vehicle {
	for _, v := range db.Data.Vehicles {
		if name == v.Name {
			return v
		}
	}
	panic(fmt.Errorf("could not find vehicle %v in the database", name))
}

// OnsiteResults calculates the from-vehicle emissions and vehicle
// resource use per unit output as part of a life cycle calculation,
// as well as requirements of other pathways.
// The returned results are emissions and resource use per meter
// driven by the vehicle.
func (v *Vehicle) OnsiteResults(_ slca.Pathway, _ slca.Output, lcadb slca.LCADB) *slca.OnsiteResults {
	v.Lock()
	defer v.Unlock()
	// Return saved results if prevously calculated.
	if v.results != nil {
		return v.results
	}
	functionalUnit := unit.New(1, unit.Meter)
	db := lcadb.(*DB)
	r := slca.NewOnsiteResults(db)

	lag := db.evalExpr(db.Data.VehicleTechnologyLag.Value).Value()

	sp := subprocess{name: "operation", scc: v.SCC}

	// Add emissions and requirements for the different
	// operating modes.
	for _, m := range v.Modes {
		modeShare := db.InterpolateValueWithLag(m.VMTShare, lag)

		// Add in power plant emissions.
		for _, e := range m.Plant.Emissions {
			g := db.gasFromID(e.Gas)
			v := db.InterpolateValueWithLag(e.Value, lag)
			v.Mul(modeShare)
			v.Mul(functionalUnit) // We want results for one meter.
			r.AddEmission(sp, g, v)
		}

		// Add in power plant fuel use
		resource := db.GetResource(m.Plant.Fuel.Resource, v)
		path := db.GetPathway(m.Plant.Fuel.Pathway)
		consumption := db.InterpolateValueWithLag(m.Plant.Fuel.Consumption, lag)
		chargingEfficiency := db.InterpolateValueWithLag(m.Plant.Fuel.ChargingEfficiency, lag)
		// Calculate the amount of energy used by this mode.
		amount := unit.Div(consumption, chargingEfficiency)
		r.AddRequirement(
			path.GetOutputProcess(resource, db),
			path,
			path.GetOutput(resource, db),
			unit.Mul(unit.Mul(amount, modeShare), functionalUnit), // Energy use to travel one meter.
			db,
		)
	}

	// Add in manufacturing requirements.
	one := unit.New(1, unit.Dimless)
	for _, mfg := range v.Manufacturing {
		for _, mat := range mfg.Materials {
			lifetimeVMT := db.InterpolateValueWithLag(v.LifetimeVMT, lag)

			resource := db.GetResource(mat.Resource, v)
			var path slca.Pathway
			switch mat.SourceType {
			case "Pathway":
				path = db.GetPathway(mat.Source)
			case "Mix":
				path = db.GetMix(mat.Source)
			default:
				panic(fmt.Errorf("greet: vehicle: invalid source type %s", mat.SourceType))
			}
			quantity := db.InterpolateValueWithLag(mat.Quantity, lag)
			replacements := db.InterpolateValueWithLag(mat.Replacements, lag)
			units := db.InterpolateValueWithLag(mat.Units, lag)

			amt := unit.Mul(quantity, units) // amount per vehicle
			// Amount per lifetime, which is the amount per vehicle
			// time one + the number of replacements.
			amt.Mul(unit.Add(replacements, one))
			// Amount for one meter, which is the amout per lifetime,
			// divided by lifetime meters traveled, times one meter.
			amt.Div(unit.Div(lifetimeVMT, functionalUnit))

			r.AddRequirement(
				path.(PathwayLike).GetOutputProcess(resource, db),
				path,
				path.(PathwayLike).GetOutput(resource, db),
				amt,
				db,
			)
		}
	}

	// Add in non-combustion emissions.
	for _, e := range v.NonCombustionEmissions {
		g := db.gasFromID(e.Gas)
		v := db.InterpolateValueWithLag(e.Values, lag)
		v.Mul(functionalUnit) // We want results for one meter.
		r.AddEmission(sp, g, v)
	}

	r.Div(functionalUnit)
	v.results = r
	return r
}

var drivingOutput = &Output{ID: "Driving", Ref: "DrivingRes"}
var drivingResource = &Resource{ID: "DrivingRes"}
var drivingPathway = &Pathway{ID: "Driving", Name: "Driving"}

// MainProcessAndOutput returns the process that outputs the main
// output of the receiver, and also returns that output.
func (v *Vehicle) MainProcessAndOutput(db slca.LCADB) (slca.Process, slca.Output) {
	return v, drivingOutput
}
