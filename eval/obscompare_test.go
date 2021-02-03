package eval

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GaryBoone/GoStats/stats"
	"github.com/ctessum/cdf"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/carto"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

const (
	// gridProj is the spatial projection of the InMAP grid.
	gridProj = "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1"
)

var varnames = []string{
	"Total PM2_5",
	"Sulfur dioxide",
	"Particulate sulfate",
	"Gaseous oxide of nitrogen compounds",
	"Particulate nitrate",
	"Gaseous ammonia",
	"Particulate ammonium",
}

var inmapVars = []string{
	"TotalPM25",
	"SOx",
	"PSO4",
	"NOx",
	"PNO3",
	"NH3",
	"PNH4",
}

var figNames = []string{
	"fig6",
	"figC13",
	"fig7",
	"figC15",
	"fig9",
	"figC14",
	"fig8",
}

var wrfVars = []string{
	"TotalPM25",
	"gS",
	"pS",
	"gNO",
	"pNO",
	"gNH",
	"pNH",
	"alt"}

var states []geom.Polygon // State shapes

// Chemical mass conversions
const (
	// grams per mole
	mwNOx = 46.0055
	mwN   = 14.0067
	mwNO3 = 62.00501
	mwNH3 = 17.03056
	mwNH4 = 18.03851
	mwS   = 32.0655
	mwSO2 = 64.0644
	mwSO4 = 96.0632
	// ratios
	NOxToN = mwN / mwNOx
	NtoNO3 = mwNO3 / mwN
	SOxToS = mwS / mwSO2
	StoSO4 = mwSO4 / mwS
	NH3ToN = mwN / mwNH3
	NtoNH4 = mwNH4 / mwN
)

// covert from μg/m3 of ion to total μg/m3 of compound
// This is necessary because we're using the wrf data
// from the inmap preprocessor output which is in the ion
// format.
var wrfConv = []float64{
	1.,
	1. / SOxToS,
	StoSO4,
	1. / NOxToN,
	NtoNO3,
	1. / NH3ToN,
	NtoNH4,
	1.,
}

var mw = []float64{
	1.,
	mwSO2,
	mwSO4,
	mwNOx,
	mwNO3,
	mwNH3,
	mwNH4,
}

var obsVars = []string{
	"PM2.5 - Local Conditions",
	"Sulfur dioxide",
	"Sulfate PM2.5 LC",
	"Nitrogen dioxide (NO2)",
	"Total Nitrate PM2.5 LC",
	"Ammonia",
	"Ammonium Ion PM2.5 LC",
}

//"Nitric oxide (NO)",

var (
	figWidth  = 7 * vg.Inch
	figHeight = 2.7 * vg.Inch
)

// obsFile is observation data from
// http://aqsdr1.epa.gov/aqsweb/aqstmp/airdata/download_files.html#Annual
func obsCompare(inmapDataLoc, wrfDataLoc, obsFile, statesLoc, outDir, fileprefix string) error {
	plot.DefaultFont = "Helvetica"

	states = getStates(statesLoc, 10000)
	fmt.Println("Getting data")
	iChan := make(chan *rtree.Rtree)
	go getInMAPdata(inmapDataLoc, iChan)
	wChan := make(chan []*rtree.Rtree)
	go getWRFdata(wrfDataLoc, wChan)
	inmapData := <-iChan
	wrfData := <-wChan
	alt := wrfData[len(wrfVars)-1]
	oChan := make(chan [][]*data)
	go getObsData(obsFile, oChan, alt)
	obsData := <-oChan
	fmt.Println("Finished getting data")

	captionChans := make([]chan *matchObsResult, len(inmapVars))

	canvases := make(map[int]draw.Canvas)
	for iPol, polName := range inmapVars {
		captionChans[iPol] = make(chan *matchObsResult)

		canvases[iPol] = draw.New(vgimg.NewWith(vgimg.UseWH(figWidth, figHeight), vgimg.UseDPI(96)))
		//canvases[iPol] = draw.New(vgimg.New(figWidth, figHeight)) //, vgimg.DPI(300)))
		//canvases[iPol] = draw.New(vgpdf.New(figWidth, figHeight))
		//canvases[iPol] = draw.New(vgsvg.New(figWidth, figHeight))

		go matchObs(polName, obsData[iPol], wrfData[iPol],
			inmapData, captionChans[iPol], canvases[iPol])
	}

	var results []matchObsResult
	for iPol, cChan := range captionChans {
		//polName := inmapVars[iPol]
		fname := filepath.Join(outDir, fileprefix+"_"+inmapVars[iPol]+".png")
		obsresult := <-cChan
		results = append(results, *obsresult)

		f, err := os.Create(fname)
		if err != nil {
			panic(err)
		}
		_, err = vgimg.PngCanvas{Canvas: canvases[iPol].Canvas.(*vgimg.Canvas)}.WriteTo(f)
		//_, err = canvases[iPol].Canvas.(*vgpdf.Canvas).WriteTo(f)
		//_, err = canvases[iPol].Canvas.(*vgsvg.Canvas).WriteTo(f)
		if err != nil {
			panic(err)
		}
		f.Close()
	}
	f, err := os.Create(filepath.Join(outDir, fileprefix+"_modelMeasurementComparisons.json"))
	if err != nil {
		return err
	}

	b, err := json.Marshal(results)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	json.Indent(&out, b, "", "\t")
	out.WriteTo(f)
	f.Close()
	return nil
}

