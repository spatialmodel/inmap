package aim

import (
	"bitbucket.org/ctessum/gis"
	"bitbucket.org/ctessum/sparse"
	"bitbucket.org/ctessum/webframework"
	"bufio"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"strconv"
	"strings"
)

func (d *AIMdata) WebServer(httpPort string) {
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
	"SOxemissions", "PM2_5emissions", "U", "V", "W", "Organicpartitioning",
	"Sulfurpartitioning", "Nitratepartitioning", "Ammoniapartitioning",
	"Particlewetdeposition", "SO2wetdeposition",
	"Non-SO2gaswetdeposition", "Kz", "M2u", "M2d", "kPblTop", "velocityImbalance"}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	webframework.RenderHeader(w, "AIM status", "")
	webframework.RenderNav(w, "AIM", []string{"Home", "Processor", "Memory"},
		[]string{"/", "/proc/", "/heap/"}, "Home", "")

	const body1 = `
	<div class="container">
		<div class="row">
			<div class="span4">
				<h5>Select variable</h5>
				<form>
					<select class="span4" id="mapvar" multiple="multiple" size=20 onchange=updateImage()>`
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
					<select class="span1" id="layer" multiple="multiple" size=20 onchange=updateImage()>`
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
			<div class="span6 pagination-centered">
				<h4 id="maptitle">PrimaryPM2_5 layer 0 status</h4>
				<div class="row">
					<img id=mapdiv src="map/PrimaryPM2_5/0" class="img-rounded">
				</div>
				<div class="row">
					<embed id=legenddiv src="/legend/PrimaryPM2_5/0" type="image/svg+xml" />
				</div>
			</div>
		</div>
	</div>`
	fmt.Fprintln(w, body3)

	const mapJS = `<script>
function updateImage() {
	var mapvar=document.getElementById("mapvar").value;
	var layer=document.getElementById("layer").value;
	document.getElementById("mapdiv").src = "map/"+mapvar+"/"+layer;
	document.getElementById("maptitle").innerHTML = mapvar+" layer "+layer+" status";
	var elem = document.getElementsByTagName("embed")[0],
	copy = elem.cloneNode();
	copy.src = "legend/"+mapvar+"/"+layer;
	elem.parentNode.replaceChild(copy, elem);
}
</script>`

	webframework.RenderFooter(w, mapJS)
}

func parseMapRequest(base string, r *http.Request) (name string, layer int, err error) {
	var layer64 int64
	request := strings.Split(r.URL.Path[len(base):], "/")
	name = request[0]
	layer64, err = strconv.ParseInt(request[1], 10, 64)
	layer = int(layer64)
	return
}

func (d *AIMdata) mapHandler(w http.ResponseWriter, r *http.Request) {
	name, layer, err := parseMapRequest("/map/", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c := d.ToArray(name, "instantaneous")
	b := bufio.NewWriter(w)
	layerSubset := c.Subset([]int{int(layer), 0, 0},
		[]int{int(layer), c.Shape[1] - 1, c.Shape[2] - 1})
	err = CreateImage(b, layerSubset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = b.Flush()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Creates a png image from an array
func CreateImage(w *bufio.Writer, c *sparse.DenseArray) error {
	cmap := gis.NewColorMap("LinCutoff")
	cmap.AddArray(c.Elements)
	cmap.Set()
	nx := c.Shape[1]
	ny := c.Shape[0]
	i := image.NewRGBA(image.Rect(0, 0, nx, ny))
	for x := 0; x < nx; x++ {
		for y := 0; y < ny; y++ {
			i.Set(x, y, cmap.GetColor(c.Get(ny-y-1, x)))
		}
	}
	return png.Encode(w, i)
}

// Creates a legend and serves it.
func (d *AIMdata) legendHandler(w http.ResponseWriter, r *http.Request) {
	name, layer, err := parseMapRequest("/legend/", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	c := d.ToArray(name, "instantaneous")
	cmap := gis.NewColorMap("LinCutoff")
	layerSubset := c.Subset([]int{int(layer), 0, 0},
		[]int{int(layer), c.Shape[1] - 1, c.Shape[2] - 1})
	cmap.AddArray(layerSubset.Elements)
	cmap.Set()
	cmap.LegendWidth = 2.1
	cmap.LegendHeight = 0.2
	cmap.LineWidth = 0.2
	cmap.FontSize = 3.5
	err = cmap.Legend(w, "concentrations (μg/m3)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
