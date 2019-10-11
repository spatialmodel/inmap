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
	"os"
	"sync"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/osm"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/requestcache"
)

// SrgSpecOSM holds OpenStreetMap spatial surrogate specification information.
type SrgSpecOSM struct {
	Region Country `json:"region"`
	Name   string  `json:"name"`
	Code   string  `json:"code"`

	OSMFile string `json:"osm_file"`

	Tags map[string][]string `json:"tags"`

	// TagMultiplier is a factors by which the tags should
	// be multiplied. If it is zero it will automatically be set to one.
	TagMultiplier float64 `json:"tag_multiplier"`

	// BackupSurrogateNames specifies names of surrogates to use if this
	// one doesn't have data for the desired location.
	BackupSurrogateNames []string `json:"backup_surrogate_names"`

	// MergeNames specify names of other surrogates that should be combined
	// to create this surrogate.
	MergeNames []string `json:"merge_names"`
	// MergeMultipliers specifies multipliers associated with the surrogates
	// in MergeNames.
	MergeMultipliers []float64 `json:"merge_multipliers"`

	// progress specifies the progress in generating the surrogate.
	progress     float64
	progressLock sync.Mutex
	// status specifies what the surrogate generator is currently doing.
	status string

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
	for _, s := range o {
		s.cache = newCache(s.readSrgData, diskCachePath, memCacheSize)
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

// Status returns information about the status of the receiver.
func (srg *SrgSpecOSM) Status() Status {
	srg.progressLock.Lock()
	o := Status{
		Name:     srg.Name,
		Code:     srg.Code,
		Status:   srg.status,
		Progress: srg.progress,
	}
	srg.progressLock.Unlock()
	return o
}

func (srg *SrgSpecOSM) setStatus(percent float64, status string) {
	srg.progressLock.Lock()
	srg.progress = percent
	srg.status = status
	srg.progressLock.Unlock()
}

func (srg *SrgSpecOSM) incrementStatus(percent float64) {
	srg.progressLock.Lock()
	srg.progress += percent
	srg.progressLock.Unlock()
}

// getSrgData returns the spatial surrogate information for this
// surrogate definition and location, where tol is tolerance for geometry simplification.
func (srg *SrgSpecOSM) getSrgData(gridData *GridDef, inputLoc *Location, tol float64) (*rtree.Rtree, error) {
	srg.setStatus(0, "getting surrogate weight data")

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

	key := fmt.Sprintf("osm_srgdata_%s_%g", gridData.Name, tol)
	request := srg.cache.NewRequest(context.TODO(), &osmReadSrgDataInput{gridData: gridData, tol: tol}, key)
	srgs, err := request.Result()
	if err != nil {
		return nil, err
	}

	srgData := rtree.NewTree(25, 50)
	for _, s := range srgs.([]*srgHolder) {
		if s.Bounds().Overlaps(srgBounds) {
			srgData.Insert(s)
		}
	}

	return srgData, nil
}

type osmReadSrgDataInput struct {
	gridData *GridDef
	tol      float64
}

// readSrgData returns all of the spatial surrogate information for this
// surrogate definition, inputI is of type *osmReadSrgDataInput and
// inputI.tol is tolerance for geometry simplification.
func (srg *SrgSpecOSM) readSrgData(ctx context.Context, inputI interface{}) (interface{}, error) {
	input := inputI.(*osmReadSrgDataInput)

	srgSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		panic(err)
	}

	srgCT, err := srgSR.NewTransform(input.gridData.SR)
	if err != nil {
		return nil, err
	}

	data, err := osm.ExtractFile(context.Background(), os.ExpandEnv(srg.OSMFile), osm.KeepTags(srg.Tags))
	if err != nil {
		return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tags %v: %v", srg.Tags, err)
	}
	if err := data.Check(); err != nil {
		return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tags %v: %v", srg.Tags, err)
	}
	geomTags, err := data.Geom()
	if err != nil {
		return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tags %v: %v", srg.Tags, err)
	}
	typ, err := osm.DominantType(geomTags)
	if err != nil {
		return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tags %v: %v", srg.Tags, err)
	}
	var srgs []*srgHolder
	for _, geomTag := range geomTags {
		switch typ { // Drop features that do not match the dominant type.
		case osm.Point:
			if _, ok := geomTag.Geom.(geom.Point); !ok {
				continue
			}
		case osm.Poly:
			if _, ok := geomTag.Geom.(geom.Polygonal); !ok {
				continue
			}
		case osm.Line:
			if _, ok := geomTag.Geom.(geom.Linear); !ok {
				continue
			}
		default:
			continue // Drop collection-type features.
		}

		g, err := geomTag.Geom.Transform(srgCT)
		if err != nil {
			return nil, fmt.Errorf("aep: processing OSM spatial surrogate data: %v", err)
		}
		if input.tol > 0 {
			switch gs := g.(type) {
			case geom.Simplifier:
				g = gs.Simplify(input.tol)
			}
		}
		var srgData *srgHolder
		if srg.TagMultiplier != 0 {
			srgData = &srgHolder{
				Geom:   g,
				Weight: srg.TagMultiplier,
			}
		} else {
			srgData = &srgHolder{
				Geom:   g,
				Weight: 1,
			}
		}
		srgs = append(srgs, srgData)
	}
	return srgs, nil
}
