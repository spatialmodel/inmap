/*
Copyright © 2018 the InMAP authors.
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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package aep

import (
	"math"
	"strings"
	"testing"
	"time"

	"bitbucket.org/ctessum/sparse"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
)

type arrayAdjuster sparse.DenseArray

func (a *arrayAdjuster) Adjustment() (*sparse.DenseArray, error) {
	return (*sparse.DenseArray)(a), nil
}

func TestSpatialAdjustRecord(t *testing.T) {
	inputSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		t.Error(err)
	}
	r := strings.NewReader(srgSpecFileString)
	srgSpecs, err := ReadSrgSpec(r, "testdata", true)
	if err != nil {
		t.Error(err)
	}
	r = strings.NewReader(gridRefFileString)
	gridRef, err := ReadGridRef(r)
	if err != nil {
		t.Fatal(err)
	}
	grid, err := createGrid()
	if err != nil {
		t.Fatal(err)
	}
	sp := NewSpatialProcessor(srgSpecs, []*GridDef{grid}, gridRef, inputSR, true)

	sourceData := SourceData{
		FIPS:    "34017",
		SCC:     "0010200501",
		Country: USA,
	}
	pointData := PointSourceData{
		Point: geom.Point{X: -73.9712, Y: 40.7831}, // Downtown Manhattan.
		SR:    longlat,
	}

	emis := new(Emissions)
	begin, _ := time.Parse("Jan 2006", "Jan 2005")
	end, _ := time.Parse("Jan 2006", "Jan 2006")
	rate, err := parseEmisRateAnnual("1", "-9", func(v float64) *unit.Unit { return unit.New(v, unit.Kilogram) })
	if err != nil {
		t.Fatal(err)
	}
	emis.Add(begin, end, "testpol", "", rate)

	a := sparse.ZerosDense(4, 4)
	for i := range a.Elements {
		a.Elements[i] = 0.5
	}
	adj := arrayAdjuster(*a)

	for i, rec := range []Record{
		&PolygonRecord{
			SourceData: sourceData,
			Emissions:  *emis,
		},
		&PointRecord{
			SourceData:      sourceData,
			PointSourceData: pointData,
			Emissions:       *emis,
		},
	} {
		if i == 0 && testing.Short() {
			continue // Skip surrogate creation for polygon record.
		}
		recAdj := &SpatialAdjustRecord{Record: rec, SpatialAdjuster: &adj}

		emis, _, err := GriddedEmissions(rec, begin, end, sp, 0)
		if err != nil {
			t.Fatalf("i: %d, err: %v", i, err)
			continue
		}
		emisAdj, _, err := GriddedEmissions(recAdj, begin, end, sp, 0)
		if err != nil {
			t.Fatalf("i: %d, err: %v", i, err)
			continue
		}
		emis2, _, err := GriddedEmissions(rec, begin, end, sp, 0)
		if err != nil {
			t.Fatalf("i: %d, err: %v", i, err)
			continue
		}

		if i == 0 { // area record
			sum := emis[Pollutant{Name: "testpol"}].Sum()
			if math.Abs(sum-1) > 0.000001 {
				t.Errorf("%d area gridded emissions should sum to 1 but "+
					"instead sums to %f", i, sum)
			}
			sumAdj := emisAdj[Pollutant{Name: "testpol"}].Sum()
			if math.Abs(sumAdj-0.5) > 0.000001 {
				t.Errorf("%d area adjusted gridded emissions should sum to 0.5 but "+
					"instead sums to %f", i, sumAdj)
			}
			sum2 := emis2[Pollutant{Name: "testpol"}].Sum()
			if math.Abs(sum2-1) > 0.000001 {
				t.Errorf("%d area gridded emissions 2 should sum to 1 but "+
					"instead sums to %f", i, sum2)
			}
		} else if i == 1 { // point record
			sum := emis[Pollutant{Name: "testpol"}].Sum()
			if math.Abs(sum-1) > 0.000001 {
				t.Errorf("%d point gridded emissions should sum to 1 but "+
					"instead sums to %f", i, sum)
			}
			sumAdj := emisAdj[Pollutant{Name: "testpol"}].Sum()
			if math.Abs(sumAdj-0.5) > 0.000001 {
				t.Errorf("%d point adjusted gridded emissions should sum to 0.5 but "+
					"instead sums to %f", i, sumAdj)
			}
			sum2 := emis2[Pollutant{Name: "testpol"}].Sum()
			if math.Abs(sum2-1) > 0.000001 {
				t.Errorf("%d point gridded emissions 2 should sum to 1 but "+
					"instead sums to %f", i, sum2)
			}
		}
	}
}
