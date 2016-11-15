/*
Copyright (C) 2012-2014 Regents of the University of Minnesota.
This file is part of AEP.

AEP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

AEP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with AEP.  If not, see <http://www.gnu.org/licenses/>.
*/

package aep

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ctessum/unit"
	"github.com/ctessum/unit/badunit"

	"bitbucket.org/ctessum/sparse"
)

const (
	nullVal       = "-9"
	commentRune   = '#'
	commentString = "#"
	endLineRune   = '\n'
)

// A Record holds data from a parsed emissions inventory record. Two types
// that implement this interface are PointRecord and PolygonRecord.
type Record interface {
	// GetSCC returns the SCC associated with this record.
	GetSCC() string

	// GetFIPS returns the FIPS associated with this record.
	GetFIPS() string

	// GetCountry returns the Country associated with this record.
	GetCountry() Country

	// GetEmissions returns the emissions associated with this record
	GetEmissions() *Emissions

	// CombineEmissions combines emissions from r2 into this record.
	CombineEmissions(r2 Record)

	// Totals returns the total emissions in units of grams.
	Totals() map[Pollutant]*unit.Unit

	// PeriodTotals returns the total emissions from this emissions source between
	// the times begin and end.
	PeriodTotals(begin, end time.Time) map[Pollutant]*unit.Unit

	// DropPols removes the pollutants that are not in polsToKeep
	// and returns the total emissions removed, in units of grams.
	// If polsToKeep is nil, all pollutants are kept.
	DropPols(polsToKeep map[string]*PolHolder) map[Pollutant]*unit.Unit

	// Spatialize takes a spatial processor (sp) and a grid index number (gi) and
	// returns a gridded spatial surrogate (gridSrg) for an emissions source,
	// as well as whether the emissions source is completely covered by the grid
	// (coveredByGrid) and whether it is in the grid all all (inGrid).
	Spatialize(sp *SpatialProcessor, gi int) (gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error)

	// GetSourceData returns the source information associated with this record.
	GetSourceData() *SourceData

	// Key returns a unique identifier for this record.
	Key() string
}

// PointSource is an emissions record of a point source.
type PointSource interface {
	// PointData returns the data specific to point sources.
	PointData() *PointSourceData
}

// EconomicRecord is any record that contains economic information.
type EconomicRecord interface {
	// GetEconomicData returns the economic information associated with this record.
	GetEconomicData() *EconomicData
}

// PointRecord holds information about an emissions source that has a point
// location.
type PointRecord struct {
	SourceData
	PointSourceData
	EconomicData
	ControlData
	Emissions
}

// Key returns a unique key for this record.
func (r *PointRecord) Key() string {
	return r.SourceData.Key() + r.PointSourceData.Key()
}

// PolygonRecord holds information about an emissions source that has an area
// (i.e., polygon) location. The polygon is designated by the SourceData.FIPS code.
type PolygonRecord struct {
	SourceData
	EconomicData
	ControlData
	Emissions
}

// nobusinessPolygonRecord is a nonpoint record that does not have any
// economic information.
type nobusinessPolygonRecord struct {
	SourceData
	ControlData
	Emissions
}

// nocontrolPolygonRecord is a polygon record without any control information.
type nocontrolPolygonRecord struct {
	SourceData
	EconomicData
	Emissions
}

type supplementalPointRecord struct {
	SourceData
	PointSourceData
	Emissions
}

// Key returns a unique key for this record.
func (r *supplementalPointRecord) Key() string {
	return r.SourceData.Key() + r.PointSourceData.Key()
}

