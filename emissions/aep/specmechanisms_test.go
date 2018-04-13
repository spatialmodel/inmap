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
	"testing"
)

var (
	mechAssignmentExample = `SAPRC99,1,ARO1,1.000E+0
SAPRC99,2,ALK5,1.000E+0
SAPRC99,3,ARO1,1.000E+0
SAPRC99,4,NROG,1.334E+2
SAPRC99,5,NROG,1.678E+2
SAPRC99,6,ALK5,1.000E+0
SAPRC99,7,ALK1,1.000E+0
SAPRC99,8,ALK5,1.000E+0
SAPRC99,9,ALK4,1.000E+0
SAPRC99,10,ALK5,1.000E+0
SAPRC99,11,ALK5,1.000E+0
SAPRC99,12,ALK5,1.000E+0
SAPRC99,13,ALK4,1.000E+0
SAPRC99,14,ALK5,1.000E+0
SAPRC99,15,NROG,1.169E+2
SAPRC99,16,ALK1,1.000E+0
SAPRC99,17,ALK5,1.000E+0
SAPRC99,18,ALK5,1.000E+0
SAPRC99,19,ALK5,1.000E+0
SAPRC99,20,ALK4,1.000E+0
SAPRC99,66,ALK3,2.247E-3
SAPRC99,66,ALK4,7.103E-2
SAPRC99,66,ALK5,4.286E-1
SAPRC99,592,ALK3,1.000E+0
SAPRC99,605,ALK4,1.000E+0
SAPRC99,2284,ALK3,2.247E-3
SAPRC99,2284,ALK4,7.103E-2
SAPRC99,2284,ALK5,4.286E-1
SAPRC99,2284,OLE1,7.933E-2
SAPRC99,2284,OLE2,3.374E-2
SAPRC99,2284,TRP1,3.920E-2
SAPRC99,2284,ARO1,6.531E-2
SAPRC99,2284,ARO2,7.409E-2
SAPRC99,2284,PROD2,4.273E-2
SAPRC99,2284,CRES,5.059E-3
SAPRC99,2284,MVK,1.172E-3
SAPRC99,2284,RCO_OH,2.249E-2
SAPRC99,2284,NROG,5.988E-2
SAPRC99,671,ALK2,1.000E+0
`

	mechMWExample = `SAPRC99,mol,CH4,16.04
SAPRC99,mol,ALK1,30.07
SAPRC99,mol,ALK2,36.73
SAPRC99,mol,ALK3,58.61
SAPRC99,mol,ALK4,77.60
SAPRC99,mol,ALK5,118.89
SAPRC99,mol,ETHENE,28.05
SAPRC99,mol,OLE1,72.34
SAPRC99,mol,OLE2,75.78
SAPRC99,mol,ISOPRENE,68.12
SAPRC99,mol,TRP1,136.24
SAPRC99,mol,SESQ,204.35
SAPRC99,mol,BENZENE,78.11
SAPRC99,mol,ARO1,95.16
SAPRC99,mol,ARO2,118.72
SAPRC99,mol,HCHO,30.03
SAPRC99,mol,CCHO,44.05
SAPRC99,mol,RCHO,58.08
SAPRC99,mol,BALD,106.13
SAPRC99,mol,ACET,58.08
`

	mechSpeciesInfoExample = `1,"(1-methylpropyl)benzene","135-98-8","","45234",0,0,"",134.21816,0,,,,,134.21816,-99,,"Known compounds or mixtures"
2,"(2-methylbutyl)cyclohexane","54105-77-0","","99052",0,0,"",154.29238,0,,,,,154.29238,-99,,"Known compounds or mixtures"
3,"(2-methylpropyl)benzene; isobutylbenzene","538-93-2","","45235",0,0,"",134.21816,0,,,,,134.21816,-99,,"Known compounds or mixtures"
4,"1,1,1-trichloroethane","71-55-6","","43814",0,1,"",133.40422,1,,,,,133.40422,-99,,"Known compounds or mixtures"
5,"1,1,2,2-tetrachloroethane","79-34-5","","99277",0,1,"",167.84928,0,,,,,167.84928,-99,,"Known compounds or mixtures"
6,"1,1,2,3-tetramethylcyclohexane","6783-92-2","","99062",0,0,"",140.26580,0,,,,,140.26580,-99,,"Known compounds or mixtures"
7,"1,1,2-trichloroethane","79-00-5","","43820",0,1,"",133.40422,0,,,,,133.40422,-99,,"Known compounds or mixtures"
8,"1,1,2-trimethylcyclohexane","7094-26-0","","91074",0,0,"",126.23922,0,,,,,126.23922,-99,,"Known compounds or mixtures"
9,"1,1,2-trimethylcyclopentane","4259-00-1","","91033",0,0,"",112.21264,0,,,,,112.21264,-99,,"Known compounds or mixtures"
10,"1,1,3,4-tetramethylcyclohexane","24612-75-7","","99043",0,0,"",140.26580,0,,,,,140.26580,-99,,"Known compounds or mixtures"
11,"1,1,3,5-tetramethylcyclohexane","4306-65-4","","99107",0,0,"",140.26580,0,,,,,140.26580,-99,,"Known compounds or mixtures"
12,"1,1,3-trimethylcyclohexane","3073-66-3","","91064",0,0,"",126.23922,0,,,,,126.23922,-99,,"Known compounds or mixtures"
13,"1,1,3-trimethylcyclopentane","4516-69-2","","91030",0,0,"",112.21264,0,,,,,112.21264,-99,,"Known compounds or mixtures"
14,"1,1,4-trimethylcyclohexane","7094-27-1","","91057",0,0,"",126.23922,0,,,,,126.23922,-99,,"Known compounds or mixtures"
15,"1,1-dichloro-1-fluoroethane","1717-00-6","","99230",0,0,"HCFC-141b",116.94962,1,,,,,116.94962,-99,,"Known compounds or mixtures"
16,"1,1-dichloroethane","75-34-3","","43813",0,1,"",98.95916,0,,,,,98.95916,-99,,"Known compounds or mixtures"
17,"1,1-dichloroethene (vinylidene chloride)","75-35-4","","99013",0,1,"",96.94328,0,,,,,96.94328,-99,,"Known compounds or mixtures"
18,"1,1-dimethyl-2-propylcyclohexane","16587-71-6","","99059",0,0,"",154.29238,0,,,,,154.29238,-99,,"Known compounds or mixtures"
19,"1,1-dimethylcyclohexane","590-66-9","","91041",0,0,"",112.21264,0,,,,,112.21264,-99,,"Known compounds or mixtures"
20,"1,1-dimethylcyclopentane","1638-26-2","","99098",0,0,"",98.18606,0,,,,,98.18606,-99,,"Known compounds or mixtures"
66,"1-decene, dimer, hydrogenated","68649-11-6","","99269",0,0,"",137.19212,0,,,,,137.19212,-99,,"Unknown mixture. Mwt of REPUNK mixture used"
592,"N-butane","106-97-8","","43212",1,0,"N_BUTA",58.12220,0,,,,,58.12220,-99,,"Known compounds or mixtures"
605,"N-pentane","109-66-0","","43220",1,0,"N_PENT",72.14878,0,,,,,72.14878,-99,,"Known compounds or mixtures"
2284,"Unidentified","N/A","","99999",0,0,"UNID",137.19212,0,,,,,137.19212,-99,,"Unknown mixture. Mwt of REPUNK mixture used"
671,"Propane","74-98-6","","43204",1,0,"N_PROP",44.09562,0,,,,,44.09562,-99,,"Known compounds or mixtures"
`
)

