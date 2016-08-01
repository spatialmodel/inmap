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

package inmap

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/carto"
	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/plotutil"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"github.com/gonum/plot/vg/vgimg"
)

// OutputOptions returns the options for output variable names and their
// descriptions.
func (d *InMAP) OutputOptions() (names []string, descriptions []string, units []string) {

	// Model pollutant concentrations
	for pol := range PolLabels {
		names = append(names, pol)
	}
	sort.Strings(names)
	for _, n := range names {
		descriptions = append(descriptions, n+" Concentration")
	}

	// Baseline pollutant concentrations
	var tempBaseline []string
	for pol := range baselinePolLabels {
		tempBaseline = append(tempBaseline, pol)
	}
	sort.Strings(tempBaseline)
	names = append(names, tempBaseline...)
	for _, n := range tempBaseline {
		descriptions = append(descriptions, n+" Concentration")
	}

	// Population and deaths
	var tempPop []string
	var tempDeaths []string
	for pop := range d.popIndices {
		tempPop = append(tempPop, pop)
		tempDeaths = append(tempDeaths, pop+" deaths")
	}
	sort.Strings(tempPop)
	names = append(names, tempPop...)
	names = append(names, tempDeaths...)
	for _, n := range tempPop {
		descriptions = append(descriptions, n+" Population")
	}
	descriptions = append(descriptions, tempDeaths...)

	// Emissions.
	var tempEmis []string
	for pol := range emisLabels {
		tempEmis = append(tempEmis, pol)
	}
	sort.Strings(tempEmis)
	names = append(names, tempEmis...)
	descriptions = append(descriptions, tempEmis...)

	// Eveything else
	t := reflect.TypeOf(*d.cells.first.Cell)
	var tempNames []string
	var tempDescriptions []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := f.Name
		desc := f.Tag.Get("desc")
		if desc != "" {
			tempDescriptions = append(tempDescriptions, desc)
			tempNames = append(tempNames, v)
		}
	}
	names = append(names, tempNames...)
	descriptions = append(descriptions, tempDescriptions...)

	units = make([]string, len(names))
	for i, n := range names {
		units[i] = d.getUnits(n)
	}

	return
}

func parseMapRequest(base string, r *http.Request) (name string,
	layer, zoom, x, y int, err error) {
	request := strings.Split(r.URL.Path[len(base):], "&")
	name = request[0]
	layer, err = s2i(request[1])
	if err != nil {
		return
	}
	zoom, err = s2i(request[2])
	if err != nil {
		return
	}
	x, err = s2i(request[3])
	if err != nil {
		return
	}
	y, err = s2i(request[4])
	if err != nil {
		return
	}
	return
}

func s2i(s string) (int, error) {
	i64, err := strconv.ParseInt(s, 10, 64)
	return int(i64), err
}

func s2f(s string) (float64, error) {
	if s == "************************" { // Null value
		return 0., nil
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err
}

func (d *InMAP) mapHandler(w http.ResponseWriter, r *http.Request) {
	name, layer, z, x, y, err := parseMapRequest("/map/", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	vals := d.toArray(name, layer)
	geometry := d.GetGeometry(layer, true)
	geom2 := make([]geom.Geom, len(geometry))
	for i, g := range geometry {
		geom2[i] = geom.Geom(g)
	}
	m := carto.NewMapData(len(vals), carto.LinCutoff)
	m.Cmap.AddArray(vals)
	m.Cmap.Set()
	m.Shapes = geom2
	m.Data = vals
	//b := bufio.NewWriter(w)
	err = m.WriteGoogleMapTile(w, z, x, y)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//err = b.Flush()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func parseLegendRequest(base string, r *http.Request) (name string,
	layer int, err error) {
	request := strings.Split(r.URL.Path[len(base):], "/")
	name = request[0]
	layer, err = s2i(request[1])
	if err != nil {
		return
	}
	return
}

// LegendHandler creates a legend and serves it.
func (d *InMAP) LegendHandler(w http.ResponseWriter, r *http.Request) {
	name, layer, err := parseLegendRequest("/legend/", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	vals := d.toArray(name, layer)
	cmap := carto.NewColorMap(carto.LinCutoff)
	cmap.AddArray(vals)
	cmap.Set()
	const LegendWidth = 6.2 * vg.Inch
	const LegendHeight = LegendWidth * 0.1067
	cmap.LegendWidth = LegendWidth
	cmap.LegendHeight = LegendHeight
	cmap.LineWidth = 0.5
	cmap.FontSize = 8

	c := vgimg.New(LegendWidth, LegendHeight)
	dc := draw.New(c)
	err = cmap.Legend(&dc, fmt.Sprintf("%v (%v)", name, d.getUnits(name)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = vgimg.PngCanvas{Canvas: c}.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func parseVerticalProfileRequest(base string, r *http.Request) (name string,
	lon, lat float64, err error) {
	request := strings.Split(r.URL.Path[len(base):], "/")
	name = request[0]
	lon, err = s2f(request[1])
	if err != nil {
		return
	}
	lat, err = s2f(request[2])
	if err != nil {
		return
	}
	return
}

func (d *InMAP) verticalProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	name, lon, lat, err := parseVerticalProfileRequest("/verticalProfile/", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// TODO: This need to be converted to grid coordinates, not lat-lon.
	height, vals, err := d.VerticalProfile(name, geom.Point{X: lon, Y: lat})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p, err := plot.New()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.Title.Text = fmt.Sprintf("%v vertical\nprofile at (%.2f, %.2f)", name, lon, lat)
	p.X.Label.Text = "Layer height (m)"
	p.Y.Label.Text = d.getUnits(name)
	xy := make(plotter.XYs, len(height))
	for i, h := range height {
		xy[i].X = h
		xy[i].Y = vals[i]
	}
	err = plotutil.AddLinePoints(p, xy)
	p.Y.Min = 0.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ww, hh := 2.*vg.Inch, 1.5*vg.Inch
	wt, err := p.WriterTo(ww, hh, "png")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = wt.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// VerticalProfile retrieves the vertical profile for a given
// variable at the given location p in the native grid projection.
func (d *InMAP) VerticalProfile(variable string, p geom.Point) (height, vals []float64, err error) {
	if err := d.checkOutputNames(variable); err != nil {
		return nil, nil, err
	}

	height = make([]float64, d.nlayers)
	vals = make([]float64, d.nlayers)
	cells := d.index.SearchIntersect(p.Bounds())
	if len(cells) == 0 {
		return nil, nil, fmt.Errorf("inmap.VerticalProfile: location %+v not in grid", p)
	}
	var c *Cell
	for _, cI := range cells {
		c = cI.(*Cell)
		if c.Layer == 0 {
			break
		}
	}
	if c.Layer != 0 {
		panic("couldn't find a ground level cell.")
	}
	i := 0
	for !c.boundary {
		vals[i] = c.getValue(variable, d.popIndices)
		height[i] = c.LayerHeight + c.Dz/2.
		c = c.above.first.Cell
		i++
	}
	return
}

// HTMLUI returns a function that serves an HTML user interface at address.
// If address is "", then the server won't run.
func HTMLUI(address string) DomainManipulator {
	return func(d *InMAP) error {
		if address != "" {
			errChan := make(chan error)
			go func() {
				errChan <- http.ListenAndServe(address, nil)
			}()
			return <-errChan
		}
		return nil
	}
}
