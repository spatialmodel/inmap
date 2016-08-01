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

import "fmt"

// cellRefs holds a cell, a reference to the next and previous
// adjacent cells in a cellList, and potentially information about the
// cell's relationship to a neighbor.
type cellRef struct {
	*Cell
	next, previous *cellRef
	info           *neighborInfo
}

// cellList is a linked list of cells.
type cellList struct {
	first *cellRef
	len   int
	index map[*Cell]*cellRef
}

// array returns a sorted array of the cells in this list.
func (l *cellList) array() []*Cell {
	o := make([]*Cell, l.len)
	c := l.first
	for i := 0; i < l.len; i++ {
		o[i] = c.Cell
		c = c.next
	}
	sortCells(o)
	return o
}

// delete deletes this cellRef from the list.
func (l *cellList) delete(c *cellRef) {
	if c.previous != nil && c.next != nil {
		c.previous.next, c.next.previous = c.next, c.previous
	} else if c.previous != nil {
		c.previous.next = nil
	} else if c.next != nil {
		c.next.previous = nil
	}
	if c == l.first {
		l.first = c.next
	}
	c.previous = nil
	c.next = nil
	l.len--
	delete(l.index, c.Cell)
}

// deleteCell deletes this Cell from the list
func (l *cellList) deleteCell(c *Cell) {
	cc, ok := l.index[c]
	if !ok {
		panic("tried to delete cell that is not in list")
	}
	l.delete(cc)
}

// addToList adds the cell to the beginning of the list.
func (l *cellList) add(c *Cell) *cellRef {
	cc := &cellRef{Cell: c}
	cc.next = l.first
	if l.first != nil {
		l.first.previous = cc
	}
	l.first = cc
	l.len++

	if l.index == nil {
		l.index = make(map[*Cell]*cellRef)
	}
	l.index[c] = cc
	return cc
}

// forwardInList returns the cell n spaces forward from c in the cell list.
func (l *cellList) forwardFrom(c *cellRef, n int) *cellRef {
	c2 := c
	for i := 0; i < n; i++ {
		if c2.next == nil {
			return nil
		}
		c2 = c2.next
	}
	return c2
}

func (l *cellList) ref(c *Cell) *cellRef {
	cc, ok := l.index[c]
	if !ok {
		panic("tried to retrieve cell that is not in list")
	}
	return cc
}

func (l *cellList) String() string {
	s := ""
	for c := l.first; c != nil; c = c.next {
		if c != l.first {
			s += "\n"
		}
		s += fmt.Sprint(c.Cell)
	}
	return s
}
