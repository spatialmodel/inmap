/*
Copyright © 2019 the InMAP authors.
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
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/ctessum/geom"
	"github.com/ctessum/unit"
)

func TestReadCOARDSFile(t *testing.T) {
	file := "testdata/emis_coards_hawaii.nc"
	begin := time.Date(2016, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2017, time.January, 1, 0, 0, 0, 0, time.UTC)
	sourceData := SourceData{}
	generator, err := ReadCOARDSFile(file, begin, end, Ton, sourceData)
	if err != nil {
		t.Fatal(err)
	}

	totalEmis := new(Emissions)

	wantGeometry := []*geom.Bounds{
		&geom.Bounds{Min: geom.Point{X: -161.25, Y: 15}, Max: geom.Point{X: -158.75, Y: 17}},
		&geom.Bounds{Min: geom.Point{X: -158.75, Y: 15}, Max: geom.Point{X: -156.25, Y: 17}},
		&geom.Bounds{Min: geom.Point{X: -156.25, Y: 15}, Max: geom.Point{X: -153.75, Y: 17}},
		&geom.Bounds{Min: geom.Point{X: -161.25, Y: 17}, Max: geom.Point{X: -158.75, Y: 19}},
		&geom.Bounds{Min: geom.Point{X: -158.75, Y: 17}, Max: geom.Point{X: -156.25, Y: 19}},
		&geom.Bounds{Min: geom.Point{X: -156.25, Y: 17}, Max: geom.Point{X: -153.75, Y: 19}},
		&geom.Bounds{Min: geom.Point{X: -161.25, Y: 19}, Max: geom.Point{X: -158.75, Y: 21}},
		&geom.Bounds{Min: geom.Point{X: -158.75, Y: 19}, Max: geom.Point{X: -156.25, Y: 21}},
		&geom.Bounds{Min: geom.Point{X: -156.25, Y: 19}, Max: geom.Point{X: -153.75, Y: 21}},
		&geom.Bounds{Min: geom.Point{X: -161.25, Y: 21}, Max: geom.Point{X: -158.75, Y: 23}},
		&geom.Bounds{Min: geom.Point{X: -158.75, Y: 21}, Max: geom.Point{X: -156.25, Y: 23}},
		&geom.Bounds{Min: geom.Point{X: -156.25, Y: 21}, Max: geom.Point{X: -153.75, Y: 23}},
		&geom.Bounds{Min: geom.Point{X: -161.25, Y: 23}, Max: geom.Point{X: -158.75, Y: 25}},
		&geom.Bounds{Min: geom.Point{X: -158.75, Y: 23}, Max: geom.Point{X: -156.25, Y: 25}},
		&geom.Bounds{Min: geom.Point{X: -156.25, Y: 23}, Max: geom.Point{X: -153.75, Y: 25}},
	}

	var i int
	for {
		rec, err := generator()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Error(err)
		}

		g := rec.(*basicPolygonRecord).Polygonal
		if !reflect.DeepEqual(g, wantGeometry[i]) {
			t.Errorf("%v != %v", g, wantGeometry[i])
		}

		totalEmis.CombineEmissions(rec)
		i++
	}
	emisWant := map[Pollutant]*unit.Unit{
		Pollutant{Name: "NH3"}:   unit.New(4.1533555064591676e+07, unit.Kilogram),
		Pollutant{Name: "NOx"}:   unit.New(4.0896043774575606e+07, unit.Kilogram),
		Pollutant{Name: "PM2_5"}: unit.New(1.3217351922194459e+08, unit.Kilogram),
		Pollutant{Name: "SOx"}:   unit.New(4.145479962381774e+07, unit.Kilogram),
		Pollutant{Name: "VOC"}:   unit.New(3.340366574798584e+07, unit.Kilogram),
	}
	emisHave := totalEmis.Totals()
	if !reflect.DeepEqual(emisWant, emisHave) {
		t.Errorf("%v != %v", emisHave, emisWant)
	}
}
