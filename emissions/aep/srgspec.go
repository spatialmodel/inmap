/*
Copyright (C) 2012-2019 the InMAP authors.
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
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
)

type SrgSpec interface {
	getSrgData(gridData *GridDef, loc *Location, tol float64) (*rtree.Rtree, error)
	backupSurrogateNames() []string
	region() Country
	code() string
	name() string
	dataShapefile() string
	dataAttribute() string
	Status() Status
	mergeNames() []string
	mergeMultipliers() []float64
	setStatus(percent float64, status string)
	incrementStatus(percent float64)
}

// SrgSpecSMOKE holds SMOKE-formatted spatial surrogate specification information.
// See the SMOKE emissions model technical documentation for additional information.
type SrgSpecSMOKE struct {
	Region          Country
	Name            string
	Code            string
	DATASHAPEFILE   string
	DATAATTRIBUTE   string
	WEIGHTSHAPEFILE string
	Details         string

	// BackupSurrogateNames specifies names of surrogates to use if this
	// one doesn't have data for the desired location.
	BackupSurrogateNames []string

	// WeightColumns specify the fields of the surogate shapefile that
	// should be used to weight the output locations.
	WeightColumns []string

	// WeightFactors are factors by which each of the WeightColumns should
	// be multiplied.
	WeightFactors []float64

	// FilterFunction specifies which rows in the surrogate shapefile should
	// be used to create this surrogate.
	FilterFunction *SurrogateFilter

	// MergeNames specify names of other surrogates that should be combined
	// to create this surrogate.
	MergeNames []string
	// MergeMultipliers specifies multipliers associated with the surrogates
	// in MergeNames.
	MergeMultipliers []float64

	// progress specifies the progress in generating the surrogate.
	progress     float64
	progressLock sync.Mutex
	// status specifies what the surrogate generator is currently doing.
	status string
}

// Status returns information about the status of the receiver.
func (srg *SrgSpecSMOKE) Status() Status {
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

func (srg *SrgSpecSMOKE) setStatus(percent float64, status string) {
	srg.progressLock.Lock()
	srg.progress = percent
	srg.status = status
	srg.progressLock.Unlock()
}

func (srg *SrgSpecSMOKE) incrementStatus(percent float64) {
	srg.progressLock.Lock()
	srg.progress += percent
	srg.progressLock.Unlock()
}

const none = "NONE"

// ReadSrgSpecSMOKE reads a SMOKE formatted spatial surrogate specification file.
// Results are returned as a map of surrogate specifications as indexed by
// their unique ID, which is Region+SurrogateCode. shapefileDir specifies the
// location of all the required shapefiles, and checkShapeFiles specifies whether
// to check if the required shapefiles actually exist. If checkShapeFiles is
// true, then it is okay for the shapefiles to be in any subdirectory of
// shapefileDir, otherwise all shapefiles must be in shapefileDir itself and
// not a subdirectory.
func ReadSrgSpecSMOKE(fid io.Reader, shapefileDir string, checkShapefiles bool) (*SrgSpecs, error) {
	srgs := NewSrgSpecs()
	reader := csv.NewReader(fid)
	reader.Comment = '#'
	reader.TrailingComma = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("in ReadSrgSpec: %v", err)
	}
	for i := 1; i < len(records); i++ {
		record := records[i]
		srg := new(SrgSpecSMOKE)
		srg.Region, err = countryFromName(record[0])
		if err != nil {
			return nil, fmt.Errorf("in ReadSrgSpec: %v", err)
		}
		srg.Name = strings.TrimSpace(record[1])
		srg.Code = record[2]
		srg.DATASHAPEFILE = record[3]
		srg.DATAATTRIBUTE = strings.TrimSpace(record[4])
		srg.WEIGHTSHAPEFILE = record[5]
		WEIGHTATTRIBUTE := record[6]
		WEIGHTFUNCTION := record[7]
		FILTERFUNCTION := record[8]
		MERGEFUNCTION := record[9]
		for i := 10; i <= 12; i++ {
			if len(record[i]) != 0 {
				srg.BackupSurrogateNames = append(srg.BackupSurrogateNames, record[i])
			}
		}
		srg.Details = record[13]

		// Parse weight function
		if WEIGHTATTRIBUTE != none && WEIGHTATTRIBUTE != "" {
			srg.WeightColumns = append(srg.WeightColumns,
				strings.TrimSpace(WEIGHTATTRIBUTE))
			srg.WeightFactors = append(srg.WeightFactors, 1.)
		}
		if WEIGHTFUNCTION != "" {
			weightfunction := strings.Split(WEIGHTFUNCTION, "+")
			for _, wf := range weightfunction {
				mulFunc := strings.Split(wf, "*")
				if len(mulFunc) == 1 {
					srg.WeightColumns = append(srg.WeightColumns,
						strings.TrimSpace(mulFunc[0]))
					srg.WeightFactors = append(srg.WeightFactors, 1.)
				} else if len(mulFunc) == 2 {
					v, err2 := strconv.ParseFloat(mulFunc[0], 64)
					if err2 != nil {
						return nil, fmt.Errorf("srgspec weight function: %v", err2)
					}
					srg.WeightColumns = append(srg.WeightColumns,
						strings.TrimSpace(mulFunc[1]))
					srg.WeightFactors = append(srg.WeightFactors, v)
				} else {
					return nil, fmt.Errorf("invalid value %s in srgspec "+
						"weighting function", wf)
				}
			}
		}

		// Parse filter function
		srg.FilterFunction = ParseSurrogateFilter(FILTERFUNCTION)

		// Parse merge function
		if MERGEFUNCTION != none && MERGEFUNCTION != "" {
			s := strings.Split(MERGEFUNCTION, "+")
			for _, s2 := range s {
				s3 := strings.Split(s2, "*")
				srg.MergeNames = append(srg.MergeNames, strings.TrimSpace(s3[1]))
				val, err2 := strconv.ParseFloat(strings.TrimSpace(s3[0]), 64)
				if err2 != nil {
					return nil, err2
				}
				srg.MergeMultipliers = append(srg.MergeMultipliers, val)
			}
		}

		// Set up the shapefile paths and
		// optionally check to make sure the shapefiles exist.
		if checkShapefiles {
			if srg.DATASHAPEFILE != "" {
				srg.DATASHAPEFILE, err = findFile(shapefileDir, srg.DATASHAPEFILE+".shp")
				if err != nil {
					return nil, err
				}
			}
			if srg.WEIGHTSHAPEFILE != "" {
				srg.WEIGHTSHAPEFILE, err = findFile(shapefileDir, srg.WEIGHTSHAPEFILE+".shp")
				if err != nil {
					return nil, err
				}
			}
		} else {
			if srg.DATASHAPEFILE != "" {
				srg.DATASHAPEFILE = filepath.Join(shapefileDir, srg.DATASHAPEFILE+".shp")
			}
			if srg.WEIGHTSHAPEFILE != "" {
				srg.WEIGHTSHAPEFILE = filepath.Join(shapefileDir, srg.WEIGHTSHAPEFILE+".shp")
			}
		}

		if checkShapefiles {
			if srg.DATASHAPEFILE != "" {
				shpf, err := shp.NewDecoder(srg.DATASHAPEFILE)
				if err != nil {
					return nil, err
				}
				shpf.Close()
			}
			if srg.WEIGHTSHAPEFILE != "" {
				shpf, err := shp.NewDecoder(srg.WEIGHTSHAPEFILE)
				if err != nil {
					return nil, err
				}
				shpf.Close()
			}
		}
		srgs.Add(srg)
	}
	return srgs, nil
}

func (srg *SrgSpecSMOKE) backupSurrogateNames() []string { return srg.BackupSurrogateNames }
func (srg *SrgSpecSMOKE) region() Country                { return srg.Region }
func (srg *SrgSpecSMOKE) code() string                   { return srg.Code }
func (srg *SrgSpecSMOKE) name() string                   { return srg.Name }
func (srg *SrgSpecSMOKE) mergeNames() []string           { return srg.MergeNames }
func (srg *SrgSpecSMOKE) mergeMultipliers() []float64    { return srg.MergeMultipliers }
func (srg *SrgSpecSMOKE) dataShapefile() string          { return srg.DATASHAPEFILE }
func (srg *SrgSpecSMOKE) dataAttribute() string          { return srg.DATAATTRIBUTE }

// InputShapes returns the input shapes associated with the receiver.
func (srg *SrgSpecSMOKE) InputShapes() (map[string]*Location, error) {
	inputShp, err := shp.NewDecoder(srg.DATASHAPEFILE)
	if err != nil {
		return nil, err
	}
	defer inputShp.Close()
	inputSR, err := inputShp.SR()
	if err != nil {
		return nil, err
	}
	inputData := make(map[string]*Location)
	for {
		g, fields, more := inputShp.DecodeRowFields(srg.dataAttribute())
		if !more {
			break
		}

		inputID := fields[srg.DATAATTRIBUTE]
		ggeom := g.(geom.Polygon)

		// Extend existing polygon if one already exists for this InputID
		if _, ok := inputData[inputID]; !ok {
			inputData[inputID] = &Location{
				Geom: ggeom,
				SR:   inputSR,
			}
		} else {
			inputData[inputID].Geom = append(inputData[inputID].Geom.(geom.Polygon), ggeom...)
		}
	}
	if inputShp.Error() != nil {
		return nil, fmt.Errorf("in file %s, %v", srg.dataShapefile(), inputShp.Error())
	}
	return inputData, nil
}

// get surrogate shapes and weights. tol is a geometry simplification tolerance.
func (srg *SrgSpecSMOKE) getSrgData(gridData *GridDef, inputLoc *Location, tol float64) (*rtree.Rtree, error) {
	srgShp, err := shp.NewDecoder(srg.WEIGHTSHAPEFILE)
	if err != nil {
		return nil, err
	}
	defer srgShp.Close()

	srgSR, err := srgShp.SR()
	if err != nil {
		return nil, err
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

	var fieldNames []string
	if srg.FilterFunction != nil {
		fieldNames = append(fieldNames, srg.FilterFunction.Column)
	}
	if srg.WeightColumns != nil {
		fieldNames = append(fieldNames, srg.WeightColumns...)
	}
	srgData := rtree.NewTree(25, 50)
	var recGeom geom.Geom
	var data map[string]string
	var keepFeature bool
	var featureVal string
	var size float64
	var more bool
	for {
		recGeom, data, more = srgShp.DecodeRowFields(fieldNames...)
		if !more {
			break
		}

		if !recGeom.Bounds().Overlaps(srgBounds) {
			continue
		}

		if srg.FilterFunction == nil {
			keepFeature = true
		} else {
			// Determine whether this feature should be kept according to
			// the filter function.
			keepFeature = false
			featureVal = strings.TrimSpace(fmt.Sprintf("%v", data[srg.FilterFunction.Column]))
			for _, filterVal := range srg.FilterFunction.Values {
				switch srg.FilterFunction.EqualNotEqual {
				case "NotEqual":
					if featureVal != filterVal {
						keepFeature = true
					}
				default:
					if featureVal == filterVal {
						keepFeature = true
					}
				}
			}
		}
		if keepFeature && recGeom != nil {
			srgH := new(srgHolder)
			srgH.Geom, err = recGeom.Transform(srgCT)
			if err != nil {
				return srgData, err
			}
			if tol > 0 {
				switch srgH.Geom.(type) {
				case geom.Simplifier:
					srgH.Geom = srgH.Geom.(geom.Simplifier).Simplify(tol)
				}
			}
			if len(srg.WeightColumns) != 0 {
				weightval := 0.
				for i, name := range srg.WeightColumns {
					var v float64
					vstring := data[name]
					if strings.Contains(vstring, "\x00\x00\x00\x00\x00\x00") || strings.Contains(vstring, "***") || vstring == "" {
						// null value
						v = 0.
					} else {
						v, err = strconv.ParseFloat(data[name], 64)
						if err != nil {
							return srgData, fmt.Errorf("aep.getSrgData: shapefile %s column %s, %v", srg.WEIGHTSHAPEFILE, name, err)
						}
						v = math.Max(v, 0) // Get rid of any negative weights.
					}
					weightval += v * srg.WeightFactors[i]
				}
				switch srgH.Geom.(type) {
				case geom.Polygonal:
					size = srgH.Geom.(geom.Polygonal).Area()
					if size == 0. {
						if tol > 0 {
							// We probably simplified the shape down to zero area.
							continue
						} else {
							// TODO: Is it okay for input shapes to have zero area? Probably....
							continue
							//err = fmt.Errorf("Area should not equal "+
							//	"zero in %v", srg.WEIGHTSHAPEFILE)
							//return srgData, err
						}
					} else if size < 0 {
						panic(fmt.Errorf("negative area: %g, geom:%#v", size, srgH.Geom))
					}
					srgH.Weight = weightval / size
				case geom.Linear:
					size = srgH.Geom.(geom.Linear).Length()
					if size == 0. {
						err = fmt.Errorf("Length should not equal "+
							"zero in %v", srg.WEIGHTSHAPEFILE)
						return srgData, err
					}
					srgH.Weight = weightval / size
				case geom.Point:
					srgH.Weight = weightval
				default:
					err = fmt.Errorf("aep: in file %s, unsupported geometry type %#v",
						srg.WEIGHTSHAPEFILE, srgH.Geom)
					return srgData, err
				}
			} else {
				srgH.Weight = 1.
			}
			if srgH.Weight < 0. || math.IsInf(srgH.Weight, 0) ||
				math.IsNaN(srgH.Weight) {
				err = fmt.Errorf("Surrogate weight is %v, which is not acceptable.", srgH.Weight)
				return srgData, err
			} else if srgH.Weight != 0. {
				srgData.Insert(srgH)
			}
		}
	}
	if srgShp.Error() != nil {
		return nil, fmt.Errorf("in file %s, %v", srg.WEIGHTSHAPEFILE, srgShp.Error())
	}
	return srgData, nil
}
