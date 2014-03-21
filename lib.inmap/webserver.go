package inmap

import (
	"bitbucket.org/ctessum/gis"
	"bitbucket.org/ctessum/webframework"
	"fmt"
	//"github.com/pmylund/go-cache"
	"net/http"
	"strconv"
	"strings"
)

func (d *InMAPdata) WebServer(httpPort string) {
	http.HandleFunc("/js/bootstrap.min.js", webframework.ServeJSmin)
	http.HandleFunc("/css/bootstrap.min.css", webframework.ServeCSS)
	http.HandleFunc("/css/bootstrap-responsive.min.css",
		webframework.ServeCSSresponsive)
	http.HandleFunc("/map/", d.mapHandler)
	http.HandleFunc("/legend/", d.legendHandler)
	http.HandleFunc("/proc/", webframework.ProcessorProf)
	http.HandleFunc("/heap/", webframework.HeapProf)
	http.HandleFunc("/", reportHandler)
	http.ListenAndServe(":"+httpPort, nil)
}

var mapOptions = []string{"PrimaryPM2_5", "VOC", "SOA", "NH3", "pNH4", "SOx",
	"pSO4", "NOx", "pNO3", "VOCemissions", "NOxemissions", "NH3emissions",
	"SOxemissions", "PM2_5emissions", "UPlusSpeed", "UMinusSpeed",
	"VPlusSpeed", "VMinusSpeed", "WPlusSpeed", "WMinusSpeed",
	"Organicpartitioning", "Sulfurpartitioning", "Nitratepartitioning",
	"Ammoniapartitioning", "Particlewetdeposition", "SO2wetdeposition",
	"Non-SO2gaswetdeposition", "Kxxyy", "Kzz", "M2u", "M2d", "PblTopLayer",
	"Total deaths", "White deaths", "Non-white deaths",
	"High income deaths", "Low income deaths",
	"High income white deaths", "Low income non-white deaths","Population",
	"Baseline mortality rate"}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	const mapStyle = `
<style>
#mapdiv {
	width: 100%;
	height: 400px;
}
</style>`
	webframework.RenderHeader(w, "AIM status", mapStyle)
	webframework.RenderNav(w, "AIM", []string{"Home", "Processor", "Memory"},
		[]string{"/", "/proc/", "/heap/"}, "Home", "")

	const body1 = `
	<div class="container">
		<div class="row">
			<div class="span3">
				<h5>Select variable</h5>
				<form>
					<select class="span3" id="mapvar" multiple="multiple" size=20 onchange=updateMap()>`
	fmt.Fprintln(w, body1)
	for i, option := range mapOptions {
		if i == 0 {
			fmt.Fprintf(w, "<option SELECTED>%v</option>", option)
		} else {
			fmt.Fprintf(w, "<option>%v</option>", option)
		}
	}
	const body2 = `
					</select>
				</form>
			</div>
			<div class="span2">
				<h5>Select layer</h5>
				<form>
					<select class="span1" id="layer" multiple="multiple" size=20 onchange=updateMap()>`
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
			<div class="span7 pagination-centered">
				<h4 id="maptitle">PrimaryPM2_5 layer 0 status</h4>
				<div class="row">
					<div id="mapdiv"></div>
				</div>
				<div class="row">
					<embed id=legenddiv src="/legend/PrimaryPM2_5/0" type="image/svg+xml" />
				</div>
			</div>
		</div>
	</div>`
	fmt.Fprintln(w, body3)

	const mapJS = `
<script src="https://maps.googleapis.com/maps/api/js?v=3.exp&sensor=false"></script>
<script>
var map;
function tileOptions(mapvar,layer) {
	var myMapOptions = {
	   getTileUrl: function(coord, zoom) {
	   return "/map/"+mapvar+"&"+layer+"&"+zoom+"&"+(coord.x)+"&"+(coord.y);
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
	var customMapType = tileOptions(mapvar,layer);
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
		zoom: 4,
		center: latlng,
		mapTypeId: google.maps.MapTypeId.ROADMAP,
		panControl: true,
		zoomControl: true,
		streetViewControl: false
	}
	map = new google.maps.Map(document.getElementById(id), mapOptions);
	map.overlayMapTypes.insertAt(0, customMapType);
	map.overlayMapTypes.insertAt(1, googleLabelLayer);
}
function updateMap() {
	var mapvar=document.getElementById("mapvar").value;
	var layer=document.getElementById("layer").value;
	var customMapType = tileOptions(mapvar,layer);
	map.overlayMapTypes.removeAt(0);
	map.overlayMapTypes.insertAt(0, customMapType);
	document.getElementById("maptitle").innerHTML = mapvar+" layer "+layer+" status";
	var elem = document.getElementsByTagName("embed")[0],
	copy = elem.cloneNode();
	copy.src = "legend/"+mapvar+"/"+layer;
	elem.parentNode.replaceChild(copy, elem);
}
google.maps.event.addDomListener(window, 'load', loadmap("PrimaryPM2_5",0,"mapdiv"))
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

func (d *InMAPdata) mapHandler(w http.ResponseWriter, r *http.Request) {
	name, layer, z, x, y, err := parseMapRequest("/map/", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	vals := d.toArray(name, layer)
	geometry := d.getGeometry(layer)
	m := gis.NewMapData(len(vals), "LinCutoff")
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
	cmap := gis.NewColorMap("LinCutoff")
	cmap.AddArray(vals)
	cmap.Set()
	cmap.LegendWidth = 2.1
	cmap.LegendHeight = 0.2
	cmap.LineWidth = 0.2
	cmap.FontSize = 3.5
	err = cmap.Legend(w, "concentrations (μg/m³")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
