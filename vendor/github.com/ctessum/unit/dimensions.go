package unit

import (
	"bytes"
	"fmt"
	"sort"
)

// Dimension is a type representing an SI base dimension or other
// orthogonal dimension. If a new dimension is desired for a
// domain-specific problem, NewDimension should be used. Integers
// should never be cast as type dimension
//	// Good: Create a package constant with an init function
//	var MyDimension unit.Dimension
//	init(){
//		MyDimension = NewDimension("my")
//	}
//	main(){
//		var := MyDimension(28.2)
//	}
type Dimension int

func (d Dimension) String() string {
	switch {
	case d == reserved:
		return "reserved"
	case d < Dimension(len(symbols)):
		return symbols[d]
	default:
		panic("unit: illegal dimension")
	}
}

const (
	// SI Base Units
	reserved Dimension = iota
	// CurrentDim is the dimmension representing current.
	CurrentDim
	// LengthDim is the dimension representing length.
	LengthDim
	// LuminousIntensityDim is the dimension representing luminous intensity
	LuminousIntensityDim
	// MassDim is the dimension representing mass
	MassDim
	// TemperatureDim is the dimension representing temperature
	TemperatureDim
	// TimeDim is the dimension representing time.
	TimeDim
	// AngleDim is the dimension representing angle.
	AngleDim
)

var (
	symbols = []string{
		CurrentDim:           "A",
		LengthDim:            "m",
		LuminousIntensityDim: "cd",
		MassDim:              "kg",
		TemperatureDim:       "K",
		TimeDim:              "s",
		AngleDim:             "rad",
	}

	// for guaranteeing there aren't two identical symbols
	dimensions = map[string]Dimension{
		"A":   CurrentDim,
		"m":   LengthDim,
		"cd":  LuminousIntensityDim,
		"kg":  MassDim,
		"K":   TemperatureDim,
		"s":   TimeDim,
		"rad": AngleDim,

		// Reserve common SI symbols
		// base units
		"mol": reserved,
		// prefixes
		"Y":  reserved,
		"Z":  reserved,
		"E":  reserved,
		"P":  reserved,
		"T":  reserved,
		"G":  reserved,
		"M":  reserved,
		"k":  reserved,
		"h":  reserved,
		"da": reserved,
		"d":  reserved,
		"c":  reserved,
		"μ":  reserved,
		"n":  reserved,
		"p":  reserved,
		"f":  reserved,
		"a":  reserved,
		"z":  reserved,
		"y":  reserved,
		// SI Derived units with special symbols
		"sr":  reserved,
		"F":   reserved,
		"C":   reserved,
		"S":   reserved,
		"H":   reserved,
		"V":   reserved,
		"Ω":   reserved,
		"J":   reserved,
		"N":   reserved,
		"Hz":  reserved,
		"lx":  reserved,
		"lm":  reserved,
		"Wb":  reserved,
		"W":   reserved,
		"Pa":  reserved,
		"Bq":  reserved,
		"Gy":  reserved,
		"Sv":  reserved,
		"kat": reserved,
		// Units in use with SI
		"ha": reserved,
		"L":  reserved,
		"l":  reserved,
		// Units in Use Temporarily with SI
		"bar": reserved,
		"b":   reserved,
		"Ci":  reserved,
		"R":   reserved,
		"rd":  reserved,
		"rem": reserved,
	}
)

// TODO: Should we actually reserve "common" SI unit symbols ("N", "J", etc.) so there isn't confusion
// TODO: If we have a fancier ParseUnit, maybe the 'reserved' symbols should be a different map
// 		map[string]string which says how they go?

// Dimensions represent the dimensionality of the unit in powers
// of that dimension. If a key is not present, the power of that
// dimension is zero. Dimensions is used in conjuction with New.
type Dimensions map[Dimension]int

