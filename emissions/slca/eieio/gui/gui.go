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
	eieiorpc "github.com/spatialmodel/inmap/emissions/slca/eieio/grpc/gopherjsgrpc"
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
	wg.Add(len(c.selectionCallbacks) + 1)
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
	wg.Wait()
}

// queryFromSelection creates a URL query from s.
func queryFromSelection(s eieiorpc.Selection) url.Values {
	v := url.Values{}
	v.Set("dg", s.DemandGroup)
	v.Set("ds", s.DemandSector)
	v.Set("pg", s.ProductionGroup)
	v.Set("ps", s.ProductionSector)
	v.Set("it", s.ImpactType)
	v.Set("dt", s.DemandType)
	v.Set("y", fmt.Sprint(s.Year))
	v.Set("pop", s.Population)
	v.Set("pol", fmt.Sprintf("%d", s.Pollutant))
	return v
}

// selectionFromQuery parses a URL query to populate a
// Selection variable.
func selectionFromQuery(q string) eieiorpc.Selection {
	v, err := url.ParseQuery(q)
	check(err)
	var s eieiorpc.Selection
	s.DemandGroup = v.Get("dg")
	s.DemandSector = v.Get("ds")
	s.ProductionGroup = v.Get("pg")
	s.ProductionSector = v.Get("ps")
	s.ImpactType = v.Get("it")
	s.DemandType = v.Get("dt")
	y, err := strconv.ParseInt(v.Get("y"), 10, 32)
	check(err)
	s.Year = int32(y)
	s.Population = v.Get("pop")
	p, err := strconv.ParseInt(v.Get("pol"), 10, 32)
	check(err)
	s.Pollutant = int32(p)
	return s
}

// selectionFromForm reads the form on the web page to populate a
// Selection variable.
func selectionFromForm() eieiorpc.Selection {
	var s eieiorpc.Selection
	s.DemandGroup = selected("#dg")
	s.DemandSector = selected("#ds")
	s.ProductionGroup = selected("#pg")
	s.ProductionSector = selected("#ps")
	s.ImpactType = selected("#it")
	s.DemandType = selected("#dt")
	y, err := strconv.ParseInt(selected("#y"), 10, 32)
	check(err)
	s.Year = int32(y)
	s.Population = selected("#pop")
	p, err := strconv.ParseInt(selected("#pol"), 10, 32)
	check(err)
	s.Pollutant = int32(p)
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
								options: c.impactOptions(c.DemandGroups, "dg")},
							&selector{c: c, id: "ds", label: "Specific use",
								options: c.impactOptions(c.DemandSectors, "ds")},
							&selector{c: c, id: "pg", label: "Emitter group",
								options: c.impactOptions(c.ProdGroups, "pg")},
							&selector{c: c, id: "ps", label: "Specific emitter",
								options: c.impactOptions(c.ProdSectors, "ps")},
							&selector{c: c, id: "it", label: "Impact type", options: c.impactTypeOptions()},
							&selector{c: c, id: "pop", label: "Impacted population", options: c.populationOptions()},
							&selector{c: c, id: "pol", label: "Pollutant", options: c.pollutantOptions()},
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
		c.router.HandleFunc("/{query}", func(ctx *router.Context) {
			go func() { c.update(ctx.Params["query"]) }()
		})
		c.router.InterceptLinks()
		c.router.Start()

		url, err := url.Parse(c.doc.BaseURI())
		check(err)

		c.router.Navigate(url.RawQuery)
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
					s.c.router.Navigate(fmt.Sprintf("/%s", queryFromSelection(sel).Encode()))
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

// userOptions returns a function that list the final demand types.
func (c *GUI) userOptions() optionFunc {
	users := []struct {
		val  string
		name string
	}{
		{val: "All", name: "All demand"},
		{val: "NonExport", name: "Domestic (All - Export)"},
		{val: "F010", name: "Personal Consumption"},
		{val: "F02S", name: "Private Structures"},
		{val: "F02E", name: "Private Equipment"},
		{val: "F02N", name: "Private IP"},
		{val: "F02R", name: "Private Residential"},
		{val: "F030", name: "Inventory Change"},
		{val: "F040", name: "Export"},
		{val: "F06C", name: "Defense Consumption"},
		{val: "F06S", name: "Defense Structures"},
		{val: "F06E", name: "Defense Equipment"},
		{val: "F06N", name: "Defense IP"},
		{val: "F07C", name: "Nondefense Consumption"},
		{val: "F07S", name: "Nondefense Structures"},
		{val: "F07E", name: "Nondefense Equipment"},
		{val: "F07N", name: "Nondefense IP"},
		{val: "F10C", name: "Local Consumption"},
		{val: "F10S", name: "Local Structures"},
		{val: "F10E", name: "Local Equipment"},
		{val: "F10N", name: "Local IP"},
	}
	return func(id string) func() {
		return func() {
			sel := c.doc.GetElementByID(id).(*dom.HTMLSelectElement)
			sel.SetInnerHTML("")
			for _, u := range users {
				sel.InsertBefore(c.createOption(u.name, u.val, u.val == c.selection.DemandType), nil)
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
				sel.InsertBefore(c.createOption(p.name, fmt.Sprintf("%d", p.val), p.val == c.selection.Pollutant), nil)
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
