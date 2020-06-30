package eval

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/carto"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/proj"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/inmaputil"
	"github.com/spatialmodel/inmap/science/chem/simplechem"

	"gonum.org/v1/plot/vg"
	vgdraw "gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

// TestAnimation_logo creates a series of images that show the progression of a simulation
// where emissions are located in a pattern representing the InMAP logo.
// This command can be used to convert the images to a video:
// avconv -framerate 20 -f image2 -i logoSimulation_%03d.png -b 65536k logoSimulation.mp4
func TestAnimation_logo(t *testing.T) {
	if testing.Short() {
		return
	}

	evalData := os.Getenv(evalDataEnv)
	if evalData == "" {
		t.Fatalf("please set the '%s' environment variable to the location of the "+
			"downloaded evaluation data and try again", evalDataEnv)
	}

	os.MkdirAll("animation_logo", os.ModePerm)

	const (
		E          = 1.0 // E is an emissions intensity constant
		TestGridSR = `PROJCS["Lambert_Conformal_Conic_2SP",GEOGCS["GCS_unnamed ellipse",DATUM["D_unknown",SPHEROID["Unknown",6370997,0]],PRIMEM["Greenwich",0],UNIT["Degree",0.017453292519943295]],PROJECTION["Lambert_Conformal_Conic_2SP"],PARAMETER["standard_parallel_1",33],PARAMETER["standard_parallel_2",45],PARAMETER["latitude_of_origin",40],PARAMETER["central_meridian",-97],PARAMETER["false_easting",0],PARAMETER["false_northing",0],UNIT["Meter",1]]`
		logoScale  = 70000.0
		delta      = 0.1
	)

	// Create a shapefile containing emissions, population, and mortality rate data.
	logo := []geom.Polygon{
		[]geom.Path{{
			{X: 449.0, Y: -564.5}, {X: 449.0, Y: -575.75},
			{X: 449.0 + delta, Y: -575.75}, {X: 449.0 + delta, Y: -564.5},
			{X: 449.0, Y: -564.5},
		}},
		[]geom.Path{{
			{X: 456, Y: -575.75}, {X: 456, Y: -564.5}, {X: 463.5, Y: -575.75 + delta}, {X: 463.5, Y: -564.5},
			{X: 463.5 + delta, Y: -564.5}, {X: 463.5 + delta, Y: -575.75}, {X: 463.5 - delta, Y: -575.75},
			{X: 456 + delta, Y: -564.5 - 2*delta}, {X: 456 + delta, Y: -575.75},
			{X: 456, Y: -575.75},
		}},
		[]geom.Path{{
			{X: 447, Y: -580}, {X: 465, Y: -580},
			{X: 465, Y: -580 - delta}, {X: 447, Y: -580 - delta},
			{X: 447, Y: -580},
		}},
		[]geom.Path{{
			{X: 468.6, Y: -586.3}, {X: 468.6, Y: -564.5}, {X: 474.22, Y: -578.32}, {X: 480, Y: -564.5}, {X: 480, Y: -586.3},
			{X: 480 - delta, Y: -586.3}, {X: 480 - delta, Y: -564.5 - 3*delta}, {X: 474.22, Y: -578.32 - delta}, {X: 468.6 + delta, Y: -564.5 - 3*delta}, {X: 468.6 + delta, Y: -586.3},
			{X: 468.6, Y: -586.3},
		}},
		[]geom.Path{{
			{X: 485, Y: -586.3}, {X: 488.7, Y: -564.5}, {X: 493, Y: -586.3},
			{X: 493 - delta, Y: -586.3}, {X: 488.7, Y: -564.5 - delta}, {X: 485 + delta, Y: -586.3},
			{X: 485, Y: -586.3},
		}},
		[]geom.Path{{
			{X: (485 + 488.7) / 2, Y: (-586.3 + -564.5) / 2}, {X: (488.7 + 493) / 2, Y: (-564.5 + -586.3) / 2},
			{X: (488.7 + 493) / 2, Y: (-564.5+-586.3)/2 - delta}, {X: (485 + 488.7) / 2, Y: (-586.3+-564.5)/2 - delta},
			{X: (485 + 488.7) / 2, Y: (-586.3 + -564.5) / 2},
		}},
		[]geom.Path{
			{
				{X: 499.2, Y: -586.3}, {X: 499.2, Y: -564.5}, {X: 505, Y: -564.5}, {X: 508, Y: (-564.2 - 577.18) / 2}, {X: 505, Y: -577.18}, {X: 499.2 + delta, Y: -577.18},
				{X: 499.2 + delta, Y: -586.3},
				{X: 499.2, Y: -586.3},
			},
			{
				{X: 499.2 + delta, Y: -577.18 + delta}, {X: 505, Y: -577.18 + delta}, {X: 508 - delta, Y: (-564.2 - 577.18) / 2}, {X: 505, Y: -564.5 - delta}, {X: 499.2 + delta, Y: -564.5 - delta},
				{X: 499.2 + delta, Y: -577.18 + delta},
			},
		},
	}

	logoArea := make([]float64, len(logo))
	for i, g := range logo {
		logoArea[i] = g.Area()
	}

	var xmax, ymax = -1.0e200, -1.0e200
	var xmin, ymin = 1.0e200, 1.0e200
	for _, mls := range logo {
		for _, ls := range mls {
			for _, p := range ls {
				xmax, xmin = math.Max(xmax, p.X), math.Min(xmin, p.X)
				ymax, ymin = math.Max(ymax, p.Y), math.Min(ymin, p.Y)
			}
		}
	}
	xcent := (xmax + xmin) / 2
	ycent := (ymax + ymin) / 2
	for i, mls := range logo {
		for j, ls := range mls {
			for k := range ls {
				logo[i][j][k].X -= xcent
				logo[i][j][k].Y -= ycent
				logo[i][j][k].X *= logoScale
				logo[i][j][k].Y *= logoScale
			}
		}
	}

	type EmisRecord struct {
		geom.Polygon
		PM25 float64 `shp:"PM2_5"` // emissions [μg/s]
		SOx  float64
		MR   float64
		Pop  float64
	}

	s, err := shp.NewEncoder("animation_logo/logo.shp", EmisRecord{})
	if err != nil {
		t.Fatal(err)
	}

	for i, g := range logo {
		err = s.Encode(EmisRecord{
			Polygon: g,
			PM25:    E * logoArea[i],
			SOx:     E * logoArea[i],
			MR:      1,
			Pop:     logoArea[i],
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	s.Close()

	f, err := os.Create("animation_logo/logo.prj")
	if err != nil {
		panic(err)
	}
	if _, err = f.Write([]byte(TestGridSR)); err != nil {
		panic(err)
	}
	f.Close()

	cfg := inmaputil.InitializeConfig()

	dynamic := true
	createGrid := false // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	cfg.Set("config", "nei2005Config.toml")
	cfg.Set("VarGrid.CensusFile", "animation_logo/logo.shp")
	cfg.Set("VarGrid.CensusPopColumns", []string{"Pop"})
	cfg.Set("VarGrid.PopGridColumn", "Pop")
	cfg.Set("VarGrid.MortalityRateFile", "animation_logo/logo.shp")
	cfg.Set("VarGrid.MortalityRateColumns", []string{"MR"})

	vgc, err := inmaputil.VarGridConfig(cfg.Viper)
	if err != nil {
		t.Fatal(err)
	}

	dataChan := make(chan []geomConc)
	errChan := make(chan error)

	states := getStates(filepath.Join(evalData, "states.shp"), 0)

	go createImages("animation_logo/logoSimulation_%03d.png", dataChan, errChan, states, false)

	// framePeriod is the interval in seconds between snapshots
	const framePeriod = 3600.0 * 3

	if err := inmaputil.Run(nil, "animation_logo/logoOut.log", "animation_logo/logoOut.shp", false,
		map[string]string{"TotalPM25": "TotalPM25"}, cfg.GetString("EmissionUnits"),
		[]string{"animation_logo/logo.shp"}, nil,
		vgc, nil, nil, cfg.GetString("InMAPData"), cfg.GetString("VariableGridData"), cfg.GetInt("NumIterations"),
		dynamic, createGrid, inmaputil.DefaultScienceFuncs, nil,
		[]inmap.DomainManipulator{inmap.RunPeriodically(framePeriod, saveConc(dataChan))}, nil, simplechem.Mechanism{}); err != nil {
		t.Fatal(err)
	}

	close(dataChan)
	if err := <-errChan; err != nil {
		t.Fatal(err)
	}
}

// TestAnimation_nei creates a series of images that show the progression of a
// simulation based on year 2005 NEI emissions.
// This command can be used to convert the images to different types of videos:
// avconv -framerate 15 -f image2 -i inmapNEI_%03d.png -c:v libx264 -crf 28 inmapNEI.mp4
// avconv -framerate 15 -f image2 -i inmapNEI_%03d.png -c:v libvpx -crf 10 -b:v 1M  inmapNEI.webm
// avconv -framerate 15 -f image2 -i inmapNEI_%03d.png -c:v libtheora -qscale:v 7  inmapNEI.ogv
func TestAnimation_nei(t *testing.T) {
	if testing.Short() {
		return
	}

	evalData := os.Getenv(evalDataEnv)
	if evalData == "" {
		t.Fatalf("please set the '%s' environment variable to the location of the "+
			"downloaded evaluation data and try again", evalDataEnv)
	}

	os.MkdirAll("animation_nei", os.ModePerm)

	cfg := inmaputil.InitializeConfig()

	dynamic := true
	createGrid := false // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	cfg.SetConfigFile("nei2005Config.toml")
	vgc, err := inmaputil.VarGridConfig(cfg.Viper)
	if err != nil {
		t.Fatal(err)
	}

	dataChan := make(chan []geomConc)
	errChan := make(chan error)

	states := getStates(filepath.Join(evalData, "states.shp"), 0)

	go createImages("animation_nei/inmapNEI_%03d.png", dataChan, errChan, states, true)

	// framePeriod is the interval in seconds between snapshots
	const framePeriod = 3600.0

	if err := inmaputil.Run(nil, "animation_nei/results.log", "animation_nei/results.shp", false,
		inmaputil.GetStringMapString("OutputVariables", cfg.Viper), cfg.GetString("EmissionUnits"),
		cfg.GetStringSlice("EmissionsShapefiles"), nil,
		vgc, nil, nil, cfg.GetString("InMAPData"), cfg.GetString("VariableGridData"), cfg.GetInt("NumIterations"),
		dynamic, createGrid, inmaputil.DefaultScienceFuncs, nil,
		[]inmap.DomainManipulator{inmap.RunPeriodically(framePeriod, saveConc(dataChan))}, nil, simplechem.Mechanism{}); err != nil {
		t.Fatal(err)
	}

	close(dataChan)
	if err := <-errChan; err != nil {
		t.Fatal(err)
	}
}

func createImages(basename string, dataChan chan []geomConc, errChan chan error, states []geom.Polygon, insets bool) {

	cmap := carto.NewColorMap(carto.LinCutoff)
	var data [][]geomConc
	var vals []float64
	for d := range dataChan {
		data = append(data, d)
		tempData := make([]float64, len(d))
		for i, dd := range d {
			tempData[i] = dd.val
		}
		vals = append(vals, tempData...)
	}
	cmap.AddArray(vals)
	cmap.Set()

	for i, d := range data {
		var err error
		if insets {
			err = createImageWithInsets(d, cmap, states, fmt.Sprintf(basename, i))
		} else {
			err = createImage(d, cmap, states, fmt.Sprintf(basename, i))
		}
		if err != nil {
			errChan <- err
			return
		}
	}

	errChan <- nil
}

type geomConc struct {
	geom.Polygonal
	val float64
}

func saveConc(outChan chan []geomConc) inmap.DomainManipulator {
	return func(d *inmap.InMAP) error {
		var m simplechem.Mechanism
		o, err := inmap.NewOutputter("", false, map[string]string{"TotalPM25": "TotalPM25"}, nil, m)
		if err != nil {
			return err
		}

		res, err := d.Results(o)
		if err != nil {
			return err
		}
		vals := res["TotalPM25"]
		g := d.GetGeometry(0, false)

		c := make([]geomConc, len(g))
		for i, gg := range g {
			c[i] = geomConc{
				Polygonal: gg,
				val:       vals[i],
			}
		}
		outChan <- c
		return nil
	}
}

func createImage(data []geomConc, cmap *carto.ColorMap, states []geom.Polygon, filename string) error {

	const (
		W = -2400000.0
		E = 2440000.0
		S = -1700000.0
		N = 1500000.0
	)
	width := 1000
	height := int(float64(width) * (N - S) / (E - W))

	img := draw.Image(image.NewRGBA(image.Rect(0, 0, int(width), int(height))))
	c := vgimg.NewWith(vgimg.UseImage(img))
	dc := vgdraw.New(c)

	m := carto.NewCanvas(N, S, E, W, dc)
	for _, d := range data {
		fill := cmap.GetColor(d.val)
		ls := vgdraw.LineStyle{
			Width: 0.1 * vg.Millimeter,
			Color: fill,
		}
		err := m.DrawVector(d.Polygonal, fill, ls, vgdraw.GlyphStyle{})
		if err != nil {
			return err
		}
	}

	lineStyle := vgdraw.LineStyle{
		Width: 0.25 * vg.Millimeter,
		Color: color.Black,
	}
	var fill = color.NRGBA{0, 255, 0, 0}
	for _, g := range states {
		m.DrawVector(g, fill, lineStyle, vgdraw.GlyphStyle{})
	}

	w, err := os.Create(filename)
	if err != nil {
		return err
	}
	png := vgimg.PngCanvas{Canvas: c}
	if _, err := png.WriteTo(w); err != nil {
		return err
	}
	w.Close()
	return nil
}

func createImageWithInsets(data []geomConc, cmap *carto.ColorMap, states []geom.Polygon, filename string) error {
	const (
		// Wmain is the western edge of the main map
		Wmain = -2736000.00
		// Smain is the southerm edge of the main map
		Smain = -2088000.00
		// Emain is the eastern edge of the main map
		Emain = 2592000.00
		// Nmain is the norther edge of the main map
		Nmain = 1944000.00

		// ybuffer and xbuffer the the sizes of the insets
		ybuffer = 40000.
		xbuffer = ybuffer * 1.31

		legendHeight = 0.4 * vg.Inch
		figHeight    = 2.65*vg.Inch + legendHeight
		mainWidth    = (figHeight - legendHeight) / (Nmain - Smain) * (Emain - Wmain)
		insetWidth   = figHeight / vg.Length(ybuffer/xbuffer) * 0.5
	)

	img := vgimg.NewWith(vgimg.UseWH(2*insetWidth+mainWidth, figHeight), vgimg.UseDPI(300))
	canvas := vgdraw.New(img)

	stateLineStyle := vgdraw.LineStyle{
		Width: 0.25 * vg.Millimeter,
		Color: color.Black,
	}
	insetLineStyle := vgdraw.LineStyle{
		Width: 0.4 * vg.Millimeter,
		Color: color.RGBA{255, 0, 0, 255},
	}
	var clearFill = color.NRGBA{0, 255, 0, 0}

	subfigs := []struct {
		b *geom.Bounds
		c vgdraw.Canvas
	}{
		{ // Main canvas
			b: &geom.Bounds{
				Min: geom.Point{X: Wmain, Y: Smain},
				Max: geom.Point{X: Emain, Y: Nmain},
			},
			c: vgdraw.Crop(canvas, insetWidth, -insetWidth, legendHeight, 0),
		},
		{ // Las Vegas
			b: getInset(36.169635, -115.130025, xbuffer, ybuffer),
			c: vgdraw.Crop(canvas, 0, -mainWidth-insetWidth, figHeight/2., 0),
		},
		{ // Los Angeles
			b: getInset(34.052932, -118.264104, xbuffer, ybuffer),
			c: vgdraw.Crop(canvas, 0, -mainWidth-insetWidth, 0, -figHeight/2.),
		},
		{ // New York
			b: getInset(40.714243, -73.998220, xbuffer, ybuffer),
			c: vgdraw.Crop(canvas, insetWidth+mainWidth, 0, figHeight/2., 0),
		},
		{ // Miami
			b: getInset(25.760527, -80.186963, xbuffer, ybuffer),
			c: vgdraw.Crop(canvas, insetWidth+mainWidth, 0, 0, -figHeight/2.),
		},
	}

	legendC := vgdraw.Crop(canvas, insetWidth, -insetWidth, 0, -figHeight+legendHeight)
	if err := cmap.Legend(&legendC, "PM2.5 concentration (μg m-3)"); err != nil {
		return err
	}

	var mainMapCanvas *carto.Canvas

	for i, subfig := range subfigs {
		rect := geom.Polygon{[]geom.Point{
			subfig.b.Min,
			{X: subfig.b.Max.X, Y: subfig.b.Min.Y},
			subfig.b.Max,
			{X: subfig.b.Min.X, Y: subfig.b.Max.Y},
		}}
		mapCanvas := carto.NewCanvas(subfig.b.Max.Y, subfig.b.Min.Y, subfig.b.Max.X, subfig.b.Min.X, subfig.c)
		if i == 0 {
			mainMapCanvas = mapCanvas
		}

		// draw data
		for _, d := range data {
			fill := cmap.GetColor(d.val)
			ls := vgdraw.LineStyle{
				Width: 0.1 * vg.Millimeter,
				Color: fill,
			}
			g := d.Intersection(rect)
			if g != nil {
				err := mapCanvas.DrawVector(g, fill, ls, vgdraw.GlyphStyle{})
				if err != nil {
					return err
				}
			}
		}

		// Draw states
		for _, g := range states {
			gg := g.Intersection(rect)
			if gg != nil {
				mapCanvas.DrawVector(gg, clearFill, stateLineStyle, vgdraw.GlyphStyle{})
			}
		}

		// Draw inset locators
		if i > 0 {
			// Draw inset boundary
			mainMapCanvas.DrawVector(rect, clearFill, insetLineStyle, vgdraw.GlyphStyle{})
			mapCanvas.DrawVector(rect, clearFill, insetLineStyle, vgdraw.GlyphStyle{})

			inSetUR := mapCanvas.Coordinates(geom.Point{X: subfig.b.Max.X, Y: subfig.b.Max.Y})
			mainUR := mainMapCanvas.Coordinates(geom.Point{X: subfig.b.Max.X, Y: subfig.b.Max.Y})
			inSetUL := mapCanvas.Coordinates(geom.Point{X: subfig.b.Min.X, Y: subfig.b.Max.Y})
			mainUL := mainMapCanvas.Coordinates(geom.Point{X: subfig.b.Min.X, Y: subfig.b.Max.Y})
			inSetLR := mapCanvas.Coordinates(geom.Point{X: subfig.b.Max.X, Y: subfig.b.Min.Y})
			mainLR := mainMapCanvas.Coordinates(geom.Point{X: subfig.b.Max.X, Y: subfig.b.Min.Y})
			mainLL := mainMapCanvas.Coordinates(geom.Point{X: subfig.b.Min.X, Y: subfig.b.Min.Y})
			inSetLL := mapCanvas.Coordinates(geom.Point{X: subfig.b.Min.X, Y: subfig.b.Min.Y})
			if i > 2 {
				canvas.StrokeLine2(insetLineStyle, inSetUL.X, inSetUL.Y, mainUR.X, mainUR.Y)
				canvas.StrokeLine2(insetLineStyle, inSetLL.X, inSetLL.Y, mainLR.X, mainLR.Y)
			} else {
				canvas.StrokeLine2(insetLineStyle, inSetUR.X, inSetUR.Y, mainUL.X, mainUL.Y)
				canvas.StrokeLine2(insetLineStyle, inSetLR.X, inSetLR.Y, mainLL.X, mainLL.Y)
			}
		}
	}

	// Write out image file.
	w, err := os.Create(filename)
	if err != nil {
		return err
	}
	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(w); err != nil {
		return err
	}
	w.Close()

	return nil
}

func getInset(lat, lon, xbuffer, ybuffer float64) (bounds *geom.Bounds) {
	src, err := proj.Parse("+proj=longlat +ellps=WGS84 +datum=WGS84 +no_defs")
	if err != nil {
		panic(err)
	}
	dst, err := proj.Parse("+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1")
	if err != nil {
		panic(err)
	}
	insetCT, err := src.NewTransform(dst)
	if err != nil {
		panic(err)
	}

	g := geom.Point{X: lon, Y: lat}
	gg, err := g.Transform(insetCT)
	if err != nil {
		panic(err)
	}
	bounds = geom.NewBounds()
	bounds.Min = geom.Point{
		X: gg.(geom.Point).X - xbuffer,
		Y: gg.(geom.Point).Y - ybuffer,
	}
	bounds.Max = geom.Point{
		X: gg.(geom.Point).X + xbuffer,
		Y: gg.(geom.Point).Y + ybuffer,
	}
	return
}
