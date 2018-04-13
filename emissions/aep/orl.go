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

package aep

import (
	"encoding/csv"
	"fmt"
	"time"

	"github.com/ctessum/unit"
)

func (f *InventoryFile) readHeaderORL(inputConverter func(float64) *unit.Unit) error {
	year, country, err := f.readHeaderGeneral()
	if err != nil {
		return err
	}
	begin, end, err := f.Period.TimeInterval(year)
	if err != nil {
		return err
	}
	r := csv.NewReader(f.ReadSeeker)
	r.Comment = '#'

	firstLine, err := r.Read()
	if err != nil {
		return fmt.Errorf("aep: in file %s: %v", f.Name, err)
	}

	var recFunc func([]string, Country, time.Time, time.Time,
		func(float64) *unit.Unit) (Record, error)
	// Infer type of ORL file based on number of records in each line.
	switch len(firstLine) {
	case 70: // point file
		recFunc = NewORLPoint
	case 37: // nonpoint file
		recFunc = NewORLNonpoint
	case 30: // nonroad file
		recFunc = NewORLNonroad
	case 16: // mobile file
		recFunc = NewORLMobile
	default:
		return fmt.Errorf("in aep.readHeaderORL: unsupported number of fields %d", len(firstLine))
	}

	// Rewind the file.
	_, err = f.ReadSeeker.Seek(0, 0)
	if err != nil {
		return err
	}
	r = csv.NewReader(f.ReadSeeker)
	r.Comment = commentRune

	f.parseLine = func() (Record, error) {
		line, err := r.Read()
		if err != nil {
			return nil, err
		}
		return recFunc(line, country, begin, end, inputConverter)
	}
	return nil
}

// NewORLPoint creates a new record from the ORL point record rec, where country is the country
// and year of the emissions, begin and end specify the time period
// this record covers, and inputConv specifies the factor
// to multiply emissions by to convert them to SI units.
func NewORLPoint(rec []string, country Country, begin, end time.Time, inputConv func(float64) *unit.Unit) (Record, error) {
	if len(rec) != 70 {
		return nil, fmt.Errorf("aep.NewORLPoint: record should have 70 fields but instead has %d", len(rec))
	}

	r := new(PointRecord)
	r.SourceData.Country = country
	r.parseFIPS(rec[0])
	r.PointSourceData.PlantID = trimString(rec[1])
	r.PointSourceData.PointID = trimString(rec[2])
	r.PointSourceData.StackID = trimString(rec[3])
	r.PointSourceData.Segment = trimString(rec[4])
	r.PointSourceData.Plant = trimString(rec[5])
	r.parseSCC(rec[6])
	r.SourceData.SourceType = trimString(rec[8])

	err := r.setStackParams(rec[9], rec[10], rec[11], rec[12], rec[13])
	if err != nil {
		return nil, err
	}

	r.parseSIC(rec[14])
	r.ControlData.MACT = trimString(rec[15])
	r.parseNAICS(rec[16])

	ctype, xloc, yloc, utmz := trimString(rec[17]), rec[18], rec[19], rec[20]
	err = r.setupLocation(xloc, yloc, ctype, utmz, "")
	if err != nil {
		return nil, err
	}

	err = r.ControlData.setCEff(rec[24])
	if err != nil {
		return nil, err
	}
	err = r.ControlData.setREff(rec[25])
	if err != nil {
		return nil, err
	}

	r.PointSourceData.ORISFacilityCode = trimString(rec[29])
	r.PointSourceData.ORISBoilerID = trimString(rec[30])

	pol, ann, avd := rec[21], rec[22], rec[23]
	emisRate, err := parseEmisRateAnnual(ann, avd, inputConv)
	if err != nil {
		return nil, err
	}
	pol, prefix := splitPol(pol)

	r.Emissions.Add(begin, end, pol, prefix, emisRate)

	return r, nil
}

