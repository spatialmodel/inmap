package greet

import (
	"fmt"
	"testing"

	"github.com/ctessum/unit"
)

func TestInterpolate(t *testing.T) {

	var y = []*ValueYear{
		&ValueYear{
			Value: "0;mass;0;mass;True;;;;",
			Year:  "2005",
		},
		&ValueYear{
			Value: "10;mass;10;mass;True;;;;",
			Year:  "2015",
		},
		&ValueYear{
			Value: "5;mass;5;mass;True;;;;",
			Year:  "2010",
		},
	}
	type test struct {
		year string
		val  *unit.Unit
	}
	var tests = []test{
		test{year: "2000", val: unit.New(0, unit.Kilogram)},
		test{year: "2005", val: unit.New(0, unit.Kilogram)},
		test{year: "2011", val: unit.New(6, unit.Kilogram)},
		test{year: "2012", val: unit.New(7, unit.Kilogram)},
		test{year: "2013", val: unit.New(8, unit.Kilogram)},
		test{year: "2014", val: unit.New(9, unit.Kilogram)},
		test{year: "2018", val: unit.New(10, unit.Kilogram)},
	}

	for _, tt := range tests {
		db := &DB{
			BasicParameters: &BasicParameters{
				YearSelected: Expression(tt.year + ";;0;;True;;;;;"),
			},
		}
		v := db.InterpolateValue(y)
		if fmt.Sprintf("%g", tt.val) != fmt.Sprintf("%g", v) {
			t.Errorf("for year %s: want %g, got %g", tt.year, tt.val, v)
		}
	}
}

func TestInterpolateEmissions(t *testing.T) {
	y := []*EmissionYear{
		&EmissionYear{
			Year: "2005",
			Emissions: []*Emission{
				&Emission{
					Ref: "84908646", Factor: "0;3;False;emission_factor;;;;;"},
			},
		},
		&EmissionYear{
			Year: "2010",
			Emissions: []*Emission{
				&Emission{Ref: "84908646", Factor: "0;2;False;emission_factor;;;;;"},
			},
		},
		&EmissionYear{
			Year: "2015",
			Emissions: []*Emission{
				&Emission{Ref: "84908646", Factor: "0;1;False;emission_factor;;;;;"},
			},
		},
	}

	type test struct {
		year string
		val  *unit.Unit
	}
	var tests = []test{
		test{year: "2000", val: unit.New(3, kgPerJoule)},
		test{year: "2005", val: unit.New(3, kgPerJoule)},
		test{year: "2006", val: unit.New(2.8, kgPerJoule)},
		test{year: "2007", val: unit.New(2.6, kgPerJoule)},
		test{year: "2008", val: unit.New(2.4, kgPerJoule)},
		test{year: "2009", val: unit.New(2.2, kgPerJoule)},
		test{year: "2010", val: unit.New(2, kgPerJoule)},
		test{year: "2011", val: unit.New(1.8, kgPerJoule)},
		test{year: "2012", val: unit.New(1.6, kgPerJoule)},
		test{year: "2013", val: unit.New(1.4, kgPerJoule)},
		test{year: "2014", val: unit.New(1.2, kgPerJoule)},
		test{year: "2015", val: unit.New(1, kgPerJoule)},
		test{year: "2018", val: unit.New(1, kgPerJoule)},
	}

	for _, tt := range tests {
		db := &DB{
			BasicParameters: &BasicParameters{
				YearSelected: Expression(tt.year + ";0;True;date;;;;;"),
			},
			Data: &Data{
				Gases: []*Gas{
					&Gas{Ref: "", Share: "", Name: "Test Gas 1", ID: "84908646",
						CRatio:                 "0;0;True;percentage;;84908646967;;;",
						SRatio:                 "0;0;True;percentage;;84908646973;;;",
						GlobalWarmingPotential: "0;0;True;percentage;;84908646961;;;"},
				},
			},
		}
		_, v := db.interpolateEmissions(y, nil)
		if fmt.Sprintf("%.4g", tt.val.Value()) != fmt.Sprintf("%.4g", v[0].Value()) {
			t.Errorf("for year %s: want %g, got %g", tt.year, tt.val, v[0])
		}
	}
}
