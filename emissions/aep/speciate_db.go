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
	"strconv"
	"strings"
)

// SpeciateDB holds information extracted from the
// SPECIATE database.
// (https://www.epa.gov/air-emissions-modeling/tools-related-air-emissions-modeling)
type SpeciateDB struct {
	// SpeciesProperties maps speciate IDs to chemical names and
	// molecular weights.
	SpeciesProperties map[string]struct {
		Name string
		MW   float64
	}

	// VOCToTOG holds the conversion factor from VOC mass to TOG mass
	// for VOCs in different speciation profiles.
	VOCToTOG map[string]float64

	// VOCProfiles maps aggregated VOCs to individual VOC species.
	// Format: map[speciation code][species ID]weight
	VOCProfiles map[string]map[string]float64

	// OtherGasProfiles maps aggregated non-VOC gases
	// such as NOx to individual chemical species.
	// Format: map[speciation code][species ID]weight
	OtherGasProfiles map[string]map[string]float64

	// PMProfiles maps aggregated particulate matter species
	// to individual chemical species.
	// Format: map[speciation code][species ID]weight
	PMProfiles map[string]map[string]float64
}

// NewSpeciateDB returns a new SpeciateDB variable filled with
// information from the SPECIATE database, where the arguments
// are readers of the similarly named tables in the database,
// exported to CSV format.
// The SPECIATE database is available at:
// https://www.epa.gov/air-emissions-modeling/speciate-version-45-through-32.
// To export the tables in CSV format in Linux, use the
// mdb-export command:
//
//	mdb-export SPECIATE.mdb SPECIES_PROPERTIES > SPECIES_PROPERTIES.csv
//	mdb-export SPECIATE.mdb GAS_PROFILE > GAS_PROFILE.csv
//	mdb-export SPECIATE.mdb GAS_SPECIES > GAS_SPECIES.csv
//	mdb-export SPECIATE.mdb OTHER\ GASES_SPECIES > OTHER_GASES_SPECIES.csv
//	mdb-export SPECIATE.mdb PM_SPECIES > PM_SPECIES.csv
func NewSpeciateDB(speciesProperties, gasProfile, gasSpecies, otherGasSpecies, pmSpecies io.Reader) (*SpeciateDB, error) {
	s := new(SpeciateDB)
	if err := s.speciesProperties(speciesProperties); err != nil {
		return nil, err
	}
	if err := s.vocToTOG(gasProfile); err != nil {
		return nil, err
	}
	var err error
	s.VOCProfiles, err = species(gasSpecies)
	if err != nil {
		return nil, fmt.Errorf("aep: gas_species: %v", err)
	}
	s.OtherGasProfiles, err = species(otherGasSpecies)
	if err != nil {
		return nil, fmt.Errorf("aep: other_gas_species: %v", err)
	}
	s.PMProfiles, err = species(pmSpecies)
	if err != nil {
		return nil, fmt.Errorf("aep: pm_species: %v", err)
	}
	return s, nil
}

// Speciation returns the speciation fractions for the given speciation
// profile code and specition type. Valid Type options are "VOC", "NOx",
// and "PM2.5". The output is in the format map[SPECIATE ID]fraction.
func (s *SpeciateDB) Speciation(code string, Type SpeciationType) (map[string]float64, error) {
	switch Type {
	case VOC, VOCUngrouped:
		o, ok := s.VOCProfiles[code]
		if !ok {
			return nil, fmt.Errorf("aep: no VOC speciation profile for code '%s'", code)
		}
		return o, nil
	case NOx:
		o, ok := s.OtherGasProfiles[code]
		if !ok {
			return nil, fmt.Errorf("aep: no NOx speciation profile for code '%s'", code)
		}
		return o, nil
	case PM25:
		o, ok := s.PMProfiles[code]
		if !ok {
			return nil, fmt.Errorf("aep: no PM2.5 speciation profile for code '%s'", code)
		}
		return o, nil
	default:
		return nil, fmt.Errorf("aep: speciate_db: invalid speciation type '%s'", Type)
	}
}

