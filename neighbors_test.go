package inmap

import (
	"reflect"
	"testing"

	"github.com/ctessum/geom"
)

func TestNewNeighborRect(t *testing.T) {

	smallb := &geom.Bounds{
		Min: geom.Point{X: 0, Y: 0},
		Max: geom.Point{X: 1.0e-10, Y: 1.0e-10},
	}

	w := newNeighborRect(smallb, west)
	e := newNeighborRect(smallb, east)
	n := newNeighborRect(smallb, north)
	s := newNeighborRect(smallb, south)
	a := newNeighborRect(smallb, aboveBelow)

	want := []*geom.Bounds{
		{Min: geom.Point{X: -1.0000000000000001e-20, Y: 5.0000000000000005e-21}, Max: geom.Point{X: -5.0000000000000005e-21, Y: 9.999999999500001e-11}},
		{Min: geom.Point{X: 1.00000000005e-10, Y: 5.0000000000000005e-21}, Max: geom.Point{X: 1.0000000001000001e-10, Y: 9.999999999500001e-11}},
		{Min: geom.Point{X: 5.0000000000000005e-21, Y: 1.00000000005e-10}, Max: geom.Point{X: 9.999999999500001e-11, Y: 1.0000000001000001e-10}},
		{Min: geom.Point{X: 5.0000000000000005e-21, Y: -1.0000000000000001e-20}, Max: geom.Point{X: 9.999999999500001e-11, Y: -5.0000000000000005e-21}},
		{Min: geom.Point{X: 5.0000000000000005e-21, Y: 5.0000000000000005e-21}, Max: geom.Point{X: 9.999999999500001e-11, Y: 9.999999999500001e-11}},
	}

	for i, have := range []*geom.Bounds{w, e, n, s, a} {
		if !reflect.DeepEqual(want[i], have) {
			t.Errorf("smallb: want %v but have %v", want[i], have)
		}
	}

	largeb := &geom.Bounds{
		Min: geom.Point{X: 1e20, Y: 1e20},
		Max: geom.Point{X: 2e20, Y: 2e20},
	}

	w = newNeighborRect(largeb, west)
	e = newNeighborRect(largeb, east)
	n = newNeighborRect(largeb, north)
	s = newNeighborRect(largeb, south)
	a = newNeighborRect(largeb, aboveBelow)

	want = []*geom.Bounds{
		{Min: geom.Point{X: 9.999999997e+19, Y: 1.00000000015e+20}, Max: geom.Point{X: 9.9999999985e+19, Y: 1.99999999985e+20}},
		{Min: geom.Point{X: 2.00000000015e+20, Y: 1.00000000015e+20}, Max: geom.Point{X: 2.0000000003e+20, Y: 1.99999999985e+20}},
		{Min: geom.Point{X: 1.00000000015e+20, Y: 2.00000000015e+20}, Max: geom.Point{X: 1.99999999985e+20, Y: 2.0000000003e+20}},
		{Min: geom.Point{X: 1.00000000015e+20, Y: 9.999999997e+19}, Max: geom.Point{X: 1.99999999985e+20, Y: 9.9999999985e+19}},
		{Min: geom.Point{X: 1.00000000015e+20, Y: 1.00000000015e+20}, Max: geom.Point{X: 1.99999999985e+20, Y: 1.99999999985e+20}},
	}

	for i, have := range []*geom.Bounds{w, e, n, s, a} {
		if !reflect.DeepEqual(want[i], have) {
			t.Errorf("largeb: want %v but have %v", want[i], have)
		}
	}

	negativeb := &geom.Bounds{
		Min: geom.Point{X: -1, Y: -1},
		Max: geom.Point{X: 0, Y: 0},
	}

	w = newNeighborRect(negativeb, west)
	e = newNeighborRect(negativeb, east)
	n = newNeighborRect(negativeb, north)
	s = newNeighborRect(negativeb, south)
	a = newNeighborRect(negativeb, aboveBelow)

	want = []*geom.Bounds{
		{Min: geom.Point{X: -1.0000000001, Y: -0.99999999995}, Max: geom.Point{X: -1.00000000005, Y: -5e-11}},
		{Min: geom.Point{X: 5e-11, Y: -0.99999999995}, Max: geom.Point{X: 1e-10, Y: -5e-11}},
		{Min: geom.Point{X: -0.99999999995, Y: 5e-11}, Max: geom.Point{X: -5e-11, Y: 1e-10}},
		{Min: geom.Point{X: -0.99999999995, Y: -1.0000000001}, Max: geom.Point{X: -5e-11, Y: -1.00000000005}},
		{Min: geom.Point{X: -0.99999999995, Y: -0.99999999995}, Max: geom.Point{X: -5e-11, Y: -5e-11}},
	}

	for i, have := range []*geom.Bounds{w, e, n, s, a} {
		if !reflect.DeepEqual(want[i], have) {
			t.Errorf("negativeb: want %v but have %v", want[i], have)
		}
	}

	centeredb := &geom.Bounds{
		Min: geom.Point{X: -1, Y: -1},
		Max: geom.Point{X: 1, Y: 1},
	}

	w = newNeighborRect(centeredb, west)
	e = newNeighborRect(centeredb, east)
	n = newNeighborRect(centeredb, north)
	s = newNeighborRect(centeredb, south)
	a = newNeighborRect(centeredb, aboveBelow)

	want = []*geom.Bounds{
		{Min: geom.Point{X: -1.0000000002, Y: -0.9999999999}, Max: geom.Point{X: -1.0000000001, Y: 0.9999999999}},
		{Min: geom.Point{X: 1.0000000001, Y: -0.9999999999}, Max: geom.Point{X: 1.0000000002, Y: 0.9999999999}},
		{Min: geom.Point{X: -0.9999999999, Y: 1.0000000001}, Max: geom.Point{X: 0.9999999999, Y: 1.0000000002}},
		{Min: geom.Point{X: -0.9999999999, Y: -1.0000000002}, Max: geom.Point{X: 0.9999999999, Y: -1.0000000001}},
		{Min: geom.Point{X: -0.9999999999, Y: -0.9999999999}, Max: geom.Point{X: 0.9999999999, Y: 0.9999999999}},
	}

	for i, have := range []*geom.Bounds{w, e, n, s, a} {
		if !reflect.DeepEqual(want[i], have) {
			t.Errorf("centeredb: want %v but have %v", want[i], have)
		}
	}
}
