/*
Copyright © 2016 the InMAP authors.
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
	"reflect"
	"testing"
)

func TestList(t *testing.T) {
	c0 := &Cell{Layer: 0}
	c1 := &Cell{Layer: 1}
	c2 := &Cell{Layer: 2}
	c3 := &Cell{Layer: 3}

	l := new(cellList)
	l2 := new(cellList)

	for _, c := range []*Cell{c0, c1, c2, c3} {
		l.add(c)
		l2.add(c)
	}

	l2.deleteCell(c0)
	l2.deleteCell(c1)
	l2.deleteCell(c2)
	l2.deleteCell(c3)
	if l2.first != nil {
		t.Error("l2 should be empty but it is not.")
	}

	want := []*Cell{c0, c1, c2, c3}
	if !reflect.DeepEqual(l.array(), want) {
		t.Errorf("have %#v, want %#v", l.array(), want)
	}

	l.deleteCell(c2)
	want = []*Cell{c0, c1, c3}
	if !reflect.DeepEqual(l.array(), want) {
		t.Errorf("have %#v, want %#v", l.array(), want)
	}

	cr0 := l.ref(c0)
	if c := l.forwardFrom(cr0, 0); c != cr0 {
		t.Errorf("have %#v, want %#v", c, cr0)
	}

	if c := l.forwardFrom(cr0, 1); c != nil {
		t.Errorf("have %#v, want nil", c)
	}

	if c := l.forwardFrom(cr0, 5); c != nil {
		t.Errorf("have %#v, want nil", c)
	}

	cr3 := l.ref(c3)
	cr1 := l.ref(c1)
	if c := l.forwardFrom(cr3, 1); c != cr1 {
		t.Errorf("have %#v, want %#v", c, c1)
	}

	if c := l.forwardFrom(cr3, 2); c != cr0 {
		t.Errorf("have %#v, want %#v", c, cr0)
	}

	if c := l.forwardFrom(cr3, 3); c != nil {
		t.Errorf("have %#v, want nil", c)
	}

	l3 := new(cellList)
	l3.add(c0)
	l3.add(c1)
	l3.delete(l3.first)
	if l3.len == 1 && l3.first == nil {
		t.Errorf("improperly formed list")
	}

}
