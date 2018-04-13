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
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ctessum/unit"
	"github.com/ctessum/unit/badunit"
)

func TestEmissions(t *testing.T) {
	e := new(Emissions)

	begin, _ := time.Parse("Jan 2006", "Jan 2005")
	end, _ := time.Parse("Jan 2006", "Jan 2006")
	rate := unit.New(1, map[unit.Dimension]int{unit.MassDim: 1, unit.TimeDim: -1})
	e.Add(begin, end, "testpol", "", rate)

	begin2, _ := time.Parse("Jan 2006", "Jun 2005")
	end2, _ := time.Parse("Jan 2006", "Jun 2006")
	rate2 := unit.New(1, map[unit.Dimension]int{unit.MassDim: 1, unit.TimeDim: -1})
	e.Add(begin2, end2, "testpol", "", rate2)
	e.Add(begin2, end2, "testpol2", "", rate2)
	e.Add(begin2, end2, "testpol3", "", rate2)

	wantTotals := map[Pollutant]*unit.Unit{
		Pollutant{Name: "testpol"}:  unit.New(6.3072e+07, unit.Kilogram),
		Pollutant{Name: "testpol2"}: unit.New(3.1536e+07, unit.Kilogram),
		Pollutant{Name: "testpol3"}: unit.New(3.1536e+07, unit.Kilogram),
	}
	haveTotals := e.Totals()
	if !reflect.DeepEqual(wantTotals, haveTotals) {
		t.Errorf("totals: want %v but have %v", wantTotals, haveTotals)
	}

	begin3, _ := time.Parse("Jan 2006", "Jun 2004")
	havePeriod1 := e.PeriodTotals(begin3, end)
	wantPeriod1 := map[Pollutant]*unit.Unit{
		Pollutant{Name: "testpol"}:  unit.New(5.00256e+07, unit.Kilogram),
		Pollutant{Name: "testpol2"}: unit.New(1.84896e+07, unit.Kilogram),
		Pollutant{Name: "testpol3"}: unit.New(1.84896e+07, unit.Kilogram),
	}
	if !reflect.DeepEqual(wantPeriod1, havePeriod1) {
		t.Errorf("period1: want %v but have %v", wantPeriod1, havePeriod1)
	}

	begin4, _ := time.Parse("Jan 2006", "Jun 2004")
	end3, _ := time.Parse("Jan 2006", "Jun 2007")
	havePeriod2 := e.PeriodTotals(begin4, end3) // should be the same as above
	if !reflect.DeepEqual(wantTotals, havePeriod2) {
		t.Errorf("period2: want %v but have %v", wantPeriod1, havePeriod2)
	}

	begin5, _ := time.Parse("Jan 2006", "Jun 2004")
	end5, _ := time.Parse("Jan 2006", "Jan 2005")
	havePeriod3 := e.PeriodTotals(begin5, end5)
	if len(havePeriod3) != 0 {
		t.Errorf("period3: want empty map but have %v", havePeriod3)
	}

	begin6, _ := time.Parse("Jan 2006", "Jun 2007")
	end6, _ := time.Parse("Jan 2006", "Jan 2008")
	havePeriod4 := e.PeriodTotals(begin6, end6)
	if len(havePeriod4) != 0 {
		t.Errorf("period4: want empty map but have %v", havePeriod4)
	}

	droppedTotals := e.DropPols(Speciation{"testpol": struct {
		SpecType  SpeciationType
		SpecNames struct {
			Names []string
			Group bool
		}
		SpecProf map[string]float64
	}{}})
	newTotals := e.Totals()
	wantDroppedTotals := map[Pollutant]*unit.Unit{
		Pollutant{Name: "testpol2"}: unit.New(3.1536e+07, unit.Kilogram),
		Pollutant{Name: "testpol3"}: unit.New(3.1536e+07, unit.Kilogram),
	}
	wantNewTotals := map[Pollutant]*unit.Unit{
		Pollutant{Name: "testpol"}: unit.New(6.3072e+07, unit.Kilogram),
	}
	wantUnits := map[Pollutant]unit.Dimensions{
		Pollutant{Name: "testpol"}: map[unit.Dimension]int{unit.MassDim: 1, unit.TimeDim: -1},
	}
	if !reflect.DeepEqual(wantDroppedTotals, droppedTotals) {
		t.Errorf("dropped totals: want %v but have %v", wantDroppedTotals, droppedTotals)
	}
	if !reflect.DeepEqual(wantNewTotals, newTotals) {
		t.Errorf("new totals: want %v but have %v", wantNewTotals, newTotals)
	}
	if !reflect.DeepEqual(wantUnits, e.units) {
		t.Errorf("new units: want %v but have %v", wantUnits, e.units)
	}

	v, err := parseEmisRateAnnual(nullVal, "1", badunit.Ton) // 1 ton/day
	if err != nil {
		t.Error(err)
	}
	want := unit.New(0.010499826388888888, unit.Dimensions{unit.MassDim: 1, unit.TimeDim: -1})
	if !reflect.DeepEqual(v, want) {
		t.Errorf("parseEmisRate: want %v but have %v", want, v)
	}

	v, err = parseEmisRateAnnual("1", "1", badunit.Ton) // 1 ton/year
	if err != nil {
		t.Error(err)
	}
	want = unit.New(2.8766647640791475e-05, unit.Dimensions{unit.MassDim: 1, unit.TimeDim: -1})
	if !reflect.DeepEqual(v, want) {
		t.Errorf("parseEmisRate: want %v but have %v", want, v)
	}

	e2 := new(Emissions)
	e2.Add(begin2, end2, "testpol8", "", rate2)
	e.CombineEmissions(&PointRecord{Emissions: *e2})
	wantUnits = map[Pollutant]unit.Dimensions{
		Pollutant{Name: "testpol", Prefix: ""}:  unit.Dimensions{4: 1, 6: -1},
		Pollutant{Name: "testpol8", Prefix: ""}: unit.Dimensions{4: 1, 6: -1},
	}
	if !reflect.DeepEqual(wantUnits, e.units) {
		t.Errorf("combined units: want %v but have %v", wantUnits, e.units)
	}
}

