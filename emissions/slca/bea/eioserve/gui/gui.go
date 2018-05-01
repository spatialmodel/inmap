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
	"context"
	"fmt"
	"io"
	"strings"

	leaflet "github.com/ctessum/go-leaflet"
	"github.com/gopherjs/gopherjs/js"
	"github.com/johanbrandhorst/protobuf/grpcweb"
	eioclientpb "github.com/spatialmodel/inmap/emissions/slca/bea/eioserve/proto/eioclientpb"
	"honnef.co/go/js/dom"
)

const All = "All"

func main() {
	c, err := newClient()
	if err != nil {
		dom.GetWindow().Alert(err.Error())
		return
	}
	go c.Load()
}

type client struct {
	eioclientpb.EIOServeClient

	doc dom.Document

	demandSectorGroup *dom.HTMLSelectElement
	demandSector      *dom.HTMLSelectElement
	prodSectorGroup   *dom.HTMLSelectElement
	prodSector        *dom.HTMLSelectElement
	impactType        *dom.HTMLSelectElement
	demandType        *dom.HTMLSelectElement
	sccButton         *dom.HTMLButtonElement
	sccModalTitle     dom.Element
	sccTable          *sccTable

	selection eioclientpb.Selection

	Map      *leaflet.Map
	Polygons *js.Object
}

// newClient creates a new client.
func newClient() (*client, error) {
	c := new(client)
	c.doc = dom.GetWindow().Document()

	serverAddr := strings.TrimSuffix(c.doc.BaseURI(), "/")
	c.EIOServeClient = eioclientpb.NewEIOServeClient(serverAddr)

	c.demandSectorGroup = c.doc.GetElementByID("demandSectorGroup").(*dom.HTMLSelectElement)
	c.demandSector = c.doc.GetElementByID("demandSector").(*dom.HTMLSelectElement)
	c.prodSectorGroup = c.doc.GetElementByID("prodSectorGroup").(*dom.HTMLSelectElement)
	c.prodSector = c.doc.GetElementByID("prodSector").(*dom.HTMLSelectElement)
	c.impactType = c.doc.GetElementByID("resultType").(*dom.HTMLSelectElement)
	c.demandType = c.doc.GetElementByID("demandType").(*dom.HTMLSelectElement)
	c.sccButton = c.doc.GetElementByID("sccButton").(*dom.HTMLButtonElement)
	c.sccModalTitle = c.doc.GetElementByID("sccModalTitle")

	go func() {
		c.sccTable = newSCCTable()

		c.addSelectListener(c.demandSectorGroup)
		c.addSelectListener(c.demandSector)
		c.addSelectListener(c.prodSectorGroup)
		c.addSelectListener(c.prodSector)
		c.addSelectListener(c.impactType)
		c.addSelectListener(c.demandType)
		c.addSCCButtonListener()
	}()

	return c, nil
}

// Load prepares the environment.
func (c *client) Load() error {
	if err := c.LoadMap("eiomap"); err != nil {
		dom.GetWindow().Alert(err.Error())
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

	if c.selection.ProductionSector == All {
		c.sccButton.Disabled = true
	} else {
		c.sccButton.Disabled = false
	}

	funcs := []func(context.Context, *eioclientpb.Selection, ...grpcweb.CallOption) (*eioclientpb.Selectors, error){
		c.DemandGroups, c.DemandSectors, c.ProdGroups, c.ProdSectors}
	selections := []string{c.selection.DemandGroup, c.selection.DemandSector, c.selection.ProductionGroup, c.selection.ProductionSector}
	objects := []*dom.HTMLSelectElement{c.demandSectorGroup, c.demandSector, c.prodSectorGroup, c.prodSector}
	for i, f := range funcs {
		go func(i int, f func(context.Context, *eioclientpb.Selection, ...grpcweb.CallOption) (*eioclientpb.Selectors, error)) {
			sectors, err := f(context.Background(), &c.selection)
			if err != nil {
				dom.GetWindow().Alert(err.Error())
			}
			c.updateSelect(objects[i], selections[i], sectors)
		}(i, f)
	}
	go func() {
		if err := c.SetMapColors(); err != nil {
			dom.GetWindow().Alert(err.Error())
		}
	}()
}

func (c *client) updateSelect(e *dom.HTMLSelectElement, selection string, s *eioclientpb.Selectors) {
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

// addSCCButtonListener creates a table of SCC codes associated with the
// current production sector when the button is clicked.
func (c *client) addSCCButtonListener() {
	c.sccButton.AddEventListener("click", false, func(_ dom.Event) {
		go func() {
			c.sccModalTitle.SetInnerHTML(fmt.Sprintf("Source Classification Codes (SCCs) associated with %s", c.selection.ProductionSector))
			c.sccTable.delete()
			c.sccTable = newSCCTable()
			sccClient, err := c.EIOServeClient.SCCs(context.Background(), &c.selection)
			if err != nil {
				dom.GetWindow().Alert(err.Error())
				return
			}
			for {
				sccInfo, err := sccClient.Recv()
				if err != nil {
					if err != io.EOF {
						dom.GetWindow().Alert(err.Error())
					}
					return
				}
				c.sccTable.addRow(sccInfo)
			}
		}()
	})
}

func (c *client) startLoading() {
	c.doc.GetElementByID("loading").(*dom.HTMLDivElement).SetClass("loading")
}

func (c *client) stopLoading() {
	c.doc.GetElementByID("loading").(*dom.HTMLDivElement).SetClass("")
}
