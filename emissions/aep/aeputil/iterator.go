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

package aeputil

import (
	"io"

	"github.com/spatialmodel/inmap/emissions/aep"
)

// Iterator is an iterface for types that can iterate through a list of
// emissions records and return totals at the end.
type Iterator interface {
	// Next returns the next record.
	Next() (aep.Record, error)

	// Report returns an emissions report on the records that have been
	// processed by this iterator.
	Report() *aep.InventoryReport
}

// IteratorFromMap creates an Iterator from a map of emissions.
// This function is meant to be temporary until Inventory.ReadEmissions
// is replaced with an iterator.
func IteratorFromMap(emissions map[string][]aep.Record) Iterator {
	var l int
	for _, rs := range emissions {
		l += len(rs)
	}
	iter := &mapIterator{emis: make([]aep.Record, l)}
	var i int
	for _, rs := range emissions {
		for _, r := range rs {
			iter.emis[i] = r
			i++
		}
	}
	return iter
}

type mapIterator struct {
	emis []aep.Record
	i    int
}

func (i *mapIterator) Next() (aep.Record, error) {
	if i.i == len(i.emis) {
		return nil, io.EOF
	}
	r := i.emis[i.i]
	i.i++
	return r, nil
}

func (i *mapIterator) Report() *aep.InventoryReport { return nil }