type data struct {
	geom.Geom
	val float64
}

// Match observations with modeled values
func matchObs(polName string, obsData []*data, wrfData,
	inmapData *rtree.Rtree,
	captionChan chan *matchObsResult, c draw.Canvas) {
	locs := make([]geom.Geom, 0, len(obsData))
	obsMatches := make([]float64, 0, len(obsData))
	inmapMatches := make([]float64, 0, len(obsData))
	wrfMatches := make([]float64, 0, len(obsData))
	inmapDiff := make([]float64, 0, len(obsData))
	wrfDiff := make([]float64, 0, len(obsData))

	for _, o := range obsData {
		matchTemp := inmapData.SearchIntersect(o.Bounds())
		if len(matchTemp) == 0 {
			continue
		} else if len(matchTemp) > 1 {
			panic("more than one match")
		}
		locs = append(locs, o.Geom)
		obsMatches = append(obsMatches, o.val)
		inmapVal := getInmapVal(polName, matchTemp[0].(*iData))
		inmapMatches = append(inmapMatches, inmapVal)
		inmapDiff = append(inmapDiff, inmapVal-o.val)

		matchTemp = wrfData.SearchIntersect(o.Bounds())
		if len(matchTemp) != 1 {
			panic(fmt.Sprintf("Wrong number of matches: %v; loc=%v",
				len(matchTemp), o.Geom))
		}
		wrfMatches = append(wrfMatches, matchTemp[0].(*data).val)
		wrfDiff = append(wrfDiff, matchTemp[0].(*data).val-o.val)
	}
	if len(inmapMatches) == 0 {
		fmt.Println("No matches for " + polName)
		return
	}

	labelFont, err := vg.MakeFont(plot.DefaultFont, vg.Points(7))
	if err != nil {
		panic(err)
	}
	ts := draw.TextStyle{
		Color: color.Black,
		Font:  labelFont,
	}

	// make maps
	mapWidth := 2.25 * vg.Inch
	mapHeight := 1.75 * vg.Inch
	scatterHeight := 2.24 * vg.Inch
	legendWidth := 4.5 * vg.Inch
	legendHeight := 0.4 * vg.Inch
	legendHspace := 2 * vg.Millimeter
	cWRF := draw.Crop(c, 0, mapWidth-figWidth, figHeight-mapHeight, 0)
	cInMAP := draw.Crop(c, mapWidth, 2*mapWidth-figWidth, figHeight-mapHeight, 0)
	cScatter := draw.Crop(c, 2*mapWidth, -2, figHeight-scatterHeight, -2)
	cConcLegend := draw.Crop(c, 0, legendWidth-figWidth, legendHeight+legendHspace,
		2*legendHeight+legendHspace-figHeight)
	cObsLegend := draw.Crop(c, 0, legendWidth-figWidth, 0, legendHeight-figHeight)
	cStats := draw.Crop(c, legendWidth, -2, 0, -scatterHeight)

	wrfmap, bounds := newMap(cWRF)
	inmapmap, _ := newMap(cInMAP)

	markerGlyph := draw.GlyphStyle{
		Radius: 0.5 * vg.Millimeter,
		Shape:  draw.CircleGlyph{},
	}
	lineStyle := draw.LineStyle{Width: 0.25 * vg.Millimeter}

	var wrfg []geom.Geom
	var wrfd []float64
	for _, valI := range wrfData.SearchIntersect(bounds) {
		val := valI.(*data)
		wrfg = append(wrfg, val.Geom)
		wrfd = append(wrfd, val.val)
	}
	var inmapg []geom.Geom
	var inmapd []float64
	for _, valI := range inmapData.SearchIntersect(bounds) {
		val := valI.(*iData)
		inmapg = append(inmapg, val.Geom)
		inmapd = append(inmapd, getInmapVal(polName, val))
	}
	cmapConc := carto.NewColorMap(carto.LinCutoff)
	cmapConc.Font = plot.DefaultFont
	//cmapConc.ColorScheme = carto.JetPosOnly
	cmapConc.AddArray(append(inmapd, wrfd...))
	cmapConc.Set()

	for i, v := range wrfd {
		color := cmapConc.GetColor(v)
		markerGlyph.Color = color
		lineStyle.Color = color
		wrfmap.DrawVector(wrfg[i], color, lineStyle, markerGlyph)
	}
	for i, v := range inmapd {
		color := cmapConc.GetColor(v)
		markerGlyph.Color = color
		lineStyle.Color = color
		inmapmap.DrawVector(inmapg[i], color, lineStyle, markerGlyph)
	}

	cmap := carto.NewColorMap(carto.Linear)
	cmap.Font = plot.DefaultFont
	cmap.AddArray(wrfDiff)
	cmap.AddArray(inmapDiff)
	cmap.Set()

	// Draw states
	for _, m := range []*carto.Canvas{wrfmap, inmapmap} {
		var stroke = color.NRGBA{0, 0, 0, 255}
		markerGlyph.Color = stroke
		lineStyle.Color = stroke
		var fill = color.NRGBA{0, 255, 0, 0}
		for _, g := range states {
			m.DrawVector(g, fill, lineStyle, markerGlyph)
		}
	}
	for i, v := range wrfDiff {
		color := cmap.GetColor(v)
		markerGlyph.Color = color
		lineStyle.Color = color
		wrfmap.DrawVector(locs[i], color, lineStyle, markerGlyph)
	}
	for i, v := range inmapDiff {
		color := cmap.GetColor(v)
		markerGlyph.Color = color
		lineStyle.Color = color
		inmapmap.DrawVector(locs[i], color, lineStyle, markerGlyph)
	}

	ts2 := ts
	ts2.XAlign = -0.5
	ts2.YAlign = -1
	wrfmap.FillText(ts2, vg.Point{X: wrfmap.X(0.5), Y: wrfmap.Max.Y - 0.1*vg.Inch}, "WRF-Chem")
	inmapmap.FillText(ts2, vg.Point{X: inmapmap.X(0.5), Y: inmapmap.Max.Y - 0.1*vg.Inch}, "InMAP")

	wrfstats, inmapstats := makePlot(obsMatches, wrfMatches, inmapMatches,
		"", cScatter)

	err = cmapConc.Legend(&cConcLegend, "Concentration (μg/m³)")
	if err != nil {
		panic(err)
	}
	err = cmap.Legend(&cObsLegend, "Model-measurement difference (μg/m³)")
	if err != nil {
		panic(err)
	}

	colspace := 0.3 * vg.Inch
	left := cStats.Min.X + 0.8*vg.Inch
	rowspace := 0.12 * vg.Inch
	top := cStats.Max.Y - 0.1*vg.Inch
	ts3 := ts
	ts3.XAlign = -0.5
	ts3.YAlign = -0.5
	for i, s := range []string{"MFB", "MFE", "MB", "ME",
		"S", "R²"} {
		cStats.FillText(ts3, vg.Point{X: left + vg.Length(i)*colspace, Y: top}, s)
	}
	types := []string{"WRF-Chem", "InMAP"}
	format := []string{"%.0f%%", "%.0f%%", "%.1f", "%.1f", "%.2f",
		"%.2f"}
	ts4 := ts
	ts4.XAlign = -1
	ts4.YAlign = -0.5
	for j, ss := range []*statistics{wrfstats, inmapstats} {
		cStats.FillText(ts4, vg.Point{X: left - 0.2*vg.Inch, Y: top - vg.Length(j+1)*rowspace}, types[j])
		for i, s := range []interface{}{ss.mfb * 100, ss.mfe * 100,
			ss.mb, ss.me, ss.slope, ss.rsquared} { //ss.intercept,
			cStats.FillText(ts3, vg.Point{X: left + vg.Length(i)*colspace,
				Y: top - vg.Length(j+1)*rowspace},
				fmt.Sprintf(format[i], s))
		}
	}

	captionChan <- &matchObsResult{
		ComparisonName:     polName,
		Locations:          locs,
		Measurements:       obsMatches,
		InMAPPredictions:   inmapMatches,
		WRFChemPredictions: wrfMatches,
	}
}