var (
	// Dimless is for dimensionless numbers.
	Dimless = Dimensions{}
	// Joule is a unit of energy [kg m2 s-2]
	Joule = Dimensions{
		MassDim:   1,
		LengthDim: 2,
		TimeDim:   -2,
	}
	// Meter is a meter.
	Meter = Dimensions{
		LengthDim: 1,
	}
	// Meter2 is a square meter
	Meter2 = Dimensions{
		LengthDim: 2,
	}
	// Meter3 is a cubic meter
	Meter3 = Dimensions{
		LengthDim: 3,
	}
	// Meter3PerSecond is a cubic meter per second.
	Meter3PerSecond = Dimensions{
		LengthDim: 3,
		TimeDim:   -1,
	}

	// Kelvin is a degree kelvin [K].
	Kelvin = Dimensions{
		TemperatureDim: 1,
	}

	// KilogramPerMeter3 is density.
	KilogramPerMeter3 = Dimensions{
		MassDim:   1,
		LengthDim: -3,
	}
	// Pascal is a unit of pressure [kg m-1 s-2]
	Pascal = Dimensions{
		MassDim:   1,
		LengthDim: -1,
		TimeDim:   -2,
	}
	// Kilogram is a kilogram.
	Kilogram = Dimensions{
		MassDim: 1,
	}
	// Watt is a unit of power [kg m2 s-3]
	Watt = Dimensions{
		MassDim:   1,
		LengthDim: 2,
		TimeDim:   -3,
	}
	// Herz is a unit of frequency [s-1]
	Herz = Dimensions{
		TimeDim: -1,
	}
	// Second is a second (time).
	Second = Dimensions{
		TimeDim: 1,
	}
	// MeterPerSecond is speed.
	MeterPerSecond = Dimensions{
		TimeDim:   -1,
		LengthDim: 1,
	}
	// MeterPerSecond2 is acceleration.
	MeterPerSecond2 = Dimensions{
		TimeDim:   -2,
		LengthDim: 1,
	}
)

func (d Dimensions) String() string {
	// Map iterates randomly, but print should be in a fixed order. Can't use
	// dimension number, because for user-defined dimension that number may
	// not be fixed from run to run.
	atoms := make(unitPrinters, 0, len(d))
	for dimension, power := range d {
		if power != 0 {
			atoms = append(atoms, atom{dimension, power})
		}
	}
	sort.Sort(atoms)
	var b bytes.Buffer
	for i, a := range atoms {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%s", a.Dimension)
		if a.pow != 1 {
			fmt.Fprintf(&b, "^%d", a.pow)
		}
	}

	return b.String()
}

type atom struct {
	Dimension
	pow int
}

type unitPrinters []atom

func (u unitPrinters) Len() int {
	return len(u)
}

func (u unitPrinters) Less(i, j int) bool {
	return (u[i].pow > 0 && u[j].pow < 0) || u[i].String() < u[j].String()
}

func (u unitPrinters) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

// NewDimension returns a new dimension variable which will have a
// unique representation across packages to prevent accidental overlap.
// The input string represents a symbol name which will be used for printing
// Unit types. This symbol may not overlap with any of the SI base units
// or other symbols of common use in SI ("kg", "J", "μ", etc.). A list of
// such symbols can be found at http://lamar.colostate.edu/~hillger/basic.htm or
// by consulting the package source. Furthermore, the provided symbol is also
// forbidden from overlapping with other packages calling NewDimension. NewDimension
// is expecting to be used only during initialization, and as such it will panic
// if the symbol matching an existing symbol
// NewDimension should only be called for unit types that are actually orthogonal
// to the base dimensions defined in this package. Please see the package-level
// documentation for further explanation. Calls to NewDimension are not thread safe.
func NewDimension(symbol string) Dimension {
	_, ok := dimensions[symbol]
	if ok {
		panic("unit: dimension string \"" + symbol + "\" already used")
	}
	symbols = append(symbols, symbol)
	d := Dimension(len(symbols) - 1)
	dimensions[symbol] = d
	return d
}
