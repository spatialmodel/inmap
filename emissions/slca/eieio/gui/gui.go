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
	"net/url"
	"strconv"
	"sync"

	leaflet "github.com/ctessum/go-leaflet"
	"github.com/go-humble/router"
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/vecty"
	"github.com/gopherjs/vecty/elem"
	"github.com/gopherjs/vecty/event"
	"github.com/gopherjs/vecty/prop"
	"github.com/johanbrandhorst/protobuf/grpcweb"
	eieiorpc "github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc/eieiorpcjs"
	"honnef.co/go/js/dom"
)

func main() {
	_, err := NewGUI()
	check(err)
}

// GUI implements a graphical user interface.
type GUI struct {
	vecty.Core
	eieiorpc.EIEIOrpcClient

	doc    dom.Document
	router *router.Router

	selection eieiorpc.Selection

	selectionCallbacks []func()

	Map      *leaflet.Map
	Polygons *js.Object
}

// NewGUI creates a new GUI.
func NewGUI() (*GUI, error) {
	c := new(GUI)
	c.doc = dom.GetWindow().Document()

	url, err := url.Parse(c.doc.BaseURI())
	check(err)

	c.EIEIOrpcClient = eieiorpc.NewEIEIOrpcClient(fmt.Sprintf("%s://%s", url.Scheme, url.Host))

	c.selectionCallbacks = []func(){}
	vecty.RenderBody(c)

	return c, nil
}

func (c *GUI) update(query string) {
	c.startLoading()
	defer c.stopLoading()
	if query == "" {
		sel, err := c.DefaultSelection(context.Background(), nil)
		check(err)
		c.selection = *sel
	} else {
		c.selection = selectionFromQuery(query)
	}
	var wg sync.WaitGroup
	wg.Add(len(c.selectionCallbacks) + 2)
	go func() {
		check(c.SetMapColors())
		wg.Done()
	}()
	for i := range c.selectionCallbacks {
		go func(i int) {
			c.selectionCallbacks[i]()
			wg.Done()
		}(i)
	}

	go func() {
		check(c.LoadGeometry())
		wg.Done()
	}()

	wg.Wait()
}

// queryFromSelection creates a URL query from s.
func queryFromSelection(s eieiorpc.Selection) url.Values {
	v := url.Values{}
	v.Set("dg", s.EndUseGroup)
	v.Set("ds", s.EndUseSector)
	v.Set("pg", s.EmitterGroup)
	v.Set("ps", s.EmitterSector)
	v.Set("it", s.ImpactType)
	v.Set("dt", fmt.Sprintf("%d", s.FinalDemandType))
	v.Set("y", fmt.Sprint(s.Year))
	v.Set("pop", s.Population)
	v.Set("aqm", s.AQM)
	switch s.ImpactType {
	case "health", "conc":
		v.Set("pol", fmt.Sprintf("%d", s.GetPollutant()))
	case "emis":
		e := s.GetEmission()
		if int32(e) == 5 {
			e = eieiorpc.Emission_PM25
		}
		v.Set("pol", fmt.Sprintf("%d", e))
	default:
		check(fmt.Errorf("invalid impact type `%s`", s.ImpactType))
	}
	return v
}

// selectionFromQuery parses a URL query to populate a
// Selection variable.
func selectionFromQuery(q string) eieiorpc.Selection {
	v, err := url.ParseQuery(q)
	check(err)
	var s eieiorpc.Selection
	s.EndUseGroup = v.Get("dg")
	s.EndUseSector = v.Get("ds")
	s.EmitterGroup = v.Get("pg")
	s.EmitterSector = v.Get("ps")
	s.AQM = v.Get("aqm")
	s.ImpactType = v.Get("it")
	dt, err := strconv.ParseInt(v.Get("dt"), 10, 32)
	check(err)
	s.FinalDemandType = eieiorpc.FinalDemandType(dt)
	y, err := strconv.ParseInt(v.Get("y"), 10, 32)
	check(err)
	s.Year = int32(y)
	s.Population = v.Get("pop")
	p, err := strconv.ParseInt(v.Get("pol"), 10, 32)
	check(err)
	switch s.ImpactType {
	case "health", "conc":
		s.Pol = &eieiorpc.Selection_Pollutant{Pollutant: eieiorpc.Pollutant(p)}
	case "emis":
		s.Pol = &eieiorpc.Selection_Emission{Emission: eieiorpc.Emission(p)}
	default:
		check(fmt.Errorf("invalid impact type `%s`", s.ImpactType))
	}
	return s
}