type matchObsResult struct {
	ComparisonName                                     string
	Locations                                          []geom.Geom
	Measurements, InMAPPredictions, WRFChemPredictions []float64
}

func getObsData(obsFile string, oChan chan [][]*data, alts *rtree.Rtree) {
	dst, err := proj.Parse(gridProj)
	if err != nil {
		panic(err)
	}
	obsData := make([][]*data, len(obsVars))
	for i := 0; i < len(obsVars); i++ {
		obsData[i] = make([]*data, 0, 10000)
	}
	f, err := os.Open(obsFile)
	if err != nil {
		panic(err)
	}
	c := csv.NewReader(f)
	for {
		rec, err := c.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		parameter := rec[8]
		polI := findIndex(parameter, obsVars)
		if polI < 0 {
			continue
		}
		metric := rec[11]
		switch metric {
		case "Daily Mean", "Observed Values", "Observed values":
		default:
			//fmt.Println(metric)
			continue
		}

		o := new(data)
		lat := s2f(rec[5])
		lon := s2f(rec[6])
		o.Geom = geom.Point{X: lon, Y: lat}
		datum := rec[7]
		if datum == "UNKNOWN" {
			datum = "WGS84"
		}
		src, err := proj.Parse("+proj=longlat +datum=" + datum)
		if err != nil {
			panic(err)
		}
		ct, err := src.NewTransform(dst)
		if err != nil {
			panic(err)
		}
		o.Geom, err = o.Geom.Transform(ct)
		if err != nil {
			panic(err)
		}
		o.val = s2f(rec[26])
		altTemp := alts.SearchIntersect(o.Bounds())
		if len(altTemp) == 0 {
			continue
		} else if len(altTemp) > 1 {
			panic("more than one match")
		}
		alt := altTemp[0].(*data).val

		units := rec[14]
		switch units {
		case "Micrograms/cubic meter (LC)":
		case "Parts per billion":
			o.val *= 1. / alt / 28.97 * mw[polI]
		case "Parts per million":
			o.val *= 1000. / alt / 28.97 * mw[polI]
		default:
			fmt.Println(rec)
			fmt.Println(rec[12])
			fmt.Println(units)
			continue
		}
		obsData[polI] = append(obsData[polI], o)
	}
	fmt.Println("Finished getting observation data")
	oChan <- obsData
}