// SpeciesProperties gets species names and molecular weights from the SPECIATE
// SPECIES_PROPERTIES table. The table must first be exported to CSV format.
func (s *SpeciateDB) speciesProperties(f io.Reader) error {
	s.SpeciesProperties = make(map[string]struct {
		Name string
		MW   float64
	})
	r := csv.NewReader(f)
	line, err := r.Read()
	if err != nil {
		return fmt.Errorf("aep: reading species_properties header: %v", err)
	}
	iName, err := colIndex("name", line)
	if err != nil {
		return fmt.Errorf("aep: reading species_properties header: %v", err)
	}
	iMW, err := colIndex("spec_mw", line)
	if err != nil {
		return fmt.Errorf("aep: reading species_properties header: %v", err)
	}
	iCode, err := colIndex("id", line)
	if err != nil {
		return fmt.Errorf("aep: reading species_properties header: %v", err)
	}
	iLine := 2
	for {
		line, err = r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("aep: reading species_properties file line %d: %v", iLine, err)
		}
		var MW float64
		if line[iMW] == "" {
			MW = math.NaN()
		} else {
			MW, err = strconv.ParseFloat(line[iMW], 64)
			if err != nil {
				return fmt.Errorf("aep: reading species_properties file line %d column %d: %v", iLine, iMW, err)
			}
		}
		code := line[iCode]
		name := line[iName]
		s.SpeciesProperties[code] = struct {
			Name string
			MW   float64
		}{
			Name: name,
			MW:   MW,
		}
		iLine++
	}
	return nil
}

// IDFromName returns the SPECIATE ID associated with the given
// species name.
func (s *SpeciateDB) IDFromName(name string) (string, error) {
	for ID, props := range s.SpeciesProperties {
		if props.Name == name {
			return ID, nil
		}
	}
	return "", fmt.Errorf("aep: no SPECIATE ID associated with name %s", name)
}

// vocToTOG gets VOC to TOG conversion information from the SPECIATE
// GAS_PROFILE table. The table must first be exported to CSV format.
func (s *SpeciateDB) vocToTOG(f io.Reader) error {
	s.VOCToTOG = make(map[string]float64)
	r := csv.NewReader(f)
	line, err := r.Read()
	if err != nil {
		return fmt.Errorf("aep: reading VOC to TOG conversion factor header: %v", err)
	}
	iConv, err := colIndex("VOCtoTOG", line)
	if err != nil {
		return fmt.Errorf("aep: reading VOC to TOG conversion factor header: %v", err)
	}
	iCode, err := colIndex("P_NUMBER", line)
	if err != nil {
		return fmt.Errorf("aep: reading VOC to TOG conversion factor header: %v", err)
	}
	iLine := 2
	for {
		line, err = r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("aep: reading VOC to TOG conversion factor file line %d: %v", iLine, err)
		}
		var conv float64
		if line[iConv] == "" {
			conv = 1
		} else {
			conv, err = strconv.ParseFloat(line[iConv], 64)
			if err != nil {
				return fmt.Errorf("aep: reading VOC to TOG conversion factor file line %d column %d: %v", iLine, iConv, err)
			}
		}
		code := line[iCode]
		s.VOCToTOG[code] = conv
		iLine++
	}
	return nil
}

// species loads speciation profile information from a SPECIATE table.
// The table must first be exported to CSV format.
func species(f io.Reader) (map[string]map[string]float64, error) {
	o := make(map[string]map[string]float64)
	r := csv.NewReader(f)
	line, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("reading SPECIATE table header: %v", err)
	}
	iSpecies, err := colIndex("species_id", line)
	if err != nil {
		return nil, fmt.Errorf("reading SPECIATE table header: %v", err)
	}
	iWeight, err := colIndex("weight_per", line)
	if err != nil {
		return nil, fmt.Errorf("reading SPECIATE table header: %v", err)
	}
	iCode, err := colIndex("P_NUMBER", line)
	if err != nil {
		return nil, fmt.Errorf("reading SPECIATE table header: %v", err)
	}
	iLine := 2
	for {
		line, err = r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("reading SPECIATE table line %d: %v", iLine, err)
		}
		var weight float64
		if line[iWeight] != "" {
			weight, err = strconv.ParseFloat(line[iWeight], 64)
			if err != nil {
				return nil, fmt.Errorf("reading SPECIATE table line %d column %d: %v", iLine, iWeight, err)
			}
		}
		code := line[iCode]
		species := line[iSpecies]
		if _, ok := o[code]; !ok {
			o[code] = make(map[string]float64)
		}
		o[code][species] = weight
		iLine++
	}
	// Normalize weights so they sum to 1 for each code.
	for pol, d := range o {
		var sum float64
		for _, v := range d {
			sum += v
		}
		for s, v := range d {
			o[pol][s] = v / sum
		}
	}
	return o, nil
}

// colIndex returns the index in header of the first case-insenstive
// match with n, if any.
func colIndex(name string, header []string) (int, error) {
	n := strings.ToLower(name)
	for i, c := range header {
		if strings.ToLower(c) == n {
			return i, nil
		}
	}
	return -1, fmt.Errorf("header does not contain column '%s'", n)
}
