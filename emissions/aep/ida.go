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
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ctessum/unit"
)

// pointRecordIDA holds information about an emissions source that has a point
// location. IDA records have pollutant-specific control information.
type pointRecordIDA struct {
	SourceData
	PointSourceData
	EconomicData
	ControlData map[string]ControlData
	Emissions
}

// Key returns a unique key for this record.
func (r *pointRecordIDA) Key() string {
	return r.SourceData.Key() + r.PointSourceData.Key()
}

// polyonRecordIDA holds information about an emissions source that has a polygon
// location. IDA records have pollutant-specific control information.
type polygonRecordIDA struct {
	SourceData
	ControlData map[string]ControlData
	Emissions
}

// PointData exists to fulfill the Record interface but always returns
// nil because this is not a point source.
func (r *polygonRecordIDA) PointData() *PointSourceData { return nil }

// mobilePolygonRecordIDA holds information about an emissions source that has a polygon
// location and only has source and emissions data.
type mobilePolygonRecordIDA struct {
	SourceData
	Emissions
}

// PointData exists to fulfill the Record interface but always returns
// nil because this is not a point source.
func (r *mobilePolygonRecordIDA) PointData() *PointSourceData { return nil }

func (f *InventoryFile) readHeaderIDA(inputConverter func(float64) *unit.Unit) error {
	year, country, err := f.readHeaderGeneral()
	if err != nil {
		return err
	}
	begin, end, err := f.Period.TimeInterval(year)
	if err != nil {
		return err
	}

	buf := bufio.NewScanner(f.ReadSeeker)

	// get pollutant IDs
	var polids []string
	var record string
	for buf.Scan() {
		record = buf.Text()
		if len(record) > 0 && record[0] != commentRune {
			break
		}
		if len(record) > 6 && record[1:6] == "POLID" && len(polids) == 0 {
			polids = strings.Split(strings.TrimSpace(record[6:]), " ")
		}
		if len(record) > 5 && record[1:5] == "DATA" && len(polids) == 0 {
			polids = strings.Split(strings.TrimSpace(record[5:]), " ")
		}
	}
	if err = buf.Err(); err != nil {
		return fmt.Errorf("aep: in file %s: %v", f.Name, err)
	}

	if len(polids) >= 10 {
		// If there are more than 10 pollutants, there are situations where
		// it can be ambiguous about which file type we are dealing with based
		// on the line length.
		return fmt.Errorf("aep: in file %s: too many pollutants (must be less than 10)", f.Name)
	}
	if err = buf.Err(); err != nil {
		return fmt.Errorf("aep: in file %s: %v", f.Name, err)
	}

	var recFunc func(string, []string, Country, time.Time, time.Time, func(float64) *unit.Unit) (Record, error)
	switch len(record) {
	case 249 + 52*len(polids): // Point record
		recFunc = NewIDAPoint
	case 15 + 47*len(polids): // Area record
		recFunc = NewIDAArea
	case 25 + 20*len(polids): // Mobile record
		recFunc = NewIDAMobile
	default:
		return fmt.Errorf("in aep.readHeaderIDA: unsupported line length %d with %d pollutants", len(record), len(polids))
	}

	// rewind the file
	_, err = f.ReadSeeker.Seek(0, 0)
	if err != nil {
		return err
	}
	buf = bufio.NewScanner(f.ReadSeeker)

	f.parseLine = func() (Record, error) {
		var line string
		var err error
		for buf.Scan() { // loop until we find a non-commented line.
			line = buf.Text()
			if line[0] != commentRune {
				break
			}
		}
		if err = buf.Err(); err != nil {
			return nil, err
		}
		if len(line) == 0 {
			return nil, io.EOF
		}
		return recFunc(line, polids, country, begin, end, inputConverter)
	}
	return nil
}

