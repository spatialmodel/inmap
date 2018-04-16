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

// Command inmap is a command-line interface for the InMAP air pollution model.
package main

import (
	"fmt"
	"os"

	"github.com/spatialmodel/inmap/inmaputil"
)

func main() {
	var commands int
	for _, arg := range os.Args { // Count the number of supplied commands.
		if arg[0] != '-' {
			commands++
		}
	}
	if commands == 1 { // If only one command was supplied, start the GUI server.
		inmaputil.StartWebServer()
	}

	// If more than one command was supplied, run in CLI mode.
	if err := inmaputil.Root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
