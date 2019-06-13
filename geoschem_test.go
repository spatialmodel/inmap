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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmap

import (
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/ctessum/cdf"
	"github.com/ctessum/geom"
)

func TestReadOlsonLandMap(t *testing.T) {
	f, err := os.Open("cmd/inmap/testdata/preproc/geoschem-new/Olson_2001_Land_Map.025x025.generic.nc")
	if err != nil {
		t.Fatal(err)
	}
	cf, err := cdf.Open(f)
	if err != nil {
		t.Fatal(err)
	}

	o, err := readOlsonLandMap(cf)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("stats", func(t *testing.T) {
		x0, y0 := -180.0, -90.0
		x1, y1 := 180.0, 90.0
		r := geom.Polygon{{
			{X: x0, Y: y0},
			{X: x1, Y: y0},
			{X: x1, Y: y1},
			{X: x0, Y: y1},
		}}

		min := math.MaxInt32
		max := math.MinInt32
		var sum int
		for _, cI := range o.data.SearchIntersect(r.Bounds()) {
			c := cI.(olsonGridCell)
			v := c.category
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
			sum += v
		}
		if max != 72 {
			t.Errorf("max: %d != 72", max)
		}
		if min != 0 {
			t.Errorf("min: %d != 0", min)
		}
		if sum != 9108986 {
			t.Errorf("sum: %d != 0", sum)
		}
	})

	t.Run("fractions", func(t *testing.T) {
		t.Run("amazon", func(t *testing.T) {
			// This rectangle is in the amazon,
			// so it should be almost all "Southern hemisphere mixed forest."
			x0, y0 := -70.0, -10.0
			x1, y1 := -60.0, 0.0
			r := geom.Polygon{{
				{X: x0, Y: y0},
				{X: x1, Y: y0},
				{X: x1, Y: y1},
				{X: x0, Y: y1},
			}}
			f := o.fractions(r)
			want := map[int]float64{
				0: 0.006875, 13: 0.001875, 33: 0.99125, 43: 0,
			}
			if !reflect.DeepEqual(f, want) {
				t.Errorf("want %v but have %v", want, f)
			}
		})
		t.Run("antarctica", func(t *testing.T) {
			// This rectangle is split between the ocean and antarctica,
			// so it should be a combination of water and "semi-desert".
			x0, y0 := 150.0, -90.0
			x1, y1 := 180.0, -60.0
			r := geom.Polygon{{
				{X: x0, Y: y0},
				{X: x1, Y: y0},
				{X: x1, Y: y1},
				{X: x0, Y: y1},
			}}
			f := o.fractions(r)
			want := map[int]float64{
				0: 0.44027777777777777, 12: 0.5597222222222222,
			}
			if !reflect.DeepEqual(f, want) {
				t.Errorf("want %v but have %v", want, f)
			}
		})

	})
}