// NewIDAPoint creates a new record from the IDA point record rec, where
// pollutants are the names of the pollutants in the record, country is the country
// and year of the emissions, begin and end specify the time period
// this record covers, and inputConv specifies the factor
// to multiply emissions by to convert them to SI units.
func NewIDAPoint(rec string, pollutants []string, country Country, begin, end time.Time, inputConv func(float64) *unit.Unit) (Record, error) {
	if len(rec) != 249+52*len(pollutants) {
		return nil, fmt.Errorf("aep.NewIDAPoint: record should have a length of "+
			"%d but instead it is %d", 249+52*len(pollutants), len(rec))
	}

	r := new(pointRecordIDA)
	r.SourceData.Country = country
	r.parseFIPS(rec[0:5])

	r.PointSourceData.PlantID = trimString(rec[5:20])
	r.PointSourceData.PointID = trimString(rec[20:35])
	r.PointSourceData.StackID = trimString(rec[35:47])
	r.PointSourceData.ORISFacilityCode = trimString(rec[47:53])
	r.PointSourceData.ORISBoilerID = trimString(rec[53:59])
	r.PointSourceData.Segment = trimString(rec[59:61])
	r.PointSourceData.Plant = trimString(rec[61:101])

	r.parseSCC(rec[101:111])

	err := r.setStackParams(rec[119:123], rec[123:129], rec[129:133], rec[133:143], rec[143:152])
	if err != nil {
		return nil, err
	}

	r.parseSIC(rec[226:230])

	lat := rec[230:239]
	lon := rec[239:248]
	err = r.setupLocation(lon, lat, "L", "", "")
	if err != nil {
		return nil, err
	}

	r.ControlData = make(map[string]ControlData)
	for i, pol := range pollutants {
		start := 249 + 52*i
		ann, avd := rec[start:start+13], rec[start+13:start+26]
		emisRate, err := parseEmisRateAnnual(ann, avd, inputConv)
		if err != nil {
			return nil, err
		}
		pol, prefix := splitPol(pol)
		r.Emissions.Add(begin, end, pol, prefix, emisRate)

		cd := new(ControlData)
		err = cd.setCEff(rec[start+26 : start+33])
		if err != nil {
			return nil, err
		}
		err = cd.setREff(rec[start+33 : start+36])
		if err != nil {
			return nil, err
		}
		r.ControlData[pol] = *cd
	}

	return r, nil
}

// NewIDAArea creates a new record from the IDA area record rec, where
// pollutants are the names of the pollutants in the record, country is the country
// and year of the emissions, begin and end specify the time period
// this record covers, and inputConv specifies the factor
// to multiply emissions by to convert them to SI units.
func NewIDAArea(rec string, pollutants []string, country Country, begin, end time.Time, inputConv func(float64) *unit.Unit) (Record, error) {

	if len(rec) != 15+47*len(pollutants) {
		return nil, fmt.Errorf("aep.NewIDAArea: record should have a length of "+
			"%d but instead it is %d", 15+47*len(pollutants), len(rec))
	}

	r := new(polygonRecordIDA)
	r.SourceData.Country = country
	r.parseFIPS(rec[0:5])
	r.parseSCC(rec[5:15])

	r.ControlData = make(map[string]ControlData)
	for i, pol := range pollutants {
		start := 15 + 47*i
		ann, avd := rec[start:start+10], rec[start+10:start+20]
		emisRate, err := parseEmisRateAnnual(ann, avd, inputConv)
		if err != nil {
			return nil, err
		}
		pol, prefix := splitPol(pol)
		r.Emissions.Add(begin, end, pol, prefix, emisRate)

		cd := new(ControlData)
		err = cd.setCEff(rec[start+31 : start+38])
		if err != nil {
			return nil, err
		}
		err = cd.setREff(rec[start+38 : start+41])
		if err != nil {
			return nil, err
		}
		err = cd.setRPen(rec[start+41 : start+47])
		if err != nil {
			return nil, err
		}
		r.ControlData[pol] = *cd
	}

	return r, nil
}

// NewIDAMobile creates a new record from the IDA mobile record rec, where
// pollutants are the names of the pollutants in the record, country is the country
// and year of the emissions, begin and end specify the time period
// this record covers, and inputConv specifies the factor
// to multiply emissions by to convert them to SI units.
func NewIDAMobile(rec string, pollutants []string, country Country, begin, end time.Time, inputConv func(float64) *unit.Unit) (Record, error) {
	if len(rec) != 25+20*len(pollutants) {
		return nil, fmt.Errorf("aep.NewORLMobile: record should have a length of "+
			"%d but instead it is %d", 25+20*len(pollutants), len(rec))
	}

	r := new(mobilePolygonRecordIDA)
	r.SourceData.Country = country
	r.parseFIPS(rec[0:5])
	r.parseSCC(rec[15:25])

	for i, pol := range pollutants {
		start := 25 + 20*i
		ann, avd := rec[start:start+10], rec[start+10:start+20]
		emisRate, err := parseEmisRateAnnual(ann, avd, inputConv)
		if err != nil {
			return nil, err
		}
		pol, prefix := splitPol(pol)
		r.Emissions.Add(begin, end, pol, prefix, emisRate)
	}

	return r, nil
}
