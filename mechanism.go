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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmap

// Mechanism is an interface for atmospheric chemical mechanisms.
type Mechanism interface {
	// AddEmisFlux adds emissions flux to Cell c based on the given
	// pollutant name and amount in units of μg/s. The units of
	// the resulting flux may vary for different chemical mechanisms.
	AddEmisFlux(c *Cell, name string, val float64) error

	// DryDep returns a dry deposition function of the type indicated by
	// name that is compatible with this chemical mechanism.
	// Valid options may vary for different chemical mechanisms.
	DryDep(name string) (CellManipulator, error)

	// WetDep returns a dry deposition function of the type indicated by
	// name that is compatible with this chemical mechanism.
	// Valid options may vary for different chemical mechanisms.
	WetDep(name string) (CellManipulator, error)

	// Species returns the names of the emission and concentration pollutant
	// species that are used by this chemical mechanism.
	Species() []string

	// Value returns the concentration or emissions value of
	// the given variable in the given Cell. It returns an
	// error if given an invalid variable name.
	Value(c *Cell, variable string) (float64, error)

	// Units returns the units of the given variable, or an
	// error if the variable name is invalid.
	Units(variable string) (string, error)

	// Chemistry returns a function that simulates chemical reactions.
	Chemistry() CellManipulator

	// Len returns the number of pollutants in the chemical mechanism.
	Len() int
}
