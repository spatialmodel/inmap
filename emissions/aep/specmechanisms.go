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
	"encoding/csv"
	"fmt"
	"io"
	"math"
)

// Mechanisms holds information on chemical speciation mechanisms
// as specified at http://www.cert.ucr.edu/~carter/emitdb/.
type Mechanisms struct {
	// MechAssigment holds chemical mechanism assignment information
	// for individiaul chemical species.
	// Data should be in the format:
	// map[mechanism][SPECIATE ID][mechanism group]ratio.
	MechAssignment map[string]map[string]map[string]float64

	// MechMW holds molecular weight information for chemical
	// mechanism species. Data should be in the format:
	// map[mechanism][mechanism group]{massOrMol,MW}
	MechMW map[string]map[string]mechData

	// SpecInfo holds name and molecular weight information
	// for individual chemical species.
	// Data should be in the format:
	// map[SPECIATE ID]{name, MW}
	SpecInfo map[string]specInfo
}

// NewMechanisms returns a new Mechanisms variable initialized with
// information from mechanism assignment, molecular weight, and
// species information files in the format specified at
// http://www.cert.ucr.edu/~carter/emitdb/. (The files on the
// website must first be converted from Microsoft Excel to CSV
// format before being read by this function.)
func NewMechanisms(mechAssignment, molarWeight, speciesInfo io.Reader) (*Mechanisms, error) {
	var m Mechanisms
	var err error
	m.MechAssignment, err = readMechAssignment(mechAssignment)
	if err != nil {
		return nil, err
	}
	m.MechMW, err = readMechMW(molarWeight)
	if err != nil {
		return nil, err
	}
	m.SpecInfo, err = readSpecInfoVOC(speciesInfo)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GroupFactors returns the factors by which to multiply the given chemical
// species by to convert it to the given chemical mechanism. If mass is true,
// the multipliers will be adjusted so as to conserve the mass of the input
// species, so the units will be [mass/mass].
// Otherwise, the factors will be computed so as to convert the
// chemical amount to a molar basis and the units will be [mol/gram].
// The species to chemical mechanism conversion
// database (http://www.cert.ucr.edu/~carter/emitdb/)
// is designed to (attempt to) conserve reactivity
// rather than moles or mass, so for molar speciation the total number of moles
// may not be conserved.
func (m *Mechanisms) GroupFactors(mechanism, speciesID string, mass bool) (map[string]float64, error) {
	groupFactors, ok := m.MechAssignment[mechanism][speciesID]
	if !ok {
		return nil, fmt.Errorf("aep: the mechanism name and species ID combination %v and %v is not in the mechanism assignment file", mechanism, speciesID)
	}
	o := make(map[string]float64)
	specinfo, ok := m.SpecInfo[speciesID]
	if !ok {
		return nil, fmt.Errorf("aep: species ID number %v is not in the mechanism species info file", speciesID)
	}
	// Convert mass amounts to molar amounts.
	if !mass {
		for k, v := range groupFactors {
			o[k] = v / specinfo.MW
		}
		return o, nil
	}

	// If we've gotten this far, we're doing mass speciation.
	for k, v := range groupFactors {
		// Multiply the factor by the appropriate molecular weight ratio.
		mechMW, ok := m.MechMW[mechanism][k]
		if !ok {
			return nil, fmt.Errorf("aep: no molecular weight for mechanism %s and mechanism species group %s", mechanism, k)
		}
		o[k] = v * mechMW.MW / specinfo.MW
	}

	// Normalize factors so they sum to 1.
	var sum float64
	for _, v := range o {
		sum += v
	}
	for k, v := range o {
		o[k] = v / sum
	}
	return o, nil
}

// Read file specifying moles of a chemical mechanism model species
// to chemicals in the SPECIATE database. Data can be obtained at
// http://www.cert.ucr.edu/~carter/emitdb/.
// Output data is in the format:
// map[mechanism][SPECIATE ID][mechanism group]ratio
func readMechAssignment(f io.Reader) (map[string]map[string]map[string]float64, error) {
	out := make(map[string]map[string]map[string]float64)
	r := csv.NewReader(f)
	for {
		rec, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("aep: reading mechanism assigment: %v", err)
		}
		mech := rec[0]
		specID := rec[1]
		if err != nil {
			return nil, fmt.Errorf("aep: reading mechanism assigment: %v", err)
		}
		mechGroup := rec[2]
		ratio, err := stringToFloat(rec[3])
		if err != nil {
			return nil, fmt.Errorf("aep: reading mechanism assigment: %v", err)
		}
		if _, ok := out[mech]; !ok {
			out[mech] = make(map[string]map[string]float64)
		}
		if _, ok := out[mech][specID]; !ok {
			out[mech][specID] = make(map[string]float64)
		}
		out[mech][specID][mechGroup] = ratio
	}
	return out, nil
}

type mechData struct {
	massOrMol string
	MW        float64
}

// Read file specifying molecular weights of mechanism model species
// Data can be obtained at
// http://www.cert.ucr.edu/~carter/emitdb/.
// Output data is in the format:
// map[mechanism][mechanism group]{massOrMol,MW}
func readMechMW(f io.Reader) (map[string]map[string]mechData, error) {
	out := make(map[string]map[string]mechData)
	r := csv.NewReader(f)
	for {
		rec, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("aep: reading mechanism molar weight: %v", err)
		}
		mech := rec[0]
		massOrMol := rec[1] // mol or mass
		mechGroup := rec[2]
		MW, err := stringToFloat(rec[3])
		if err != nil {
			return nil, fmt.Errorf("aep: reading mechanism molar weight: %v", err)
		}
		if MW < 0. {
			MW = math.NaN()
		}
		if _, ok := out[mech]; !ok {
			out[mech] = make(map[string]mechData)
		}
		out[mech][mechGroup] = mechData{massOrMol, MW}
	}
	return out, nil
}

type specInfo struct {
	name string
	MW   float64
}

// Read species info file. This data in this file is mainly the same as
// what is in the SPECIATE database, but molecular weights for some
// mixtures are apparently different.
// Data can be obtained at
// http://www.cert.ucr.edu/~carter/emitdb/.
// Output data is in the format:
// map[specID]{name, MW}
func readSpecInfoVOC(f io.Reader) (map[string]specInfo, error) {
	out := make(map[string]specInfo)
	r := csv.NewReader(f)
	for {
		rec, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("aep: reading mechanism species info: %v", err)
		}
		id := rec[0]
		if err != nil {
			return nil, fmt.Errorf("aep: reading mechanism species info: %v", err)
		}
		name := rec[1]
		MW, err := stringToFloat(rec[8])
		if err != nil {
			return nil, fmt.Errorf("aep: reading mechanism species info: %v", err)
		}
		if MW < 0. {
			MW = math.NaN()
		}
		out[id] = specInfo{name, MW}
	}
	return out, nil
}
