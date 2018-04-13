//+build js

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
	"net/rpc"
	"net/rpc/jsonrpc"

	leaflet "github.com/ctessum/go-leaflet"
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/websocket"
	"honnef.co/go/js/dom"
)

func main() {

	c, err := newClient(address, port)
	if err != nil {
		dom.GetWindow().Alert(err.Error())
		return
	}
	if err := c.Load(); err != nil {
		dom.GetWindow().Alert(err.Error())
	}
}

type client struct {
	doc    dom.Document
	client *rpc.Client

	demandSectorGroup *dom.HTMLSelectElement
	demandSector      *dom.HTMLSelectElement
	prodSectorGroup   *dom.HTMLSelectElement
	prodSector        *dom.HTMLSelectElement
	impactType        *dom.HTMLSelectElement
	demandType        *dom.HTMLSelectElement

	selection Selection

	Map      *leaflet.Map
	Polygons *js.Object
}

// newClient creates a new client.
func newClient(address, port string) (*client, error) {
	c := new(client)
	c.doc = dom.GetWindow().Document()
	var err error
	conn, err := websocket.Dial("ws://" + address + port + "/ws-rpc")
	if err != nil {
		return nil, err
	}
	c.client = jsonrpc.NewClient(conn)

	c.demandSectorGroup = c.doc.GetElementByID("demandSectorGroup").(*dom.HTMLSelectElement)
	c.demandSector = c.doc.GetElementByID("demandSector").(*dom.HTMLSelectElement)
	c.prodSectorGroup = c.doc.GetElementByID("prodSectorGroup").(*dom.HTMLSelectElement)
	c.prodSector = c.doc.GetElementByID("prodSector").(*dom.HTMLSelectElement)
	c.impactType = c.doc.GetElementByID("resultType").(*dom.HTMLSelectElement)
	c.demandType = c.doc.GetElementByID("demandType").(*dom.HTMLSelectElement)

	c.addSelectListener(c.demandSectorGroup)
	c.addSelectListener(c.demandSector)
	c.addSelectListener(c.prodSectorGroup)
	c.addSelectListener(c.prodSector)
	c.addSelectListener(c.impactType)
	c.addSelectListener(c.demandType)

	return c, nil
}

// Load prepares the environment.
func (c *client) Load() error {
	if err := c.LoadMap("eiomap"); err != nil {
		return err
	}

	c.updateSelection()
	return nil
}

// selection returns the selected value of a selector
func selection(e *dom.HTMLSelectElement) (string, error) {
	sel := e.SelectedOptions()
	if len(sel) != 1 {
		return "", fmt.Errorf("exactly one option needs to be selected, instead have %d", len(sel))
	}
	return sel[0].GetAttribute("value"), nil
}

func (c *client) addSelectListener(o *dom.HTMLSelectElement) {
	o.AddEventListener("input", false, func(_ dom.Event) {
		go func() {
			c.updateSelection()
		}()
	})
}

func (c *client) updateSelection() {
	c.startLoading()
	defer c.stopLoading()
	if err := c.sectorSelection(); err != nil {
		dom.GetWindow().Alert(err.Error())
	}
	funcNames := []string{"DemandGroups", "DemandSectors", "ProdGroups", "ProdSectors"}
	selections := []string{c.selection.DemandGroup, c.selection.DemandSector, c.selection.ProductionGroup, c.selection.ProductionSector}
	objects := []*dom.HTMLSelectElement{c.demandSectorGroup, c.demandSector, c.prodSectorGroup, c.prodSector}
	for i := range funcNames {
		sectors, err := c.Sectors(funcNames[i])
		if err != nil {
			dom.GetWindow().Alert(err.Error())
		}
		c.updateSelect(objects[i], selections[i], sectors)
	}
	if err := c.SetMapColors(); err != nil {
		dom.GetWindow().Alert(err.Error())
	}
}

func (c *client) updateSelect(e *dom.HTMLSelectElement, selection string, s *Selectors) {
	e.SetInnerHTML("")
	for i, name := range s.Names {
		if s.Values[i] == 0 {
			continue
		}
		o := c.doc.CreateElement("option").(*dom.HTMLOptionElement)
		o.SetAttribute("value", name)
		if name == selection {
			o.Selected = true
			switch c.selection.ImpactType {
			case "health_total", "conc_totalPM25", "health_white", "health_black", "health_native", "health_asian", "health_latino",
				"conc_PNH4", "conc_PNO3", "conc_PSO4", "conc_SOA", "conc_PrimaryPM25":
				o.SetInnerHTML(fmt.Sprintf("%s (%.3g deaths)", name, s.Values[i]))
			case "emis_PM25", "emis_NH3", "emis_NOx", "emis_SOx", "emis_VOC":
				o.SetInnerHTML(fmt.Sprintf("%s (%.3g μg s<sup>-1</sup>)", name, s.Values[i]))
			default:
				dom.GetWindow().Alert(fmt.Sprintf("invalid impact type request: %s", c.selection.ImpactType))
			}
		} else {
			o.SetInnerHTML(fmt.Sprintf("%s (%.3g)", name, s.Values[i]))
		}
		e.InsertBefore(o, nil)
	}
}

// sectorSelection returns the sector combination that has been selected.
func (c *client) sectorSelection() error {
	old := c.selection
	var err error
	c.selection.DemandGroup, err = selection(c.demandSectorGroup)
	if err != nil {
		return err
	}
	c.selection.DemandSector, err = selection(c.demandSector)
	if err != nil {
		return err
	}
	c.selection.ProductionGroup, err = selection(c.prodSectorGroup)
	if err != nil {
		return err
	}
	c.selection.ProductionSector, err = selection(c.prodSector)
	if err != nil {
		return err
	}
	c.selection.ImpactType, err = selection(c.impactType)
	if err != nil {
		return err
	}
	c.selection.DemandType, err = selection(c.demandType)
	if err != nil {
		return err
	}
	if c.selection.DemandGroup != old.DemandGroup {
		c.selection.DemandSector = All
	}
	if c.selection.ProductionGroup != old.ProductionGroup {
		c.selection.ProductionSector = All
	}
	return nil
}

// Sectors returns the sectors and their amounts returned by calling
// the given rpc funcName.
func (c *client) Sectors(funcName string) (*Selectors, error) {
	var o Selectors
	err := c.client.Call("Server."+funcName, c.selection, &o)
	return &o, err
}

func (c *client) startLoading() {
	c.doc.GetElementByID("loading").(*dom.HTMLDivElement).SetClass("loading")
}

func (c *client) stopLoading() {
	c.doc.GetElementByID("loading").(*dom.HTMLDivElement).SetClass("")
}