// The ParsedRecord type is a container for all of the needed
// information in the FF10, ORL, and IDA files.
// See the SMOKE manual for file format specification.
// The struct tags indicate the location of the record in each
// input file type.
type ParsedRecord struct {
	// Five digit FIPS code for state and county (required)
	FIPS string `pointorl:"0" areaorl:"0" nonroadorl:"0" mobileorl:"0" pointida:"0:5" areaida:"0:5" mobileida:"0:5" pointff10:"1" areaff10:"1"`

	// Plant Identification Code (15 characters maximum) (required,
	// this is the same as the State Facility Identifier in the NIF)
	PLANTID string `pointorl:"1" pointida:"5:20" pointff10:"3"`

	// Point Identification Code (15 characters maximum) (required,
	// this is the same as the Emission Unit ID in the NIF)
	POINTID string `pointorl:"2" pointida:"20:35" pointff10:"4"`

	// Stack Identification Code (15 characters maximum) (recommended,
	// this is the same as the Emissions Release Point ID in the NIF)
	STACKID string `pointorl:"3" pointida:"35:47" pointff10:"5"`

	// DOE Plant ID (15 characters maximum) (recommended, this is the
	// same as the Process ID in the NIF)
	SEGMENT string `pointorl:"4" pointida:"59:61" pointff10:"6"`

	// Plant Name (40 characters maximum) (recommended)
	PLANT string `pointorl:"5" pointida:"61:101" pointff10:"15"`

	// Ten character Source Classification Code (required)
	SCC string `pointorl:"6" areaorl:"1" nonroadorl:"1" mobileorl:"1" pointida:"101:111" areaida:"5:15" mobileida:"15:25" pointff10:"11" areaff10:"5"`

	// Source type (2 characters maximum), used by SMOKE in determining
	// applicable MACT-based controls (required)
	// 	01 = major
	// 	02 = Section 12 area source
	// 	03 = nonroad
	// 	04 = onroad
	SRCTYPE string `pointorl:"8" areaorl:"4" nonroadorl:"8" mobileorl:"5" pointff10:"16"`

	// Stack Height (ft) (required)
	STKHGT float64 `pointorl:"9" pointida:"119:123" pointff10:"17"`

	// Stack Diameter (ft) (required)
	STKDIAM float64 `pointorl:"10" pointida:"123:129" pointff10:"18"`

	// Stack Gas Exit Temperature (°F) (required)
	STKTEMP float64 `pointorl:"11" pointida:"129:133" pointff10:"19"`

	// Stack Gas Flow Rate (ft3/sec) (optional)
	STKFLOW float64 `pointorl:"12" pointida:"133:143" pointff10:"20"`

	// Stack Gas Exit Velocity (ft/sec) (required)
	STKVEL float64 `pointorl:"13" pointida:"143:152" pointff10:"21"`

	// Standard Industrial Classification Code (recommended)
	SIC string `pointorl:"14" areaorl:"2" pointida:"226:230"`

	// Maximum Available Control Technology Code
	// (6 characters maximum) (optional)
	MACT string `pointorl:"15" areaorl:"3"`

	// North American Industrial Classification System Code
	// (6 characters maximum) (optional)
	NAICS string `pointorl:"16" areaorl:"5" pointff10:"22"`

	// Coordinate system type (1 character maximum) (required)
	// U = Universal Transverse Mercator
	// L = Latitude/longitude
	CTYPE string `pointorl:"17" default:"L"`

	// X location (required)
	// If CTYPE = U, Easting value (meters)
	// If CTYPE = L, Longitude (decimal degrees)
	XLOC float64 `pointorl:"18" pointida:"239:248" pointff10:"23"`

	// Y location (required)
	// If CTYPE = U, Northing value (meters)
	// If CTYPE = L, Latitude (decimal degrees)
	YLOC float64 `pointorl:"19" pointida:"230:239" pointff10:"24"`

	//	UTM zone (required if CTYPE = U)
	UTMZ int `pointorl:"20"`

	// ANN_EMIS is Annual Emissions (tons/year) (required)
	// Emissions values must be positive because negative numbers are used
	// to represent missing data.
	// In the struct tags, there are three numbers. for ORL records,
	// the first number is
	// the pollutant location, the second number is the annual emissions
	// location, and the third number is the average day emissions location.
	// For IDA records, the first number is the start of the first pollutant,
	// and the second two numbers are offsets for the ends of the annual and
	// average day emissions fields.
	// For FF10 records, the first number is the location of the pollutant, the
	// second number is the location of the annual emissions, and
	// the third number (followed by "...") is the location of January emissions.
	ANN_EMIS map[Period]map[string]*SpecValUnits `pointorl:"21,22,23" areaorl:"6,7,8" nonroadorl:"2,3,4" mobileorl:"2,3,4" pointida:"249:13:26" areaida:"15:10:20" mobileida:"25:10:20" pointff10:"12,13,52..." areaff10:"7,8,20..."`

	// Control efficiency percentage (give value of 0-100) (recommended,
	// if left blank, SMOKE default is 0).
	// This can have different values for different pollutants.
	CEFF map[string]float64 `pointorl:"24" areaorl:"9" nonroadorl:"5" pointida:"249:26:33" areaida:"15:31:38" mobileida:"25:20:20"`

	// Rule Effectiveness percentage (give value of 0-100) (recommended,
	// if left blank, SMOKE default is 100)
	// This can have different values for different pollutants.
	REFF map[string]float64 `pointorl:"25" areaorl:"10" nonroadorl:"6" pointida:"249:26:33" areaida:"15:38:41" mobileida:"25:20:20" default:"100"`

	// Rule Penetration percentage (give value of 0-100) (recommended,
	// if left blank, SMOKE default is 100)
	// This can have different values for different pollutants.
	RPEN map[string]float64 `areaorl:"11" nonroadorl:"7" pointida:"249:33:40" areaida:"14:41:47" mobileida:"25:20:20" default:"100"`

	//DOE Plant ID (generally recommended, and required if matching
	// to hour-specific CEM data)
	ORIS_FACILITY_CODE string `pointorl:"29" pointida:"47:53" pointff10:"41"`

	// Boiler Identification Code (recommended)
	ORIS_BOILER_ID string `pointorl:"30" pointida:"53:59" pointff10:"42"`

	PointXcoord float64 // Projected coordinate for point sources
	PointYcoord float64 // Projected coordinate for point sources

	// Pols that should not be included in the speciation of this record
	// to avoid double counting
	DoubleCountPols []string

	// The country that this record applies to.
	// TODO: Acount for the fact that FF10 files can have record-specific countries.
	Country Country

	// Surrogate to apply emissions to grid cells
	GridSrgs []*sparse.SparseArray

	// inGrid specifies whether the record is at least partially within
	// each grid.
	inGrid []bool
	// coveredByGrid specifies whether the record is completely covered by each
	// grid.
	coveredByGrid []bool

	err error

	inputConv float64 // Conversion from input units to grams.

}

