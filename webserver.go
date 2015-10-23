/*
Copyright (C) 2013-2014 Regents of the University of Minnesota.
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
	"strconv"
	"strings"

	"bitbucket.org/ctessum/webframework"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/carto"
	"github.com/ctessum/geom/op"
	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/plotutil"
	"github.com/gonum/plot/vg"
)

//  Descriptions of web server map variables
var mapDescriptions []string

// Variable names that go along with descriptions (map[description]variable)
var mapOptions map[string]string

// Names of population types.
var popNames map[string]string

// WebServer provides a HTML user interface for the model.
func (d *InMAPdata) WebServer(httpPort string) {

	// First, set up the options of variables to make maps of
	mapOptions = make(map[string]string)
	mapDescriptions = make([]string, 0)
	for pol := range polLabels { // Concentrations
		mapOptions[pol] = pol
		mapDescriptions = append(mapDescriptions, pol)
	}
	for emis := range emisLabels { // Emissions
		mapDescriptions = append(mapDescriptions, emis)
		mapOptions[emis] = emis
	}
	popNames = make(map[string]string)
	for _, c := range d.Data { // Population and mortalities
		if len(c.PopData) != 0 {
			for pop := range c.PopData {
				popNames[pop] = ""
				mapDescriptions = append(mapDescriptions, pop)
				mapOptions[pop] = pop
				mapDescriptions = append(mapDescriptions, pop+" deaths")
				mapOptions[pop+" deaths"] = pop + " deaths"
			}
			break
		}
	}
	t := reflect.TypeOf(*d.Data[0]) // Everything else
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := f.Name
		desc := f.Tag.Get("desc")
		if desc != "" {
			mapDescriptions = append(mapDescriptions, desc)
			mapOptions[desc] = v
		}
	}

	http.HandleFunc("/js/bootstrap.min.js", webframework.ServeJSmin)
	http.HandleFunc("/css/bootstrap.min.css", webframework.ServeCSS)
	http.HandleFunc("/css/bootstrap-responsive.min.css",
		webframework.ServeCSSresponsive)
	http.HandleFunc("/map/", d.mapHandler)
	http.HandleFunc("/legend/", d.legendHandler)
	http.HandleFunc("/verticalProfile/", d.verticalProfileHandler)
	http.HandleFunc("/proc/", webframework.ProcessorProf)
	http.HandleFunc("/heap/", webframework.HeapProf)
	http.HandleFunc("/", reportHandler)
	http.ListenAndServe(":"+httpPort, nil)
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	const mapStyle = `
<style>
#mapdiv {
	width: 100%;
	height: 600px;
	position: absolute;
	top: 40px;
	z-index: -2;
}
#legendholder {
	position: fixed;
	bottom: 0;
	width: 630px;
	left: 50%;
	margin-left: -315px;
	background:rgba(255,255,255,0.8);
	border-radius:10px;
	z-index: -1;
}
#titleholder {
	position: fixed;
	top: 50px;
	width: 630px;
	left: 50%;
	margin-left: -315px;
	background:rgba(255,255,255,0.8);
	border-radius:10px;
	z-index: -1;
}
#varholder {
	position: fixed;
	left: 80px;
	top: 50px;
	z-index: -1;
}
#layerholder {
	position: fixed;
	left: 80px;
	top: 130px;
	z-index: -1;
}
#mapdiv img {
	max-width: none;
}
</style>`
	webframework.RenderHeader(w, "InMAP status", mapStyle)
	webframework.RenderNav(w, "InMAP", []string{"Home", "Processor", "Memory"},
		[]string{"/", "/proc/", "/heap/"}, "Home", "")

	const body1 = `
	<div id="mapdiv"></div>
	<div id="legendholder">
		<embed id=legenddiv src="/legend/TotalPM2_5/0" type="image/svg+xml" />
	</div>
	<div id="varholder">
		<h5>Select variable</h5>
		<form>
			<select class="span3" id="mapvar" onchange=updateMap()>`
	fmt.Fprintln(w, body1)
	for i, desc := range mapDescriptions {
		if i == 0 {
			fmt.Fprintf(w, "<option value='%v' SELECTED>%v</option>", mapOptions[desc], desc)
		} else {
			fmt.Fprintf(w, "<option value='%v'>%v</option>", mapOptions[desc], desc)
		}
	}
	const body2 = `
			</select>
		</form>
	</div>
	<div id="layerholder">
		<h5>Select layer</h5>
		<form>
			<select class="span1" id="layer" onchange=updateMap()>`
	fmt.Fprintln(w, body2)
	for k := 0; k < 27; k++ {
		if k == 0 {
			fmt.Fprintf(w, "<option SELECTED>%v</option>", k)
		} else {
			fmt.Fprintf(w, "<option>%v</option>", k)
		}
	}
	const body3 = `
			</select>
		</form>
	</div>
	<div id="titleholder">
		<h4 id="maptitle" style="text-align:center">TotalPM2_5 layer 0 status</h4>
	</div>`
	fmt.Fprintln(w, body3)

	const mapJS = `
<script src="https://maps.googleapis.com/maps/api/js?v=3.exp&sensor=false"></script>
<script>
var map;
function tileOptions(layer) {
	var myMapOptions = {
	   getTileUrl: function(coord, zoom) {
	   return "/map/"+window.mapvar+"&"+layer+"&"+zoom+"&"+(coord.x)+"&"+(coord.y);
	   },
	tileSize: new google.maps.Size(256, 256),
	isPng: true,
	opacity: 1,
	name: "custom"
	};
	var customMapType = new google.maps.ImageMapType(myMapOptions);
	return customMapType;
}
function loadmap(mapvar,layer,id) {
	window.mapvar = mapvar
	var windowheight = $(window).height();
	$('#mapdiv').css('height', windowheight-40);
	var customMapType = tileOptions(layer);
	var labelTiles = {
		getTileUrl: function(coord, zoom) {
			return "http://mt0.google.com/vt/v=apt.116&hl=en-US&" +
			"z=" + zoom + "&x=" + coord.x + "&y=" + coord.y + "&client=api";
		},
		tileSize: new google.maps.Size(256, 256),
		isPng: true
	};
	var googleLabelLayer = new google.maps.ImageMapType(labelTiles);

	var latlng = new google.maps.LatLng(40, -97);
	var mapOptions = {
		zoom: 5,
		center: latlng,
		mapTypeId: google.maps.MapTypeId.ROADMAP,
		panControl: true,
		zoomControl: true,
		streetViewControl: false
	}
	map = new google.maps.Map(document.getElementById(id), mapOptions);
	map.overlayMapTypes.insertAt(0, customMapType);
	map.overlayMapTypes.insertAt(1, googleLabelLayer);

	google.maps.event.addListener(map, 'click', function(event) {
		var infoString = '<img src=/verticalProfile/'+window.mapvar+'/'+
			event.latLng.lng()+'/'+event.latLng.lat()+'>'
		new google.maps.InfoWindow({
			position: event.latLng,
			content: infoString
		}).open(map);
	});
}
function updateMap() {
	window.mapvar=document.getElementById("mapvar").value;
	var layer=document.getElementById("layer").value;
	var customMapType = tileOptions(layer);
	map.overlayMapTypes.removeAt(0);
	map.overlayMapTypes.insertAt(0, customMapType);
	document.getElementById("maptitle").innerHTML = window.mapvar+
		" layer "+layer+" status";
	var elem = document.getElementsByTagName("embed")[0],
	copy = elem.cloneNode();
	copy.src = "legend/"+window.mapvar+"/"+layer;
	elem.parentNode.replaceChild(copy, elem);
}
google.maps.event.addDomListener(window, 'load', loadmap("TotalPM2_5",0,"mapdiv"))
</script>`

	webframework.RenderFooter(w, mapJS)
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
	f, err := strconv.ParseFloat(s, 64)
	return f, err
}

func (d *InMAPdata) mapHandler(w http.ResponseWriter, r *http.Request) {
	name, layer, z, x, y, err := parseMapRequest("/map/", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	vals := d.toArray(name, layer)
	geometry := d.GetGeometry(layer)
	m := carto.NewMapData(len(vals), carto.LinCutoff)
	m.Cmap.AddArray(vals)
	m.Cmap.Set()
	m.Shapes = geometry
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

// Creates a legend and serves it.
func (d *InMAPdata) legendHandler(w http.ResponseWriter, r *http.Request) {
	name, layer, err := parseLegendRequest("/legend/", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	vals := d.toArray(name, layer)
	cmap := carto.NewColorMap(carto.LinCutoff)
	cmap.AddArray(vals)
	cmap.Set()
	cmap.LegendWidth = 2.1
	cmap.LegendHeight = 0.2
	cmap.LineWidth = 0.2
	cmap.FontSize = 3.5
	c := carto.NewDefaultLegendCanvas()
	err = cmap.Legend(&c.Canvas, fmt.Sprintf("%v (%v)", name, d.getUnits(name)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = c.WriteTo(w)
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

func (d *InMAPdata) verticalProfileHandler(w http.ResponseWriter,
	r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	name, lon, lat, err := parseVerticalProfileRequest("/verticalProfile/", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	height, vals := d.VerticalProfile(name, lon, lat)
	p, err := plot.New()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.Title.Text = fmt.Sprintf("%v vertical\nprofile at (%.2f, %.2f)",
		name, lon, lat)
	//p.X.Label.Text = "Layer height (m)"
	p.X.Label.Text = "Layer index"
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
// variable at a given location.
func (d *InMAPdata) VerticalProfile(variable string, lon, lat float64) (
	height, vals []float64) {
	height = make([]float64, d.Nlayers)
	vals = make([]float64, d.Nlayers)
	x, y := carto.Degrees2meters(lon, lat)
	loc := geom.Point{X: x, Y: y}
	for _, cell := range d.Data {
		in, err := op.Within(loc, cell.WebMapGeom)
		if err != nil {
			panic(err)
		}
		if in {
			for i := 0; i < d.Nlayers; i++ {
				vals[i] = cell.getValue(variable)
				height[i] = float64(i)
				//if i == 0 {
				//	height[i] = cell.Dz / 2.
				//} else {
				//	height[i] = height[i-1] + cell.DzMinusHalf[0]
				//}
				cell = cell.Above[0]
			}
			return
		}
	}
	return
}
