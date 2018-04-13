/*
Copyright (C) 2012-2014 the InMAP authors.
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
	nullVal     = "-9"
	commentRune = '#'
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
	DropPols(polsToKeep Speciation) map[Pollutant]*unit.Unit

	// Spatialize takes a spatial processor (sp) and a grid index number (gi) and
	// returns a gridded spatial surrogate (gridSrg) for an emissions source,
	// as well as whether the emissions source is completely covered by the grid
	// (coveredByGrid) and whether it is in the grid all all (inGrid).
	Spatialize(sp *SpatialProcessor, gi int) (gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error)

	// GetSourceData returns the source information associated with this record.
	GetSourceData() *SourceData

	// PointData returns the data specific to point sources. If the record is
	// not a point source, it should return nil.
	PointData() *PointSourceData

	// Key returns a unique identifier for this record.
	Key() string
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

// PointData exists to fulfill the Record interface but always returns
// nil because this is not a point source.
func (r *PolygonRecord) PointData() *PointSourceData { return nil }

// nobusinessPolygonRecord is a nonpoint record that does not have any
// economic information.
type nobusinessPolygonRecord struct {
	SourceData
	ControlData
	Emissions
}

// PointData exists to fulfill the Record interface but always returns
// nil because this is not a point source.
func (r *nobusinessPolygonRecord) PointData() *PointSourceData { return nil }

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
	polsToKeep Speciation
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
func NewEmissionsReader(polsToKeep Speciation, freq InventoryFrequency, InputUnits InputUnits) (*EmissionsReader, error) {
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
