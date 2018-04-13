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

package slca

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ctessum/unit"
	"github.com/spatialmodel/inmap/emissions/aep"
)

// Speciate returns a chemically speciated copy of r.
func (db *DB) Speciate(r *Results) (*Results, error) {
	if err := db.CSTConfig.lazyLoadSpeciator(); err != nil {
		return nil, err
	}
	o := NewResults(db.LCADB)

	o.Nodes = r.Nodes
	o.Edges = make([]*ResultEdge, len(r.Edges))

	for i, oldE := range r.Edges {
		newE := ResultEdge{
			FromID: oldE.ToID,
			ToID:   oldE.ToID,
			ID:     oldE.ID,
		}
		var err error
		newE.FromResults, err = db.speciate(oldE.FromResults)
		if err != nil {
			return nil, err
		}
		o.Edges[i] = &newE
	}
	return o, nil
}

func (db *DB) speciate(r *OnsiteResults) (*OnsiteResults, error) {
	year := int(db.LCADB.GetYear() + 0.5)
	begin, end, err := aep.Annual.TimeInterval(fmt.Sprintf("%d", year))
	interval := unit.New(end.Sub(begin).Seconds(), unit.Second)
	if err != nil {
		return nil, err
	}
	o := NewOnsiteResults(db)
	o.Resources = r.Resources
	o.Requirements = r.Requirements
	for sp, spData := range r.Emissions {
		rec := new(aep.PolygonRecord)
		rec.SourceData.SCC = string(sp.GetSCC())
		rec.SourceData.FIPS = "01001"
		rec.SourceData.Country = aep.USA
		for g, v := range spData {
			p := splitPol(g.GetName())
			rec.Emissions.Add(begin, end, p.Name, p.Prefix, unit.Div(v, interval))
		}
		emis, _, err := db.CSTConfig.speciator.Speciate(
			rec,
			db.CSTConfig.NEIData.ChemicalMechanism,
			db.CSTConfig.NEIData.MassSpeciation,
			!db.CSTConfig.NEIData.SCCExactMatch,
		)
		if err != nil {
			return nil, err
		}
		totals := emis.Totals()
		for p, v := range totals {
			o.AddEmission(sp, gas(p.Name), v)
		}
	}
	return o, nil
}

type gas string

func (g gas) GetName() string { return string(g) }
func (g gas) GetID() string   { return string(g) }

func splitPol(p string) aep.Pollutant {
	if strings.Contains(p, "_TBW") {
		return aep.Pollutant{Name: strings.TrimRight(p, "_TBW"), Prefix: "TIR"}
	}
	if strings.Contains(p, "_evap") {
		return aep.Pollutant{Name: strings.TrimRight(p, "_evap"), Prefix: "EVP"}
	}
	return aep.Pollutant{Name: p}
}

func (c *CSTConfig) lazyLoadSpeciator() error {
	var err error
	c.speciateOnce.Do(func() {
		files := []string{c.NEIData.SpecRef, c.NEIData.SpecRefCombo, c.NEIData.SpeciesProperties,
			c.NEIData.GasProfile, c.NEIData.GasSpecies, c.NEIData.OtherGasSpecies, c.NEIData.PMSpecies,
			c.NEIData.MechAssignment, c.NEIData.MolarWeight, c.NEIData.SpeciesInfo}
		f := make([]io.Reader, len(files))
		for i, file := range files {
			var fid io.Reader
			fid, err = os.Open(os.ExpandEnv(file))
			if err != nil {
				err = fmt.Errorf("aep: loading speciator: %v", err)
				return
			}
			f[i] = fid
		}
		c.speciator, err = aep.NewSpeciator(f[0], f[1], f[2], f[3], f[4], f[5], f[6],
			f[7], f[8], f[9], c.InventoryConfig.PolsToKeep)
	})
	return err
}