// selectionFromForm reads the form on the web page to populate a
// Selection variable.
func selectionFromForm() eieiorpc.Selection {
	var s eieiorpc.Selection
	s.EndUseGroup = selected("#dg")
	s.EndUseSector = selected("#ds")
	s.EmitterGroup = selected("#pg")
	s.EmitterSector = selected("#ps")
	s.ImpactType = selected("#it")
	s.AQM = selected("#aqm")
	dt, err := strconv.ParseInt(selected("#dt"), 10, 32)
	check(err)
	s.FinalDemandType = eieiorpc.FinalDemandType(dt)
	y, err := strconv.ParseInt(selected("#y"), 10, 32)
	check(err)
	s.Year = int32(y)
	s.Population = selected("#pop")
	p, err := strconv.ParseInt(selected("#pol"), 10, 32)
	check(err)
	switch s.ImpactType {
	case "health", "conc":
		s.Pol = &eieiorpc.Selection_Pollutant{Pollutant: eieiorpc.Pollutant(p)}
	case "emis":
		s.Pol = &eieiorpc.Selection_Emission{Emission: eieiorpc.Emission(p)}
	default:
		check(fmt.Errorf("invalid impact type `%s`", s.ImpactType))
	}
	return s
}

// Render creates the page view.
func (c *GUI) Render() vecty.ComponentOrHTML {
	vecty.SetTitle("EIEIO")
	return elem.Body(
		&Navigator{Title: "EIEIO"},
		elem.Div(vecty.Markup(vecty.Class("container-fluid")),
			elem.Div(vecty.Markup(vecty.Class("row")),
				elem.Div(vecty.Markup(vecty.Class("col-xs-12", "col-md-3")),
					elem.Heading3(vecty.Text("Air pollution health impacts of human activity")),
					elem.Form(vecty.Markup(prop.ID("selection-form")),
						elem.Div(vecty.Markup(vecty.Class("form-group")),
							&selector{c: c, id: "y", label: "Year", options: c.yearOptions()},
							&selector{c: c, id: "dt", label: "User", options: c.userOptions()},
							&selector{c: c, id: "dg", label: "Use group",
								options: c.impactOptions(c.EndUseGroups, "dg")},
							&selector{c: c, id: "ds", label: "Specific use",
								options: c.impactOptions(c.EndUseSectors, "ds")},
							&selector{c: c, id: "pg", label: "Emitter group",
								options: c.impactOptions(c.EmitterGroups, "pg")},
							&selector{c: c, id: "ps", label: "Specific emitter",
								options: c.impactOptions(c.EmitterSectors, "ps")},
							&selector{c: c, id: "it", label: "Impact type", options: c.impactTypeOptions()},
							&selector{c: c, id: "pop", label: "Impacted population", options: c.populationOptions()},
							&selector{c: c, id: "pol", label: "Pollutant", options: c.pollutantOptions()},
							&selector{c: c, id: "aqm", label: "Air quality model", options: c.aqmOptions()},
						),
					),
					elem.Div(vecty.Markup(prop.ID("eiolegend"))), // Div for the legend.
					elem.Paragraph(vecty.Markup(),
						vecty.Text("The largest 0.1% of values on the map are shown in green."),
					),
				),
				elem.Div(vecty.Markup(vecty.Class("col-xs-12", "col-md-9")),
					elem.Div(vecty.Markup(prop.ID("eiomap"))),
				),
			),
		),
		elem.Div(vecty.Markup(vecty.Class("loading"), prop.ID("loading"))),
	)
}