func TestDropTotals(t *testing.T) {
	e := new(Emissions)

	begin, _ := time.Parse("Jan 2006", "Jan 2005")
	end, _ := time.Parse("Jan 2006", "Jan 2006")
	rate := unit.New(1, map[unit.Dimension]int{unit.MassDim: 1, unit.TimeDim: -1})
	e.Add(begin, end, "testpol", "", rate)
	e.Add(begin, end, "testpol", "", rate)
	e.Add(begin, end, "testpol2", "", rate)

	polsToKeep := Speciation{"testpol2": struct {
		SpecType  SpeciationType
		SpecNames struct {
			Names []string
			Group bool
		}
		SpecProf map[string]float64
	}{}}

	droppedPols := e.DropPols(polsToKeep)

	droppedPolsWant := map[Pollutant]*unit.Unit{
		Pollutant{Name: "testpol", Prefix: ""}: unit.New(6.3072e+07, unit.Dimensions{4: 1}),
	}
	eWant := &Emissions{
		e: []*emissionsPeriod{
			&emissionsPeriod{
				begin:     begin,
				end:       end,
				rate:      rate.Value(),
				Pollutant: Pollutant{Name: "testpol2"},
			},
		},
		units: map[Pollutant]unit.Dimensions{Pollutant{Name: "testpol2", Prefix: ""}: unit.Dimensions{6: -1, 4: 1}},
	}

	if !reflect.DeepEqual(droppedPols, droppedPolsWant) {
		t.Errorf("droppedPols have %#v, want %#v", droppedPols, droppedPolsWant)
	}

	if !reflect.DeepEqual(e, eWant) {
		t.Errorf("e have %#v, want %#v", e, eWant)
	}

	const scale = 0.5
	eScaledWant := &Emissions{
		e: []*emissionsPeriod{
			&emissionsPeriod{
				begin:     begin,
				end:       end,
				rate:      rate.Value() * scale,
				Pollutant: Pollutant{Name: "testpol2"},
			},
		},
		units: map[Pollutant]unit.Dimensions{Pollutant{Name: "testpol2", Prefix: ""}: unit.Dimensions{6: -1, 4: 1}},
	}
	err := e.Scale(func(p Pollutant) (float64, error) {
		if p.Name == "testpol2" {
			return scale, nil
		}
		return 0, fmt.Errorf("invalid pollutant %v", p)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(e, eScaledWant) {
		t.Errorf("e scaled: have %#v, want %#v", e, eScaledWant)
	}

}

func TestEmissions_Clone(t *testing.T) {
	e := new(Emissions)
	begin, _ := time.Parse("Jan 2006", "Jan 2005")
	end, _ := time.Parse("Jan 2006", "Jan 2006")
	rate := unit.New(1, map[unit.Dimension]int{unit.MassDim: 1, unit.TimeDim: -1})
	e.Add(begin, end, "testpol", "", rate)

	e2 := e.Clone()
	scaleFunc := func(Pollutant) (float64, error) { return 0.5, nil }
	if err := e2.Scale(scaleFunc); err != nil {
		t.Fatal(err)
	}

	eWant := map[Pollutant]*unit.Unit{
		Pollutant{Name: "testpol", Prefix: ""}: unit.New(3.1536e+07, unit.Dimensions{4: 1}),
	}
	if !reflect.DeepEqual(e.Totals(), eWant) {
		t.Errorf("e = %v, want %v", e.Totals(), eWant)
	}
	e2Want := map[Pollutant]*unit.Unit{
		Pollutant{Name: "testpol", Prefix: ""}: unit.New(3.1536e+07/2, unit.Dimensions{4: 1}),
	}
	if !reflect.DeepEqual(e2.Totals(), e2Want) {
		t.Errorf("e2 = %v, want %v", e2.Totals(), e2Want)
	}
}
