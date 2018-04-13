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

package bea

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/spatialmodel/inmap/emissions/slca"
)

func TestNEISpatialRefs(t *testing.T) {
	s := loadSpatial(t)
	spatialRefs, ok := s.SpatialRefs[2007]
	if !ok {
		t.Fatal(fmt.Errorf("missing spatial refs for year %d", 2007))
	}

	zeroWant := &slca.SpatialRef{
		SCCs: []slca.SCC{
			"2801000003", "2801700099", "2270005015", "2461850000", "0030202601", "0030202070",
			"0030202080", "2801500264", "2801500171", "2801000005", "2801000000", "2801700005",
			"2801700006", "2801700007", "2801700001", "2801700013", "2801700015", "2801700014",
			"2801700003", "2801700010", "2801700004", "2268005060", "2268005055", "2267005060",
			"2267005055", "2270005010", "2270005030", "2270005025", "2270005020", "2270005060",
			"2270005055", "2270005035", "2270005045", "2270005040", "2260005035", "2265005010",
			"2265005030", "2265005015", "2265005025", "2265005020", "2265005060", "2265005055",
			"2265005035", "2265005045", "2265005040",
		},
		SCCFractions: []float64{
			0.1391898672468795, 0.1391898672468795, 0.1391898672468795, 0.1391898672468795,
			0.1391898672468795, 0.1391898672468795, 0.1391898672468795, 0.2784349496221354,
			0.18706598258126522, 0.1391898672468795, 0.1391898672468795, 0.1391898672468795,
			0.1391898672468795, 0.1391898672468795, 0.1391898672468795, 0.1391898672468795,
			0.1391898672468795, 0.1391898672468795, 0.1391898672468795, 0.1391898672468795,
			0.1391898672468795, 0.1391898672468795, 0.1391898672468795, 0.1391898672468795,
			0.1391898672468795, 0.1391898672468795, 0.1391898672468795, 0.1391898672468795,
			0.1391898672468795, 0.1391898672468795, 0.1391898672468795, 0.1391898672468795,
			0.1391898672468795, 0.1391898672468795, 0.1391898672468795, 0.1391898672468795,
			0.1391898672468795, 0.1391898672468795, 0.1391898672468795, 0.1391898672468795,
			0.1391898672468795, 0.1391898672468795, 0.1391898672468795, 0.1391898672468795,
			0.1391898672468795,
		},
		Type:            slca.Stationary,
		EmisYear:        2007,
		NoNormalization: true,
	}
	if !reflect.DeepEqual(zeroWant, spatialRefs[0]) {
		t.Errorf("ref 0: have %#v, want %v", spatialRefs[0], zeroWant)
	}

	threeEightyEightWant := &slca.SpatialRef{
		SCCs:            []slca.SCC{},
		SCCFractions:    []float64{},
		EmisYear:        2007,
		Type:            slca.Stationary,
		NoNormalization: true,
	}
	if !reflect.DeepEqual(threeEightyEightWant, spatialRefs[388]) {
		t.Errorf("ref 388: have %v, want %#v", spatialRefs[388], threeEightyEightWant)
	}

	codes := make(map[slca.SCC]struct{})
	for _, r := range spatialRefs {
		for _, scc := range r.SCCs {
			codes[scc] = struct{}{}
		}
	}

	for code := range codes {
		var codeTotal float64
		for _, r := range spatialRefs {
			for i, scc := range r.SCCs {
				if scc == code {
					codeTotal += r.SCCFractions[i]
				}
			}
		}
		if different(codeTotal, 1) {
			t.Errorf("code %s total: %g != 1", code, codeTotal)
		}
	}
}
