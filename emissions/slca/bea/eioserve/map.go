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
	"log"

	"honnef.co/go/js/dom"

	"github.com/ctessum/geom"
	leaflet "github.com/ctessum/go-leaflet"
	"github.com/ctessum/go-leaflet/plugin/glify"
	"github.com/gopherjs/gopherjs/js"
)

func (c *client) LoadMap(div string) error {
	c.setCSS()
	dom.GetWindow().AddEventListener("resize", false, func(arg3 dom.Event) {
		c.setCSS()
	})

	mapOptions := leaflet.DefaultMapOptions()
	mapOptions.PreferCanvas = true
	c.Map = leaflet.NewMap(div, mapOptions)
	c.Map.SetView(leaflet.NewLatLng(39.8282, -98.5795), 4)

	pane := c.Map.CreatePane("labels")
	pane.SetZIndex(650)
	options := leaflet.DefaultTileLayerOptions()
	options.Attribution = `Map data &copy; <a href=\"http://openstreetmap.org">OpenStreetMap</a> contributors, <a href="http://creativecommons.org/licenses/by-sa/2.0/">CC-BY-SA</a>, Imagery © <a href="http://mapbox.com">Mapbox</a>`
	options.Pane = "labels"

	layer := leaflet.NewTileLayer("https://api.mapbox.com/styles/v1/ctessum/cixuwgf55003e2roe7z5ouk2w/tiles/256/{z}/{x}/{y}?access_token=pk.eyJ1IjoiY3Rlc3N1bSIsImEiOiJjaXh1dnZxYjAwMDRjMzNxcWczZ3JqZDd4In0.972k4y-Xc-PpYTdeUTbufA", options)
	layer.AddTo(c.Map)

	if err := c.LoadGeometry(); err != nil {
		return err
	}

	return nil
}

type gridCell struct {
	geom.Polygon
	i int
}

func (c *client) LoadGeometry() error {
	log.Println("calling Server.Geometry")
	var o []byte
	if err := c.client.Call("Server.Geometry", &Empty{}, &o); err != nil {
		return err
	}
	log.Println("received from Server.Geometry; parsing polygons")
	c.Polygons = js.Global.Get("JSON").Call("parse", string(o))
	log.Println("finished parsing polygons")
	return nil
}

func (c *client) SetMapColors() error {

	var m MapInfo
	log.Println("calling Server.MapInfo")
	if err := c.client.Call("Server.MapInfo", &c.selection, &m); err != nil {
		return err
	}
	log.Println("received from Server.MapInfo; setting polygon colors")

	options := glify.DefaultShapeOptions()
	options.Map = c.Map
	options.Data = c.Polygons
	options.Opacity = 1
	options.Color = func(i int) *glify.RGB {
		rgb := glify.NewRGB()
		c := m.Color[i]
		rgb.R = c.R
		rgb.G = c.G
		rgb.B = c.B
		return rgb
	}
	glify.NewShapes(options)

	log.Println("finished setting polygon colors; setting legend")

	c.doc.GetElementByID("eiolegend").SetInnerHTML(`<img id="legendimg" alt="Embedded Image" src="data:image/png;base64,` + m.Legend + `" />`)
	c.setLegendCSS()
	return nil
}

func (c *client) setCSS() {
	c.setLegendCSS()
	c.setEIOMapCSS()
}

func (c *client) setLegendCSS() {
	img := c.doc.GetElementByID("legendimg")
	if img != nil {
		rect := c.doc.GetElementByID("eiolegend").GetBoundingClientRect()
		img2 := img.(*dom.HTMLImageElement)
		img2.Width = int(rect.Width)
	}
}

func (c *client) setEIOMapCSS() {
	const mapMargin = 50 // This is the height of the nav bar.
	height := dom.GetWindow().InnerHeight()
	eiomap := c.doc.GetElementByID("eiomap").(*dom.HTMLDivElement)
	eiomap.Style().SetProperty("background-color", "black", "")
	eiomap.Style().SetProperty("height", fmt.Sprintf("%dpx", height-mapMargin), "")

}