type iData struct {
	geom.Geom
	TotalPM25                       float64
	SOx, PSO4, NOx, PNO3, NH3, PNH4 float64
}

func getInMAPdata(inmapDataLoc string, iChan chan *rtree.Rtree) {
	out := rtree.NewTree(25, 50)

	// open a shapefile for reading
	shape, err := shp.NewDecoder(inmapDataLoc)
	if err != nil {
		panic(err)
	}
	defer shape.Close()

	mapData := make([]*iData, shape.AttributeCount())
	// loop through all features in the shapefile
	n := 0
	for {
		var o iData
		if !shape.DecodeRow(&o) {
			break
		}

		out.Insert(&o)
		mapData[n] = &o
		n++
	}
	if shape.Error() != nil {
		panic(shape.Error())
	}

	fmt.Println("Finished getting InMAP data")
	iChan <- out
}

func getWRFdata(wrfDataLoc string, wChan chan []*rtree.Rtree) {

	const (
		W  = -2736000.00
		S  = -2088000.00
		E  = 2592000.00
		N  = 1944000.00
		dx = 12000.
		dy = 12000.
		nx = 444
		ny = 336
	)

	ff, err := os.Open(wrfDataLoc)
	if err != nil {
		panic(err)
	}
	defer ff.Close()
	f, err := cdf.Open(ff)
	if err != nil {
		panic(err)
	}
	out := make([]*rtree.Rtree, len(wrfVars))
	for ii, polName := range wrfVars {
		r := f.Reader(polName, []int{0, 0, 0}, []int{0, ny - 1, nx - 1})
		buf := r.Zero(-1)
		_, err = r.Read(buf)
		if err != nil {
			panic(err)
		}
		out[ii] = rtree.NewTree(25, 50)
		mapData := make([]*data, len(buf.([]float32)))
		for i, v := range buf.([]float32) {
			o := new(data)
			o.val = float64(v) * wrfConv[ii]
			x := W + float64(i%nx)*dx
			y := S + float64(i/nx)*dy
			o.Geom = geom.Polygon([]geom.Path{{{X: x, Y: y}, {X: x + dx, Y: y},
				{X: x + dx, Y: y + dy}, {X: x, Y: y + dy}, {X: x, Y: y}}})
			out[ii].Insert(o)
			mapData[i] = o
		}
	}
	fmt.Println("Finished getting WRF data")
	wChan <- out
}

