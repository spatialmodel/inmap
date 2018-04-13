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
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// SpecRef holds speciation reference information extracted
// from SMOKE model gsref files.
type SpecRef struct {
	sRef      map[string]map[string]interface{}            // map[SCC][pol]code
	sRefCombo map[Period]map[string]map[string]interface{} // map[Period][pol][FIPS][code]frac
}

// NewSpecRef returns a new SpecRef variable created from the
// information in the given readers.
func NewSpecRef(ref, refCombo io.Reader) (*SpecRef, error) {
	sp := new(SpecRef)
	var err error
	sp.sRef, err = specRef(ref)
	if err != nil {
		return nil, err
	}
	sp.sRefCombo, err = specRefCombo(refCombo)
	if err != nil {
		return nil, err
	}
	return sp, nil
}

// Codes returns the speciation profile code(s) that match the given SCC
// code, pollutant (pol), time period (start and end), country, and FIPS location code,
// along with the fraction of speciation that
// should be attributed to each code.
// If partialMatch is false, only codes exactly matching the given
// SCC will be returned, otherwise if no match is found an attempt will
// be made to return a code matching a more general SCC.
// If no direct match is found for pol, an attempt will be made to
// find a match for a version of pol without its prefix.
func (sp *SpecRef) Codes(SCC string, pol Pollutant, start, end time.Time, c Country, FIPS string, partialMatch bool) (map[string]float64, error) {
	codes, err := sp.code(SCC, pol, partialMatch)
	if err != nil {
		return nil, err
	}
	if codes["COMBO"] != 0 {
		return sp.comboCode(SCC, pol, start, end, c, FIPS, partialMatch)
	}
	return codes, nil
}

// code returns a speciation profile code for a normal speciation profile.
func (sp *SpecRef) code(SCC string, pol Pollutant, partialMatch bool) (map[string]float64, error) {
	if !partialMatch {
		matchedVal, ok := sp.sRef[SCC][pol.String()]
		if ok {
			return map[string]float64{matchedVal.(string): 1}, nil
		}
		matchedVal, ok = sp.sRef[SCC][pol.Name]
		if ok {
			return map[string]float64{matchedVal.(string): 1}, nil
		}
		str := fmt.Sprintf("aep: invalid exact match combination of SCC code '%s' and pollutant '%s'", SCC, pol.Name)
		if pol.Prefix != "" {
			str += fmt.Sprintf(" (or '%s')", pol.String())
		}
		return nil, errors.New(str)
	}
	// Look for a partial match.
	_, _, matchedVal, err := MatchCodeDouble(SCC, pol.String(), sp.sRef)
	if err != nil {
		_, _, matchedVal, err = MatchCodeDouble(SCC, pol.Name, sp.sRef)
		if err != nil {
			str := fmt.Sprintf("aep: invalid partial match combination of SCC code '%s' and pollutant '%s'", SCC, pol.String())
			if pol.Prefix != "" {
				str += fmt.Sprintf(" (or '%s')", pol.Name)
			}
			return nil, errors.New(str)
		}
	}
	return map[string]float64{matchedVal.(string): 1}, nil
}

// comboCode returns a speciation profile code for a normal speciation profile.
func (sp *SpecRef) comboCode(SCC string, pol Pollutant, start, end time.Time, c Country, FIPS string, partialMatch bool) (map[string]float64, error) {
	p, err := PeriodFromTimeInterval(start, end)
	if err != nil {
		return nil, err
	}
	countryFIPS := getCountryCode(c) + FIPS
	periodCodes, ok := sp.sRefCombo[p]
	if !ok {
		return nil, fmt.Errorf("aep: no speciation profiles for period %v", p)
	}
	if !partialMatch {
		codes, ok := periodCodes[pol.String()][countryFIPS]
		if ok {
			return codes.(map[string]float64), nil
		}
		codes, ok = periodCodes[pol.Name][countryFIPS]
		if ok {
			return codes.(map[string]float64), nil
		}
		str := fmt.Sprintf("aep: invalid combo exact match combination country+FIPS %s and pollutant '%s'", countryFIPS, pol.String())
		if pol.Prefix != "" {
			str += fmt.Sprintf(" (or '%s')", pol.Name)
		}
		return nil, errors.New(str)
	}
	_, _, codes, err := MatchCodeDouble(pol.String(), countryFIPS, periodCodes)
	if err != nil {
		_, _, codes, err = MatchCodeDouble(pol.Name, countryFIPS, periodCodes)
		if err != nil {
			str := fmt.Sprintf("aep: invalid combo partial match combination country+FIPS %s and pollutant '%s'", countryFIPS, pol.String())
			if pol.Prefix != "" {
				str += fmt.Sprintf(" (or '%s')", pol.Name)
			}
			return nil, errors.New(str)
		}
	}
	return codes.(map[string]float64), nil
}

