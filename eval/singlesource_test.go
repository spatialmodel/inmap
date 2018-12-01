package eval

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/ctessum/cdf"
	"github.com/GaryBoone/GoStats/stats"

	"github.com/ctessum/atmos/evalstats"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/carto"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"

	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/spatialmodel/inmap/inmaputil"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

func TestSingleSource(t *testing.T) {

	if testing.Short() {
		return
	}

	evalData := os.Getenv(evalDataEnv)
	if evalData == "" {
		t.Fatalf("please set the '%s' environment variable to the location of the "+
			"downloaded evaluation data and try again", evalDataEnv)
	}

	os.MkdirAll("singleSource", os.ModePerm)

	cfg := inmaputil.InitializeConfig()

	for _, filename := range []string{"configSingleSource_9km.toml", "configSingleSource_nested.toml"} {
		cfg.Set("config", filename)

		cfg.Root.SetArgs([]string{"run", "steady"})
		if err := cfg.Root.Execute(); err != nil {
			t.Fatal(err)
		}
	}

	const (
		figWidth  = 5.75 * vg.Inch
		figHeight = figWidth * 4.65 / 7
		legendH   = 0.4 * vg.Inch
		statsW    = 0.6 * vg.Inch
	)
	states := getShp(filepath.Join(evalData, "states"))
	pop := getPop(evalData)

	c := vgimg.New(figWidth, figHeight)
	dc := draw.New(c)
	statsc := draw.Crop(dc, figWidth-statsW, 0, legendH, 0)
	mainc := draw.Crop(dc, 0, -statsW, legendH, 0)
	legendc1 := draw.Crop(dc, 0, -statsW-(figWidth-statsW)/3., 0, -figHeight+legendH)
	legendc2 := draw.Crop(dc, (figWidth-statsW)/3.*2, -statsW, 0, -figHeight+legendH)
	tiles := draw.Tiles{
		Cols:      3,
		Rows:      2,
		PadLeft:   vg.Points(2),
		PadTop:    vg.Points(2),
		PadBottom: vg.Points(1),
		PadX:      2 * vg.Millimeter,
		PadY:      2 * vg.Millimeter,
	}
	cmap1 := carto.NewColorMap(carto.LinCutoff)
	cmap2 := carto.NewColorMap(carto.LinCutoff)

	gridss := [][]string{
		{"9km", "9km"},
		{"1km", "nested"},
	}

	// Get data.
	gridIndexes := []int{0, 2}
	data := make([]dataHolder, len(gridIndexes))
	for i, grids := range gridss {
		wrfConc, wrfGrid, W, E, S, N := getWRFSingleSource(grids[0], gridIndexes[i], evalData)
		tempInmap := getInMAP(grids[1])
		inmapConc := make([]float64, len(wrfConc))
		for j, g := range wrfGrid {
			for _, iCell := range tempInmap.SearchIntersect(g.Bounds()) {
				c := iCell.(*inmapData)
				isect := g.Intersection(c.Geom.(geom.Polygonal))
				ga := g.Area()
				ia := isect.Area()
				frac := ia / ga
				inmapConc[j] += c.TotalPM25 * frac
			}
		}
		diff := make([]float64, len(wrfConc))
		for j, v := range inmapConc {
			diff[j] = v - wrfConc[j]
		}

		cmap1.AddArray(wrfConc)
		cmap1.AddArray(inmapConc)
		cmap2.AddArray(diff)

		wrfc := carto.NewCanvas(N, S, E, W, tiles.At(mainc, 0, i))
		inmapc := carto.NewCanvas(N, S, E, W, tiles.At(mainc, 1, i))
		diffc := carto.NewCanvas(N, S, E, W, tiles.At(mainc, 2, i))

		data[i] = dataHolder{
			wrfConc:   wrfConc,
			inmapConc: inmapConc,
			diff:      diff,
			wrfGrid:   wrfGrid,
			c:         []*carto.Canvas{wrfc, inmapc, diffc},
			N:         N,
			S:         S,
			E:         E,
			W:         W,
		}
	}

	cmap1.NumDivisions = 8
	cmap1.Set()
	cmap1.Legend(&legendc1, "PM2.5 concentration (μg m-3)")
	cmap2.NumDivisions = 5
	cmap2.Set()
	cmap2.Legend(&legendc2, "Conc. difference (μg m-3)")

	// Create maps.
	for _, d := range data {
		for j, g := range d.wrfGrid {
			bc := cmap1.GetColor(d.wrfConc[j])
			ls := draw.LineStyle{Color: bc, Width: 0.1}
			d.c[0].DrawVector(g, bc, ls, draw.GlyphStyle{})

			bc = cmap1.GetColor(d.inmapConc[j])
			ls = draw.LineStyle{Color: bc, Width: 0.1}
			d.c[1].DrawVector(g, bc, ls, draw.GlyphStyle{})

			bc = cmap2.GetColor(d.diff[j])
			ls = draw.LineStyle{Color: bc, Width: 0.1}
			d.c[2].DrawVector(g, bc, ls, draw.GlyphStyle{})
		}
		// Draw state borders
		stateLineStyle := draw.LineStyle{
			Width: 0.25 * vg.Millimeter,
			Color: color.NRGBA{100, 100, 100, 255},
		}
		var clearFill = color.NRGBA{0, 255, 0, 0}
		for _, cc := range d.c {
			b := geom.Polygon{[]geom.Point{
				{X: d.W, Y: d.S}, {X: d.E, Y: d.S},
				{X: d.E, Y: d.N}, {X: d.W, Y: d.N},
				{X: d.W, Y: d.S}},
			}
			for _, g := range states.SearchIntersect(b.Bounds()) {
				gg := g.(geom.Polygonal).Simplify(1000).(geom.Polygonal)
				gg = gg.Intersection(b)
				cc.DrawVector(gg, clearFill, stateLineStyle, draw.GlyphStyle{})
			}
		}
	}

	// Show insets.
	insetLineStyle := draw.LineStyle{
		Width: 0.4 * vg.Millimeter,
		Color: color.RGBA{0, 0, 0, 255},
	}
	for i := 0; i < 3; i++ {
		inSetUR := data[1].c[i].Coordinates(geom.Point{X: data[1].E, Y: data[1].N})
		inSetUL := data[1].c[i].Coordinates(geom.Point{X: data[1].W, Y: data[1].N})
		inSetLR := data[1].c[i].Coordinates(geom.Point{X: data[1].E, Y: data[1].S})
		inSetLL := data[1].c[i].Coordinates(geom.Point{X: data[1].W, Y: data[1].S})
		mainUL := data[0].c[i].Coordinates(geom.Point{X: data[1].W, Y: data[1].N})
		mainUR := data[0].c[i].Coordinates(geom.Point{X: data[1].E, Y: data[1].N})
		mainLL := data[0].c[i].Coordinates(geom.Point{X: data[1].W, Y: data[1].S})
		mainLR := data[0].c[i].Coordinates(geom.Point{X: data[1].E, Y: data[1].S})
		mainc.StrokeLine2(insetLineStyle, inSetUL.X, inSetUL.Y,
			mainUL.X, mainUL.Y)
		mainc.StrokeLine2(insetLineStyle, inSetUR.X, inSetUR.Y,
			mainUR.X, mainUR.Y)

		mainc.StrokeLine2(insetLineStyle, mainUR.X, mainUR.Y,
			mainUL.X, mainUL.Y)
		mainc.StrokeLine2(insetLineStyle, mainUL.X, mainUL.Y,
			mainLL.X, mainLL.Y)
		mainc.StrokeLine2(insetLineStyle, mainLL.X, mainLL.Y,
			mainLR.X, mainLR.Y)
		mainc.StrokeLine2(insetLineStyle, mainLR.X, mainLR.Y,
			mainUR.X, mainUR.Y)

		mainc.StrokeLine2(insetLineStyle, inSetUR.X, inSetUR.Y,
			inSetUL.X, inSetUL.Y)
		mainc.StrokeLine2(insetLineStyle, inSetUL.X, inSetUL.Y,
			inSetLL.X, inSetLL.Y)
		mainc.StrokeLine2(insetLineStyle, inSetLL.X, inSetLL.Y,
			inSetLR.X, inSetLR.Y)
		mainc.StrokeLine2(insetLineStyle, inSetLR.X, inSetLR.Y,
			inSetUR.X, inSetUR.Y)
	}

	// Write statistics
	statsTiles := draw.Tiles{
		Cols:      1,
		Rows:      2,
		PadLeft:   1 * vg.Millimeter,
		PadTop:    0 * vg.Centimeter,
		PadBottom: 0 * vg.Centimeter,
		PadY:      2 * vg.Millimeter,
	}
	statsTiles2 := draw.Tiles{
		Cols:      1,
		Rows:      2,
		PadTop:    0 * vg.Centimeter,
		PadBottom: 0 * vg.Centimeter,
		PadY:      0.5 * vg.Millimeter,
	}
	statsTiles3 := draw.Tiles{
		Cols: 1,
		Rows: 8,
	}
	labelFont, err := vg.MakeFont("Helvetica", vg.Points(7))
	if err != nil {
		panic(err)
	}
	ts := draw.TextStyle{
		Color: color.Black,
		Font:  labelFont,
	}
	for i, d := range data {
		statc := statsTiles.At(statsc, 0, i)
		slope, _, rsquared, _, _, _ := stats.LinearRegression(d.wrfConc, d.inmapConc)
		mb := evalstats.MB(d.wrfConc, d.inmapConc)
		mfb := evalstats.MFB(d.wrfConc, d.inmapConc) * 100.
		me := evalstats.ME(d.wrfConc, d.inmapConc)
		mfe := evalstats.MFE(d.wrfConc, d.inmapConc) * 100.
		mr := evalstats.MR(d.wrfConc, d.inmapConc)

		vals := []float64{mfb, mfe, mb, me, mr, slope, rsquared}
		labels := []string{"MFB", "MFE", "MB", "ME", "MR", "S", "R2"}
		format := []string{"%.0f%%", "%.0f%%", "%.2f", "%.2f", "%.2f", "%.2f", "%.2f"}
		statcc := statsTiles3.At(statsTiles2.At(statc, 0, 0), 0, 0)
		ts2 := ts
		ts2.YAlign = -0.5
		statcc.FillText(ts2, vg.Point{X: statcc.X(0), Y: statcc.Y(0.5)}, "Area:")
		for j := 1; j < 8; j++ {
			statcc = statsTiles3.At(statsTiles2.At(statc, 0, 0), 0, j)
			statcc.FillText(ts2, vg.Point{X: statcc.X(0), Y: statcc.Y(0.5)},
				fmt.Sprintf("%s: "+format[j-1], labels[j-1], vals[j-1]))
		}

		wrfConcPop, inmapConcPop, gridPop := d.popWeight(pop)
		slopePop, _, rsquaredPop, _, _, _ := stats.LinearRegression(wrfConcPop, inmapConcPop)
		mbPop := evalstats.MBWeighted(d.wrfConc, d.inmapConc, gridPop)
		mfbPop := evalstats.MFBWeighted(d.wrfConc, d.inmapConc, gridPop) * 100.
		mePop := evalstats.MEWeighted(d.wrfConc, d.inmapConc, gridPop)
		mfePop := evalstats.MFEWeighted(d.wrfConc, d.inmapConc, gridPop) * 100.
		mrPop := evalstats.MRWeighted(d.wrfConc, d.inmapConc, gridPop)

		vals = []float64{mfbPop, mfePop, mbPop, mePop, mrPop, slopePop, rsquaredPop}
		statcc = statsTiles3.At(statsTiles2.At(statc, 0, 1), 0, 0)
		statcc.FillText(ts2, vg.Point{X: statcc.X(0), Y: statcc.Y(0.5)}, "Population:")
		for j := 1; j < 8; j++ {
			statcc := statsTiles3.At(statsTiles2.At(statc, 0, 1), 0, j)
			statcc.FillText(ts2, vg.Point{X: statcc.X(0), Y: statcc.Y(0.5)},
				fmt.Sprintf("%s: "+format[j-1], labels[j-1], vals[j-1]))
		}

		panelLabels := [][]string{
			{"a) WRF-Chem 9 km", "b) InMAP 9 km", "c) InMAP minus WRF-Chem"},
			{"d) WRF-Chem 1 km", "e) InMAP 1–27 km variable grid", "f) InMAP minus WRF-Chem"},
		}
		// write labels
		ts3 := ts
		ts3.YAlign = -1
		for i, d := range data {
			for ii, cc := range d.c {
				cc.FillText(ts3, vg.Point{X: cc.X(0) + vg.Points(2), Y: cc.Y(1) - vg.Points(2)}, panelLabels[i][ii])
			}
		}
	}

	f, err := os.Create("singleSource/comparison.png")
	handle(err)
	png := vgimg.PngCanvas{Canvas: c}
	_, err = png.WriteTo(f)
	handle(err)
}

