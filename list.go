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
	"fmt"
	"sort"
)

// cellRefs holds a cell, a reference to the next and previous
// adjacent cells in a cellList, and potentially information about the
// cell's relationship to a neighbor.
type cellRef struct {
	*Cell
	info *neighborInfo
}

// cellList is a list of cells.
type cellList []*cellRef

func (l *cellList) len() int {
	return len(*l)
}

// array returns a sorted array of the cells in this list.
func (l *cellList) array() []*Cell {
	o := make([]*Cell, len(*l))
	for i, c := range *l {
		o[i] = c.Cell
	}
	return o
}

// delete deletes this cellRef from the list.
func (l *cellList) delete(c *cellRef) {
	l.deleteCell(c.Cell)
}

// deleteCell deletes this Cell from the list
func (l *cellList) deleteCell(c *Cell) {

	// Find the index where the cell should be.
	i := sort.Search(len(*l), func(i int) bool {
		return c.before((*l)[i].Cell)
	})

	cref := (*l)[i]
	if cref.Cell != c {
		panic("tried to delete cell that is not in list")
	}

	copy((*l)[i:], (*l)[i+1:])
	(*l)[len(*l)-1] = nil
	(*l) = (*l)[:len(*l)-1]
}

// addToList adds the cell to the beginning of the list.
func (l *cellList) add(c *Cell) *cellRef {
	cc := &cellRef{Cell: c}

	// Find the correct location to insert the cell
	i := sort.Search(len(*l), func(i int) bool {
		return c.before((*l)[i].Cell)
	})

	// Insert the cell.
	(*l) = append((*l), nil)
	copy((*l)[i+1:], (*l)[i:])
	(*l)[i] = cc

	return cc
}

// index returns the index of c and whether it was found
func (l *cellList) index(c *Cell) (int, bool) {
	// Find the index where the cell should be.
	i := sort.Search(len(*l), func(i int) bool {
		return c.before((*l)[i].Cell)
	})

	cref := (*l)[i]
	if cref.Cell != c {
		return -1, false
	}
	return i, true
}

func (l *cellList) ref(c *Cell) *cellRef {

	// Find the index where the cell should be.
	i := sort.Search(len(*l), func(i int) bool {
		return c.before((*l)[i].Cell)
	})

	cref := (*l)[i]
	if cref.Cell != c {
		panic("tried to retrieve cell that is not in list")
	}
	return cref
}

func (l *cellList) String() string {
	s := ""
	for i, c := range *l {
		if i != 0 {
			s += "\n"
		}
		s += fmt.Sprint(c.Cell)
	}
	return s
}

// before returns whether c should be sorted before c2.
func (c *Cell) before(c2 *Cell) bool {
	if c == c2 {
		return true
	}
	if c.Layer != c2.Layer {
		return c.Layer < c2.Layer
	}

	icent := c.Centroid()
	jcent := c2.Centroid()

	if icent.X != jcent.X {
		return icent.X < jcent.X
	}
	if icent.Y != jcent.Y {
		return icent.Y < jcent.Y
	}
	// We apparently have concentric cells if we get to here.
	panic(fmt.Errorf("problem sorting: i: %v, j: %v", c, c2))
}
