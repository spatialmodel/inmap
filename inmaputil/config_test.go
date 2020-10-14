/*
Copyright © 2020 the InMAP authors.
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

package inmaputil

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/ctessum/geom"
)

func TestParseMask(t *testing.T) {
	t.Run("polygon", func(t *testing.T) {
		f, err := os.Create("tmp_mask.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove("tmp_mask.json")
		fmt.Fprint(f, `{"type": "Polygon","coordinates": [ [ [1, 1], [1, 1], [1, 1], [1, 1] ] ] }`)
		mask, err := parseMask("tmp_mask.json")
		if err != nil {
			t.Fatal(err)
		}
		want := geom.Polygon{geom.Path{geom.Point{X: 1, Y: 1}, geom.Point{X: 1, Y: 1}, geom.Point{X: 1, Y: 1}, geom.Point{X: 1, Y: 1}}}
		if !reflect.DeepEqual(mask, want) {
			t.Errorf("%v != %v", mask, want)
		}
	})
	t.Run("multipolygon", func(t *testing.T) {
		f, err := os.Create("tmp_mask.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove("tmp_mask.json")
		fmt.Fprint(f, `{"type": "MultiPolygon","coordinates": [ [ [ [1, 1], [1, 1], [1, 1], [1, 1] ] ], [ [ [1, 1], [1, 1], [1, 1], [1, 1] ] ] ] }`)
		mask, err := parseMask("tmp_mask.json")
		if err != nil {
			t.Fatal(err)
		}
		want := geom.Polygon{
			geom.Path{geom.Point{X: 1, Y: 1}, geom.Point{X: 1, Y: 1}, geom.Point{X: 1, Y: 1}, geom.Point{X: 1, Y: 1}},
			geom.Path{geom.Point{X: 1, Y: 1}, geom.Point{X: 1, Y: 1}, geom.Point{X: 1, Y: 1}, geom.Point{X: 1, Y: 1}},
		}
		if !reflect.DeepEqual(mask, want) {
			t.Errorf("%v != %v", mask, want)
		}
	})

}
