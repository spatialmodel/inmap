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
	"bytes"
	"math"
	"testing"
	"time"
)

var (
	specRefExample = `#EXPORT_DATE=Tue Dec 22 11:25:43 EST 2015
#EXPORT_VERSION_NAME=add RFL__BENZENE
#EXPORT_VERSION_NUMBER=2
#REV_HISTORY v1(02/08/2009)  Madeleine Strum.   removed all but integrate HAPs  need xrefs for integrate HAPs when running non-multi-pol version of CMAQ-- CMAQ4.7  N1a
#REV_HISTORY v2(02/10/2009)  Madeleine Strum.   added RFL__BENZENE  in nonroad and we will be integrating it
/POINT DEFN/ 4 4
10100202;"95014";"SO2";;;;;;;
10100200;"95015";"SO2";;;;;;;
2280003100;"HONO";"EXH__NOX";;;;;;;! Added for a new invtable with mode-specific NOX
30532003;"91112";"PM2_5";;;;;;;! Profile name: Sand & Gravel - Simplified; Assignment basis: EPA recommendation from Pechan 5jun2006 x-walk
2310011503;"2487";"VOC";48033;;;;;;
2202420000;"8774";"EXH__VOC";211000;;;;;;! Added for Mexico othon 2008
2201001000;"COMBO";"VOC";;;;;;;! Profile name: Combination of base exhaust and evap; Assignment basis: Canada/Mexico SCC set to similar U.S. SCC
2260002009;"COMBO";"EVP__VOC";;;;;;;! Profile name: Uses gspro_combo file; Assignment basis: OTAQ recommendation, v3.1 platform
`

	specRefComboExample = `#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"EVP__NONHAPVOC";01001;0;2;8869;0.1256;8870;0.8744;1
"EVP__VOC";01001;0;2;8869;0.1256;8870;0.8744;1
"NONHAPVOC";01001;0;2;8869;0.1256;8870;0.8744;1
"EVP__NONHAPVOC";01003;0;2;8869;0.1256;8870;0.8744;1
"VOC";01003;0;2;8869;0.1256;8870;0.8744;1
`
)

func TestSpecRef(t *testing.T) {
	r1 := bytes.NewBuffer([]byte(specRefExample))
	r2 := bytes.NewBuffer([]byte(specRefComboExample))

	ref, err := NewSpecRef(r1, r2)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("check specRefCombo sum", func(t *testing.T) {
		for p, periodData := range ref.sRefCombo {
			for pol, polData := range periodData {
				for fips, fipsData := range polData {
					var sum float64
					for _, v := range fipsData.(map[string]float64) {
						sum += v
					}
					if math.Abs(1-sum) > 1e-8 {
						t.Errorf("period %v, pol %s, fips %s sum: have %g, want 1", p, pol, fips, sum)
					}
				}
			}
		}
	})
}

func TestSpecRef_Codes(t *testing.T) {
	specRef := bytes.NewBuffer([]byte(specRefExample))
	specRefCombo := bytes.NewBuffer([]byte(specRefComboExample))

	ref, err := NewSpecRef(specRef, specRefCombo)
	if err != nil {
		t.Fatal(err)
	}

	start, end, err := Annual.TimeInterval("2011")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		SCC          string
		pol          Pollutant
		start, end   time.Time
		c            Country
		FIPS         string
		partialMatch bool
		result       map[string]float64
	}{
		{
			SCC:          "0010100202",
			pol:          Pollutant{Name: "SO2"},
			start:        start,
			end:          end,
			c:            USA,
			FIPS:         "010101",
			partialMatch: false,
			result:       map[string]float64{"95014": 1},
		},
		{
			SCC:          "0010100202",
			pol:          Pollutant{Name: "SO2", Prefix: "EXH"},
			start:        start,
			end:          end,
			c:            USA,
			FIPS:         "010101",
			partialMatch: false,
			result:       map[string]float64{"95014": 1},
		},
		{
			SCC:          "0010100203",
			pol:          Pollutant{Name: "SO2"},
			start:        start,
			end:          end,
			c:            USA,
			FIPS:         "010101",
			partialMatch: true,
			result:       map[string]float64{"95015": 1},
		},
		{
			SCC:          "0010100203",
			pol:          Pollutant{Name: "SO2", Prefix: "EXH"},
			start:        start,
			end:          end,
			c:            USA,
			FIPS:         "010101",
			partialMatch: true,
			result:       map[string]float64{"95015": 1},
		},
		{
			SCC:          "2280003100",
			pol:          Pollutant{Name: "NOX", Prefix: "EXH"},
			start:        start,
			end:          end,
			c:            USA,
			FIPS:         "010101",
			partialMatch: false,
			result:       map[string]float64{"HONO": 1},
		},
		{
			SCC:          "2201001000",
			pol:          Pollutant{Name: "VOC", Prefix: "EVP"},
			start:        start,
			end:          end,
			c:            USA,
			FIPS:         "01003",
			partialMatch: false,
			result:       map[string]float64{"8870": 0.8744, "8869": 0.1256},
		},
		{
			SCC:          "2201001000",
			pol:          Pollutant{Name: "VOC"},
			start:        start,
			end:          end,
			c:            USA,
			FIPS:         "01003",
			partialMatch: false,
			result:       map[string]float64{"8870": 0.8744, "8869": 0.1256},
		},
		{
			SCC:          "2201001001",
			pol:          Pollutant{Name: "VOC", Prefix: "EVP"},
			start:        start,
			end:          end,
			c:            USA,
			FIPS:         "01003",
			partialMatch: true,
			result:       map[string]float64{"8870": 0.8744, "8869": 0.1256},
		},
		{
			SCC:          "2201001001",
			pol:          Pollutant{Name: "VOC"},
			start:        start,
			end:          end,
			c:            USA,
			FIPS:         "01003",
			partialMatch: true,
			result:       map[string]float64{"8870": 0.8744, "8869": 0.1256},
		},
	}
	for _, test := range tests {
		profiles, err := ref.Codes(test.SCC, test.pol, test.start, test.end, test.c, test.FIPS, test.partialMatch)
		if err != nil {
			t.Error(err)
		}
		if mapDifferent(test.result, profiles) {
			t.Errorf("test %+v: want %v, got %v", test, test.result, profiles)
		}
	}
}
