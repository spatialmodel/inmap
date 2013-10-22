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
)

func (m *MetData) WebServer() {
	http.HandleFunc("/js/bootstrap.min.js", webframework.ServeJSmin)
	http.HandleFunc("/css/bootstrap.min.css", webframework.ServeCSS)
	http.HandleFunc("/css/bootstrap-responsive.min.css",
		webframework.ServeCSSresponsive)
	http.HandleFunc("/map/", m.mapHandler)
	http.HandleFunc("/", reportHandler)
	http.ListenAndServe(":8080", nil)
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	webframework.RenderHeader(w, "AIM status", "")
	webframework.RenderNav(w, "AIM", nil, nil, "", "")
	const body1 = `
	<div class="container">
		<div class="row">
			<div class="span8">
				<img src="map/PM2_5" class="img-rounded">
			</div>
		</div>
	</div>`
	fmt.Fprint(w, body1)
	webframework.RenderFooter(w, "")
}

func (m *MetData) mapHandler(w http.ResponseWriter, r *http.Request) {
	m.arrayLock.RLock()
	c := m.initialConc[iPM2_5].Copy()
	m.arrayLock.RUnlock()
	pol := "PM2.5"
	b := bufio.NewWriter(w)
	CreateImage(b, c.Subset([]int{0, 0, 0},
		[]int{0, c.Shape[1] - 1, c.Shape[2] - 1}), pol)
	err := b.Flush()
	if err != nil {
		panic(err)
	}
}

// Creates a png image from an array
func CreateImage(w *bufio.Writer, Cf *sparse.DenseArray, pol string) {
	cmap := gis.NewColorMap("LinCutoff")
	cmap.AddArray(Cf.Elements)
	cmap.Set()
	nx := Cf.Shape[1]
	ny := Cf.Shape[0]
	//	cmap.Legend(pol+"_legend.svg", "concentrations (μg/m3)")
	i := image.NewRGBA(image.Rect(0, 0, nx, ny))
	for x := 0; x < nx; x++ {
		for y := 0; y < ny; y++ {
			i.Set(x, y, cmap.GetColor(Cf.Get(ny-y-1, x)))
		}
	}
	err := png.Encode(w, i)
	if err != nil {
		panic(err)
	}
}
