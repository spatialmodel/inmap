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

package main

import (
	"fmt"

	"github.com/gopherjs/gopherjs/js"
	eioclientpb "github.com/spatialmodel/inmap/emissions/slca/bea/eioserve/proto/eioclientpb"
)

type sccTable struct {
	div   *js.Object
	table *js.Object
	doc   *js.Object
}

func newSCCTable() *sccTable {
	t := &sccTable{
		doc: js.Global.Get("window").Get("document"),
	}
	t.div = t.doc.Call("getElementById", "sccTable")
	t.table = t.doc.Call("createElement", "table")
	t.table.Set("class", "table table-striped")
	t.header()
	t.div.Call("appendChild", t.table)
	return t
}

func (t *sccTable) header() {
	row := t.doc.Call("createElement", "tr")
	for _, name := range []string{"SCC", "Description", "Fraction"} {
		th := t.doc.Call("createElement", "th")
		textnode := t.doc.Call("createTextNode", name)
		th.Call("appendChild", textnode)
		row.Call("appendChild", th)
	}
	(*t.table).Call("appendChild", row)
}

// addRow adds a row to the reciever table.
func (t *sccTable) addRow(r *eioclientpb.SCCInfo) {
	row := t.doc.Call("createElement", "tr")
	for _, v := range []interface{}{r.SCC, r.Desc, fmt.Sprintf("%.3g", r.Frac)} {
		col := t.doc.Call("createElement", "td")
		textnode := t.doc.Call("createTextNode", v)
		col.Call("appendChild", textnode)
		row.Call("appendChild", col)
	}
	t.table.Call("appendChild", row)
}

func (t *sccTable) delete() {
	for t.div.Call("hasChildNodes").Bool() {
		t.div.Call("removeChild", t.div.Get("lastChild"))
	}
}
