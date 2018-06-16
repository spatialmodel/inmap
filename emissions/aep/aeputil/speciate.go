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
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/ctessum/unit"
	"github.com/spatialmodel/inmap/emissions/aep"
)

// SpeciateConfig holds speciation configuration information.
type SpeciateConfig struct {
	// These variables specify the locations of files used for
	// chemical speciation.
	SpecRef, SpecRefCombo, SpeciesProperties, GasProfile   string
	GasSpecies, OtherGasSpecies, PMSpecies, MechAssignment string
	MolarWeight, SpeciesInfo                               string

	// ChemicalMechanism specifies which chemical mechanism to
	// use for speciation.
	ChemicalMechanism string

	// MassSpeciation specifies whether to use mass speciation.
	// If false, speciation will convert values to moles.
	MassSpeciation bool

	// SCCExactMatch specifies whether SCCs should be expected to match
	// exactly with the the speciation reference, or if partial matches
	// are acceptable.
	SCCExactMatch bool

	Speciation aep.Speciation

	loadOnce  sync.Once
	speciator *aep.Speciator
}

// SpeciatedRecord is an emissions record where chemical speciation
// has been performed. It should be created using SpeciateConfig.Speciate().
type SpeciatedRecord struct {
	aep.Record

	c             *SpeciateConfig
	emis, dropped *aep.Emissions
}

// GetEmissions returns the speciated emissions.
func (r *SpeciatedRecord) GetEmissions() *aep.Emissions {
	return r.emis
}

// Totals returns emissions totals.
func (r *SpeciatedRecord) Totals() map[aep.Pollutant]*unit.Unit {
	return r.emis.Totals()
}

// PeriodTotals returns total emissions for the given time period.
func (r *SpeciatedRecord) PeriodTotals(begin, end time.Time) map[aep.Pollutant]*unit.Unit {
	return r.emis.PeriodTotals(begin, end)
}

// CombineEmissions combines emissions from r2 with the receiver.
func (r *SpeciatedRecord) CombineEmissions(r2 aep.Record) { r.emis.CombineEmissions(r2) }

// DroppedEmissions returns emissions that were dropped from
// the analysis during speciation to avoid double counting.
func (r *SpeciatedRecord) DroppedEmissions() *aep.Emissions {
	return r.dropped
}

// speciate performs the speciation.
func (r *SpeciatedRecord) speciate() error {
	if r.emis == nil {
		err := r.c.lazyLoadSpeciator()
		if err != nil {
			return err
		}
		r.emis, r.dropped, err = r.c.speciator.Speciate(r.Record, r.c.ChemicalMechanism, r.c.MassSpeciation, !r.c.SCCExactMatch)
		if err != nil {
			return err
		}
	}
	return nil
}

// Speciate chemically speciates the given record.
func (c *SpeciateConfig) Speciate(r aep.Record) (*SpeciatedRecord, error) {
	s := &SpeciatedRecord{
		Record: r,
		c:      c,
	}
	if err := s.speciate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (c *SpeciateConfig) lazyLoadSpeciator() error {
	var err error
	c.loadOnce.Do(func() {
		files := []string{c.SpecRef, c.SpecRefCombo, c.SpeciesProperties,
			c.GasProfile, c.GasSpecies, c.OtherGasSpecies, c.PMSpecies,
			c.MechAssignment, c.MolarWeight, c.SpeciesInfo}
		f := make([]io.Reader, len(files))
		for i, file := range files {
			var fid io.Reader
			fid, err = os.Open(os.ExpandEnv(file))
			if err != nil {
				err = fmt.Errorf("aeputil: loading speciator: %v", err)
				return
			}
			f[i] = fid
		}
		c.speciator, err = aep.NewSpeciator(f[0], f[1], f[2], f[3], f[4], f[5], f[6],
			f[7], f[8], f[9], c.Speciation)
	})
	return err
}

// Iterator creates a new iterator that consumes records from the
// given iterators and chemically speciates them.
func (c *SpeciateConfig) Iterator(parent Iterator) Iterator {
	return &speciateIterator{
		c:             c,
		parent:        parent,
		totals:        make(map[aep.Pollutant]*unit.Unit),
		droppedTotals: make(map[aep.Pollutant]*unit.Unit),
	}
}

type speciateIterator struct {
	c                     *SpeciateConfig
	parent                Iterator
	totals, droppedTotals map[aep.Pollutant]*unit.Unit
}

func (i *speciateIterator) Next() (aep.Record, error) {
	r, err := i.parent.Next()
	if err != nil {
		return nil, err
	}
	s, err := i.c.Speciate(r)
	if err != nil {
		return nil, err
	}
	for pol, val := range s.GetEmissions().Totals() {
		if _, ok := i.totals[pol]; !ok {
			i.totals[pol] = val
		} else {
			i.totals[pol].Add(val)
		}
	}
	for pol, val := range s.DroppedEmissions().Totals() {
		if _, ok := i.droppedTotals[pol]; !ok {
			i.droppedTotals[pol] = val
		} else {
			i.droppedTotals[pol].Add(val)
		}
	}
	return s, nil
}

func (i *speciateIterator) Totals() map[aep.Pollutant]*unit.Unit        { return i.totals }
func (i *speciateIterator) DroppedTotals() map[aep.Pollutant]*unit.Unit { return i.droppedTotals }
func (i *speciateIterator) Group() string                               { return "" }
func (i *speciateIterator) Name() string                                { return "Speciation" }

func (i *speciateIterator) Report() *aep.InventoryReport {
	return &aep.InventoryReport{Data: []aep.Totaler{i}}
}
