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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/osm"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
)

// SrgSpecOSM holds OpenStreetMap spatial surrogate specification information.
type SrgSpecOSM struct {
	Region        Country `json:"region"`
	Name          string  `json:"name"`
	Code          string  `json:"code"`
	DataShapefile string  `json:"data_shapefile"`
	DataAttribute string  `json:"data_attribute"`

	OSMFile string `json:"osm_file"`

	Tags map[string][]string `json:"tags"`

	// TagMultipliers are factors by which each of the tags should
	// be multiplied. If empty, all weights are set equal to one.
	TagMultipliers map[string]float64 `json:"tag_multipliers"`

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
}

// ReadSrgSpec reads a OpenStreetMap surrogate specification formated as a
// JSON array of SrgSpecOSM objects.
func ReadSrgSpecOSM(r io.Reader) (*SrgSpecs, error) {
	d := json.NewDecoder(r)
	var o []*SrgSpecOSM
	err := d.Decode(&o)
	if err != nil {
		return nil, err
	}
	srgs := NewSrgSpecs()
	for _, s := range o {
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
func (srg *SrgSpecOSM) dataShapefile() string          { return srg.DataShapefile }
func (srg *SrgSpecOSM) dataAttribute() string          { return srg.DataAttribute }

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
// surrogate definition, where tol is tolerance for geometry simplification.
func (srg *SrgSpecOSM) getSrgData(gridData *GridDef, inputLoc *Location, tol float64) (*rtree.Rtree, error) {
	srg.setStatus(0, "getting surrogate weight data")

	f, err := os.Open(srg.OSMFile)
	if err != nil {
		return nil, fmt.Errorf("aep: opening spatial surrogate OSM file: %v", err)
	}

	srgSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		panic(err)
	}

	srgCT, err := srgSR.NewTransform(gridData.SR)
	if err != nil {
		return nil, err
	}

	gridSrgCT, err := gridData.SR.NewTransform(srgSR)
	if err != nil {
		return nil, err
	}

	// Calculate the area of interest for our surrogate data.
	inputShapeT, err := inputLoc.Reproject(srgSR)
	if err != nil {
		return nil, err
	}
	inputShapeBounds := inputShapeT.Bounds()
	srgBounds := inputShapeBounds.Copy()
	for _, cell := range gridData.Cells {
		cellT, err := cell.Transform(gridSrgCT)
		if err != nil {
			return nil, err
		}
		b := cellT.Bounds()
		if b.Overlaps(inputShapeBounds) {
			srgBounds.Extend(b)
		}
	}

	srgData := rtree.NewTree(25, 50)

	for t, v := range srg.Tags {
		data, err := osm.ExtractTag(f, t, v...)
		if err != nil {
			return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tag `%s:%v`: %v", t, v, err)
		}
		if err := data.Check(); err != nil {
			return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tag `%s:%v`: %v", t, v, err)
		}
		geomTags, err := data.Geom()
		if err != nil {
			return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tag `%s:%v`: %v", t, v, err)
		}
		typ, err := osm.DominantType(geomTags)
		if err != nil {
			return nil, fmt.Errorf("aep: extracting OSM spatial surrogate data for tag `%s:%v`: %v", t, v, err)
		}
		for _, geomTag := range geomTags {
			if !geomTag.Bounds().Overlaps(srgBounds) {
				continue
			}

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
			if tol > 0 {
				switch g.(type) {
				case geom.Simplifier:
					g = g.(geom.Simplifier).Simplify(tol)
				}
			}
			if srg.TagMultipliers != nil {
				if m, ok := srg.TagMultipliers[t]; ok {
					srgData.Insert(&srgHolder{
						Geom:   g,
						Weight: m,
					})
				}
			} else {
				srgData.Insert(&srgHolder{
					Geom:   g,
					Weight: 1,
				})
			}
		}
	}
	return srgData, nil
}
