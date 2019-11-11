/*
Copyright © 2019 the InMAP authors.
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

package hash

import (
	"encoding/gob"
	"fmt"
	"hash/fnv"

	"github.com/davecgh/go-spew/spew"
)

// Hash returns a hash key for the specified object.
func Hash(object interface{}) string {
	if s, ok := object.(fmt.Stringer); ok {
		return s.String()
	}
	h := fnv.New128a()

	e := gob.NewEncoder(h)
	if err := e.Encode(object); err == nil {
		bKey := h.Sum([]byte{})
		return fmt.Sprintf("%x", bKey[0:h.Size()])
	}
	// If there is an error (e.g., there are NaN values)
	// use spew instead of gob.
	printer := spew.ConfigState{
		Indent:                  " ",
		SortKeys:                true,
		DisableMethods:          true,
		SpewKeys:                true,
		DisablePointerAddresses: true,
		DisableCapacities:       true,
	}
	printer.Fprintf(h, "%#v", object)
	bKey := h.Sum([]byte{})
	return fmt.Sprintf("%x", bKey[0:h.Size()])
}
