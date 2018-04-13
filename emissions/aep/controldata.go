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

package aep

// ControlData holds information about how emissions from this source can be
// controlled.
type ControlData struct {
	// Maximum Available Control Technology Code
	// (6 characters maximum) (optional)
	MACT string

	// Control efficiency percentage (give value of 0-100) (recommended,
	// if left blank, default is 0).
	CEff float64

	// Rule Effectiveness percentage (give value of 0-100) (recommended,
	// if left blank, default is 100)
	REff float64

	// Rule Penetration percentage (give value of 0-100) (optional;
	// if missing will result in default of 100)
	RPen float64
}

func (r *ControlData) setCEff(s string) error {
	if s == "" {
		r.CEff = 0.
		return nil
	}
	var err error
	r.CEff, err = stringToFloat(s)
	return err
}

func (r *ControlData) setREff(s string) error {
	if s == "" {
		r.REff = 100.
		return nil
	}
	var err error
	r.REff, err = stringToFloat(s)
	return err
}

func (r *ControlData) setRPen(s string) error {
	if s == "" {
		r.RPen = 100.
		return nil
	}
	var err error
	r.RPen, err = stringToFloat(s)
	return err
}
