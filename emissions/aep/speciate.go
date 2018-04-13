/*
Copyright (C) 2012 the InMAP authors.
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

package aep

import (
	"fmt"
	"io"

	"github.com/ctessum/unit"
)

// Speciation specifies the pollutant names that should be kept for
// analysis and information about how to chemically speciate each one.
// PolInfo holds the configuration of chemical speciation settings for individual pollutants.
// Only one of the fields should be used for each pollutant.
type Speciation map[string]struct {
	// SpecType specifies a type of speciation to be applied.
	// Options are "VOC","PM2.5", and "NOx".
	// If empty, the pollutant will be carried through to
	// the output without speciation, or grouped as
	// if it were the pollutants in "SpecNames".
	SpecType SpeciationType

	// SpecNames contains names of pollutants in the SPECIATE
	// database which are equivalent to this pollutant.
	// For records containing this pollutant, the pollutants
	// included in "SpecNames.Names" will be left out of any
	// speciation that occurs to avoid double counting.
	SpecNames struct {
		// Names are the SPECIATE names that are equivalent to this
		// pollutant.
		Names []string
		// Group specifies whether this pollutant should be
		// grouped into a chemical mechanism.
		Group bool
	}

	// SpecProf can be used to directly specify speciation
	// factors. Conversion from mass to moles, if required,
	// must be included in the provided factors.
	//
	// BUG: The current implementation won't change the units of the emissions, so
	// if a unit conversion was included in the factor it won't be reflected
	// in the speciated result.
	SpecProf map[string]float64
}

// SpeciationType specifies the available speciation types.
type SpeciationType string

// These are the currently supported speciation types.
const (
	VOC           SpeciationType = "VOC"
	VOCUngrouped                 = "VOCUngrouped" // Do not group VOCs into a chemical mechanism.
	NOx                          = "NOx"
	PM25                         = "PM2.5"
	SingleSpecies                = "Single" // This pollutant is already speciated.
	Direct                       = "Direct" // The speciation profile is directly specified in the configuration.
)

// Type returns the SpeciationType of p. If there is not
// an exact match, it will return the speciation type of p
// without its prefix. If that fails, it will return an error.
func (s Speciation) Type(p Pollutant) (SpeciationType, error) {
	t, ok := s[p.String()]
	if !ok {
		t, ok = s[p.Name]
		if !ok {
			return "", fmt.Errorf("aep: no SpeciationType is specified for %s", p.String())
		}
	}
	if t.SpecType != "" {
		return t.SpecType, nil
	}
	if len(t.SpecNames.Names) > 0 {
		return SingleSpecies, nil
	}
	if len(t.SpecProf) > 0 {
		return Direct, nil
	}
	return "", fmt.Errorf("aep: invalid Speciation for %s", p.String())
}

// Names returns the equivalent SPECIATE names of p and whether
// they should each be grouped into a chemical mechanism. If there is not
// an exact match for p, it will return the speciation type of p
// without its prefix. If that fails, or if the SpeciationType
// of p is not SingleSpecies, it will return an error.
func (s Speciation) Names(p Pollutant) ([]string, bool, error) {
	Type, err := s.Type(p)
	if err != nil {
		return nil, false, err
	}
	if Type != SingleSpecies {
		return nil, false, fmt.Errorf("aep: pollutant %v is not a SingleSpecies SpeciationType and therefore does not have equivalent SPECIATE names", p)
	}

	t, ok := s[p.String()]
	if !ok {
		t, ok = s[p.Name]
		if !ok {
			return nil, false, fmt.Errorf("aep: no speciation is specified for %s", p.String())
		}
	}
	return t.SpecNames.Names, t.SpecNames.Group, nil
}

// Profile returns the directly specified speciation profile for p.
// If there is not
// an exact match, it will return the speciation profile for p
// without its prefix. If that fails, or if the SpeciationType
// of p is not Direct, it will return an error.
func (s Speciation) Profile(p Pollutant) (map[string]float64, error) {
	Type, err := s.Type(p)
	if err != nil {
		return nil, err
	}
	if Type != Direct {
		return nil, fmt.Errorf("aep: pollutant %v is not a Direct SpeciationType and therefore does not have a directly specified speciation profile", p)
	}

	t, ok := s[p.String()]
	if !ok {
		t, ok = s[p.Name]
		if !ok {
			return nil, fmt.Errorf("aep: no speciation is specified for %s", p.String())
		}
	}
	return t.SpecProf, nil
}

// Speciate disaggregates the lumped emissions species in the given Record into
// the species groups in the given chemical mechanism. If mass is true,
// speciation is done on a mass basis, otherwise emissions will be converted
// to molar units for gases (but not for particulates). If partialMatch is true,
// matches to speciation profile codes will be made based on parital matches
// to SCC codes if full matches are not available.
func (s *Speciator) Speciate(r Record, mechanism string, mass, partialMatch bool) (emis, dropped *Emissions, err error) {
	e := r.GetEmissions()
	emis = new(Emissions)
	dropped = new(Emissions)

	// Get the list of pollutants that will be double-counted.
	doubleCountIDs, err := s.doubleCountIDs(e)
	if err != nil {
		return nil, nil, err
	}

	// Range through each emissions period.
	for _, ep := range e.e {
		Type, err := s.Speciation.Type(ep.Pollutant)
		if err != nil {
			return nil, nil, err
		}

		switch Type {
		case VOC, VOCUngrouped, NOx, PM25:
			// Get the speciation codes.
			var codes map[string]float64
			codes, err = s.SpecRef.Codes(r.GetSCC(), ep.Pollutant, ep.begin, ep.end, r.GetCountry(), r.GetFIPS(), partialMatch)
			if err != nil {
				return nil, nil, err
			}
			for code, codeFrac := range codes {
				// Get the species fractions for this code.
				var speciesFracs map[string]float64
				speciesFracs, err = s.SpeciateDB.Speciation(code, Type)
				if err != nil {
					return nil, nil, err
				}
				switch Type {
				case VOC, VOCUngrouped:
					// Get the VOC to TOG conversion
					vocToTOG, ok := s.SpeciateDB.VOCToTOG[code]
					if !ok {
						return nil, nil, fmt.Errorf("aep: no VOC to TOG conversion for speciation profile '%v'", code)
					}
					for speciesID, frac := range speciesFracs {
						var drop bool
						drop, err = dropped.addDoubleCounted(doubleCountIDs, ep, e.units, vocToTOG*codeFrac*frac, speciesID, s)
						if err != nil {
							return nil, nil, err
						}
						if drop {
							continue
						}
						switch Type {
						case VOC: // Group VOCs into chemical mechanism.
							if err = emis.addVOC(ep, vocToTOG*codeFrac*frac, speciesID, mechanism, e.units, mass, s); err != nil {
								return nil, nil, err
							}
						case VOCUngrouped: // Do not group VOCs.
							if err = emis.addUngroupedGas(ep, vocToTOG*codeFrac*frac, speciesID, e.units, mass, s); err != nil {
								return nil, nil, err
							}
						default:
							panic("this shouldn't happen")
						}
					}
				case NOx, PM25:
					for speciesID, frac := range speciesFracs {
						var drop bool
						drop, err = dropped.addDoubleCounted(doubleCountIDs, ep, e.units, codeFrac*frac, speciesID, s)
						if err != nil {
							return nil, nil, err
						}
						if drop {
							continue
						}
						switch Type {
						case NOx:
							if err = emis.addUngroupedGas(ep, codeFrac*frac, speciesID, e.units, mass, s); err != nil {
								return nil, nil, err
							}
						case PM25:
							if err = emis.addPM(ep, codeFrac*frac, speciesID, e.units, s); err != nil {
								return nil, nil, err
							}
						default:
							panic("this shouldn't happen")
						}
					}
				default:
					panic("this shouldn't happen")
				}
			}
		case SingleSpecies: // Divide a single species into the chemical mechanism groups.
			if err = emis.addSingleSpecies(ep, mechanism, e.units, mass, s); err != nil {
				return nil, nil, err
			}
		case Direct:
			if err := emis.addDirect(ep, e.units, s); err != nil {
				return nil, nil, err
			}
		default:
			panic("this shouldn't happen")
		}
	}
	return emis, dropped, nil
}

// doubleCountIDs creates a list of the pollutants that are included in the
// inventory as explicit species. If we end up generating
// any mass of any of these pollutants while speciating,
// we will include that mass in the double counted totals
// rather than in the main result to avoid double counting.
// We assume that if a species is explicitly tracked during
// one time period, it is explicitly tracked during all
// time periods.
func (s *Speciator) doubleCountIDs(e *Emissions) (map[string]int, error) {
	doubleCountIDs := make(map[string]int)
	for _, ep := range e.e {
		Type, err := s.Speciation.Type(ep.Pollutant)
		if err != nil {
			return nil, err
		}
		if Type == SingleSpecies {
			specNames, _, err := s.Speciation.Names(ep.Pollutant)
			if err != nil {
				return nil, err
			}
			for _, n := range specNames {
				id, err := s.SpeciateDB.IDFromName(n)
				if err != nil {
					return nil, err
				}
				doubleCountIDs[id] = 1
			}
		}
	}
	return doubleCountIDs, nil
}

// addDoubleCounted checks if speciesID is in the list of double counted
// pollutants. If it is, the emissions associated with speciesID are added
// to the receiver, which is expected to be a holder for double counted
// emissions, and a true value is returned signifying that the
// emissions should not be added to the main output.
// Double counted emissions are not converted to moles.
func (e *Emissions) addDoubleCounted(doubleCountIDs map[string]int, ep *emissionsPeriod, u map[Pollutant]unit.Dimensions, factor float64, speciesID string, s *Speciator) (bool, error) {
	if doubleCountIDs[speciesID] == 1 {
		// This species is double counted, so don't include
		// it in the total.
		speciesInfo, ok := s.SpeciateDB.SpeciesProperties[speciesID]
		if !ok {
			return true, fmt.Errorf("aep: no properties in SPECIATE db for species ID '%s'", speciesID)
		}
		emisRate := unit.Mul(unit.New(ep.rate, u[ep.Pollutant]), unit.New(factor, unit.Dimless))
		e.Add(ep.begin, ep.end, speciesInfo.Name, "", emisRate)
		return true, nil
	}
	return false, nil
}

// addVOC adds speciated VOCs to the receiver after grouping them into
// a chemical mechanism.
func (e *Emissions) addVOC(ep *emissionsPeriod, factor float64, speciesID, mechanism string, u map[Pollutant]unit.Dimensions, mass bool, s *Speciator) error {
	groupFactors, err := s.Mechanisms.GroupFactors(mechanism, speciesID, mass)
	if err != nil {
		return err
	}
	for group, groupFactor := range groupFactors {
		// Create a new emissions variable for this group.
		var specFactor *unit.Unit
		if mass {
			specFactor = unit.New(factor*groupFactor, unit.Dimless)
		} else {
			specFactor = unit.New(factor*groupFactor, unit.Dimensions{kiloMol: 1, unit.MassDim: -1})
		}
		e.Add(ep.begin, ep.end, group, "", unit.Mul(unit.New(ep.rate, u[ep.Pollutant]), specFactor))
	}
	return nil
}

// addUngroupedGas adds speciated emissions for a gas that is not meant to
// be grouped into a chemical mechanism.
func (e *Emissions) addUngroupedGas(ep *emissionsPeriod, factor float64, speciesID string, u map[Pollutant]unit.Dimensions, mass bool, s *Speciator) error {
	speciesInfo, ok := s.SpeciateDB.SpeciesProperties[speciesID]
	if !ok {
		return fmt.Errorf("aep: no properties in SPECIATE db for species ID '%s'", speciesID)
	}
	var specFactor *unit.Unit
	if !mass {
		specFactor = unit.New(factor/speciesInfo.MW, unit.Dimensions{kiloMol: 1, unit.MassDim: -1})
	} else {
		specFactor = unit.New(factor, unit.Dimless)
	}
	e.Add(ep.begin, ep.end, speciesInfo.Name, "", unit.Mul(unit.New(ep.rate, u[ep.Pollutant]), specFactor))
	return nil
}

// addPM adds speciated emissions without the option of converting to moles.
func (e *Emissions) addPM(ep *emissionsPeriod, factor float64, speciesID string, u map[Pollutant]unit.Dimensions, s *Speciator) error {
	speciesInfo, ok := s.SpeciateDB.SpeciesProperties[speciesID]
	if !ok {
		return fmt.Errorf("aep: no properties in SPECIATE db for species ID '%s'", speciesID)
	}
	e.Add(ep.begin, ep.end, speciesInfo.Name, "", unit.Mul(unit.New(ep.rate, u[ep.Pollutant]), unit.New(factor, unit.Dimless)))
	return nil
}

// addSingleSpecies adds an already speciated emission into a chemical
// mechanism in the receiver.
func (e *Emissions) addSingleSpecies(ep *emissionsPeriod, mechanism string, u map[Pollutant]unit.Dimensions, mass bool, s *Speciator) error {
	names, group, err := s.Speciation.Names(ep.Pollutant)
	if err != nil {
		return err
	}
	speciesID, err := s.SpeciateDB.IDFromName(names[0])
	if err != nil {
		return err
	}

	if !group { // Don't group into a species mechanism.
		speciesInfo, ok := s.SpeciateDB.SpeciesProperties[speciesID]
		if !ok {
			return fmt.Errorf("aep: no properties in SPECIATE db for species ID '%s'", speciesID)
		}
		var specFactor *unit.Unit
		if !mass {
			specFactor = unit.New(1/speciesInfo.MW, unit.Dimensions{kiloMol: 1, unit.MassDim: -1})
		} else {
			specFactor = unit.New(1, unit.Dimless)
		}
		e.Add(ep.begin, ep.end, speciesInfo.Name, "", unit.Mul(unit.New(ep.rate, u[ep.Pollutant]), specFactor))
		return nil // If we're not grouping into a chemical mechanism, we're done.
	}

	groupFactors, err := s.Mechanisms.GroupFactors(mechanism, speciesID, mass)
	if err != nil {
		return err
	}
	for group, factor := range groupFactors {
		// Create a new emissions variable for this group.
		var specFactor *unit.Unit
		if mass {
			specFactor = unit.New(factor, unit.Dimless)
		} else {
			specFactor = unit.New(factor, unit.Dimensions{kiloMol: 1, unit.MassDim: -1})
		}
		e.Add(ep.begin, ep.end, group, "", unit.Mul(unit.New(ep.rate, u[ep.Pollutant]), specFactor))
	}
	return nil
}

// addDirect performs Direct speciation to ep and adds the result
// to the receiver.
func (e *Emissions) addDirect(ep *emissionsPeriod, u map[Pollutant]unit.Dimensions, s *Speciator) error {
	prof, err := s.Speciation.Profile(ep.Pollutant)
	if err != nil {
		return err
	}
	for name, val := range prof {
		e.Add(ep.begin, ep.end, name, "", unit.New(ep.rate*val, u[ep.Pollutant]))
	}
	return nil
}

var kiloMol = unit.NewDimension("kmol")

// Speciator speciates emissions in Records
// from more aggregated chemical groups to more specific chemical
// groups.
type Speciator struct {
	SpecRef    *SpecRef
	SpeciateDB *SpeciateDB
	Mechanisms *Mechanisms
	Speciation Speciation
}

// NewSpeciator returns a new Speciator variable.
func NewSpeciator(specRef, specRefCombo, speciesProperties, gasProfile, gasSpecies, otherGasSpecies, pmSpecies, mechAssignment, molarWeight, speciesInfo io.Reader, speciation Speciation) (*Speciator, error) {
	var s Speciator
	var err error
	s.SpecRef, err = NewSpecRef(specRef, specRefCombo)
	if err != nil {
		return nil, err
	}
	s.SpeciateDB, err = NewSpeciateDB(speciesProperties, gasProfile, gasSpecies, otherGasSpecies, pmSpecies)
	if err != nil {
		return nil, err
	}
	s.Mechanisms, err = NewMechanisms(mechAssignment, molarWeight, speciesInfo)
	if err != nil {
		return nil, err
	}
	s.Speciation = speciation
	return &s, nil
}