type dataHolder struct {
	wrfConc, inmapConc, diff []float64
	N, S, E, W               float64
	wrfGrid                  []geom.Polygonal
	c                        []*carto.Canvas
}

func (d *dataHolder) popWeight(pop *rtree.Rtree) (wrf, inmap, totalPop []float64) {
	totalPop = make([]float64, len(d.wrfGrid))
	wrf = make([]float64, len(d.wrfGrid))
	inmap = make([]float64, len(d.wrfGrid))
	for i, g := range d.wrfGrid {
		for _, gPg := range pop.SearchIntersect(g.Bounds()) {
			p := gPg.(popHolder)
			isect := p.Geom.(geom.Polygonal).Intersection(g)
			aI := isect.Area()
			aP := p.Geom.(geom.Polygonal).Area()
			totalPop[i] += p.TotalPop * aI / aP
		}
	}
	popSum := floats.Sum(totalPop)

	for i, v := range d.wrfConc {
		wrf[i] = v * totalPop[i] / popSum
	}
	for i, v := range d.inmapConc {
		inmap[i] = v * totalPop[i] / popSum
	}

	return
}

type inmapData struct {
	geom.Geom
	TotalPM25 float64
}

func getInMAP(gridType string) *rtree.Rtree {
	t := rtree.NewTree(25, 50)
	filename := fmt.Sprintf("singleSource/LosAngeles_%s.shp", gridType)
	e, err := shp.NewDecoder(filename)
	handle(err)
	for {
		var rec inmapData
		more := e.DecodeRow(&rec)
		if !more {
			break
		}
		t.Insert(&rec)
	}
	handle(e.Error())
	return t
}

