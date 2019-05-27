package eval

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/GaryBoone/GoStats/stats"
	"github.com/ctessum/atmos/evalstats"
	"github.com/ctessum/cdf"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/carto"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/inmaputil"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

func TestFuelScenarios(t *testing.T) {

	if testing.Short() {
		return
	}

	evalData := os.Getenv(evalDataEnv)
	if evalData == "" {
		t.Fatalf("please set the '%s' environment variable to the location of the "+
			"downloaded evaluation data and try again", evalDataEnv)
	}

	os.MkdirAll("FuelScenarios", os.ModePerm)

	var scenarios = []string{"gasoline", "hev", "diesel", "cng", "cornEtoh",
		"stoverEtoh", "avgEV", "coalEV", "ngEV", "stoverEV", "windEV", "battery"}

	var emissionsNames = []string{
		"gasoline.Results.BaselineGasolineVehicle",
		"hev.Results.BaselineHEV",
		"diesel.Results.BaselineDieselVehicle",
		"cng.Results.CNGVehicle",
		"cornEtoh.Results.LowLevelEtOHBlendVehicle",
		"stoverEtoh.Results.LowLevelEtOHBlendVehicle",
		"avgEV.Results.ElectricVehicle",
		"coalEV.Results.ElectricVehicle",
		"ngEV.Results.ElectricVehicle",
		"stoverEV.Results.ElectricVehicle",
		"windEV.Results.ElectricVehicle",
		"battery.VehicleCycle.EVbatteryLiIonPerMile",
	}

	cfg := inmaputil.InitializeConfig()

	for i, scenario := range scenarios {
		cfg.Set("config", "nei2005Config.toml")
		emisName := emissionsNames[i]
		cfg.Set("EmissionsShapefiles", []string{
			filepath.Join(evalData, "FuelScenarios", "emissions", fmt.Sprintf("%s.elevated.shp", emisName)),
			filepath.Join(evalData, "FuelScenarios", "emissions", fmt.Sprintf("%s.groundlevel.shp", emisName)),
		})
		cfg.Set("OutputFile", filepath.Join("FuelScenarios", fmt.Sprintf("%s_vargrid.shp", scenario)))

		cfg.Root.SetArgs([]string{"run", "steady"})

		if err := cfg.Root.Execute(); err != nil {
			t.Fatal(err)
		}
	}

	var inmapGeom, states []geom.Polygon
	var alt, griddedPop []float64
	var err error

	fmt.Println("Getting states...")
	states = getStates(filepath.Join(evalData, "states.shp"), 10000)

	fmt.Println("Getting alt...")
	alt = getAlt(evalData)

	fmt.Println("Getting geometry")
	inmapGeom = wrfGeom()

	pop := getPopulation(evalData)
	griddedPop = gridPop(pop, inmapGeom)

	dataChans := make(map[string]chan map[string]*gridStats)
	for _, scenario := range scenarios {
		dataChans[scenario] = process(scenario, states, inmapGeom, alt, griddedPop, t, evalData)
	}
	outData := make(map[string]map[string]*gridStats)
	for _, scenario := range scenarios {
		fmt.Println(scenario)
		outData[scenario] = <-dataChans[scenario]
	}
	f, err := os.Create("FuelScenarios/stats.json")
	if err != nil {
		panic(err)
	}
	e := json.NewEncoder(f)
	err = e.Encode(outData)
	if err != nil {
		panic(err)
	}
	f.Close()
}

