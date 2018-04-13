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

// SourceData holds information about the type of an emissions source.
type SourceData struct {
	// Five digit FIPS code for state and county (required)
	FIPS string

	// Ten character Source Classification Code (required)
	SCC string

	// Source type (2 characters maximum), used by SMOKE in determining
	// applicable MACT-based controls (required)
	// 	01 = major
	// 	02 = Section 12 area source
	// 	03 = nonroad
	// 	04 = onroad
	SourceType string

	// The country that this record applies to.
	Country Country
}

// GetSCC returns the SCC associated with this record.
func (r SourceData) GetSCC() string {
	return r.SCC
}

// GetFIPS returns the FIPS associated with this record.
func (r SourceData) GetFIPS() string {
	return r.FIPS
}

// GetCountry returns the Country associated with this record.
func (r SourceData) GetCountry() Country {
	return r.Country
}

// GetSourceData returns r.
func (r SourceData) GetSourceData() *SourceData {
	return &r
}

// Get rid of extra quotation marks, replace spaces with
// zeros, make sure it is five digits long.
func (r *SourceData) parseFIPS(FIPS string) {
	r.FIPS = strings.Replace(trimString(FIPS), " ", "0", -1)
	if len(r.FIPS) < 5 {
		if len(r.FIPS) == 4 {
			r.FIPS = "0" + r.FIPS
		} else if len(r.FIPS) == 3 {
			r.FIPS = "00" + r.FIPS
		} else if len(r.FIPS) == 2 {
			r.FIPS = "000" + r.FIPS
		} else if len(r.FIPS) == 1 {
			r.FIPS = "0000" + r.FIPS
		}
	}
}

// Add zeros to 8 digit SCCs so that all SCCs are 10 digits
// If SCC is less than 8 digits, add 2 zeros to the front and
// the rest to the end.
func (r *SourceData) parseSCC(SCC string) {
	r.SCC = trimString(SCC)
	if len(r.SCC) == 8 {
		r.SCC = "00" + r.SCC
	} else if len(r.SCC) == 7 {
		r.SCC = "00" + r.SCC + "0"
	} else if len(r.SCC) == 6 {
		r.SCC = "00" + r.SCC + "00"
	} else if len(r.SCC) == 5 {
		r.SCC = "00" + r.SCC + "000"
	} else if len(r.SCC) == 4 {
		r.SCC = "00" + r.SCC + "0000"
	} else if len(r.SCC) == 3 {
		r.SCC = "00" + r.SCC + "00000"
	} else if len(r.SCC) == 2 {
		r.SCC = "00" + r.SCC + "000000"
	}
}

// Key returns a unique key for this record.
func (r *SourceData) Key() string {
	return fmt.Sprintf("%s%s%d", r.FIPS, r.SCC, r.Country)
}