func getWRFSingleSource(gridType string, gridIndex int, evalData string) ([]float64, []geom.Polygonal, float64, float64, float64, float64) {
	filename := fmt.Sprintf(filepath.Join(evalData, "la_test/InMAPData_%s.ncf"), gridType)
	f := openNCF(filename)
	data := f.readVar("TotalPM25")

	wrfNamelist := "la_test/namelist.input"
	wpsNamelist := "la_test/namelist.wps"

	d, err := aep.ParseWRFConfig(wpsNamelist, wrfNamelist)
	if err != nil {
		panic(err)
	}
	sr, err := proj.Parse("+proj=longlat") // placeholder
	handle(err)
	grid := aep.NewGridRegular(d.DomainNames[gridIndex], d.Nx[gridIndex], d.Ny[gridIndex],
		d.Dx[gridIndex], d.Dy[gridIndex], d.W[gridIndex], d.S[gridIndex], sr)
	g := make([]geom.Polygonal, len(grid.Cells))
	for i, c := range grid.Cells {
		g[i] = c.Polygonal
	}
	W := d.W[gridIndex]
	E := d.W[gridIndex] + d.Dx[gridIndex]*float64(d.Nx[gridIndex])
	S := d.S[gridIndex]
	N := d.S[gridIndex] + d.Dy[gridIndex]*float64(d.Ny[gridIndex])
	return data, g, W, E, S, N
}