// Mount loads other components after the page has been rendered.
func (c *GUI) Mount() {
	go func() {
		check(c.LoadMap("eiomap"))
		c.router = router.New()
		c.router.HandleFunc("/eieio/{query}", func(ctx *router.Context) {
			go func() { c.update(ctx.Params["query"]) }()
		})
		c.router.InterceptLinks()
		c.router.Start()

		url, err := url.Parse(c.doc.BaseURI())
		check(err)

		c.router.Navigate("/eieio/" + url.RawQuery)
	}()
}

// selector implements a selector control
type selector struct {
	vecty.Core
	c       *GUI
	options func(id string) func()
	id      string
	label   string
}

// Render renders the view of the selector.
func (s *selector) Render() vecty.ComponentOrHTML {
	s.c.selectionCallbacks = append(s.c.selectionCallbacks, s.options(s.id))
	return elem.Span(
		elem.Label(vecty.Markup(prop.For(s.id)), vecty.Text(s.label)),
		elem.Select(
			vecty.Markup(
				vecty.Class("form-control"),
				prop.ID(s.id),
				event.Change(func(e *vecty.Event) {
					sel := selectionFromForm()
					s.c.router.Navigate(fmt.Sprintf("/eieio/%s", queryFromSelection(sel).Encode()))
				}),
			),
		),
	)
}

type impactFunc func(context.Context, *eieiorpc.Selection, ...grpcweb.CallOption) (*eieiorpc.Selectors, error)
type optionFunc func(id string) func()

// impactOptions returns a function that makes a list of the results of
// the given function.
func (c *GUI) impactOptions(f impactFunc, typeID string) optionFunc {
	return func(id string) func() {
		return func() {
			selection := queryFromSelection(c.selection)[typeID][0]
			selElem := c.doc.GetElementByID(id).(*dom.HTMLSelectElement)
			selElem.SetInnerHTML("")
			sel, err := f(context.Background(), &c.selection)
			check(err)
			for i, name := range sel.Names {
				if sel.Values[i] == 0 {
					continue // Skip options with zero impacts.
				}
				var val string
				if len(sel.Codes) != 0 {
					val = sel.Codes[i]
				} else {
					val = name
				}
				if val == selection {
					var text string
					switch c.selection.ImpactType {
					case "health", "conc":
						text = fmt.Sprintf("%s (%.3g deaths)", name, sel.Values[i])
					case "emis":
						text = fmt.Sprintf("%s (%.3g μg s<sup>-1</sup>)", name, sel.Values[i])
					default:
						dom.GetWindow().Alert(fmt.Sprintf("invalid impact type request: %s", c.selection.ImpactType))
					}
					selElem.InsertBefore(c.createOption(text, val, true), nil)
				} else {
					text := fmt.Sprintf("%s (%.3g)", name, sel.Values[i])
					selElem.InsertBefore(c.createOption(text, val, false), nil)
				}
			}
		}
	}
}

// yearOptions returns a function that lists the available analysis years.
func (c *GUI) yearOptions() optionFunc {
	return func(id string) func() {
		return func() {
			sel := c.doc.GetElementByID(id).(*dom.HTMLSelectElement)
			sel.SetInnerHTML("")
			years, err := c.Years(context.Background(), &c.selection)
			check(err)
			for _, y := range years.Years {
				sel.InsertBefore(c.createOption(fmt.Sprint(y), "", y == c.selection.Year), nil)
			}
		}
	}
}