func s2f(s string) float64 {
	s = strings.Trim(s, "\x00")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return f
}
func rearrangeData(x, y []float64) plotter.XYs {
	out := make(plotter.XYs, len(x))
	for i, yy := range y {
		out[i].X = x[i]
		out[i].Y = yy
	}
	return out
}

type statistics struct {
	mfb, mfe, mb, me, slope, intercept, rsquared float64
}

func makePlot(x, yWRF, yInMAP []float64,
	caption string, c draw.Canvas) (*statistics, *statistics) {

	labelFont, err := vg.MakeFont(plot.DefaultFont, vg.Points(7))
	if err != nil {
		panic(err)
	}

	// Calculate stats
	wrfstats := new(statistics)
	inmapstats := new(statistics)
	wrfstats.slope, wrfstats.intercept, wrfstats.rsquared, _, _, _ =
		stats.LinearRegression(x, yWRF)
	wrfstats.mfb = mfb(x, yWRF)
	wrfstats.mfe = mfe(x, yWRF)
	wrfstats.mb = mb(x, yWRF)
	wrfstats.me = me(x, yWRF)
	inmapstats.slope, inmapstats.intercept, inmapstats.rsquared, _, _, _ =
		stats.LinearRegression(x, yInMAP)
	inmapstats.mfb = mfb(x, yInMAP)
	inmapstats.mfe = mfe(x, yInMAP)
	inmapstats.mb = mb(x, yInMAP)
	inmapstats.me = me(x, yInMAP)

	allDataWRF := append(x, yWRF...)
	allDataInMAP := append(x, yInMAP...)
	max := stats.StatsMax(append(allDataWRF, allDataInMAP...))
	min := stats.StatsMin(append(allDataWRF, allDataInMAP...))

	// Make plot
	xyWRF := rearrangeData(x, yWRF)
	xyInMAP := rearrangeData(x, yInMAP)
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	ts := draw.TextStyle{
		Color: color.Black,
		Font:  labelFont,
	}

	p.X.Label.TextStyle = ts
	p.X.Tick.Label = ts
	p.Y.Label.TextStyle = ts
	p.Y.Tick.Label = ts
	p.X.Label.Text = "Measurements (μg/m³)"
	p.Y.Label.Text = "Model (μg/m³)"
	p.Legend = plot.Legend{
		TextStyle:      ts,
		Top:            true,
		Left:           true,
		ThumbnailWidth: .15 * vg.Inch,
		Padding:        0.75 * vg.Millimeter,
	}
	s1, err := plotter.NewScatter(xyWRF)
	if err != nil {
		panic(err)
	}
	s1.Color = color.NRGBA{127, 127, 127, 255}
	s1.Radius = 0.75
	s1.Shape = draw.CircleGlyph{}
	s2, err := plotter.NewScatter(xyInMAP)
	if err != nil {
		panic(err)
	}
	s2.Color = color.NRGBA{0, 0, 0, 255}
	s2.Radius = 0.75
	s2.Shape = draw.CircleGlyph{}
	l1, err := plotter.NewLine(plotter.XYs{{min, min}, {max, max}})
	if err != nil {
		panic(err)
	}
	l1.Color = color.NRGBA{255, 0, 0, 255}
	l2, err := plotter.NewLine(plotter.XYs{{0, wrfstats.intercept},
		{max, max*wrfstats.slope + wrfstats.intercept}})
	if err != nil {
		panic(err)
	}
	l2.Color = color.NRGBA{127, 127, 127, 255}
	l3, err := plotter.NewLine(plotter.XYs{{0, inmapstats.intercept},
		{max, max*inmapstats.slope + inmapstats.intercept}})
	if err != nil {
		panic(err)
	}
	l3.Color = color.NRGBA{0, 0, 0, 255}
	p.Add(s1, s2, l1, l2, l3)
	p.X.Max = max
	p.X.Min = min
	p.Y.Max = max
	p.Y.Min = min
	p.Legend.Add("WRF-Chem", s1)
	p.Legend.Add("WRF-Chem fit", l2)
	p.Legend.Add("InMAP", s2)
	p.Legend.Add("InMAP fit", l3)
	p.Legend.Add("1:1", l1)

	p.Draw(c)
	return wrfstats, inmapstats
}