// SpecRef reads the SMOKE gsref file, which maps SCC codes to chemical speciation profiles.
func specRef(fid io.Reader) (map[string]map[string]interface{}, error) {
	specRef := make(map[string]map[string]interface{})
	// map[SCC][pol]code
	buf := bufio.NewReader(fid)
	for {
		record, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		// Get rid of comments at end of line.
		if i := strings.Index(record, "!"); i != -1 {
			record = record[0:i]
		}

		if record[0] != '#' && record[0] != '/' && record[0] != '\n' {
			// for point sources, only match to SCC code.
			splitLine := strings.Split(record, ";")
			SCC := strings.Trim(splitLine[0], "\"")
			if len(SCC) == 8 {
				SCC = "00" + SCC
			}
			code := strings.Trim(splitLine[1], "\"")
			pol := strings.Trim(splitLine[2], "\"\n")

			if _, ok := specRef[SCC]; !ok {
				specRef[SCC] = make(map[string]interface{})
			}
			specRef[SCC][pol] = code
		}
	}
	return specRef, nil
}

// SpecRefCombo reads the SMOKE gspro_combo file, which maps location
// codes to chemical speciation profiles for mobile sources.
func specRefCombo(fid io.Reader) (map[Period]map[string]map[string]interface{}, error) {
	specRef := make(map[Period]map[string]map[string]interface{})
	// map[Period][pol][FIPS][code]frac

	buf := bufio.NewReader(fid)
	for {
		record, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, fmt.Errorf("aep: gspro_combo: reading line %v: %v", record, err)
			}
		}
		// Get rid of comments at end of line.
		if i := strings.Index(record, "!"); i != -1 {
			record = record[0:i]
		}

		if record[0] == '#' || record[0] == '/' || record[0] == '\n' {
			continue
		}
		// for point sources, only match to SCC code.
		splitLine := strings.Split(record, ";")
		pol := strings.Trim(splitLine[0], "\" ")
		// The FIPS number here is 6 characters instead of the usual 5.
		// The first character is a country code.
		FIPS := strings.Trim(splitLine[1], "\" ")
		if len(FIPS) == 5 {
			FIPS = "0" + FIPS
		}

		period, err := specRefPeriod(splitLine[2])
		if err != nil {
			return nil, fmt.Errorf("aep: gspro_combo: reading line %v: %v", splitLine, err)
		}
		if _, ok := specRef[period]; !ok {
			specRef[period] = make(map[string]map[string]interface{})
		}

		if _, ok := specRef[period][pol]; !ok {
			specRef[period][pol] = make(map[string]interface{})
		}
		if _, ok := specRef[period][pol][FIPS]; !ok {
			specRef[period][pol][FIPS] = make(map[string]float64)
		}
		total, err := strconv.ParseFloat(strings.Trim(splitLine[len(splitLine)-1], "\n\" "), 64)
		if err != nil {
			return nil, fmt.Errorf("aep: gspro_combo: reading line %v: %v", splitLine, err)
		}
		for i := 4; i < len(splitLine)-1; i += 2 {
			code := strings.Trim(splitLine[i], "\n\" ")
			frac, err := strconv.ParseFloat(strings.Trim(splitLine[i+1], "\n\" "), 64)
			if err != nil {
				return nil, fmt.Errorf("aep: gspro_combo: reading line %v: %v", splitLine, err)
			}
			specRef[period][pol][FIPS].(map[string]float64)[code] = frac / total
		}
	}
	return specRef, nil
}

// specRefPeriod convert a speciation reference period string
// into a Period variable.
func specRefPeriod(p string) (Period, error) {
	switch p {
	case "0":
		return Annual, nil
	case "1":
		return Jan, nil
	case "2":
		return Feb, nil
	case "3":
		return Mar, nil
	case "4":
		return Apr, nil
	case "5":
		return May, nil
	case "6":
		return Jun, nil
	case "7":
		return Jul, nil
	case "8":
		return Aug, nil
	case "9":
		return Sep, nil
	case "10":
		return Oct, nil
	case "11":
		return Nov, nil
	case "12":
		return Dec, nil
	default:
		return -1, fmt.Errorf("aep: invalid SpecRef period '%s'", p)
	}
}
