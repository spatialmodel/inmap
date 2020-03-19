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
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
	"github.com/ctessum/unit/badunit"
)

const (
	nullVal     = "-9"
	commentRune = '#'
)

func init() {
	gob.Register(geom.Polygon{})
	gob.Register(geom.Point{})
	gob.Register(geom.LineString{})
}

type Location struct {
	geom.Geom
	SR   *proj.SR
	Name string
}

func (l *Location) String() string {
	if l.Name == "" {
		panic("location must have name")
	}
	return l.Name
}

func (l *Location) Reproject(sr *proj.SR) (geom.Geom, error) {
	ct, err := l.SR.NewTransform(sr)
	if err != nil {
		return nil, err
	}
	return l.Geom.Transform(ct)
}

// A Record holds data from a parsed emissions inventory record.
type Record interface {
	// Key returns a unique identifier for this record.
	Key() string

	// Location returns the polygon representing the location of emissions.
	Location() *Location

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
}

// RecordElevated describes emissions that are released from above ground.
type RecordElevated interface {
	Record

	// StackParameters describes the parameters of the emissions release
	// from a elevated stack.
	StackParameters() (StackHeight, StackDiameter, StackTemp, StackFlow, StackVelocity *unit.Unit)

	// GroundLevel returns true if the receiver emissions are
	// at ground level and false if they are elevated.
	GroundLevel() bool
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
	SourceDataLocation
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
	SourceDataLocation
	ControlData
	Emissions
}

// PointData exists to fulfill the Record interface but always returns
// nil because this is not a point source.
func (r *nobusinessPolygonRecord) PointData() *PointSourceData { return nil }

// nocontrolPolygonRecord is a polygon record without any control information.
type nocontrolPolygonRecord struct {
	SourceDataLocation
	EconomicData
	Emissions
}

// basicPolygonRecord is a basic polygon record information.
type basicPolygonRecord struct {
	geom.Polygonal
	SR *proj.SR
	SourceData
	Emissions
	LocationName string
}

// PointData exists to fulfill the Record interface but always returns
// nil because this is not a point source.
func (r *basicPolygonRecord) PointData() *PointSourceData { return nil }

// Location returns the polygon representing the location of emissions.
func (r *basicPolygonRecord) Location() *Location {
	return &Location{Geom: r.Polygonal, SR: r.SR, Name: r.LocationName}
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

	sourceDataLocator *sourceDataLocator

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

// ParseInputUnits parses a string representation of an input unit
// type. Currently supported options are "tons", "tonnes", "kg", "lbs", and "g".
func ParseInputUnits(units string) (InputUnits, error) {
	switch units {
	case "tons":
		return Ton, nil
	case "tonnes":
		return Tonne, nil
	case "kg":
		return Kg, nil
	case "lbs":
		return Lb, nil
	case "g":
		return G, nil
	default:
		return -1, fmt.Errorf("aep.ParseInputUnits: invalid input units '%s'", units)
	}
}

// Conversion returns a function that converts a value to units of kilograms.
// factor reprents an additional factor the value should be multiplied by.
func (u InputUnits) Conversion(factor float64) func(v float64) *unit.Unit {
	switch u {
	case Ton:
		return func(v float64) *unit.Unit { return badunit.Ton(v * factor) }
	case Tonne:
		return func(v float64) *unit.Unit { return unit.New(v/1000.*factor, unit.Kilogram) }
	case Kg:
		return func(v float64) *unit.Unit { return unit.New(v*factor, unit.Kilogram) }
	case G:
		return func(v float64) *unit.Unit { return unit.New(v/1000.*factor, unit.Kilogram) }
	case Lb:
		return func(v float64) *unit.Unit { return badunit.Pound(v * factor) }
	default:
		panic(fmt.Errorf("aep.NewEmissionsReader: unknown value %d"+
			" for variable InputUnits. Acceptable values are Ton, "+
			"Tonne, Kg, G, and Lb.", u))
	}
}

// NewEmissionsReader creates a new EmissionsReader. polsToKeep specifies which
// pollutants from the inventory to keep. If it is nil, all pollutants are kept.
// InputUnits is the units of input data. Acceptable values are `tons',
// `tonnes', `kg', `g', and `lbs'.
// gr and sp are used to reference emissions records to geographic locations;
// if they are both nil, the location referencing is skipped.
func NewEmissionsReader(polsToKeep Speciation, freq InventoryFrequency, InputUnits InputUnits, gr *GridRef, sp *SrgSpecs) (*EmissionsReader, error) {
	e := new(EmissionsReader)
	e.polsToKeep = polsToKeep
	e.freq = freq
	e.sourceDataLocator = newSourceDataLocator(gr, sp)
	var monthlyConv = 1.
	if freq == Monthly {
		// If the freqency is monthly, the emissions need to be divided by 12 because
		// they are represented as annual emissions in the files but they only happen
		// for 1/12 of the year.
		monthlyConv = 1. / 12.
	}
	e.inputConv = InputUnits.Conversion(monthlyConv)
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
				return nil, fmt.Errorf("aep: opening emissions file: %v", err)
			}
			files[i], err = NewInventoryFile(file, f, p, e.inputConv)
			if err != nil {
				return nil, err
			}
			files[i].group = e.Group
		}
		return files, nil
	}
	if e.freq == Annually {
		file := os.ExpandEnv(filetemplate)
		f, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("aep: opening emissions file: %v", err)
		}
		invF, err := NewInventoryFile(file, f, Annual, e.inputConv)
		if err != nil {
			return nil, err
		}
		invF.group = e.Group
		return []*InventoryFile{invF}, nil
	}
	panic(fmt.Errorf("unsupported inventory frequency '%v'", e.freq))
}

// RecFilter is a class of functions that return true if a record should be kept
// and processed.
type RecFilter func(Record) bool

type sourceDataLocationer interface {
	getSourceDataLocation() *SourceDataLocation
}

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
	recordList := make([]Record, 0, len(records))
	for _, r := range records {
		if ar, ok := r.(sourceDataLocationer); ok {
			if err := e.sourceDataLocator.Locate(ar.getSourceDataLocation()); err != nil {
				log.Println(err)
				continue // Drop records we can't find a location for.
				// return nil, nil, err
			}
		}
		recordList = append(recordList, r)
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
			err = fmt.Errorf("aep.InventoryFile.parseLines: file: %s\nerr: %v", f.Name(), err)
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
		droppedTotals := record.GetEmissions().DropPols(e.polsToKeep)
		for pol, val := range droppedTotals {
			if _, ok := f.droppedTotals[pol]; !ok {
				f.droppedTotals[pol] = val
			} else {
				f.droppedTotals[pol].Add(val)
			}
		}
		totals := record.Totals()
		for pol, val := range totals {
			if _, ok := f.totals[pol]; !ok {
				f.totals[pol] = val
			} else {
				f.totals[pol].Add(val)
			}
		}
		recordChan <- recordErr{rec: record, err: nil}
	}
	wg.Done()
}
