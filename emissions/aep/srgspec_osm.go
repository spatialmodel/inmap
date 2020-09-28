/*
Copyright (C) 2019 the InMAP authors.
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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/osm"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/requestcache/v2"
)

// SrgSpecOSM holds OpenStreetMap spatial surrogate specification information.
type SrgSpecOSM struct {
	Region Country `json:"region"`
	Name   string  `json:"name"`
	Code   string  `json:"code"`

	OSMFile string `json:"osm_file"`

	Tags map[string][]string `json:"tags"`

	// BackupSurrogateNames specifies names of surrogates to use if this
	// one doesn't have data for the desired location.
	BackupSurrogateNames []string `json:"backup_surrogate_names"`

	// MergeNames specify names of other surrogates that should be combined
	// to create this surrogate.
	MergeNames []string `json:"merge_names"`
	// MergeMultipliers specifies multipliers associated with the surrogates
	// in MergeNames.
	MergeMultipliers []float64 `json:"merge_multipliers"`

	cache *requestcache.Cache
}

// ReadSrgSpec reads a OpenStreetMap surrogate specification formated as a
// JSON array of SrgSpecOSM objects.
// diskCachePath specifies a path to a directory where an on-disk cache should
// be created (if "", no cache will be created), and memCacheSize specifies the
// number of surrogate data entries to hold in an in-memory cache.
func ReadSrgSpecOSM(r io.Reader, diskCachePath string, memCacheSize int) (*SrgSpecs, error) {
	d := json.NewDecoder(r)
	var o []*SrgSpecOSM
	err := d.Decode(&o)
	if err != nil {
		return nil, err
	}
	srgs := NewSrgSpecs()
	cache, err := newCacheV2(diskCachePath, memCacheSize, marshalSrgHolders, unmarshalSrgHolders)
	if err != nil {
		return nil, err
	}
	for _, s := range o {
		s.cache = cache
		srgs.Add(s)
	}
	return srgs, nil
}

func (srg *SrgSpecOSM) backupSurrogateNames() []string { return srg.BackupSurrogateNames }
func (srg *SrgSpecOSM) region() Country                { return srg.Region }
func (srg *SrgSpecOSM) code() string                   { return srg.Code }
func (srg *SrgSpecOSM) name() string                   { return srg.Name }
func (srg *SrgSpecOSM) mergeNames() []string           { return srg.MergeNames }
func (srg *SrgSpecOSM) mergeMultipliers() []float64    { return srg.MergeMultipliers }

// getSrgData returns the spatial surrogate information for this
// surrogate definition and location, where tol is tolerance for geometry simplification.
func (srg *SrgSpecOSM) getSrgData(gridData *GridDef, inputLoc *Location, tol float64) (*rtree.Rtree, error) {
	// Calculate the area of interest for our surrogate data.
	inputShapeT, err := inputLoc.Reproject(gridData.SR)
	if err != nil {
		return nil, err
	}
	inputShapeBounds := inputShapeT.Bounds()
	srgBounds := inputShapeBounds.Copy()
	for _, cell := range gridData.Cells {
		b := cell.Bounds()
		if b.Overlaps(inputShapeBounds) {
			srgBounds.Extend(b)
		}
	}

	sd := &readSrgDataOSMInput{gridData: gridData, tol: tol, srg: srg}
	request := srg.cache.NewRequest(context.TODO(), sd)
	srgs, err := request.Result()
	if err != nil {
		return nil, err
	}
	return srgs.(readSrgDataOutput).index, nil
}

type readSrgDataOSMInput struct {
	gridData *GridDef
	tol      float64
	srg      *SrgSpecOSM
}

func (sd *readSrgDataOSMInput) Key() string {
	return fmt.Sprintf("osm_srgdata_%s%s_%s_%g", sd.srg.region(), sd.srg.code(),
		sd.gridData.SR.Name, sd.tol)
}

// Run returns all of the spatial surrogate information for this
// surrogate definition, inputI is of type *osmReadSrgDataInput and
// inputI.tol is tolerance for geometry simplification.
func (input *readSrgDataOSMInput) Run(ctx context.Context) (interface{}, error) {
	srg := input.srg
	log.Printf("processing surrogate `%s` spatial data", srg.Name)

	srgSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		panic(err)
	}

	srgCT, err := srgSR.NewTransform(input.gridData.SR)
	if err != nil {
		return nil, err
	}

	data, err := osm.ExtractFile(context.Background(), os.ExpandEnv(srg.OSMFile), osm.KeepTags(srg.Tags), false)
	if err != nil {
		return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tags %v: %v", srg.Tags, err)
	}
	geomTags, err := data.Geom()
	if err != nil {
		return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tags %v: %v", srg.Tags, err)
	}
	dominantType, err := osm.DominantType(geomTags)
	if err != nil {
		return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tags %v: %v", srg.Tags, err)
	}
	srgs := readSrgDataOutput{
		index: rtree.NewTree(25, 50),
	}
	for _, geomTag := range geomTags {
		g, err := osmGeometry(geomTag.Geom, dominantType)
		if err != nil {
			return nil, fmt.Errorf("aep: processing OSM spatial surrogate data: %v", err)
		}
		if g == nil {
			continue // ignore geometry that is not the dominant type.
		}
		g, err = g.Transform(srgCT)
		if err != nil {
			return nil, fmt.Errorf("aep: processing OSM spatial surrogate data: %v", err)
		}
		if input.tol > 0 {
			switch gs := g.(type) {
			case geom.Simplifier:
				g = gs.Simplify(input.tol)
			}
		}
		srgData := &srgHolder{
			Geom:   g,
			Weight: 1,
		}
		srgs.srgs = append(srgs.srgs, srgData)
		srgs.index.Insert(srgData)
	}
	return srgs, nil
}

func geomCollectionToMultiPoint(gc geom.GeometryCollection, dominantType osm.GeomType) (geom.MultiPoint, error) {
	o := geom.MultiPoint{}
	for _, f := range gc {
		if gc2, ok := f.(geom.GeometryCollection); ok {
			var err error
			f, err = osmGeometry(gc2, dominantType)
			if err != nil {
				return nil, err
			}
		}
		if p, ok := f.(geom.Point); ok {
			o = append(o, p)
		}
		if p, ok := f.(geom.MultiPoint); ok {
			o = append(o, p...)
		}
	}
	if len(o) > 0 {
		return o, nil
	}
	return nil, nil
}

func geomCollectionToMultiPolygon(gc geom.GeometryCollection, dominantType osm.GeomType) (geom.MultiPolygon, error) {
	o := geom.MultiPolygon{}
	for _, f := range gc {
		if gc2, ok := f.(geom.GeometryCollection); ok {
			var err error
			f, err = osmGeometry(gc2, dominantType)
			if err != nil {
				return nil, err
			}
		}
		if p, ok := f.(geom.Polygon); ok {
			o = append(o, p)
		}
		if p, ok := f.(geom.MultiPolygon); ok {
			o = append(o, p...)
		}
	}
	if len(o) > 0 {
		return o, nil
	}
	return nil, nil
}

func geomCollectionToMultiLineString(gc geom.GeometryCollection, dominantType osm.GeomType) (geom.MultiLineString, error) {
	o := geom.MultiLineString{}
	for _, f := range gc {
		if gc2, ok := f.(geom.GeometryCollection); ok {
			var err error
			f, err = osmGeometry(gc2, dominantType)
			if err != nil {
				return nil, err
			}
		}
		if l, ok := f.(geom.LineString); ok {
			o = append(o, l)
		}
		if l, ok := f.(geom.MultiLineString); ok {
			o = append(o, l...)
		}
	}
	if len(o) > 0 {
		return o, nil
	}
	return nil, nil
}

func osmGeometry(g geom.Geom, dominantType osm.GeomType) (geom.Geom, error) {
	if gc, ok := g.(geom.GeometryCollection); ok {
		switch dominantType { // Drop features that do not match the dominant type.
		case osm.Point:
			return geomCollectionToMultiPoint(gc, dominantType)
		case osm.Poly:
			return geomCollectionToMultiPolygon(gc, dominantType)
		case osm.Line:
			return geomCollectionToMultiLineString(gc, dominantType)
		default:
			return nil, fmt.Errorf("invalid geometry type %v", dominantType)
		}
	}
	switch dominantType { // Drop features that do not match the dominant type.
	case osm.Point:
		if _, ok := g.(geom.Point); ok {
			return g, nil
		}
	case osm.Poly:
		if _, ok := g.(geom.Polygonal); ok {
			return g, nil
		}
	case osm.Line:
		if _, ok := g.(geom.Linear); ok {
			return g, nil
		}
	default:
		return nil, fmt.Errorf("invalid geometry type %v", dominantType)
	}
	return nil, nil
}