// aqmOptions returns a function that lists the available air quality models.
func (c *GUI) aqmOptions() optionFunc {
	return func(id string) func() {
		return func() {
			sel := c.doc.GetElementByID(id).(*dom.HTMLSelectElement)
			sel.SetInnerHTML("")
			sel.InsertBefore(c.createOption("InMAP", "isrm", "isrm" == c.selection.AQM), nil)
			sel.InsertBefore(c.createOption("APSCA Annual", "apsca_q0", "apsca_q0" == c.selection.AQM), nil)
			sel.InsertBefore(c.createOption("APSCA Q1", "apsca_q1", "apsca_q1" == c.selection.AQM), nil)
			sel.InsertBefore(c.createOption("APSCA Q2", "apsca_q2", "apsca_q2" == c.selection.AQM), nil)
			sel.InsertBefore(c.createOption("APSCA Q3", "apsca_q3", "apsca_q3" == c.selection.AQM), nil)
			sel.InsertBefore(c.createOption("APSCA Q4", "apsca_q4", "apsca_q4" == c.selection.AQM), nil)
		}
	}
}

// userOptions returns a function that list the final demand types.
func (c *GUI) userOptions() optionFunc {
	users := []struct {
		val  eieiorpc.FinalDemandType
		name string
	}{
		{val: eieiorpc.FinalDemandType_AllDemand, name: "All demand"},
		{val: eieiorpc.FinalDemandType_NonExport, name: "Domestic (All - Export)"},
		{val: eieiorpc.FinalDemandType_PersonalConsumption, name: "Personal Consumption"},
		{val: eieiorpc.FinalDemandType_PrivateStructures, name: "Private Structures"},
		{val: eieiorpc.FinalDemandType_PrivateEquipment, name: "Private Equipment"},
		{val: eieiorpc.FinalDemandType_PrivateIP, name: "Private IP"},
		{val: eieiorpc.FinalDemandType_PrivateResidential, name: "Private Residential"},
		{val: eieiorpc.FinalDemandType_InventoryChange, name: "Inventory Change"},
		{val: eieiorpc.FinalDemandType_Export, name: "Export"},
		{val: eieiorpc.FinalDemandType_DefenseConsumption, name: "Defense Consumption"},
		{val: eieiorpc.FinalDemandType_DefenseStructures, name: "Defense Structures"},
		{val: eieiorpc.FinalDemandType_DefenseEquipment, name: "Defense Equipment"},
		{val: eieiorpc.FinalDemandType_DefenseIP, name: "Defense IP"},
		{val: eieiorpc.FinalDemandType_NondefenseConsumption, name: "Non-Defense Federal Government Consumption"},
		{val: eieiorpc.FinalDemandType_NondefenseStructures, name: "Non-Defense Federal Government Structures"},
		{val: eieiorpc.FinalDemandType_NondefenseEquipment, name: "Non-Defense Federal Government Equipment"},
		{val: eieiorpc.FinalDemandType_NondefenseIP, name: "Non-Defense Federal Government IP"},
		{val: eieiorpc.FinalDemandType_LocalConsumption, name: "Local Government Consumption"},
		{val: eieiorpc.FinalDemandType_LocalStructures, name: "Local Government Structures"},
		{val: eieiorpc.FinalDemandType_LocalEquipment, name: "Local Government Equipment"},
		{val: eieiorpc.FinalDemandType_LocalIP, name: "Local Government IP"},
	}
	return func(id string) func() {
		return func() {
			sel := c.doc.GetElementByID(id).(*dom.HTMLSelectElement)
			sel.SetInnerHTML("")
			for _, u := range users {
				sel.InsertBefore(c.createOption(u.name, fmt.Sprintf("%d", u.val), u.val == c.selection.FinalDemandType), nil)
			}
		}
	}
}

func (c *GUI) createOption(name, value string, selected bool) *dom.HTMLOptionElement {
	o := c.doc.CreateElement("option").(*dom.HTMLOptionElement)
	if value != "" {
		o.SetAttribute("value", value)
	}
	o.Selected = selected
	o.SetInnerHTML(name)
	return o
}

