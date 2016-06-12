package inmap

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
)

const (
	TestEmisFilename   = "testEmis.shp"
	TestOutputFilename = "testOutput.shp"
)

func WriteTestEmis() error {
	type emisHolder struct {
		geom.Polygon
		VOC, NOx, NH3, SOx float64 // emissions [tons/year]
		PM25               float64 `shp:"PM2_5"` // emissions [tons/year]
		Height             float64 // stack height [m]
		Diam               float64 // stack diameter [m]
		Temp               float64 // stack temperature [K]
		Velocity           float64 // stack velocity [m/s]
	}

	const (
		massConv = 907184740000.       // μg per short ton
		timeConv = 3600. * 8760.       // seconds per year
		emisConv = massConv / timeConv // convert tons/year to μg/s
		ETons    = E / emisConv        // emissions in tons per year
	)

	eShp, err := shp.NewEncoder(TestEmisFilename, emisHolder{})
	if err != nil {
		return err
	}

	emis := []emisHolder{
		{
			Polygon: geom.Polygon{{
				geom.Point{X: -3999, Y: -3999},
				geom.Point{X: -3001, Y: -3001},
				geom.Point{X: -3001, Y: -3999},
			}},
			VOC:  ETons,
			NOx:  ETons,
			NH3:  ETons,
			SOx:  ETons,
			PM25: ETons,
		},
		{
			Polygon: geom.Polygon{{
				geom.Point{X: -3999, Y: -3999},
				geom.Point{X: -3001, Y: -3001},
				geom.Point{X: -3001, Y: -3999},
			}},
			PM25:   ETons,
			Height: 20, // Layer 0
		},
		{
			Polygon: geom.Polygon{{
				geom.Point{X: -3999, Y: -3999},
				geom.Point{X: -3001, Y: -3001},
				geom.Point{X: -3001, Y: -3999},
			}},
			PM25:   ETons,
			Height: 150, // Layer 2
		},
		{
			Polygon: geom.Polygon{{
				geom.Point{X: -3999, Y: -3999},
				geom.Point{X: -3001, Y: -3001},
				geom.Point{X: -3001, Y: -3999},
			}},
			PM25:   ETons,
			Height: 2000, // Layer 9
		},
		{
			Polygon: geom.Polygon{{
				geom.Point{X: -3999, Y: -3999},
				geom.Point{X: -3001, Y: -3001},
				geom.Point{X: -3001, Y: -3999},
			}},
			PM25:   ETons,
			Height: 3000, // Above layer 9
		},
	}

	for _, e := range emis {
		if err = eShp.Encode(e); err != nil {
			return err
		}
	}
	eShp.Close()

	f, err := os.Create(strings.TrimSuffix(TestEmisFilename, filepath.Ext(TestEmisFilename)) + ".prj")
	if err != nil {
		panic(err)
	}
	if _, err = f.Write([]byte(TestGridSR)); err != nil {
		panic(err)
	}
	f.Close()

	return nil
}

func TestEmissions(t *testing.T) {
	const tol = 1.e-8 // test tolerance

	if err := WriteTestEmis(); err != nil {
		t.Error(err)
		t.FailNow()
	}
	sr, err := proj.Parse(TestGridSR)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	emis, err := ReadEmissionShapefiles(sr, "tons/year", nil, TestEmisFilename)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	cfg, ctmdata, pop, popIndices, mr := VarGridData()

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}

	type test struct {
		cellIndex int
		polIndex  []int
		values    []float64
	}
	var tests = []test{
		{
			cellIndex: 0,
			polIndex:  []int{igOrg, igS, igNH, igNO, iPM2_5},
			values:    []float64{E, E * SOxToS, E * NH3ToN, E * NOxToN, E * 2},
		},
		{
			cellIndex: 2 * 4, // layer 2, 4 cells per layer
			polIndex:  []int{iPM2_5},
			values:    []float64{E},
		},
		{
			cellIndex: 9 * 4, // layer 9, 4 cells per layer
			polIndex:  []int{iPM2_5},
			values:    []float64{E * 2},
		},
	}

	nonzero := make(map[int]map[int]int)
	for _, tt := range tests {
		c := d.Cells[tt.cellIndex]
		nonzero[tt.cellIndex] = make(map[int]int)
		for i, ii := range tt.polIndex {
			nonzero[tt.cellIndex][ii] = 0
			if different(c.EmisFlux[ii]*c.Volume, tt.values[i], tol) {
				t.Errorf("emissions value for cell %d pollutant %d should be %g but is %g",
					tt.cellIndex, ii, tt.values[i], c.EmisFlux[ii]*c.Volume)
			}
		}
	}
	for i, c := range d.Cells {
		for ii, e := range c.EmisFlux {
			if _, ok := nonzero[i][ii]; !ok {
				if e != 0 {
					t.Errorf("emissions for cell %d pollutant %d should be zero but is %g",
						i, ii, e*c.Volume)
				}
			}
		}
	}
	DeleteShapefile(TestEmisFilename)
}

func TestOutput(t *testing.T) {
	cfg, ctmdata, pop, popIndices, mr := VarGridData()

	emis := &Emissions{
		data: rtree.NewTree(25, 50),
	}
	emis.data.Insert(emisRecord{
		PM25: E,
		Geom: geom.Point{X: -3999, Y: -3999.},
	}) // ground level emissions

	d := &InMAP{
		InitFuncs: []DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, emis),
		},
		CleanupFuncs: []DomainManipulator{
			Output(TestOutputFilename, false, "TotalPop deaths", "TotalPop",
				"Total PM2.5", "PM2.5 emissions", "Baseline Total PM2.5", "WindSpeed"),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
	if err := d.Cleanup(); err != nil {
		t.Error(err)
	}
	type outData struct {
		BaselineTotalPM25 float64 `shp:"Baseline To"`
		PM25Emissions     float64 `shp:"PM2.5 emiss"`
		TotalPM25         float64 `shp:"Total PM2.5"`
		TotalPop          float64
		Deaths            float64 `shp:"TotalPop de"`
		WindSpeed         float64
	}
	dec, err := shp.NewDecoder(TestOutputFilename)
	if err != nil {
		t.Fatal(err)
	}
	var recs []outData
	for {
		var rec outData
		if more := dec.DecodeRow(&rec); !more {
			break
		}
		recs = append(recs, rec)
	}
	if err := dec.Error(); err != nil {
		t.Fatal(err)
	}

	want := []outData{
		{
			BaselineTotalPM25: 4.90770054,
			PM25Emissions:     0.00112376, //E / d.Cells[0].Volume,
			TotalPop:          100000.,
			WindSpeed:         2.16334701,
		},
		{
			BaselineTotalPM25: 4.2574172,
			WindSpeed:         2.7272017,
		},
		{
			BaselineTotalPM25: 10.34742928,
			WindSpeed:         1.88434911,
		},
		{
			BaselineTotalPM25: 5.36232233,
			WindSpeed:         2.56135321,
		},
	}

	if len(recs) != len(want) {
		t.Errorf("want %d records but have %d", len(want), len(recs))
	}
	for i, w := range want {
		if i >= len(recs) {
			continue
		}
		h := recs[i]
		if !reflect.DeepEqual(w, h) {
			t.Errorf("record %d: want %+v but have %+v", i, w, h)
		}
	}
	dec.Close()
	DeleteShapefile(TestOutputFilename)
}
