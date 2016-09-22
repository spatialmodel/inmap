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

package cmd

import (
	"os"
	"testing"
)

func TestCreateGrid(t *testing.T) {
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := Grid(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPStaticCreateGrid(t *testing.T) {
	dynamic := false
	createGrid := true
	os.Setenv("InMAPRunType", "static")
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := Run(dynamic, createGrid, DefaultScienceFuncs, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPStaticLoadGrid(t *testing.T) {
	dynamic := false
	createGrid := false
	os.Setenv("InMAPRunType", "staticLoadGrid")
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := Run(dynamic, createGrid, DefaultScienceFuncs, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPDynamic(t *testing.T) {
	dynamic := true
	createGrid := false // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if err := Run(dynamic, createGrid, DefaultScienceFuncs, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}
