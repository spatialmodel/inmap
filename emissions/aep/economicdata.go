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

import (
	"fmt"
	"strings"
)

// EconomicData holds industry information about an emissions source
type EconomicData struct {
	// SIC is the Standard Industrial Classification Code (recommended)
	SIC string

	// North American Industrial Classification System Code
	// (6 characters maximum) (optional)
	NAICS string
}

// clean up NAICS code so it either has 0 or 6 characters
func (r *EconomicData) parseNAICS(NAICS string) {
	NAICS = trimString(NAICS)
	if NAICS == "" || NAICS == nullVal {
		r.NAICS = ""
	} else {
		r.NAICS = strings.Replace(fmt.Sprintf("%-6s", NAICS), " ", "0", -1)
	}
}

// clean up SIC code so it either has 0 or 4 characters
func (r *EconomicData) parseSIC(SIC string) {
	r.SIC = trimString(SIC)
	if r.SIC == "" || r.SIC == nullVal {
		r.SIC = ""
	} else {
		r.SIC = strings.Replace(fmt.Sprintf("%-4s", r.SIC), " ", "0", -1)
	}
}

// GetEconomicData returns r.
func (r *EconomicData) GetEconomicData() *EconomicData {
	return r
}

func (r *nobusinessPolygonRecord) GetEconomicData() *EconomicData {
	return nil
}
