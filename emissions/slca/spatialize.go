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
	"bytes"
	"context"
	"crypto/md5"
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"

	"bitbucket.org/ctessum/sparse"

	"github.com/ctessum/geom"
	"github.com/ctessum/requestcache"
)

const populationSrg = "100"

// EmissionsSurrogate returns the spatially-explicit emissions caused by
// spatialRef.
func (c *CSTConfig) EmissionsSurrogate(ctx context.Context, pol Pollutant, spatialRef *SpatialRef) (*sparse.SparseArray, error) {
	recs, err := c.spatialSurrogate(ctx, spatialRef)
	if err != nil {
		return nil, err
	}
	return c.scaleFlattenSrg(recs, pol, 1)
}

// spatialSurrogate gets spatial information for a given spatial reference.
func (c *CSTConfig) spatialSurrogate(ctx context.Context, spatialRef *SpatialRef) ([]*inmap.EmisRecord, error) {
	c.loadSpatialOnce.Do(func() {
		c.spatializeRequestCache = loadCacheOnce(c.spatialSurrogateWorker, 1, c.MaxCacheEntries, c.SpatialCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	rr := c.spatializeRequestCache.NewRequest(ctx, spatialRef, spatialRef.Key())
	resultI, err := rr.Result()
	if err != nil {
		return nil, err
	}
	return resultI.([]*inmap.EmisRecord), nil
}

// spatialSurrogateWorker gets spatial information for a given spatial reference
// request.
func (c *CSTConfig) spatialSurrogateWorker(ctx context.Context, request interface{}) (interface{}, error) {
	if err := c.lazyLoadSR(); err != nil { // We need the SR geometry.
		return nil, err
	}

	spatialRef := request.(*SpatialRef)
	switch spatialRef.Type {
	case Stationary:
		if spatialRef.NoSpatial {
			// Allocate NoSpatial processes using population density
			return c.neiSpatialSrg(populationSrg, "00000")
		} else if spatialRef.Surrogate != "" {
			return c.neiSpatialSrg(spatialRef.Surrogate, spatialRef.SurrogateFIPS)
		} else if len(spatialRef.SCCs) > 0 {
			return c.neiEmisSrg(spatialRef)
		}
		return nil, fmt.Errorf("in slca.spatialSurrogate: no spatial information for spatial reference %#v", spatialRef)
	case Transportation, Vehicle:
		// surrogate 240 is total road miles
		return c.neiSpatialSrg("240", c.DefaultFIPS) //TODO: make proper surrogates for TransportationProcesses
	case NoSpatial:
		// Allocate mixes using population density
		return c.neiSpatialSrg(populationSrg, c.DefaultFIPS)

	default:
		return nil, fmt.Errorf("in slca.spatialSurrogate: unsupported Type %v", spatialRef.Type)
	}
}

// SpatialRef holds reference information about the spatial location of
// this process.
type SpatialRef struct {
	// EPA Source Classification codes corresponding to this process.
	SCCs []SCC `xml:"scc"`

	// SCCFractions specifies adjustment factors to multiply emissions by
	// for each SCC code in SCCs. If SCCFractions is nil, no adjustments
	// are applied.
	SCCFractions []float64

	// EmisYear specifies the year to adjust emissions amounts to
	// for the SCCs codes corresponding to this process.
	EmisYear int

	// Surrogate code for this process.
	Surrogate string `xml:"spatial_srg"`

	// The FIPS code to be used with Surrogate for this process.
	SurrogateFIPS string `xml:"spatial_srg_fips"`

	// NoSpatial is true if this process is intentionally lacking spatial information.
	NoSpatial bool

	Type ProcessType

	// NoNormalization specifies whether the spatial surrogate
	// should be normalized so that its sum==1. The default is
	// to perform normalization.
	NoNormalization bool
}

// Key returns a unique identifier for this SpatialRef.
func (sr *SpatialRef) Key() string {
	b := bytes.NewBuffer(nil)
	e := gob.NewEncoder(b)
	if err := e.Encode(sr); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", md5.Sum(b.Bytes()))
}

// neiSpatialSrg creates a spatial surrogate from the spatial surrogate associated
// with srgCode.
func (c *CSTConfig) neiSpatialSrg(srgCode, FIPS string) ([]*inmap.EmisRecord, error) {
	sp, err := c.SpatialConfig.SpatialProcessor()
	if err != nil {
		return nil, err
	}
	srgSpec, err := sp.SrgSpecs.GetByCode(aep.USA, srgCode)
	if err != nil {
		return nil, err
	}
	if FIPS == "" {
		FIPS = c.DefaultFIPS
	}
	srg, _, err := sp.Surrogate(srgSpec, sp.Grids[0], FIPS)
	if err != nil {
		return nil, err
	}
	if srg == nil {
		return nil, fmt.Errorf("in slca.neiSpatialSrg: nil surrogate for FIPS %v in %#v", FIPS, srgSpec)
	}
	recs := make([]*inmap.EmisRecord, len(c.SpatialConfig.GridCells))
	for i, v := range srg.Elements {
		recs[i] = &inmap.EmisRecord{
			PM25: v,
			NH3:  v,
			NOx:  v,
			SOx:  v,
			VOC:  v,
			Geom: c.SpatialConfig.GridCells[i].Centroid(),
		}
	}
	return recs, nil
}

// neiEmisSrg creates a spatial surrogate from emissions records in the NEI
// matching the SCCs in spatialRef.
func (c *CSTConfig) neiEmisSrg(spatialRef *SpatialRef) ([]*inmap.EmisRecord, error) {
	c.loadInventoryOnce.Do(func() {
		// Initialize emissions record holder.
		c.emis = make(map[int]map[string][]aep.Record)
	})
	if _, ok := c.emis[spatialRef.EmisYear]; !ok {
		fmt.Println("Filtering out New York State commercial cooking emissions.")
		c.InventoryConfig.FilterFunc = func(r aep.Record) bool {
			switch r.GetSCC() {
			case "2302002000", "2302002100", "2302002200", "2302003000", "2302003100", "2302003200":
				// Commercial meat cooking
				fips := r.GetFIPS()
				if fips[0:2] == "36" { // New York State
					return false
				}
			}
			return true
		}
		// Load emissions for the requested year if they haven't
		// already been loaded.
		emis, _, err := c.InventoryConfig.ReadEmissions()
		if err != nil {
			return nil, err
		}

		emis, err = c.groupBySCCAndApplyAdj(emis)
		if err != nil {
			return nil, err
		}

		if c.NEIBaseYear != 0 && spatialRef.EmisYear != 0 {
			// Scale emissions for the requested year.
			f, err := os.Open(c.SCCReference)
			if err != nil {
				return nil, fmt.Errorf("slca: opening SCCReference: %v", err)
			}
			emisScale, err := aeputil.ScaleNEIStateTrends(c.NEITrends, f, c.NEIBaseYear, spatialRef.EmisYear)
			if err != nil {
				return nil, fmt.Errorf("slca: Scaling NEI emissions: %v", err)
			}
			if err := aeputil.Scale(emis, emisScale); err != nil {
				return nil, err
			}
		}
		c.emis[spatialRef.EmisYear] = emis
	}
	emis := c.emis[spatialRef.EmisYear]
	sp, err := c.SpatialConfig.SpatialProcessor()
	if err != nil {
		return nil, err
	}

	foundData := false
	var aepRecs []aep.Record
	for i, scc := range spatialRef.SCCs {
		recs, ok := emis[string(scc)]
		if ok {
			foundData = true
		}
		scaledRecs := make([]aep.Record, len(recs))
		if spatialRef.SCCFractions != nil {
			for j, rec := range recs {
				scaledRecs[j] = &scaledEmissionsRecord{
					Record: rec,
					scale:  spatialRef.SCCFractions[i],
				}
			}
		} else {
			scaledRecs = recs
		}
		aepRecs = append(aepRecs, scaledRecs...)
	}
	if !foundData {
		return nil, fmt.Errorf("in slca.neiEmisSrg: couldn't find emissions matching any of these SCCs: %v", spatialRef.SCCs)
	}
	VOC, NOx, NH3, SOx, PM25, err := inmapPols(c.InventoryConfig.PolsToKeep)
	if err != nil {
		return nil, err
	}
	outData, err := inmap.FromAEP(aepRecs, sp, 0, VOC, NOx, NH3, SOx, PM25)
	if err != nil {
		return nil, err
	}
	if !spatialRef.NoNormalization {
		normalizeSrg(outData)
	}
	return outData, nil
}

// scaledEmissionsRecord is a wrapper for an emissions record that
// applies a scaling factor before returning the emissions total
type scaledEmissionsRecord struct {
	aep.Record
	scale float64
}

// GetEmissions overrides the corresponding Record method to
// return scaled emissions.
func (r *scaledEmissionsRecord) GetEmissions() *aep.Emissions {
	e := r.Record.GetEmissions().Clone()
	e.Scale(func(arg1 aep.Pollutant) (float64, error) { return r.scale, nil })
	return e
	//return r.Record.GetEmissions()
}

// normalizeSrg transforms the input so that all emissions across
// all pollutants sum to 1.
func normalizeSrg(d []*inmap.EmisRecord) {
	var grandTotal float64
	for _, r := range d {
		grandTotal += r.NH3 + r.NOx + r.PM25 + r.SOx + r.VOC
	}
	for _, r := range d {
		v := (r.NH3 + r.NOx + r.PM25 + r.SOx + r.VOC) / grandTotal
		r.PM25 = v
		r.NH3 = v
		r.NOx = v
		r.SOx = v
		r.VOC = v
	}
}

// Pollutant specifies air pollutant names
type Pollutant int

// These are the valid air pollutant names.
const (
	PM25 Pollutant = iota
	NH3
	NOx
	SOx
	VOC
)

// scaleFlattenSrg converts srg into a spatial array by multiplying the
// PM2.5 emissions value in each record by scale and allocating it
// the the grid cell(s) it falls within.
func (c *CSTConfig) scaleFlattenSrg(srg []*inmap.EmisRecord, pol Pollutant, scale float64) (*sparse.SparseArray, error) {
	if err := c.lazyLoadSR(); err != nil {
		return nil, err
	}
	o := sparse.ZerosSparse(len(c.SpatialConfig.GridCells))
	for _, rec := range srg {
		cells := c.gridIndex.SearchIntersect(rec.Geom.Bounds())
		for _, cI := range cells {
			c := cI.(gridIndex)
			if rec.Geom.(geom.Point).Within(c) == geom.Outside {
				panic("only rectangular grid cells are supported")
			}
			var v float64
			switch pol {
			case PM25:
				v = rec.PM25
			case NH3:
				v = rec.NH3
			case NOx:
				v = rec.NOx
			case SOx:
				v = rec.SOx
			case VOC:
				v = rec.VOC
			default:
				return nil, fmt.Errorf("slca: invalid pollutant: %v", pol)
			}
			o.AddVal(scale*v/float64(len(cells)), c.i)
		}
	}
	return o, nil
}

type arrayAdjuster sparse.DenseArray

func (a *arrayAdjuster) Adjustment() (*sparse.DenseArray, error) {
	return (*sparse.DenseArray)(a), nil
}

// groupBySCCAndApplyAdj groups the records by SCC code instead of by sector
// and applies a fugitive dust adjustment.
func (c *CSTConfig) groupBySCCAndApplyAdj(emis map[string][]aep.Record) (map[string][]aep.Record, error) {
	// Read the fugitive dust adjustment file.
	f, err := os.Open(c.FugitiveDustAdjustment)
	if err != nil {
		return nil, fmt.Errorf("slca: opening FugitiveDustAdjustment file: %v", err)
	}
	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("slca: reading FugitiveDustAdjustment file: %v", err)
	}

	// Convert the list of fugitive dust sectors into a map index.
	fugitiveDustSectors := make(map[string]struct{})
	for _, s := range c.FugitiveDustSectors {
		fugitiveDustSectors[s] = struct{}{}
	}

	d := sparse.ZerosDense(len(lines), len(lines[0]))
	for j := 0; j < len(lines); j++ {
		for i := 0; i < len(lines[j]); i++ {
			v, err := strconv.ParseFloat(strings.TrimSpace(lines[j][i]), 64)
			if err != nil {
				return nil, fmt.Errorf("slca: reading FugitiveDustAdjustment file: %v", err)
			}
			d.Set(v, j, i)
		}
	}
	adj := arrayAdjuster(*d)

	// Reorganize records and apply adjustments.
	o := make(map[string][]aep.Record)
	for sector, recs := range emis {
		for _, rec := range recs {
			if _, ok := fugitiveDustSectors[sector]; ok {
				o[rec.GetSCC()] = append(o[rec.GetSCC()], &aep.SpatialAdjustRecord{
					Record:          rec,
					SpatialAdjuster: &adj,
				})
			} else {
				o[rec.GetSCC()] = append(o[rec.GetSCC()], rec)
			}
		}
	}
	return o, nil
}

// unmarshalGob unmarshals an interface from a byte array and fulfills
// the requirements for the Disk cache unmarshalFunc input.
func unmarshalGob(b []byte) (interface{}, error) {
	r := bytes.NewBuffer(b)
	d := gob.NewDecoder(r)
	var data []*inmap.EmisRecord
	if err := d.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// marshalGob marshals an interface to a byte array and fulfills
// the requirements for the Disk cache marshalFunc input.
func marshalGob(data interface{}) ([]byte, error) {
	w := bytes.NewBuffer(nil)
	e := gob.NewEncoder(w)
	d := *data.(*interface{})
	dd := d.([]*inmap.EmisRecord)
	if err := e.Encode(dd); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
