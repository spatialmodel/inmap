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

	"github.com/ctessum/unit"
)

// An InventoryFile reads information from and stores information about an
// emissions inventory file.
type InventoryFile struct {
	io.ReadSeeker

	// Name is the name of this file. It can be the path to the file or something else.
	Name string

	// Group is a label for the group of files this is part of. It is used for reporting.
	Group string

	// Period is the time period that the emissions in this file apply to.
	Period

	// parseLine reads one record from the file. If the file has ended,
	// the error will be io.EOF.
	parseLine func() (Record, error)

	// Totals holds the total emissions in this file, disaggregated by pollutant.
	Totals map[Pollutant]*unit.Unit

	// DroppedTotals holds the total emissions in this file that are not being
	// kept for analysis.
	DroppedTotals map[Pollutant]*unit.Unit
}

// NewInventoryFile sets up a new InventoryFile with name name and reader
// r. p specifies the time period that the emissions are effective during, and
// inputConverter is a function to convert the input data to SI units.
func NewInventoryFile(name string, r io.ReadSeeker, p Period, inputConverter func(float64) *unit.Unit) (*InventoryFile, error) {
	f := new(InventoryFile)
	f.Name = name
	f.ReadSeeker = r
	f.Period = p

	f.Totals = make(map[Pollutant]*unit.Unit)
	f.DroppedTotals = make(map[Pollutant]*unit.Unit)

	if err := f.readHeader(inputConverter); err != nil {
		return nil, err
	}

	return f, nil
}

func getTotals(f *InventoryFile) map[Pollutant]*unit.Unit {
	return f.Totals
}

func getDroppedTotals(f *InventoryFile) map[Pollutant]*unit.Unit {
	return f.DroppedTotals
}

// readHeader extracts header information from the file and sets up a function
// for reading the records from the file.
func (f *InventoryFile) readHeader(inputConverter func(float64) *unit.Unit) error {
	buf := bufio.NewScanner(f.ReadSeeker)
	buf.Scan()
	firstRec := buf.Text()
	if err := buf.Err(); err != nil {
		return err
	}

	if strings.Contains(firstRec, "ORL") {
		return f.readHeaderORL(inputConverter)
	}
	if strings.Contains(firstRec, "IDA") {
		return f.readHeaderIDA(inputConverter)
	}
	if strings.Contains(firstRec, "FF10_POINT") {
		return f.readHeaderFF10Point(inputConverter)
	}
	if strings.Contains(firstRec, "FF10_DAILY_POINT") {
		return f.readHeaderFF10DailyPoint(inputConverter)
	}
	if strings.Contains(firstRec, "FF10_NONPOINT") {
		return f.readHeaderFF10Nonpoint(inputConverter)
	}
	if strings.Contains(firstRec, "FF10_NONROAD") {
		return f.readHeaderFF10Nonroad(inputConverter)
	}
	if strings.Index(firstRec, "FF10_ONROAD") >= 0 {
		return f.readHeaderFF10Onroad(inputConverter)
	}
	return fmt.Errorf("aep.InventoryFile.readHeader: unknown file type for: '%s'", f.Name)
}

func (f *InventoryFile) readHeaderGeneral() (year string, country Country, err error) {
	f.ReadSeeker.Seek(0, 0)
	buf := bufio.NewScanner(f.ReadSeeker)
	var record string
	for buf.Scan() {
		record = buf.Text()
		if err = buf.Err(); err != nil {
			err = fmt.Errorf("aep.InventoryFile.readHeaderGeneral: in file %s: %v", f.Name, err)
			return
		}
		if len(record) > 0 && record[0] != commentRune {
			break
		}
		if len(record) > 8 && record[1:8] == "COUNTRY" {
			country, err = countryFromName(strings.Trim(record[8:], "\n ="))
			if err != nil {
				err = fmt.Errorf("aep.InventoryFile.readHeaderGeneral: in file %s: %v", f.Name, err)
				return
			}
		}
		if len(record) > 5 && record[1:5] == "YEAR" {
			year = trimString(strings.Trim(record[5:], "\n =\t"))
		}
	}
	f.ReadSeeker.Seek(0, 0)
	return
}