type ncfSingleSource struct {
	ff *os.File
	f  *cdf.File
}

func openNCF(fname string) *ncfSingleSource {
	f := new(ncfSingleSource)
	var err error
	f.ff, err = os.Open(fname)
	if err != nil {
		panic(err)
	}
	f.f, err = cdf.Open(f.ff)
	if err != nil {
		panic(err)
	}
	return f
}

func (f *ncfSingleSource) readVar(name string) []float64 {
	const nx, ny = 33, 33
	r := f.f.Reader(name, nil, nil)
	buf := r.Zero(-1)
	_, err := r.Read(buf)
	if err != nil {
		panic(err)
	}
	dat32 := buf.([]float32)
	dat64 := make([]float64, nx*ny)
	ii := 0
	for i := 0; i < nx; i++ {
		for j := 0; j < ny; j++ {
			dat64[ii] = float64(dat32[j*nx+i])
			ii++
		}
	}
	return dat64
}

func getShp(filename string) *rtree.Rtree {
	s, err := shp.NewDecoder(filename + ".shp")
	handle(err)
	defer s.Close()

	src, err := s.SR()
	handle(err)
	const (
		// GridProj is the grid projection
		GridProj = "+proj=lcc +lat_1=26.0 +lat_2=39.0 +lat_0=34.11 +lon_0=-118.18 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1"
	)
	dst, err := proj.Parse(GridProj)
	handle(err)
	ct, err := src.NewTransform(dst)
	handle(err)

	r := rtree.NewTree(25, 50)
	for {
		var o struct {
			geom.Geom
		}
		if !s.DecodeRow(&o) {
			break
		}
		gg, err := o.Geom.Transform(ct)
		if err != nil {
			continue // probably the north pole
		}
		r.Insert(gg)
	}
	handle(s.Error())
	return r
}

func getPop(evalData string) *rtree.Rtree {
	filename := filepath.Join(evalData, "census2015blckgrp")
	s, err := shp.NewDecoder(filename + ".shp")
	handle(err)
	defer s.Close()

	src, err := s.SR()
	handle(err)
	const (
		// GridProj is the grid projection
		GridProj = "+proj=lcc +lat_1=26.0 +lat_2=39.0 +lat_0=34.11 +lon_0=-118.18 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1"
	)
	dst, err := proj.Parse(GridProj)
	handle(err)
	ct, err := src.NewTransform(dst)
	handle(err)

	r := rtree.NewTree(25, 50)
	for {
		var o popHolder
		if !s.DecodeRow(&o) {
			break
		}
		o.Geom, err = o.Geom.Transform(ct)
		if err != nil {
			continue // probably the north pole
		}
		r.Insert(o)
	}
	handle(s.Error())
	return r
}

func handle(err error) {
	if err != nil {
		panic(err)
	}
}