// TODO: This is obsolete and should be deleted.
func cleanPol(pol string) (polname string) {
	pol = trimString(pol)
	if strings.Index(pol, "__") != -1 {
		polname = strings.Split(pol, "__")[1]
	} else {
		polname = pol
	}
	return
}

//TODO: delete this.
func (r *ParsedRecord) GriddedEmissions(sp *SpatialProcessor, gi int, p Period) (map[string]*sparse.SparseArray, map[string]string, error) {
	panic("this code is out-of-date and should not be used")
}

// SpecValUnits holds emissions species type, value, and units information.
// TODO: Delete this?
type SpecValUnits struct {
	Val     float64
	Units   string
	PolType *PolHolder
}

// stringToFloat converts a string to a floating point number.
// If the string is "" or "-9" it returns 0.
func stringToFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == nullVal {
		return 0, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err
}

// Get rid of extra quotation marks and copy the string so the
// whole line from the input file isn't held in memory
func trimString(s string) string {
	return string([]byte(strings.Trim(s, "\" ")))
}

// InventoryFrequency describes how many often new inventory files are required.
type InventoryFrequency string

// Inventory frequencies can either be annual or monthly
const (
	Annually InventoryFrequency = "annual"
	Monthly  InventoryFrequency = "monthly"
)

// An EmissionsReader reads SMOKE formatted emissions files.
type EmissionsReader struct {
	polsToKeep map[string]*PolHolder
	freq       InventoryFrequency

	inputConv func(float64) *unit.Unit

	// Group specifies a group name for files read by this reader.
	// It is used for report creation
	Group string
}

// InputUnits specify available options for emissions input units.
type InputUnits int

const (
	// Ton is short tons
	Ton InputUnits = iota
	// Tonne is metric tons
	Tonne
	// Kg is kilograms
	Kg
	// G is grams
	G
	// Lb is pounds
	Lb
)

// NewEmissionsReader creates a new EmissionsReader. polsToKeep specifies which
// pollutants from the inventory to keep. If it is nil, all pollutants are kept.
// InputUnits is the units of input data. Acceptable values are `tons',
// `tonnes', `kg', `g', and `lbs'.
func NewEmissionsReader(polsToKeep map[string]*PolHolder, freq InventoryFrequency, InputUnits InputUnits) (*EmissionsReader, error) {
	e := new(EmissionsReader)
	e.polsToKeep = polsToKeep
	e.freq = freq
	var monthlyConv = 1.
	if freq == Monthly {
		// If the freqency is monthly, the emissions need to be divided by 12 because
		// they are represented as annual emissions in the files but they only happen
		// for 1/12 of the year.
		monthlyConv = 1. / 12.
	}
	switch InputUnits {
	case Ton:
		e.inputConv = func(v float64) *unit.Unit { return badunit.Ton(v * monthlyConv) }
	case Tonne:
		e.inputConv = func(v float64) *unit.Unit { return unit.New(v/1000.*monthlyConv, unit.Kilogram) }
	case Kg:
		e.inputConv = func(v float64) *unit.Unit { return unit.New(v*monthlyConv, unit.Kilogram) }
	case G:
		e.inputConv = func(v float64) *unit.Unit { return unit.New(v/1000.*monthlyConv, unit.Kilogram) }
	case Lb:
		e.inputConv = func(v float64) *unit.Unit { return badunit.Pound(v * monthlyConv) }
	default:
		return nil, fmt.Errorf("aep.NewEmissionsReader: unknown value %d"+
			" for variable InputUnits. Acceptable values are Ton, "+
			"Tonne, Kg, G, and Lb.", InputUnits)
	}
	return e, nil
}