func process(scenario string, states, inmapGeom []geom.Polygon, alt, griddedPop []float64, t *testing.T, evalData string) (outChan chan map[string]*gridStats) {
	outChan = make(chan map[string]*gridStats)

	var polNames = []string{"Total PM2.5", "Primary PM2.5",
		"Particulate Nitrate", "Particulate Ammonium",
		"Particulate Sulfate", "SOA"}

	var abbrevs = []string{"totalpm", "primarypm", "pNO",
		"pNH", "pS", "SOA"}
	var inmapVars = []string{"TotalPM25", "PrimPM25", "PNO3",
		"PNH4", "PSO4", "SOA"}
	var WRFvars = [][]string{{"PM2_5_DRY_Avg"},
		{"p25i_Avg", "p25j_Avg", "eci_Avg", "ecj_Avg",
			"orgpai_Avg", "orgpaj_Avg"},
		{"no3ai_Avg", "no3aj_Avg"},
		{"nh4ai_Avg", "nh4aj_Avg"}, {"so4ai_Avg", "so4aj_Avg"},
		{"asoa1i_Avg", "asoa1j_Avg", "asoa2i_Avg", "asoa2j_Avg",
			"asoa3i_Avg", "asoa3j_Avg", "asoa4i_Avg", "asoa4j_Avg"}}
	var mapLabels = []string{"WRF-Chem", "InMAP", "InMAP–WRF-Chem"}

	labelFont, err := vg.MakeFont("Helvetica", vg.Points(7))
	if err != nil {
		panic(err)
	}
	ts := draw.TextStyle{
		Color: color.Black,
		Font:  labelFont,
	}
	labelFontStats, err := vg.MakeFont("Helvetica", vg.Points(7))
	if err != nil {
		panic(err)
	}
	tsStats := draw.TextStyle{
		Color: color.Black,
		Font:  labelFontStats,
	}
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
	const (
		figWidth  = 7 * vg.Inch
		figHeight = 4 * vg.Inch
		legendH   = 0.4 * vg.Inch
		statsH    = 0.4 * vg.Inch
		leftPad   = 10
		rightPad  = 2
	)
	c := vgimg.NewWith(vgimg.UseWH(figWidth, figHeight), vgimg.UseDPI(96))
	dc := draw.New(c)
	mainc := draw.Crop(dc, 0, 0, legendH+2*statsH, 0)
	legendc := draw.Crop(dc, 0, 0, 0, legendH-figHeight)
	statsc := draw.Crop(dc, 0, 0, legendH, (statsH*2+legendH)-figHeight)
	tiles := draw.Tiles{
		Rows:     3,
		Cols:     6,
		PadTop:   vg.Points(10),
		PadLeft:  leftPad,
		PadRight: rightPad,
		//PadBottom: vg.Point(2),
	}
	statsTiles4 := draw.Tiles{
		Rows: 2,
		Cols: 1,
		PadY: vg.Points(3),
	}
	statsTiles := draw.Tiles{
		Rows:     1,
		Cols:     6,
		PadLeft:  leftPad,
		PadRight: rightPad,
	}
	statsTiles2 := draw.Tiles{
		Rows: 3,
		Cols: 2,
	}
	statsTiles3 := draw.Tiles{
		Rows: 1,
		Cols: 2,
	}
	legendTiles := draw.Tiles{
		Rows:   1,
		Cols:   2,
		PadTop: vg.Points(3),
		PadX:   vg.Points(5),
	}

	// Write labels
	for i, name := range polNames {
		lc := tiles.At(mainc, i, 0)
		ts.XAlign = -0.5
		ts.YAlign = 0.1
		lc.FillText(ts, vg.Point{X: lc.X(0.5), Y: lc.Y(1)}, name)
	}
	for i, name := range mapLabels {
		ts2 := ts
		ts2.Rotation = math.Pi / 2.
		lc := tiles.At(mainc, 0, i)
		ts2.XAlign = -0.5
		ts2.YAlign = 0.1
		lc.FillText(ts2, vg.Point{X: lc.X(0), Y: lc.Y(0.5)}, name)
	}

	go func() {
		outData := make(map[string]*gridStats)
		inmap := getInMAPFuelScenarios(scenario, "vargrid")
		cmap := carto.NewColorMap(carto.LinCutoff)
		cmap2 := carto.NewColorMap(carto.LinCutoff)
		cmap.NumDivisions = 4
		cmap2.NumDivisions = 4
		for i, pol := range abbrevs {
			gs := new(gridStats)
			fmt.Println(scenario, pol)
			gs.inmap, err = inmap.GetProperty(inmapVars[i], inmapGeom)
			if err != nil {
				t.Fatal(err)
			}
			if i == 0 { // Total PM2.5 is already ug/m3, others
				// are ug/kg
				gs.wrf = getWRFFuelScenarios(scenario, nil, WRFvars[i], evalData)
			} else {
				gs.wrf = getWRFFuelScenarios(scenario, alt, WRFvars[i], evalData)
			}
			for ii, vv := range gs.inmap {
				gs.inmap[ii] = vv * 1000 // convert from μg to ng
			}
			for ii, vv := range gs.wrf {
				gs.wrf[ii] = vv * 1000 // convert from μg to ng
			}
			gs.diff = make([]float64, len(gs.inmap))
			for j, val := range gs.wrf {
				gs.diff[j] = gs.inmap[j] - val
			}
			if floats.Sum(gs.wrf) == 0 && floats.Sum(gs.inmap) == 0 {
				continue
			}

			gs.calcStats(griddedPop)

			cmap.AddArray(gs.wrf)
			cmap.AddArray(gs.inmap)
			cmap2.AddArray(gs.diff)
			outData[pol] = gs
		}
		cmap.Set()
		cmap2.Set()
		lc1 := legendTiles.At(legendc, 0, 0)
		cmap.Legend(&lc1, "Concentration (ng/m³)")
		lc2 := legendTiles.At(legendc, 1, 0)
		cmap2.Legend(&lc2, "Difference (ng/m³)")

		for i, pol := range abbrevs {
			gs, ok := outData[pol]
			if !ok {
				continue
			}
			wrfmap := carto.NewCanvas(N, S, E, W, tiles.At(mainc, i, 0))
			inmapmap := carto.NewCanvas(N, S, E, W, tiles.At(mainc, i, 1))
			diffmap := carto.NewCanvas(N, S, E, W, tiles.At(mainc, i, 2))
			for j := 0; j < len(gs.inmap); j++ {
				bc := cmap.GetColor(gs.wrf[j])
				ls := draw.LineStyle{Color: bc, Width: 0.1}
				wrfmap.DrawVector(inmapGeom[j], bc, ls, draw.GlyphStyle{})
				bc = cmap.GetColor(gs.inmap[j])
				ls = draw.LineStyle{Color: bc, Width: 0.1}
				inmapmap.DrawVector(inmapGeom[j], bc, ls, draw.GlyphStyle{})
				bc = cmap2.GetColor(gs.diff[j])
				ls = draw.LineStyle{Color: bc, Width: 0.1}
				diffmap.DrawVector(inmapGeom[j], bc, ls, draw.GlyphStyle{})
			}
			for _, m := range []*carto.Canvas{wrfmap, inmapmap, diffmap} {
				for _, g := range states {
					var fill = color.NRGBA{0, 0, 0, 0}
					ls := draw.LineStyle{Color: color.Black, Width: 0.1}
					m.DrawVector(g, fill, ls, draw.GlyphStyle{})
				}
			}
			// Write statistics.
			clabels := []string{"Area", "People"}
			labels := []string{"MFB", "MB", "MFE", "ME", "S", "R²"}
			format := []string{"%.0f%%", "%.2f", "%.0f%%", "%.2f", "%.2f", "%.2f", "%.2f"}
			stats := [][]float64{
				{gs.MFB, gs.MB, gs.MFE, gs.ME, gs.S, gs.R2},
				{gs.MFBWeighted, gs.MBWeighted, gs.MFEWeighted,
					gs.MEWeighted, gs.SWeighted, gs.R2Weighted},
			}
			for ii, clabel := range clabels {
				for j, v := range stats[ii] {
					sc := statsTiles2.At(statsTiles.At(statsTiles4.At(statsc, 0, ii), i, 0), j%2, j/2)
					scc1 := statsTiles3.At(sc, 0, 0)
					scc2 := statsTiles3.At(sc, 1, 0)
					tsStats2 := tsStats
					tsStats2.XAlign = -1
					tsStats2.YAlign = -0.5
					scc1.FillText(tsStats2, vg.Point{X: scc1.X(1), Y: sc.Y(0.5)},
						fmt.Sprintf("%s: ", labels[j]))
					tsStats2.XAlign = 0
					scc2.FillText(tsStats2, vg.Point{X: scc2.X(0), Y: sc.Y(0.5)},
						fmt.Sprintf(format[j], v))
				}
				if i == 0 {
					ts2 := ts
					ts2.Rotation = math.Pi / 2.
					ts2.XAlign = -0.5
					ts2.YAlign = 0.1
					sc := statsTiles.At(statsTiles4.At(statsc, 0, ii), i, 0)
					statsc.FillText(ts2, vg.Point{X: sc.X(0), Y: sc.Y(0.5)},
						clabel)
				}
			}
		}
		f, err := os.Create(fmt.Sprintf("FuelScenarios/%s.png", scenario))
		handle(err)
		_, err = vgimg.PngCanvas{Canvas: c}.WriteTo(f)
		handle(err)
		outChan <- outData
	}()
	return outChan
}