// impactTypeOptions returns a function that lists the available impact types.
func (c *GUI) impactTypeOptions() optionFunc {
	impacts := []struct {
		val  string
		name string
	}{
		{val: "health", name: "Deaths"},
		{val: "conc", name: "PM<sub>2.5<sub> Concentrations"},
		{val: "emis", name: "Emissions"},
	}
	return func(id string) func() {
		return func() {
			sel := c.doc.GetElementByID(id).(*dom.HTMLSelectElement)
			sel.SetInnerHTML("")
			for _, it := range impacts {
				sel.InsertBefore(c.createOption(it.name, it.val, it.val == c.selection.ImpactType), nil)
			}
		}
	}
}

// pollutantOptions returns a function that lists the available
// pollutants.
func (c *GUI) pollutantOptions() optionFunc {
	type holder struct {
		val  int32
		name string
	}
	concPols := []holder{
		{val: 5, name: "Total PM<sub>2.5</sub>"},
		{val: 4, name: "Primary PM<sub>2.5</sub>"},
		{val: 0, name: "Particulate ammonium"},
		{val: 1, name: "Particulate nitrate"},
		{val: 2, name: "Particulate sulfate"},
		{val: 3, name: "Secondary organic aerosol"},
	}
	emisPols := []holder{
		{val: 0, name: "PM<sub>2.5</sub>"},
		{val: 1, name: "NH<sub>3</sub>"},
		{val: 2, name: "NO<sub>x</sub>"},
		{val: 3, name: "SO<sub>x</sub>"},
		{val: 4, name: "VOC"},
	}
	return func(id string) func() {
		return func() {
			var pols []holder
			switch c.selection.ImpactType {
			case "health", "conc":
				pols = concPols
			case "emis":
				pols = emisPols
			default:
				dom.GetWindow().Alert(fmt.Sprintf("invalid impact type request: %s", c.selection.ImpactType))
			}
			sel := c.doc.GetElementByID(id).(*dom.HTMLSelectElement)
			sel.SetInnerHTML("")
			for _, p := range pols {
				var match bool
				switch c.selection.ImpactType {
				case "health", "conc":
					match = eieiorpc.Pollutant(p.val) == c.selection.GetPollutant()
				case "emis":
					match = eieiorpc.Emission(p.val) == c.selection.GetEmission()
				default:
					dom.GetWindow().Alert(fmt.Sprintf("invalid impact type request: %s", c.selection.ImpactType))
				}
				sel.InsertBefore(c.createOption(p.name, fmt.Sprintf("%d", p.val), match), nil)
			}
		}
	}
}

// populationOptions returns a function that lists the available
// exposed populations.
func (c *GUI) populationOptions() optionFunc {
	return func(id string) func() {
		return func() {
			sel := c.doc.GetElementByID(id).(*dom.HTMLSelectElement)
			sel.SetInnerHTML("")
			if c.selection.ImpactType == "emis" {
				return
			}
			pops, err := c.Populations(context.Background(), nil)
			check(err)
			for _, p := range pops.Names {
				sel.InsertBefore(c.createOption(p, "", p == c.selection.Population), nil)
			}
		}
	}
}

// selected returns the selected value of a select input when given
// the ID of the input in the for "#id".
func selected(id string) string {
	document := dom.GetWindow().Document()
	e := document.QuerySelector(id).(*dom.HTMLSelectElement)
	sel := e.SelectedOptions()
	if len(sel) != 1 {
		check(fmt.Errorf("exactly one option needs to be selected, instead have %d", len(sel)))
	}
	v := sel[0].Value
	if v == "" {
		v = sel[0].Text
	}
	return v
}

// startLoading makes the loading symbol visible.
func (c *GUI) startLoading() {
	c.doc.GetElementByID("loading").(*dom.HTMLDivElement).SetClass("loading")
}

// stopLoading makes the loading symbol invisible.
func (c *GUI) stopLoading() {
	c.doc.GetElementByID("loading").(*dom.HTMLDivElement).SetClass("")
}

func check(err error) {
	if err != nil {
		dom.GetWindow().Alert(err.Error())
	}
}
