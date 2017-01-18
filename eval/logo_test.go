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
	"github.com/gonum/plot/vg"
	vgdraw "github.com/gonum/plot/vg/draw"
	"github.com/gonum/plot/vg/vgimg"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/inmap/cmd"
)

// TestLogo creates a series of images that show the progression of a simulation
// where emissions are located in a pattern representing the InMAP logo.
// This command can be used to convert the images to a video:
// avconv -framerate 20 -f image2 -i logoSimulation_%03d.png -b 65536k logoSimulation.mp4
func TestLogo(t *testing.T) {
	if testing.Short() {
		return
	}

	evalData := os.Getenv(evalDataEnv)
	if evalData == "" {
		t.Fatalf("please set the '%s' environment variable to the location of the "+
			"downloaded evaluation data and try again", evalDataEnv)
	}

	os.MkdirAll("logo", os.ModePerm)

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

	s, err := shp.NewEncoder("logo/logo.shp", EmisRecord{})
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

	f, err := os.Create("logo/logo.prj")
	if err != nil {
		panic(err)
	}
	if _, err = f.Write([]byte(TestGridSR)); err != nil {
		panic(err)
	}
	f.Close()

	dynamic := true
	createGrid := false // this isn't used for the dynamic grid
	os.Setenv("InMAPRunType", "dynamic")
	if err := cmd.Startup("nei2005Config.toml"); err != nil {
		t.Fatal(err)
	}
	cmd.Config.EmissionsShapefiles = []string{"logo/logo.shp"}
	cmd.Config.VarGrid.CensusFile = "logo/logo.shp"
	cmd.Config.VarGrid.CensusPopColumns = []string{"Pop"}
	cmd.Config.VarGrid.PopGridColumn = "Pop"
	cmd.Config.VarGrid.MortalityRateFile = "logo/logo.shp"
	cmd.Config.VarGrid.MortalityRateColumn = "MR"
	cmd.Config.OutputFile = "logo/logoOut.shp"
	//cmd.Config.VarGrid.PopConcThreshold *= 10

	dataChan := make(chan []geomConc)
	errChan := make(chan error)

	states := getStates(filepath.Join(evalData, "states.shp"))

	go createImages(dataChan, errChan, states)

	// framePeriod is the interval in seconds between snapshots
	const framePeriod = 3600.0 * 3

	if err := cmd.Run(dynamic, createGrid, cmd.DefaultScienceFuncs, nil,
		[]inmap.DomainManipulator{inmap.RunPeriodically(framePeriod, saveConc(dataChan))}, nil); err != nil {
		t.Fatal(err)
	}

	close(dataChan)
	if err := <-errChan; err != nil {
		t.Fatal(err)
	}
}

func createImages(dataChan chan []geomConc, errChan chan error, states []geom.Geom) {

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
		err := createImage(d, cmap, states, fmt.Sprintf("logo/logoSimulation_%03d.png", i))
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
		res, err := d.Results(false, true, map[string]string{"TotalPM25": "TotalPM25"})
		if err != nil {
			return err
		}
		vals := res["TotalPM25"]
		g := d.GetGeometry(0, false)

		o := make([]geomConc, len(g))
		for i, gg := range g {
			o[i] = geomConc{
				Polygonal: gg,
				val:       vals[i],
			}
		}
		outChan <- o
		return nil
	}
}

func createImage(data []geomConc, cmap *carto.ColorMap, states []geom.Geom, filename string) error {

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
