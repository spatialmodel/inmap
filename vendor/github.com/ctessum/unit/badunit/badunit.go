package badunit

import "github.com/ctessum/unit"

// HorsePower creates a new unit from an amount of horsepower hp.
func HorsePower(hp float64) *unit.Unit {
	return unit.New(hp*745.699872, unit.Watt)
}

// Ton creates a new unit from a number of short tons t.
func Ton(t float64) *unit.Unit {
	return unit.New(t*907.185, unit.Kilogram)
}

// Pound creates a new unit from a number of (mass) pounds p.
func Pound(p float64) *unit.Unit {
	return unit.New(p*0.45359237, unit.Kilogram)
}

// Mile creates a new unit from a number of miles m.
func Mile(m float64) *unit.Unit {
	return unit.New(m*1609.34, unit.Meter)
}

// Hour creates a new unit from a number of hours h.
func Hour(h float64) *unit.Unit {
	return unit.New(h*60*60, unit.Second)
}

// Minute creates a new unit from a number of minutes m.
func Minute(m float64) *unit.Unit {
	return unit.New(m*60, unit.Second)
}

// MilePerHour creates a new unit from a number of mi/h mph.
func MilePerHour(mph float64) *unit.Unit {
	return unit.New(mph*0.44704, unit.MeterPerSecond)
}

// KiloWattHour creates a new unit from a number of kwh.
func KiloWattHour(kwh float64) *unit.Unit {
	return unit.New(kwh*3600000, unit.Joule)
}

// Btu creates a new unit from a number of british thermal units.
func Btu(btu float64) *unit.Unit {
	return unit.New(btu*1055.06, unit.Joule)
}

// MmBtu creates a new unit from a number of millions of british thermal units.
func MmBtu(mmbtu float64) *unit.Unit {
	return unit.New(mmbtu*1055.06*1.e6, unit.Joule)
}

// Gallon creates a new unit from a number of gallons.
func Gallon(g float64) *unit.Unit {
	return unit.New(g*0.00378541, unit.Meter3)
}

// Foot3 creates a new unit from a number of cubic feet.
func Foot3(f float64) *unit.Unit {
	return unit.New(f*0.0283168, unit.Meter3)
}

// Foot creates a new unit from a number of feet.
func Foot(f float64) *unit.Unit {
	return unit.New(f*0.3048, unit.Meter)
}

// FootPerSecond creates a new unit from a number of feet per second.
func FootPerSecond(f float64) *unit.Unit {
	return unit.New(f*0.3048, unit.MeterPerSecond)
}

// Foot3PerSecond creates a new unit from a number of cubic feet per second.
func Foot3PerSecond(f float64) *unit.Unit {
	return unit.New(f*0.0283168, unit.Meter3PerSecond)
}

// Fahrenheit creates a new unit from a number of degrees Fahrenheit.
func Fahrenheit(f float64) *unit.Unit {
	return unit.New((f+459.67)*5./9., unit.Kelvin)
}
