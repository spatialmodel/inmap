/*
Copyright © 2013 the InMAP authors.
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

package inmap_test

import (
	"reflect"
	"testing"

	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
)

func TestVerticalProfile(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr, mortIndices := inmap.VarGridTestData()
	emis := inmap.NewEmissions()
	var m simplechem.Mechanism

	mutator, err := inmap.PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
		},
	}
	if err = d.Init(); err != nil {
		t.Error(err)
	}

	height, wind, err := d.VerticalProfile("WindSpeed", geom.Point{X: -500, Y: -500}, m)
	if err != nil {
		t.Fatal(err)
	}

	wantHeight := []float64{27.808368682861328, 95.64196014404297, 188.1898956298828, 306.10257720947266, 454.42835998535156, 642.9596405029297, 873.7414474487305, 1224.0262756347656, 1684.805648803711, 2168.831771850586}
	wantWind := []float64{2.163347005844116, 2.466365337371826, 2.3336946964263916, 2.100137948989868, 2.0755155086517334, 1.9850538969039917, 1.9812132120132446, 3.3489553928375244, 5.816560745239258, 7.861310005187988}

	if !reflect.DeepEqual(height, wantHeight) {
		t.Errorf("height: want %v, got %v", wantHeight, height)
	}
	if !reflect.DeepEqual(wind, wantWind) {
		t.Errorf("wind: want %v, got %v", wantWind, wind)
	}
}
