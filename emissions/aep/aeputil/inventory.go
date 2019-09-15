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
	"io"
	"os"
	"time"

	"github.com/ctessum/unit"
	"github.com/spatialmodel/inmap/emissions/aep"
)

// InventoryConfig holds emissions inventory configuration information.
type InventoryConfig struct {
	// NEIFiles lists National Emissions Inventory emissions files.
	// The file names can include environment variables.
	// The format is map[sector name][list of files].
	NEIFiles map[string][]string

	// COARDSFiles lists COARDS-compliant NetCDF emission files
	// (NetCDF 4 and greater not supported).
	// Information regarding the COARDS NetCDF conventions are
	// available here: https://ferret.pmel.noaa.gov/Ferret/documentation/coards-netcdf-conventions.
	// The file names can include environment variables.
	// The format is map[sector name][list of files].
	// For COARDS files, the sector name will also be used
	// as the SCC code.
	COARDSFiles map[string][]string

	// COARDSYear specifies the year of emissions for COARDS emissions files.
	// COARDS emissions are assumed to be in units of mass of emissions per year.
	// The year will not be used for NEI emissions files.
	COARDSYear int

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
	// It is only used when SrgSpecType == "SMOKE".
	SrgShapefileDirectory string

	// GridRef specifies the locations of the spatial surrogate gridding
	// reference files used for processing emissions.
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
	srgSpecs, err := readSrgSpec(c.SrgSpec, c.SrgShapefileDirectory, c.SrgSpecType, c.SCCExactMatch, "", 0)
	if err != nil {
		return nil, nil, err
	}

	gridRef, err := readGridRef(c.GridRef, c.SCCExactMatch)
	if err != nil {
		return nil, nil, err
	}

	units, err := aep.ParseInputUnits(c.InputUnits)
	if err != nil {
		return nil, nil, err
	}

	// Read NEI emissions.
	r, err := aep.NewEmissionsReader(c.PolsToKeep, aep.Annually, units, gridRef, srgSpecs)
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

	// Read COARDS files.
	coardsBegin := time.Date(c.COARDSYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	coardsEnd := time.Date(c.COARDSYear+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	for sector, files := range c.COARDSFiles {
		sourceData := aep.SourceData{
			SCC:     sector,
			Country: aep.Global,
			FIPS:    "00000",
		}
		for _, file := range files {
			if c.COARDSYear <= 0 {
				return nil, nil, fmt.Errorf("aeputil: COARDSYear == %d, but must be set to a positive value when COARDS files are present", c.COARDSYear)
			}
			file = os.ExpandEnv(file)
			recordGenerator, err := aep.ReadCOARDSFile(file, coardsBegin, coardsEnd, units, sourceData)
			if err != nil {
				return nil, nil, fmt.Errorf("aeputil: reading COARDS file: %v", err)
			}

			t := &recordTotaler{
				name:  file,
				group: sector,
			}

			for {
				rec, err := recordGenerator()
				if err == io.EOF {
					break
				} else if err != nil {
					return nil, nil, fmt.Errorf("aeputil: reading COARDS file: %v", err)
				}
				t.add(rec)
				records[sector] = append(records[sector], rec)
			}
			inventoryReport.AddData(t)
		}
	}
	return records, inventoryReport, nil
}

// A recordTotaler stores information about records.
type recordTotaler struct {

	// Name is the name of this file. It can be the path to the file or something else.
	name string

	// Group is a label for the group of files this is part of. It is used for reporting.
	group string

	// totals holds the total emissions in this file, disaggregated by pollutant.
	totals map[aep.Pollutant]*unit.Unit

	// droppedTotals holds the total emissions in this file that are not being
	// kept for analysis.
	droppedTotals map[aep.Pollutant]*unit.Unit
}

// Name is the name of this file. It can be the path to the file or something else.
func (f *recordTotaler) Name() string {
	return f.name
}

// Group is a label for the group of files this is part of. It is used for reporting.
func (f *recordTotaler) Group() string {
	return f.group
}

// Totals returns the total emissions in this file, disaggregated by pollutant.
func (f *recordTotaler) Totals() map[aep.Pollutant]*unit.Unit {
	return f.totals
}

// DroppedTotals returns the total emissions in this file that are not being
// kept for analysis.
func (f *recordTotaler) DroppedTotals() map[aep.Pollutant]*unit.Unit {
	return f.droppedTotals
}

func (f *recordTotaler) add(r aep.Record) {
	if f.totals == nil {
		f.totals = make(map[aep.Pollutant]*unit.Unit)
	}
	totals := r.Totals()
	for pol, val := range totals {
		if _, ok := f.totals[pol]; !ok {
			f.totals[pol] = val
		} else {
			f.totals[pol].Add(val)
		}
	}
}
