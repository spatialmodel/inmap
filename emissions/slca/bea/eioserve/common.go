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

package main

const (
	address = "localhost"
	port    = ":8080"
)

// All specifies that all sectors are to be considered
const All = "All"

// Selection specifies an analysis of interest.
type Selection struct {
	DemandGroup      string
	DemandSector     string
	ProductionGroup  string
	ProductionSector string
	ImpactType       string
	DemandType       string
}

// Selectors represents a set of Selectors.
type Selectors struct {
	Names  []string
	Values []float64
}

// Len fulfils sort.Sort.
func (s *Selectors) Len() int { return len(s.Names) }

// Less fulfils sort.Sort.
func (s *Selectors) Less(i, j int) bool {
	if s.Names[i] == All {
		return true
	}
	if s.Names[j] == All {
		return false
	}
	return s.Values[i] > s.Values[j]
}

// Swap fulfills sort.Sort.
func (s *Selectors) Swap(i, j int) {
	s.Names[i], s.Names[j] = s.Names[j], s.Names[i]
	s.Values[i], s.Values[j] = s.Values[j], s.Values[i]
}

// Empty is an empty struct.
type Empty struct{}

// MapInfo holds the grid cell colors and an SVG legend for a map.
type MapInfo struct {
	Color  []RGB
	Legend string
}

// RGB holds RGB color information.
type RGB struct {
	R, G, B float64
}