type gridStats struct {
	MB, ME, MFB, MFE, MR, R2, S                      float64
	MBWeighted, MEWeighted, MFBWeighted, MFEWeighted float64
	MRWeighted, R2Weighted, SWeighted                float64
	inmap, wrf, diff                                 []float64
}

func (gs *gridStats) calcStats(griddedPop []float64) {
	gs.S, _, gs.R2, _, _, _ = stats.LinearRegression(gs.wrf, gs.inmap)
	gs.MB = evalstats.MB(gs.wrf, gs.inmap)
	gs.MFB = evalstats.MFB(gs.wrf, gs.inmap) * 100.
	gs.ME = evalstats.ME(gs.wrf, gs.inmap)
	gs.MFE = evalstats.MFE(gs.wrf, gs.inmap) * 100.
	gs.MR = evalstats.MR(gs.wrf, gs.inmap)

	gs.MBWeighted = evalstats.MBWeighted(gs.wrf, gs.inmap, griddedPop)
	gs.MFBWeighted = evalstats.MFBWeighted(gs.wrf, gs.inmap, griddedPop) * 100.
	gs.MEWeighted = evalstats.MEWeighted(gs.wrf, gs.inmap, griddedPop)
	gs.MFEWeighted = evalstats.MFEWeighted(gs.wrf, gs.inmap, griddedPop) * 100.
	gs.MRWeighted = evalstats.MRWeighted(gs.wrf, gs.inmap, griddedPop)

	popSum := floats.Sum(griddedPop)
	wrfWeighted := make([]float64, len(gs.wrf))
	inmapWeighted := make([]float64, len(gs.inmap))
	for i, p := range griddedPop {
		wrfWeighted[i] = gs.wrf[i] * p / popSum
		inmapWeighted[i] = gs.inmap[i] * p / popSum
	}
	gs.SWeighted, _, gs.R2Weighted, _, _, _ = stats.LinearRegression(wrfWeighted,
		inmapWeighted)
	if math.IsNaN(gs.R2Weighted) {
		gs.R2Weighted = 0
	}
	if math.IsNaN(gs.R2) {
		gs.R2 = 0
	}
}