func TestSpecMechanisms(t *testing.T) {
	mechAssignment := bytes.NewBuffer([]byte(mechAssignmentExample))
	mechMW := bytes.NewBuffer([]byte(mechMWExample))
	mechSpeciesInfo := bytes.NewBuffer([]byte(mechSpeciesInfoExample))
	mechanisms, err := NewMechanisms(mechAssignment, mechMW, mechSpeciesInfo)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name                 string
		mechanism, speciesID string
		mass                 bool
		result               map[string]float64
	}{
		{
			name:      "1",
			mechanism: "SAPRC99",
			speciesID: "1",
			mass:      true,
			result:    map[string]float64{"ARO1": 1},
		},
		{
			name:      "2",
			mechanism: "SAPRC99",
			speciesID: "1",
			mass:      false,
			result:    map[string]float64{"ARO1": 1 / 134.21816000000001},
		},
		{
			name:      "3",
			mechanism: "SAPRC99",
			speciesID: "66",
			mass:      true,
			result: map[string]float64{
				"ALK3": 2.247E-3 * 58.61 / 137.19212 / (2.247E-3*58.61/137.19212 + 7.103E-2*77.60/137.19212 + 4.286E-1*118.89/137.19211999999999),
				"ALK4": 7.103E-2 * 77.60 / 137.19212 / (2.247E-3*58.61/137.19212 + 7.103E-2*77.60/137.19212 + 4.286E-1*118.89/137.19211999999999),
				"ALK5": 4.286E-1 * 118.89 / 137.19211999999999 / (2.247E-3*58.61/137.19212 + 7.103E-2*77.60/137.19212 + 4.286E-1*118.89/137.19211999999999),
			},
		},
		{
			name:      "4",
			mechanism: "SAPRC99",
			speciesID: "66",
			mass:      false,
			result: map[string]float64{
				"ALK3": 2.247E-3 / 137.19212,
				"ALK4": 7.103E-2 / 137.19212,
				"ALK5": 4.286E-1 / 137.19211999999999,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factors, err := mechanisms.GroupFactors(test.mechanism, test.speciesID, test.mass)
			if err != nil {
				t.Error(err)
			}
			if mapDifferent(factors, test.result) {
				t.Errorf("test %+v: want %v, got %v", test, test.result, factors)
			}
		})
	}
}
