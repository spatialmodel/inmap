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

func TestSR(t *testing.T) {
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	Config.SROutputFile = "tempSR.ncf"
	begin := 8
	end := 9
	layers := []int{0}
	if err := RunSR(begin, end, layers); err != nil {
		t.Fatal(err)
	}
	os.Remove(Config.SROutputFile)
}

func TestWorkerInit(t *testing.T) {
	if err := Startup("../configExample.toml"); err != nil {
		t.Fatal(err)
	}
	if _, err := InitWorker(); err != nil {
		t.Fatal(err)
	}
}