func getWRFFuelScenarios(scenario string, alt []float64, varNames []string, evalData string) []float64 {
	f := openNCFFuelScenarios(filepath.Join(evalData, "FuelScenarios", "concentrations", fmt.Sprintf("%v.ncf", scenario)))
	var out []float64
	for i, name := range varNames {
		temp := f.readVar(name, 444, 336)
		if i == 0 {
			out = make([]float64, len(temp))
		}
		if alt != nil {
			for i, val := range temp {
				// convert ug/kg air to ug/m3 air
				out[i] += val / alt[i]
			}
		} else {
			for i, val := range temp {
				out[i] += val
			}
		}
	}
	return out
}

type ncfFuelScenarios struct {
	ff *os.File
	f  *cdf.File
}

func openNCFFuelScenarios(fname string) *ncfFuelScenarios {
	f := new(ncfFuelScenarios)
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

func (f *ncfFuelScenarios) readVar(name string, nx, ny int) []float64 {
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

type inmapDataFuelScenarios string

func getInMAPFuelScenarios(scenario, grid string) inmapDataFuelScenarios {
	return inmapDataFuelScenarios(fmt.Sprintf("FuelScenarios/%v_%v.shp", scenario, grid))
}

func (id inmapDataFuelScenarios) GetGeometry() ([]geom.Geom, error) {
	d, err := shp.NewDecoder(string(id))
	if err != nil {
		return nil, err
	}
	defer d.Close()
	var g []geom.Geom

	// loop through all features in the shapefile
	for {
		gg, _, more := d.DecodeRowFields()
		if !more {
			break
		}
		g = append(g, gg)
	}
	if err := d.Error(); err != nil {
		return nil, err
	}
	return g, nil
}

func (id inmapDataFuelScenarios) GetProperty(name string, inmapGeom []geom.Polygon) ([]float64, error) {
	var data []float64
	g, err := id.GetGeometry()
	if err != nil {
		return nil, err
	}
	g2 := make([]geom.Polygonal, len(g))
	for i, gg := range g {
		g2[i] = gg.(geom.Polygonal)
	}

	d, err := shp.NewDecoder(string(id))
	if err != nil {
		return nil, err
	}
	defer d.Close()

	// loop through all features in the shapefile
	for {
		_, d, more := d.DecodeRowFields(name)
		if !more {
			break
		}
		data = append(data, s2f(d[name]))
	}
	if err := d.Error(); err != nil {
		return nil, err
	}
	ig2 := make([]geom.Polygonal, len(inmapGeom))
	for i, gg := range inmapGeom {
		ig2[i] = gg
	}
	return inmap.Regrid(g2, ig2, data)
}

type popHolder struct {
	geom.Geom
	TotalPop float64
}

func getPopulation(evalData string) *rtree.Rtree {

	p := rtree.NewTree(25, 50)

	filename := filepath.Join(evalData, "census2015blckgrp")
	f1, err := shp.NewDecoder(filename + ".shp")
	if err != nil {
		panic(err)
	}
	for {
		pp := new(popHolder)
		more := f1.DecodeRow(pp)
		if !more {
			break
		}
		if math.IsNaN(pp.TotalPop) {
			pp.TotalPop = 0.
		}
		p.Insert(pp)
	}
	handle(f1.Error())
	f1.Close()
	return p
}

func gridPop(pop *rtree.Rtree, g []geom.Polygon) []float64 {
	var wg sync.WaitGroup
	popOut := make([]float64, len(g))
	nprocs := runtime.GOMAXPROCS(-1)
	wg.Add(nprocs)
	for ip := 0; ip < nprocs; ip++ {
		go func(ip int) {
			defer wg.Done()
			for i := ip; i < len(g); i += nprocs {
				gg := g[i]
				pp := 0.
				for _, ppp := range pop.SearchIntersect(gg.Bounds()) {
					pppp := ppp.(*popHolder)
					g2 := pppp.Geom.(geom.Polygonal).Intersection(gg)
					f := g2.Area() / pppp.Geom.(geom.Polygonal).Area()
					pp += pppp.TotalPop * f
				}
				popOut[i] = pp
			}
		}(ip)
	}
	wg.Wait()
	return popOut
}

func getAlt(evalData string) []float64 {
	filename := filepath.Join(evalData, "InMAPData_v1.2.0.ncf")
	f := openNCFFuelScenarios(filename)
	return f.readVar("alt", 444, 336)
}

func wrfGeom() []geom.Polygon {
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
	g := make([]geom.Polygon, nx*ny)
	for i := 0; i < nx*ny; i++ {
		x := W + float64(i/ny)*dx
		y := S + float64(i%ny)*dy
		g[i] = geom.Polygon([]geom.Path{{{X: x, Y: y}, {X: x + dx, Y: y},
			{X: x + dx, Y: y + dy}, {X: x, Y: y + dy}, {X: x, Y: y}}})
	}
	return g
}
