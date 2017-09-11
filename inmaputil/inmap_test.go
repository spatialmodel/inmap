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

package inmaputil

import (
	"os"
	"testing"
)

func TestCreateGrid(t *testing.T) {
	Cfg.Set("config", "../inmap/configExample.toml")
	Root.SetArgs([]string{"grid"})
	if err := Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPStaticCreateGrid(t *testing.T) {
	Cfg.Set("static", true)
	Cfg.Set("createGrid", true)
	os.Setenv("InMAPRunType", "static")
	Cfg.Set("config", "../inmap/configExample.toml")
	Root.SetArgs([]string{"run", "steady"})
	if err := Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPStaticLoadGrid(t *testing.T) {
	Cfg.Set("static", true)
	Cfg.Set("createGrid", false)
	os.Setenv("InMAPRunType", "staticLoadGrid")
	Cfg.Set("config", "../inmap/configExample.toml")
	Root.SetArgs([]string{"run", "steady"})
	if err := Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInMAPDynamic(t *testing.T) {
	Cfg.Set("static", false)
	Cfg.Set("createGrid", false) // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	Cfg.Set("config", "../inmap/configExample.toml")
	if err := Root.PersistentPreRunE(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := steadyCmd.RunE(nil, nil); err != nil {
		t.Fatal(err)
	}
}