// OpenFilesFromTemplate opens the files that match the template.
func (e *EmissionsReader) OpenFilesFromTemplate(filetemplate string) ([]*InventoryFile, error) {
	var files []*InventoryFile
	if e.freq == Monthly {
		files = make([]*InventoryFile, 12)
		for i, p := range []Period{Jan, Feb, Mar, Apr, May, Jun, Jul, Aug, Sep, Oct, Nov, Dec} {
			file := strings.Replace(filetemplate, "[month]", strings.ToLower(p.String()), -1)
			file = os.ExpandEnv(file)
			f, err := os.Open(file)
			if err != nil {
				return nil, err
			}
			files[i], err = NewInventoryFile(file, f, p, e.inputConv)
			if err != nil {
				return nil, err
			}
			files[i].Group = e.Group
		}
		return files, nil
	}
	if e.freq == Annually {
		file := os.ExpandEnv(filetemplate)
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		invF, err := NewInventoryFile(file, f, Annual, e.inputConv)
		if err != nil {
			return nil, err
		}
		invF.Group = e.Group
		return []*InventoryFile{invF}, nil
	}
	panic(fmt.Errorf("unsupported inventory frequency '%v'", e.freq))
}

// RecFilter is a class of functions that return true if a record should be kept
// and processed.
type RecFilter func(Record) bool

// TODO: Double-counting tracking has been removed from here.
// Make sure to add it back somewhere else.

// ReadFiles reads emissions associated with period p from the specified files,
// and returns emissions records and a summary report.
// The specified filenames are only used for reporting. If multiple files have
// data for the same specific facility (for instance, if one file has CAPs
// emissions and the other has HAPs emissions) they need to be processed in this
// function together to avoid double counting in speciation. (If you will
// not be speciating the emissions, then it doesn't matter.) f is an optional
// filter function to determine which records should be kept. If f is nil, all
// records will be kept. The caller is responsible for closing the files.
func (e *EmissionsReader) ReadFiles(files []*InventoryFile, f RecFilter) ([]Record, *InventoryReport, error) {
	report := new(InventoryReport)

	records := make(map[string]Record)

	fileRecordChan := make(chan recordErr)
	var wg sync.WaitGroup
	wg.Add(len(files))
	for _, file := range files {
		go file.parseLines(e, f, fileRecordChan, &wg)
	}
	go func() {
		wg.Wait()
		close(fileRecordChan)
	}()
	for recordErr := range fileRecordChan {
		if recordErr.err != nil {
			return nil, nil, recordErr.err
		}
		record := recordErr.rec
		key := record.Key()
		if rec, ok := records[key]; !ok {
			// We don't yet have a record for this key
			records[key] = record
		} else {
			rec.CombineEmissions(record)
			records[key] = rec
		}
	}
	for _, file := range files {
		report.AddData(file)
	}
	recordList := make([]Record, len(records))
	i := 0
	for _, r := range records {
		recordList[i] = r
		i++
	}
	return recordList, report, nil
}

// splitPol splits a pollutant name from its prefix.
func splitPol(pol string) (polname, prefix string) {
	pol = trimString(pol)
	if strings.Index(pol, "__") != -1 {
		polname = strings.Split(pol, "__")[1]
		prefix = strings.Split(pol, "__")[0]
	} else {
		polname = pol
	}
	return
}

type recordErr struct {
	rec Record
	err error
}

// parseLines parses the lines of a file. f, if non-nil, specifies which
// records should be kept.
func (f *InventoryFile) parseLines(e *EmissionsReader, filter RecFilter, recordChan chan recordErr, wg *sync.WaitGroup) {

	for {
		record, err := f.parseLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			err = fmt.Errorf("aep.InventoryFile.parseLines: file: %s\nerr: %v", f.Name, err)
			recordChan <- recordErr{rec: nil, err: err}
			return
		}

		if record == nil {
			continue
		}

		if filter != nil && !filter(record) {
			continue // Skip this record if it doesn't match our filter.
		}

		// add emissions to totals for report
		droppedTotals := record.DropPols(e.polsToKeep)
		for pol, val := range droppedTotals {
			if _, ok := f.DroppedTotals[pol]; !ok {
				f.DroppedTotals[pol] = val
			} else {
				f.DroppedTotals[pol].Add(val)
			}
		}
		totals := record.Totals()
		for pol, val := range totals {
			if _, ok := f.Totals[pol]; !ok {
				f.Totals[pol] = val
			} else {
				f.Totals[pol].Add(val)
			}
		}
		recordChan <- recordErr{rec: record, err: nil}
	}
	wg.Done()
}
