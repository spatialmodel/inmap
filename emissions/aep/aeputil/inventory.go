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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package aeputil

import (
	"fmt"
	"os"

	"github.com/spatialmodel/inmap/emissions/aep"
)

// InventoryConfig holds emissions inventory configuration information.
type InventoryConfig struct {
	// NEIFiles lists National Emissions Inventory emissions files to use
	// for making SCC-based spatial surrogates. The file names can include
	// environment variables. The format is map[sector name][list of files].
	NEIFiles map[string][]string

	// PolsToKeep lists pollutants from the NEI that should be kept.
	PolsToKeep aep.Speciation

	// InputUnits specifies the units of input data. Acceptable
	// values are `tons', `tonnes', `kg', `g', and `lbs'.
	InputUnits string

	// SrgSpec gives the location of the surrogate specification file.
	// It is used for assigning spatial locations to emissions records.
	SrgSpec string

	// SrgSpecType specifies the type of data the gridding surrogates
	// are being created from. It can be "SMOKE" or "OSM".
	SrgSpecType string

	// SrgShapefileDirectory gives the location of the directory holding
	// the shapefiles used for creating spatial surrogates.
	// It is used for assigning spatial locations to emissions records.
	SrgShapefileDirectory string

	// GridRef specifies the locations of the spatial surrogate gridding
	// reference files used for processing the NEI.
	// It is used for assigning spatial locations to emissions records.
	GridRef []string

	// SCCExactMatch specifies whether SCC codes must match exactly when processing
	// emissions.
	SCCExactMatch bool

	// FilterFunc specifies which records should be kept.
	// If it is nil, all records are kept.
	FilterFunc aep.RecFilter
}

// ReadEmissions returns emissions records for the files specified
// in the NEIFiles field in the receiver. The returned records are
// split up by sector.
func (c *InventoryConfig) ReadEmissions() (map[string][]aep.Record, *aep.InventoryReport, error) {
	srgSpecs, err := readSrgSpec(c.SrgSpec, c.SrgShapefileDirectory, c.SrgSpecType, c.SCCExactMatch)
	if err != nil {
		return nil, nil, err
	}

	gridRef, err := readGridRef(c.GridRef, c.SCCExactMatch)
	if err != nil {
		return nil, nil, err
	}

	var r *aep.EmissionsReader
	switch c.InputUnits {
	case "tons":
		r, err = aep.NewEmissionsReader(c.PolsToKeep, aep.Annually, aep.Ton, gridRef, srgSpecs)
	case "tonnes":
		r, err = aep.NewEmissionsReader(c.PolsToKeep, aep.Annually, aep.Tonne, gridRef, srgSpecs)
	case "kg":
		r, err = aep.NewEmissionsReader(c.PolsToKeep, aep.Annually, aep.Kg, gridRef, srgSpecs)
	case "lbs":
		r, err = aep.NewEmissionsReader(c.PolsToKeep, aep.Annually, aep.Lb, gridRef, srgSpecs)
	case "g":
		r, err = aep.NewEmissionsReader(c.PolsToKeep, aep.Annually, aep.G, gridRef, srgSpecs)
	default:
		return nil, nil, fmt.Errorf("aeputil.ReadEmissions: invalid input units '%s'", c.InputUnits)
	}
	if err != nil {
		return nil, nil, err
	}

	records := make(map[string][]aep.Record)
	inventoryReport := new(aep.InventoryReport)
	for sector, fileTemplates := range c.NEIFiles {
		r.Group = sector

		var files []*aep.InventoryFile
		for _, filetemplate := range fileTemplates {
			tempFiles, err := r.OpenFilesFromTemplate(filetemplate)
			if err != nil {
				return nil, nil, err
			}
			files = append(files, tempFiles...)
		}

		recs, sectorReport, err := r.ReadFiles(files, c.FilterFunc)
		if err != nil {
			return nil, nil, err
		}
		for _, f := range files { // Close files.
			f.ReadSeeker.(*os.File).Close()
		}
		inventoryReport.AddData(sectorReport.Data...)

		for _, rec := range recs {
			records[sector] = append(records[sector], rec)
		}
	}
	return records, inventoryReport, nil
}
