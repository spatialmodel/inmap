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
	"strconv"
	"strings"
	"time"

	"github.com/ctessum/unit"
)

func (f *InventoryFile) readHeaderFF10Point(inputConverter func(float64) *unit.Unit) error {
	return f.readff10(inputConverter, NewFF10Point)
}
func (f *InventoryFile) readHeaderFF10DailyPoint(inputConverter func(float64) *unit.Unit) error {
	return f.readff10(inputConverter, NewFF10DailyPoint)
}

func (f *InventoryFile) readHeaderFF10Nonpoint(inputConverter func(float64) *unit.Unit) error {
	return f.readff10(inputConverter, NewFF10Nonpoint)
}
func (f *InventoryFile) readHeaderFF10Onroad(inputConverter func(float64) *unit.Unit) error {
	// Onroad file format is the same as nonpoint.
	return f.readff10(inputConverter, NewFF10Nonpoint)
}
func (f *InventoryFile) readHeaderFF10Nonroad(inputConverter func(float64) *unit.Unit) error {
	// Nonroad file format is the same as nonpoint.
	return f.readff10(inputConverter, NewFF10Nonpoint)
}

type monthPeriod struct {
	begin, end time.Time  // begin and end of the month.
	seconds    *unit.Unit // number of seconds between begin and end.
}

// ff10Periods gets the time periods associated with the 12 months of the
// given year.
func ff10Periods(year string) ([12]*monthPeriod, error) {
	var periods [12]*monthPeriod
	for i, p := range []Period{Jan, Feb, Mar, Apr, May, Jun, Jul,
		Aug, Sep, Oct, Nov, Dec} {

		begin, end, err := p.TimeInterval(year)
		if err != nil {
			return [12]*monthPeriod{}, err
		}
		periods[i] = &monthPeriod{
			begin:   begin,
			end:     end,
			seconds: unit.New(end.Sub(begin).Seconds(), unit.Second),
		}
	}
	return periods, nil
}

func (f *InventoryFile) readff10(inputConverter func(float64) *unit.Unit,
	recFunc func([]string, time.Time, time.Time, [12]*monthPeriod, func(float64) *unit.Unit) (Record, error)) error {

	year, _, err := f.readHeaderGeneral()
	if err != nil {
		return err
	}

	annualBegin, annualEnd, err := Annual.TimeInterval(year)
	if err != nil {
		return err
	}

	periods, err := ff10Periods(year)
	if err != nil {
		return err
	}

	r := csv.NewReader(f.ReadSeeker)
	r.Comment = commentRune

	f.parseLine = func() (Record, error) {
		line, err := r.Read()
		if err != nil {
			return nil, err
		}
		return recFunc(line, annualBegin, annualEnd, periods, inputConverter)
	}
	return nil
}

// NewFF10Point creates a new record from the FF10 point record rec, where
// annualBegin and annualEnd specify the period that annual total emissions
// occur over, 'periods' specifies the periods when the monthly emissions occur
//  and inputConv specifies the factor
// to multiply emissions by to convert them to SI units.
func NewFF10Point(rec []string, annualBegin, annualEnd time.Time,
	periods [12]*monthPeriod, inputConv func(float64) *unit.Unit) (Record, error) {

	if len(rec) != 77 {
		return nil, fmt.Errorf("aep.NewFF10Point: record should have 77 fields but instead has %d", len(rec))
	}

	if strings.Contains(rec[0], "country_cd") {
		// This record is an uncommented header so ignore it.
		return nil, nil
	}

	r := new(PointRecord)

	var err error
	r.Country, err = countryFromName(trimString(rec[0]))
	if err != nil {
		return nil, err
	}
	r.parseFIPS(rec[1])

	r.PointSourceData.PlantID = trimString(rec[3])
	r.PointSourceData.PointID = trimString(rec[4])
	r.PointSourceData.StackID = trimString(rec[5])
	r.PointSourceData.Segment = trimString(rec[6])

	r.parseSCC(rec[11])

	pol, prefix := splitPol(rec[12])
	annualEmisRate, err := parseEmisRateAnnual(rec[13], "", inputConv)
	if err != nil {
		return nil, err
	}

	err = r.setREff(rec[14]) // ANN_PCT_RED in the SMOKE manual.
	if err != nil {
		return nil, err
	}

	r.PointSourceData.Plant = trimString(rec[15])

	err = r.setStackParams(rec[17], rec[18], rec[19], rec[20], rec[21])
	if err != nil {
		return nil, err
	}

	r.parseNAICS(rec[22])

	err = r.setupLocation(rec[23], rec[24], "L", "", rec[25]) // long, lat, ctype, utmz, datum
	if err != nil {
		return nil, err
	}

	r.PointSourceData.ORISFacilityCode = trimString(rec[41])
	r.PointSourceData.ORISBoilerID = trimString(rec[42])

	// Check if any of the months have data present. If any of them do, use the
	// monthly data. Otherwise, use the annual data.
	monthlyEmisPresent := false
	for i := range periods {
		v := rec[52+i]
		if !(v == "" || v == nullVal) {
			monthlyEmisPresent = true
		}
	}

	if monthlyEmisPresent {
		for i, p := range periods {
			emis := rec[52+i]

			monthlyEmisRate, err := parseEmisRate(emis, p.seconds, inputConv)
			if err != nil {
				return nil, err
			}
			r.Emissions.Add(p.begin, p.end, pol, prefix, monthlyEmisRate)
		}
	} else {
		r.Emissions.Add(annualBegin, annualEnd, pol, prefix, annualEmisRate)
	}

	return r, nil
}

