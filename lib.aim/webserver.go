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

func (d *AIMdata) WebServer() {
	http.HandleFunc("/js/bootstrap.min.js", webframework.ServeJSmin)
	http.HandleFunc("/css/bootstrap.min.css", webframework.ServeCSS)
	http.HandleFunc("/css/bootstrap-responsive.min.css",
		webframework.ServeCSSresponsive)
	http.HandleFunc("/map/", d.mapHandler)
	http.HandleFunc("/proc/", webframework.ProcessorProf)
	http.HandleFunc("/heap/", webframework.HeapProf)
	http.HandleFunc("/", reportHandler)
	http.ListenAndServe(":8080", nil)
}

var mapOptions = []string{"PrimaryPM2_5", "VOC", "SOA", "NH3", "pNH4", "SOx",
	"pSO4", "NOx", "pNO3", "VOCemissions", "NOxemissions", "NH3emissions",
	"SOxemissions", "PM2_5emissions", "U", "V", "W", "Organicpartitioning",
	"Sulfurpartitioning", "Nitratepartitioning", "Ammoniapartitioning",
	"Particlewetdeposition", "SO2wetdeposition",
	"Non-SO2gaswetdeposition", "Kz", "M2u", "M2d", "kPblTop"}

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
			<div class="span6">
				<h4 id="maptitle">PrimaryPM2_5 layer 0 status</h4>
				<div class="row">
					<img id=mapdiv src="map/PrimaryPM2_5/0" class="img-rounded">
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
}
</script>`

	webframework.RenderFooter(w, mapJS)
}

func (d *AIMdata) mapHandler(w http.ResponseWriter, r *http.Request) {
	request := strings.Split(r.URL.Path[len("/map/"):], "/")
	name := request[0]
	layer, err := strconv.ParseInt(request[1], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c := d.ToArray(name)
	pol := "PM2.5"
	b := bufio.NewWriter(w)
	groundlevel := c.Subset([]int{int(layer), 0, 0},
		[]int{int(layer), c.Shape[1] - 1, c.Shape[2] - 1})
	err = CreateImage(b, groundlevel, pol)
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
func CreateImage(w *bufio.Writer, c *sparse.DenseArray, pol string) error {
	cmap := gis.NewColorMap("LinCutoff")
	cmap.AddArray(c.Elements)
	cmap.Set()
	nx := c.Shape[1]
	ny := c.Shape[0]
	//	cmap.Legend(pol+"_legend.svg", "concentrations (μg/m3)")
	i := image.NewRGBA(image.Rect(0, 0, nx, ny))
	for x := 0; x < nx; x++ {
		for y := 0; y < ny; y++ {
			i.Set(x, y, cmap.GetColor(c.Get(ny-y-1, x)))
		}
	}
	return png.Encode(w, i)
}
