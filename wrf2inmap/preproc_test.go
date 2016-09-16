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

package main

import (
	"os"
	"flag"
	"testing"
)

const evalDataEnv = "evaldata"

func TestWRF2InMAP(t *testing.T) {
	if testing.Short() {
		return
	}

	evalData := os.Getenv(evalDataEnv)
	if evalData == "" {
		t.Fatalf("please set the '%s' environment variable to the location of the "+
			"downloaded evaluation data and try again", evalDataEnv)
	}

	err := flag.Set("config", "configExample.json")
	if err != nil {
		t.Fatal(err)
	}
	main()
}