// NewFF10DailyPoint creates a new record from the FF10 daily point record rec, where
// periods specify the periods when the emissions occur (the 12
// months) and inputConv specifies the factor
// to multiply emissions by to convert them to SI units.
func NewFF10DailyPoint(rec []string, annualBegin, annualEnd time.Time, periods [12]*monthPeriod, inputConv func(float64) *unit.Unit) (Record, error) {

	if len(rec) != 46 {
		return nil, fmt.Errorf("aep.NewFF10DailyPoint: record should have 46 fields but instead has %d", len(rec))
	}

	if strings.Contains(rec[0], "country_cd") {
		// This record is an uncommented header so ignore it.
		return nil, nil
	}

	r := new(supplementalPointRecord)

	var err error
	r.Country, err = countryFromName(trimString(rec[0]))
	if err != nil {
		return nil, err
	}
	r.parseFIPS(rec[1])

	r.PointSourceData.PlantID = trimString(rec[3])
	r.PointSourceData.PointID = trimString(rec[4])
	r.PointSourceData.StackID = trimString(rec[5])
	r.PointSourceData.Segment = trimString(rec[6])

	r.parseSCC(rec[7])

	pol, prefix := splitPol(rec[8])

	if rec[12] == "" {
		// This record is missing required information so ignore it.
		return nil, nil
	}

	month64, err := strconv.ParseInt(rec[12], 10, 64)
	if err != nil {
		return nil, err
	}
	month := int(month64)
	monthlyEmisRate, err := parseEmisRate(rec[13], periods[month-1].seconds, inputConv)
	if err != nil {
		return nil, err
	}

	// Check if any of the days have data present. If any of them do, use the
	// monthly data. Otherwise, use the annual data.
	dailyEmisPresent := false
	for i := 0; i < 31; i++ {
		v := rec[14+i]
		if !(v == "" || v == nullVal) {
			dailyEmisPresent = true
		}
	}

	monthEnded := false
	if dailyEmisPresent {
		for i := 1; i < 32; i++ {

			if monthEnded { // Month is over.
				break
			}

			emis := rec[13+i]

			begin := time.Date(annualBegin.Year(), time.Month(month), i, 0, 0, 0, 0, time.UTC)
			end := time.Date(annualBegin.Year(), time.Month(month), i+1, 0, 0, 0, 0, time.UTC)
			if end.Month() != begin.Month() {
				monthEnded = true // This is the last day of the month.
			}

			seconds := unit.New(end.Sub(begin).Seconds(), unit.Second)
			dailyEmisRate, err := parseEmisRate(emis, seconds, inputConv)
			if err != nil {
				return nil, err
			}
			r.Emissions.Add(begin, end, pol, prefix, dailyEmisRate)
		}
	} else {
		r.Emissions.Add(periods[month-1].begin, periods[month-1].end, pol, prefix, monthlyEmisRate)
	}

	return r, nil
}

// NewFF10Nonpoint creates a new record from the FF10 nonpoint record rec, where
// annualBegin and annualEnd specify the period that annual total emissions
// occur over, 'periods' specifies the periods when the monthly emissions occur
//  and inputConv specifies the factor
// to multiply emissions by to convert them to SI units.
func NewFF10Nonpoint(rec []string, annualBegin, annualEnd time.Time,
	periods [12]*monthPeriod, inputConv func(float64) *unit.Unit) (Record, error) {

	if len(rec) != 45 {
		return nil, fmt.Errorf("aep.NewFF10Nonpoint: record should have 45 fields but instead has %d", len(rec))
	}

	if rec[0] == "country_cd" {
		// This record is an uncommented header so ignore it.
		return nil, nil
	}

	r := new(nobusinessPolygonRecord)

	var err error
	r.Country, err = countryFromName(trimString(rec[0]))
	if err != nil {
		return nil, err
	}
	r.parseFIPS(rec[1])

	r.parseSCC(rec[5])

	pol, prefix := splitPol(rec[7])
	annualEmisRate, err := parseEmisRateAnnual(rec[8], "", inputConv)
	if err != nil {
		return nil, err
	}

	err = r.setREff(rec[9]) // ANN_PCT_RED in the SMOKE manual.
	if err != nil {
		return nil, err
	}

	// Check if any of the months have data present. If any of them do, use the
	// monthly data. Otherwise, use the annual data.
	monthlyEmisPresent := false
	for i := range periods {
		v := rec[20+i]
		if !(v == "" || v == nullVal) {
			monthlyEmisPresent = true
		}
	}

	if monthlyEmisPresent {
		for i, p := range periods {
			emis := rec[20+i]

			monthlyEmisRate, err := parseEmisRate(emis, p.seconds, inputConv)
			if err != nil {
				return nil, err
			}
			r.Emissions.Add(p.begin, p.end, pol, prefix, monthlyEmisRate)
		}
	} else {
		r.Emissions.Add(annualBegin, annualEnd, pol, prefix, annualEmisRate)
	}
	return r, nil
}