func newMap(c draw.Canvas) (*carto.Canvas, *geom.Bounds) {
	const (
		W = -2736000.00
		S = -2088000.00
		E = 2592000.00
		N = 1944000.00
	)

	bounds := geom.Bounds{Min: geom.Point{X: W, Y: S}, Max: geom.Point{X: E, Y: N}}
	return carto.NewCanvas(N, S, E, W, c), &bounds
}

func mfb(a, b []float64) float64 {
	r := 0.
	for i, v1 := range a {
		v2 := b[i]
		r += 2 * (v2 - v1) / (v1 + v2)
	}
	return r / float64(len(a))
}
func mfe(a, b []float64) float64 {
	r := 0.
	for i, v1 := range a {
		v2 := b[i]
		r += 2 * math.Abs(v2-v1) / math.Abs(v1+v2)
	}
	return r / float64(len(a))
}
func mb(a, b []float64) float64 {
	r := 0.
	for i, v1 := range a {
		v2 := b[i]
		r += (v2 - v1)
	}
	return r / float64(len(a))
}
func me(a, b []float64) float64 {
	r := 0.
	for i, v1 := range a {
		v2 := b[i]
		r += math.Abs(v2 - v1)
	}
	return r / float64(len(a))
}

func findIndex(s string, sa []string) int {
	for i, ss := range sa {
		if s == ss {
			return i
		}
	}
	return -1
}

type gg struct {
	geom.Geom
}

func getStates(filename string, simplifyThreshold float64) []geom.Polygon {
	s, err := shp.NewDecoder(filename)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	src, err := s.SR()
	if err != nil {
		panic(err)
	}
	dst, err := proj.Parse(gridProj)
	if err != nil {
		panic(err)
	}
	ct, err := src.NewTransform(dst)
	if err != nil {
		panic(err)
	}

	g := make([]geom.Polygon, 0, 100)
	for {
		var o gg
		if !s.DecodeRow(&o) {
			break
		}
		gg, err := o.Geom.Transform(ct)
		if err != nil {
			panic(err)
		}
		gg = gg.(geom.Simplifier).Simplify(simplifyThreshold)
		g = append(g, gg.(geom.Polygon))
	}
	if s.Error() != nil {
		panic(s.Error())
	}
	return g
}

func getInmapVal(polName string, d *iData) float64 {
	var val float64
	switch polName {
	case "TotalPM25":
		val = d.TotalPM25
	case "SOx":
		val = d.SOx
	case "PSO4":
		val = d.PSO4
	case "NOx":
		val = d.NOx
	case "PNO3":
		val = d.PNO3
	case "NH3":
		val = d.NH3
	case "PNH4":
		val = d.PNH4
	default:
		panic(polName)
	}
	return val
}
