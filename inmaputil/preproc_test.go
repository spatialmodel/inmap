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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmaputil

import (
	"os"
	"testing"
)

func TestPreprocWRFChem(t *testing.T) {
	cfg := InitializeConfig()
	// Here we only test whether the program runs. We
	// check whether the output is correct elsewhere.
	cfg.Set("config", "../cmd/inmap/configExampleWRFChem.toml")
	cfg.Root.SetArgs([]string{"preproc"})
	defer os.Remove("../cmd/inmap/testdata/preproc/inmapData_WRFChem.ncf")
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestPreprocGEOSChem(t *testing.T) {
	cfg := InitializeConfig()
	// Here we only test whether the program runs. We
	// check whether the output is correct elsewhere.
	cfg.Set("config", "../cmd/inmap/configExampleGEOSChem.toml")
	cfg.Root.SetArgs([]string{"preproc"})
	defer os.Remove("../cmd/inmap/testdata/preproc/inmapData_GEOSChem.ncf")
	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestPreprocCombine(t *testing.T) {
	cfg := InitializeConfig()
	// Here we only test whether the program runs. We
	// check whether the output is correct elsewhere.
	cfg.Set("preprocessed_inputs", []string{
		"../cmd/inmap/testdata/inmapData_combine_outerNest.ncf",
		"../cmd/inmap/testdata/inmapData_combine_innerNest.ncf",
	})
	cfg.Root.SetArgs([]string{"preproc", "combine"})
	defer os.Remove("inmapdata_combined.ncf")

	if err := cfg.Root.Execute(); err != nil {
		t.Fatal(err)
	}
}