// NewORLNonpoint creates a new record from the ORL nonpoint record rec, where country is the country
// and year of the emissions, begin and end specify the time period
// this record covers, and inputConv specifies the factor
// to multiply emissions by to convert them to SI units.
func NewORLNonpoint(rec []string, country Country, begin, end time.Time, inputConv func(float64) *unit.Unit) (Record, error) {
	r := new(PolygonRecord)

	if len(rec) != 37 {
		return nil, fmt.Errorf("aep.NewORLNonpoint: record should have 37 fields but instead has %d", len(rec))
	}

	r.SourceData.Country = country
	r.parseFIPS(rec[0])
	r.parseSCC(rec[1])
	r.parseSIC(rec[2])
	r.ControlData.MACT = trimString(rec[3])
	r.SourceData.SourceType = trimString(rec[4])
	r.parseNAICS(rec[5])

	pol, ann, avd := rec[6], rec[7], rec[8]
	emisRate, err := parseEmisRateAnnual(ann, avd, inputConv)
	if err != nil {
		return nil, err
	}
	pol, prefix := splitPol(pol)
	r.Emissions.Add(begin, end, pol, prefix, emisRate)

	err = r.ControlData.setCEff(rec[9])
	if err != nil {
		return nil, err
	}
	err = r.ControlData.setREff(rec[10])
	if err != nil {
		return nil, err
	}
	err = r.ControlData.setRPen(rec[11])
	if err != nil {
		return nil, err
	}

	return r, nil
}

// NewORLNonroad creates a new record from the ORL nonroad record rec, where country is the country
// and year of the emissions, begin and end specify the time period
// this record covers, inputConv specifies the factor
// to multiply emissions by to convert them to SI units.
func NewORLNonroad(rec []string, country Country, begin, end time.Time, inputConv func(float64) *unit.Unit) (Record, error) {
	r := new(nobusinessPolygonRecord)

	if len(rec) != 30 {
		return nil, fmt.Errorf("aep.NewORLNonroad: record should have 30 fields but instead has %d", len(rec))
	}

	r.SourceData.Country = country
	r.parseFIPS(rec[0])
	r.parseSCC(rec[1])

	pol, ann, avd := rec[2], rec[3], rec[4]
	emisRate, err := parseEmisRateAnnual(ann, avd, inputConv)
	if err != nil {
		return nil, err
	}
	pol, prefix := splitPol(pol)
	r.Emissions.Add(begin, end, pol, prefix, emisRate)

	err = r.ControlData.setCEff(rec[5])
	if err != nil {
		return nil, err
	}
	err = r.ControlData.setREff(rec[6])
	if err != nil {
		return nil, err
	}
	err = r.ControlData.setRPen(rec[7])
	if err != nil {
		return nil, err
	}

	r.SourceData.SourceType = trimString(rec[8])

	return r, nil
}

// NewORLMobile creates a new record from the ORL mobile record rec, where country is the country
// and year of the emissions, begin and end specify the time period
// this record covers, inputConv specifies the factor
// to multiply emissions by to convert them to SI units.
func NewORLMobile(rec []string, country Country, begin, end time.Time, inputConv func(float64) *unit.Unit) (Record, error) {
	r := new(nobusinessPolygonRecord)

	if len(rec) != 16 {
		return nil, fmt.Errorf("aep.NewORLNonroad: record should have 16 fields but instead has %d", len(rec))
	}

	r.SourceData.Country = country
	r.parseFIPS(rec[0])
	r.parseSCC(rec[1])

	pol, ann, avd := rec[2], rec[3], rec[4]
	emisRate, err := parseEmisRateAnnual(ann, avd, inputConv)
	if err != nil {
		return nil, err
	}
	pol, prefix := splitPol(pol)
	r.Emissions.Add(begin, end, pol, prefix, emisRate)

	r.SourceData.SourceType = trimString(rec[5])

	err = r.ControlData.setCEff(rec[9])
	if err != nil {
		return nil, err
	}
	err = r.ControlData.setREff(rec[10])
	if err != nil {
		return nil, err
	}
	err = r.ControlData.setRPen(rec[11])
	if err != nil {
		return nil, err
	}

	return r, nil
}
