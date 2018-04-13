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

package aep

import (
	"bytes"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/ctessum/unit"
)

func TestSpeciate(t *testing.T) {
	speciation := Speciation{
		"x__VOC": {
			SpecType: "VOCUngrouped",
		},
		"VOC": {
			SpecType: "VOC",
		},
		"NOX": {
			SpecType: "NOx",
		},
		"PM2_5": {
			SpecType: "PM2.5",
		},
		"Toxin": {
			SpecProf: map[string]float64{
				"Toxin2": 0.5,
			},
		},
		"pentane": {
			SpecNames: struct {
				Names []string
				Group bool
			}{
				Names: []string{"N-pentane"},
				Group: false,
			},
		},
		"butane": {
			SpecNames: struct {
				Names []string
				Group bool
			}{
				Names: []string{"N-butane"},
				Group: true,
			},
		},
	}
	specRef := bytes.NewBuffer([]byte(specRefExample))
	specRefCombo := bytes.NewBuffer([]byte(specRefComboExample))
	speciesProperties := bytes.NewBuffer([]byte(speciesPropertiesExample))
	gasProfile := bytes.NewBuffer([]byte(gasProfileExample))
	gasSpecies := bytes.NewBuffer([]byte(gasSpeciesExample))
	otherGasesSpecies := bytes.NewBuffer([]byte(otherGasesSpeciesExample))
	pmSpecies := bytes.NewBuffer([]byte(pmSpeciesExample))
	mechAssignment := bytes.NewBuffer([]byte(mechAssignmentExample))
	mechMW := bytes.NewBuffer([]byte(mechMWExample))
	mechSpeciesInfo := bytes.NewBuffer([]byte(mechSpeciesInfoExample))

	s, err := NewSpeciator(specRef, specRefCombo, speciesProperties, gasProfile, gasSpecies, otherGasesSpecies, pmSpecies, mechAssignment, mechMW, mechSpeciesInfo, speciation)
	if err != nil {
		t.Fatal(err)
	}

	// newEmissions creates a new Emissions variable from the given inputs.
	newEmissions := func(v float64, pol Pollutant, p Period) *Emissions {
		inputConv := func(v float64) *unit.Unit { return unit.New(v, unit.Kilogram) }
		e := new(Emissions)
		var rate *unit.Unit
		rate, err = parseEmisRateAnnual(fmt.Sprintf("%g", v), "-9", inputConv)
		if err != nil {
			t.Fatal(err)
		}
		var begin, end time.Time
		begin, end, err = Annual.TimeInterval("2011")
		if err != nil {
			t.Fatal(err)
		}
		e.Add(begin, end, pol.Name, pol.Prefix, rate)
		return e
	}

	newEmissionsDouble := func(v1, v2 float64, pol1, pol2 Pollutant, p Period) *Emissions {
		e := newEmissions(v1, pol1, p)
		e2 := newEmissions(v2, pol2, p)
		e.combine(*e2)
		return e
	}

	const (
		VOCToTOG          = 1.04351461e+00
		nButaneFrac       = 2.45e+01 / (2.45e+01 + 1.277e+01)
		nPentaneFrac      = 1.277e+01 / (2.45e+01 + 1.277e+01)
		butaneMW          = 5.81222e+01
		pentaneMW         = 7.214878e+01
		butaneALK3factor  = 1
		pentaneALK4factor = 1
		butaneALK3mol     = butaneALK3factor * VOCToTOG * nButaneFrac / butaneMW
		pentaneALK4mol    = pentaneALK4factor * VOCToTOG * nPentaneFrac / pentaneMW
		ALK3MW            = 58.61
		ALK4MW            = 77.60
		butaneALK3mass    = butaneALK3factor * VOCToTOG * nButaneFrac
		pentaneALK4mass   = pentaneALK4factor * VOCToTOG * nPentaneFrac

		pmSum = 2.5e+01 + 3.8e+01 + 2.1e+00 + 8.5e+00 + 9.8e+00 + 2.8 + 16.0
	)

	tests := []struct {
		name          string
		rec           Record
		mechanism     string
		mass          bool
		partialMatch  bool
		emis, dropped map[Pollutant]*unit.Unit
	}{
		{
			name: "VOC ungrouped mol",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "2310011503",
				},
				Emissions: *newEmissions(1, Pollutant{Name: "VOC", Prefix: "x"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         false,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "N-butane"}:  unit.New(VOCToTOG*nButaneFrac/butaneMW, unit.Dimensions{kiloMol: 1}),
				Pollutant{Name: "N-pentane"}: unit.New(VOCToTOG*nPentaneFrac/pentaneMW, unit.Dimensions{kiloMol: 1}),
			},
		},
		{
			name: "VOC ungrouped mass",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "2310011503",
				},
				Emissions: *newEmissions(1, Pollutant{Name: "VOC", Prefix: "x"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         true,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "N-butane"}:  unit.New(VOCToTOG*nButaneFrac, unit.Dimensions{unit.MassDim: 1}),
				Pollutant{Name: "N-pentane"}: unit.New(VOCToTOG*nPentaneFrac, unit.Dimensions{unit.MassDim: 1}),
			},
		},
		{
			name: "VOC double count ungrouped mass",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "2310011503",
				},
				Emissions: *newEmissionsDouble(1, 1, Pollutant{Name: "VOC", Prefix: "x"}, Pollutant{Name: "pentane"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         true,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "N-butane"}:  unit.New(VOCToTOG*nButaneFrac, unit.Dimensions{unit.MassDim: 1}),
				Pollutant{Name: "N-pentane"}: unit.New(1, unit.Dimensions{unit.MassDim: 1}),
			},
			dropped: map[Pollutant]*unit.Unit{
				Pollutant{Name: "N-pentane"}: unit.New(VOCToTOG*nPentaneFrac, unit.Dimensions{unit.MassDim: 1}),
			},
		},
		{
			name: "VOC double count ungrouped mol",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "2310011503",
				},
				Emissions: *newEmissionsDouble(1, 1, Pollutant{Name: "VOC", Prefix: "x"}, Pollutant{Name: "pentane"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         false,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "N-butane"}:  unit.New(VOCToTOG*nButaneFrac/butaneMW, unit.Dimensions{kiloMol: 1}),
				Pollutant{Name: "N-pentane"}: unit.New(1/pentaneMW, unit.Dimensions{kiloMol: 1}),
			},
			dropped: map[Pollutant]*unit.Unit{
				Pollutant{Name: "N-pentane"}: unit.New(VOCToTOG*nPentaneFrac, unit.Dimensions{unit.MassDim: 1}),
			},
		},
		{
			name: "VOC grouped mol",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "2310011503",
				},
				Emissions: *newEmissions(1, Pollutant{Name: "VOC"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         false,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "ALK3"}: unit.New(butaneALK3mol, unit.Dimensions{kiloMol: 1}),
				Pollutant{Name: "ALK4"}: unit.New(pentaneALK4mol, unit.Dimensions{kiloMol: 1}),
			},
		},
		{
			name: "VOC grouped mass",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "2310011503",
				},
				Emissions: *newEmissions(1, Pollutant{Name: "VOC"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         true,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "ALK3"}: unit.New(butaneALK3mass, unit.Dimensions{unit.MassDim: 1}),
				Pollutant{Name: "ALK4"}: unit.New(pentaneALK4mass, unit.Dimensions{unit.MassDim: 1}),
			},
		},
		{
			name: "VOC double count grouped mol",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "2310011503",
				},
				Emissions: *newEmissionsDouble(1, 1, Pollutant{Name: "VOC"}, Pollutant{Name: "butane"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         false,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "ALK3"}: unit.New(1/butaneMW*butaneALK3factor, unit.Dimensions{kiloMol: 1}),
				Pollutant{Name: "ALK4"}: unit.New(pentaneALK4mol, unit.Dimensions{kiloMol: 1}),
			},
			dropped: map[Pollutant]*unit.Unit{
				Pollutant{Name: "N-butane"}: unit.New(VOCToTOG*nButaneFrac*butaneALK3factor, unit.Dimensions{unit.MassDim: 1}),
			},
		},
		{
			name: "NOx mass",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "2280003100",
				},
				Emissions: *newEmissions(1, Pollutant{Name: "NOX", Prefix: "EXH"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         true,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "Nitrogen Monoxide (Nitric Oxide)"}: unit.New(9e+01/(9e+01+15e+00), unit.Dimensions{unit.MassDim: 1}),
				Pollutant{Name: "Nitrogen Dioxide"}:                 unit.New(15e+00/(9e+01+15e+00), unit.Dimensions{unit.MassDim: 1}),
			},
		},
		{
			name: "NOx mol",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "2280003100",
				},
				Emissions: *newEmissions(1, Pollutant{Name: "NOX", Prefix: "EXH"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         false,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "Nitrogen Monoxide (Nitric Oxide)"}: unit.New(9e+01/(9e+01+15e+00)/30, unit.Dimensions{kiloMol: 1}),
				Pollutant{Name: "Nitrogen Dioxide"}:                 unit.New(15e+00/(9e+01+15e+00)/46, unit.Dimensions{kiloMol: 1}),
			},
		},
		{
			name: "PM2.5",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "0030532003",
				},
				Emissions: *newEmissions(1, Pollutant{Name: "PM2_5"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         false,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "Organic carbon"}:                        unit.New(2.5e+01/pmSum, unit.Dimensions{unit.MassDim: 1}),
				Pollutant{Name: "Elemental Carbon"}:                      unit.New(3.8e+01/pmSum, unit.Dimensions{unit.MassDim: 1}),
				Pollutant{Name: "Nitrate"}:                               unit.New(2.1e+00/pmSum, unit.Dimensions{unit.MassDim: 1}),
				Pollutant{Name: "Sulfate"}:                               unit.New(8.5e+00/pmSum, unit.Dimensions{unit.MassDim: 1}),
				Pollutant{Name: "Particulate Non-Carbon Organic Matter"}: unit.New(9.8e+00/pmSum, unit.Dimensions{unit.MassDim: 1}),
				Pollutant{Name: "Sulfur"}:                                unit.New(2.8e+00/pmSum, unit.Dimensions{unit.MassDim: 1}),
				Pollutant{Name: "Other Unspeciated PM2.5"}:               unit.New(1.6e+01/pmSum, unit.Dimensions{unit.MassDim: 1}),
			},
		},

		{
			name: "Direct",
			rec: &PolygonRecord{
				SourceData: SourceData{
					FIPS: "01001",
					SCC:  "0030532003",
				},
				Emissions: *newEmissions(1, Pollutant{Name: "Toxin"}, Annual),
			},
			mechanism:    "SAPRC99",
			mass:         false,
			partialMatch: false,
			emis: map[Pollutant]*unit.Unit{
				Pollutant{Name: "Toxin2"}: unit.New(0.5, unit.Dimensions{unit.MassDim: 1}),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			emis, dropped, err := s.Speciate(test.rec, test.mechanism, test.mass, test.partialMatch)
			if err != nil {
				t.Fatal(err)
			}
			if polMapDifferent(emis, test.emis) {
				t.Errorf("emis: have %v, want %v", emis.Totals(), test.emis)
			}
			if polMapDifferent(dropped, test.dropped) {
				t.Errorf("dropped: have %v, want %v", dropped.Totals(), test.dropped)
			}
		})
	}
}

// polMapDifferent returns true if ae and b are significantly different.
func polMapDifferent(ae *Emissions, b map[Pollutant]*unit.Unit) bool {
	if ae == nil && len(b) != 0 {
		return true
	}
	if ae == nil && len(b) == 0 {
		return false
	}

	a := ae.Totals()
	if len(a) != len(b) {
		return true
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return true
		}
		diff := 2 * math.Abs(va.Value()-vb.Value()) / (va.Value() + vb.Value())
		if diff > 1.e-8 || math.IsNaN(diff) {
			return true
		}
		if !va.Dimensions().Matches(vb.Dimensions()) {
			return true
		}
	}
	return false
}
