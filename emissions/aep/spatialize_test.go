/*
Copyright © 2017 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.*/

package aep

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/op"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
	"github.com/kr/pretty"
)

const (
	srgSpecFileString = `"REGION","SURROGATE","SURROGATE CODE","DATA SHAPEFILE","DATA ATTRIBUTE","WEIGHT SHAPEFILE","WEIGHT ATTRIBUTE","WEIGHT FUNCTION","FILTER FUNCTION","MERGE FUNCTION","SECONDARY SURROGATE","TERTIARY SURROGATE","QUARTERNARY SURROGATE","DETAILS","COMMENTS"
"USA","Population",100,"cty_pophu2k_revised","FIPSSTCO","pophu_bg2010","POP2010",,,,,,,"Total population from Census 2010 blocks",
"USA","Housing Change",137,"cty_pophu2k_revised","FIPSSTCO","pophu_bg2010","HUCH1000",,,,"Housing","Population","Land","Housing change from 2000 to 2010 census blocks",
"USA","Housing Change and Population",140,"cty_pophu2k_revised","FIPSSTCO",,,,,"0.5*Housing Change+0.5*Population","Population",,,"Combination of the change in housing between 2000 and 2010 and year 2010 population",
"USA","Commercial Land",500,"county_lu2k","FIPSSTCO","fema_bsf_2002bnd",,"COM1+COM2+COM3+COM4+COM5+COM6+COM7+COM8+COM9",,,"Population","Land",,"Sum of building square footage from the following FEMA categories:  COM1 + COM2 + COM3 + COM4 + COM5 + COM6 + COM7 + COM8 + COM9",
"USA","Urban Primary Road Miles",200,"cty_pophu2k_revised","FIPSSTCO","rd_ps_tiger2010","NONE",,"RDTYPE = 1",,"Total Road Miles","Population",,"Road Miles of Urban Primary Roads"," "
`

	gridRefFileString = `# Created by C. Allen (CSC), 1 Jul 2013, for 2011eb.
#EXPORT_DATE=Mon Jan 05 14:10:11 EST 2015
#EXPORT_VERSION_NAME=add afdust SCC
000000;0010200501;100
000000;2101006002;137! profile added for new SCC in 2002 inventory
000000;2102001000;140
000000;2102001001;100
034023;2102001001;500
036047;2102001001;200
`
)

// Whether each county is completely covered by our grid (determined through
// manual inspection).
var coveredByGrid = map[string]bool{
	"09001": false, "36119": false, "36087": false, "34031": false,
	"34003": false, "34027": false, "34013": true, "36005": true,
	"34017": true, "36061": true, "34035": false, "34039": true, "36085": true,
	"34023": false, "34025": false, "36047": true, "36081": true, "36059": false,
	"36103": false,
}

func TestReadSrgSpec(t *testing.T) {
	r := strings.NewReader(srgSpecFileString)
	srgSpecs, err := ReadSrgSpec(r, "testdata", true)
	if err != nil {
		t.Fatal(err)
	}
	testResult := ""
	for _, code := range []string{"100", "137", "140", "500", "200"} {
		srgSpec, err := srgSpecs.GetByCode(USA, code)
		if err != nil {
			t.Fatal(err)
		}
		testResult += fmt.Sprintf("&{Region:%s Name:%s Code:%s DATASHAPEFILE:%s "+
			"DATAATTRIBUTE:%s WEIGHTSHAPEFILE:%s Details:%s "+
			"BackupSurrogateNames:%v WeightColumns:%v MergeNames:%v MergeMultipliers:%v}\n",
			srgSpec.Region, srgSpec.Name, srgSpec.Code, strings.Replace(srgSpec.DATASHAPEFILE, "\\", "/", -1),
			srgSpec.DATAATTRIBUTE, strings.Replace(srgSpec.WEIGHTSHAPEFILE, "\\", "/", -1),
			srgSpec.Details, srgSpec.BackupSurrogateNames, srgSpec.WeightColumns,
			srgSpec.MergeNames, srgSpec.MergeMultipliers)
		if srgSpec.FilterFunction != nil {
			testResult += fmt.Sprintf("%+v\n", srgSpec.FilterFunction)
		}
	}
	result := `&{Region:USA Name:Population Code:100 DATASHAPEFILE:testdata/cty_pophu2k_revised.shp DATAATTRIBUTE:FIPSSTCO WEIGHTSHAPEFILE:testdata/pophu_bg2010.shp Details:Total population from Census 2010 blocks BackupSurrogateNames:[] WeightColumns:[POP2010] MergeNames:[] MergeMultipliers:[]}
&{Region:USA Name:Housing Change Code:137 DATASHAPEFILE:testdata/cty_pophu2k_revised.shp DATAATTRIBUTE:FIPSSTCO WEIGHTSHAPEFILE:testdata/pophu_bg2010.shp Details:Housing change from 2000 to 2010 census blocks BackupSurrogateNames:[Housing Population Land] WeightColumns:[HUCH1000] MergeNames:[] MergeMultipliers:[]}
&{Region:USA Name:Housing Change and Population Code:140 DATASHAPEFILE:cty_pophu2k_revised DATAATTRIBUTE:FIPSSTCO WEIGHTSHAPEFILE: Details:Combination of the change in housing between 2000 and 2010 and year 2010 population BackupSurrogateNames:[Population] WeightColumns:[] MergeNames:[Housing Change Population] MergeMultipliers:[0.5 0.5]}
&{Region:USA Name:Commercial Land Code:500 DATASHAPEFILE:testdata/county_lu2k.shp DATAATTRIBUTE:FIPSSTCO WEIGHTSHAPEFILE:testdata/fema_bsf_2002bnd.shp Details:Sum of building square footage from the following FEMA categories:  COM1 + COM2 + COM3 + COM4 + COM5 + COM6 + COM7 + COM8 + COM9 BackupSurrogateNames:[Population Land] WeightColumns:[COM1 COM2 COM3 COM4 COM5 COM6 COM7 COM8 COM9] MergeNames:[] MergeMultipliers:[]}
&{Region:USA Name:Urban Primary Road Miles Code:200 DATASHAPEFILE:testdata/cty_pophu2k_revised.shp DATAATTRIBUTE:FIPSSTCO WEIGHTSHAPEFILE:testdata/rd_ps_tiger2010.shp Details:Road Miles of Urban Primary Roads BackupSurrogateNames:[Total Road Miles Population] WeightColumns:[] MergeNames:[] MergeMultipliers:[]}
&{Column:RDTYPE EqualNotEqual:Equal Values:[1]}
`
	if testResult != result {
		t.Fatalf("expected:\n%s\ngot:\n%s", result, testResult)
	}
}

func TestReadGridRef(t *testing.T) {
	r := strings.NewReader(gridRefFileString)
	gridRef, err := ReadGridRef(r)
	if err != nil {
		t.Fatal(err)
	}
	result := &GridRef{
		0: map[string]map[string]interface{}{
			"2102001001": map[string]interface{}{"34023": "500", "36047": "200", "00000": "100"},
			"0010200501": map[string]interface{}{"00000": "100"},
			"2101006002": map[string]interface{}{"00000": "137"},
			"2102001000": map[string]interface{}{"00000": "140"}}}

	diff := pretty.Diff(result, gridRef)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
}

func createGrid() (*GridDef, error) {
	sr, err := proj.Parse("+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1")
	if err != nil {
		return nil, err
	}
	grid := NewGridRegular("test grid", 4, 4, 20000, 20000, 1870000, 280000, sr)
	return grid, err
}

func TestNewGridRegular(t *testing.T) {
	grid, err := createGrid()
	if err != nil {
		t.Fatal(err)
	}
	err = grid.WriteToShp("testdata")
	if err != nil {
		panic(err)
	}
	const gridArea = 6.4e9
	if op.Area(grid.Extent) != gridArea {
		t.Errorf("grid extent = %g, want %g", op.Area(grid.Extent), gridArea)
	}
	cellResult := []*GridCell{
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.87e+06, Y: 280000}, geom.Point{X: 1.89e+06, Y: 280000}, geom.Point{X: 1.89e+06, Y: 300000}, geom.Point{X: 1.87e+06, Y: 300000}, geom.Point{X: 1.87e+06, Y: 280000}}}, Row: 0, Col: 0, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.87e+06, Y: 300000}, geom.Point{X: 1.89e+06, Y: 300000}, geom.Point{X: 1.89e+06, Y: 320000}, geom.Point{X: 1.87e+06, Y: 320000}, geom.Point{X: 1.87e+06, Y: 300000}}}, Row: 1, Col: 0, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.87e+06, Y: 320000}, geom.Point{X: 1.89e+06, Y: 320000}, geom.Point{X: 1.89e+06, Y: 340000}, geom.Point{X: 1.87e+06, Y: 340000}, geom.Point{X: 1.87e+06, Y: 320000}}}, Row: 2, Col: 0, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.87e+06, Y: 340000}, geom.Point{X: 1.89e+06, Y: 340000}, geom.Point{X: 1.89e+06, Y: 360000}, geom.Point{X: 1.87e+06, Y: 360000}, geom.Point{X: 1.87e+06, Y: 340000}}}, Row: 3, Col: 0, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.89e+06, Y: 280000}, geom.Point{X: 1.91e+06, Y: 280000}, geom.Point{X: 1.91e+06, Y: 300000}, geom.Point{X: 1.89e+06, Y: 300000}, geom.Point{X: 1.89e+06, Y: 280000}}}, Row: 0, Col: 1, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.89e+06, Y: 300000}, geom.Point{X: 1.91e+06, Y: 300000}, geom.Point{X: 1.91e+06, Y: 320000}, geom.Point{X: 1.89e+06, Y: 320000}, geom.Point{X: 1.89e+06, Y: 300000}}}, Row: 1, Col: 1, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.89e+06, Y: 320000}, geom.Point{X: 1.91e+06, Y: 320000}, geom.Point{X: 1.91e+06, Y: 340000}, geom.Point{X: 1.89e+06, Y: 340000}, geom.Point{X: 1.89e+06, Y: 320000}}}, Row: 2, Col: 1, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.89e+06, Y: 340000}, geom.Point{X: 1.91e+06, Y: 340000}, geom.Point{X: 1.91e+06, Y: 360000}, geom.Point{X: 1.89e+06, Y: 360000}, geom.Point{X: 1.89e+06, Y: 340000}}}, Row: 3, Col: 1, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.91e+06, Y: 280000}, geom.Point{X: 1.93e+06, Y: 280000}, geom.Point{X: 1.93e+06, Y: 300000}, geom.Point{X: 1.91e+06, Y: 300000}, geom.Point{X: 1.91e+06, Y: 280000}}}, Row: 0, Col: 2, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.91e+06, Y: 300000}, geom.Point{X: 1.93e+06, Y: 300000}, geom.Point{X: 1.93e+06, Y: 320000}, geom.Point{X: 1.91e+06, Y: 320000}, geom.Point{X: 1.91e+06, Y: 300000}}}, Row: 1, Col: 2, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.91e+06, Y: 320000}, geom.Point{X: 1.93e+06, Y: 320000}, geom.Point{X: 1.93e+06, Y: 340000}, geom.Point{X: 1.91e+06, Y: 340000}, geom.Point{X: 1.91e+06, Y: 320000}}}, Row: 2, Col: 2, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.91e+06, Y: 340000}, geom.Point{X: 1.93e+06, Y: 340000}, geom.Point{X: 1.93e+06, Y: 360000}, geom.Point{X: 1.91e+06, Y: 360000}, geom.Point{X: 1.91e+06, Y: 340000}}}, Row: 3, Col: 2, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.93e+06, Y: 280000}, geom.Point{X: 1.95e+06, Y: 280000}, geom.Point{X: 1.95e+06, Y: 300000}, geom.Point{X: 1.93e+06, Y: 300000}, geom.Point{X: 1.93e+06, Y: 280000}}}, Row: 0, Col: 3, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.93e+06, Y: 300000}, geom.Point{X: 1.95e+06, Y: 300000}, geom.Point{X: 1.95e+06, Y: 320000}, geom.Point{X: 1.93e+06, Y: 320000}, geom.Point{X: 1.93e+06, Y: 300000}}}, Row: 1, Col: 3, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.93e+06, Y: 320000}, geom.Point{X: 1.95e+06, Y: 320000}, geom.Point{X: 1.95e+06, Y: 340000}, geom.Point{X: 1.93e+06, Y: 340000}, geom.Point{X: 1.93e+06, Y: 320000}}}, Row: 2, Col: 3, Weight: 0},
		&GridCell{Polygonal: geom.Polygon{{geom.Point{X: 1.93e+06, Y: 340000}, geom.Point{X: 1.95e+06, Y: 340000}, geom.Point{X: 1.95e+06, Y: 360000}, geom.Point{X: 1.93e+06, Y: 360000}, geom.Point{X: 1.93e+06, Y: 340000}}}, Row: 3, Col: 3, Weight: 0},
	}
	diff := pretty.Diff(cellResult, grid.Cells)
	if len(diff) != 0 {
		t.Fatal(diff)
	}
}

func TestCreateSurrogates(t *testing.T) {
	if testing.Short() {
		return
	}
	inputSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		t.Error(err)
	}
	r := strings.NewReader(srgSpecFileString)
	srgSpecs, err := ReadSrgSpec(r, "testdata", true)
	if err != nil {
		t.Error(err)
	}
	r = strings.NewReader(gridRefFileString)
	gridRef, err := ReadGridRef(r)
	if err != nil {
		t.Fatal(err)
	}
	grid, err := createGrid()
	if err != nil {
		t.Fatal(err)
	}
	matchFullSCC := false
	sp := NewSpatialProcessor(srgSpecs, []*GridDef{grid}, gridRef, inputSR, matchFullSCC)
	sp.load()

	// surrogates that should be nil based on manual inspection.
	nilSrgs := map[string]map[string]bool{
		"200": map[string]bool{"09001": true, "36103": true},
	}

	for _, code := range []string{"100", "137", "140", "500", "200"} {
		srgSpec, err := srgSpecs.GetByCode(USA, code)
		if err != nil {
			t.Fatal(err)
		}
		srgsI, err := sp.createSurrogate(context.Background(), &srgGrid{srg: srgSpec, gridData: grid})
		if err != nil {
			t.Errorf("creating surrogate %s: %v", code, err)
		}
		srgs := srgsI.(*GriddingSurrogate)
		if len(srgs.Srg) != 19 {
			t.Errorf("in code %s: there should be %d surrogates instead of %d",
				code, 19, len(srgs.Srg))
		}
		for fips, covered := range coveredByGrid {
			if _, ok := srgs.Srg[fips]; !ok {
				t.Errorf("county %s is not in surrogate %s", fips, code)
				continue
			}
			if srgs.Srg[fips].CoveredByGrid != covered {
				t.Errorf("county %s should %v be covered by the grid but it is %v",
					fips, covered, srgs.Srg[fips].CoveredByGrid)
			}
			srg, ok := srgs.Srg[fips]
			if !ok {
				t.Errorf("missing surrogate %s for fips %s", code, fips)
			}
			sum := 0.
			for _, cell := range srg.Cells {
				if cell.Weight < 0 {
					t.Errorf("a surrogate grid cell equals <0; this should never happen")
				}
				sum += cell.Weight
			}
			if covered {
				if math.Abs(sum-1) > 0.001 {
					t.Errorf("surrogate %s should sum to 1 for fips %s but "+
						"instead sums to %f", code, fips, sum)
				}
			} else if sum > 1. {
				t.Errorf("surrogate %s should sum to less than 1 for fips %s but "+
					"instead sums to %f", code, fips, sum)
			}
			gridded, _ := srgs.ToGrid(fips)
			if gridded == nil {
				if _, ok := nilSrgs[code][fips]; ok {
					continue
				} else {
					t.Errorf("gridded surrogate %s fips %s is nil but shouldn't be",
						code, fips)
					continue
				}
			}
			if gridded.Get(0, 3) != 0 {
				t.Errorf("gridded surrogate %s fips %s grid cell (0,3) should equal zero "+
					"because because it is over the ocean but instead it equals %f",
					code, fips, gridded.Get(0, 3))
			}
			sum = gridded.Sum()
			if covered {
				if math.Abs(sum-1) > 0.000001 {
					t.Errorf("gridded surrogate %s should sum to 1 for fips %s but "+
						"instead sums to %f", code, fips, sum)
				}
			} else if sum > 1. {
				t.Errorf("gridded surrogate %s should sum to less than 1 for fips %s but "+
					"instead sums to %f", code, fips, sum)
			}
		}
	}
}

func TestSpatializeRecord(t *testing.T) {
	inputSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		t.Error(err)
	}
	r := strings.NewReader(srgSpecFileString)
	srgSpecs, err := ReadSrgSpec(r, "testdata", true)
	if err != nil {
		t.Error(err)
	}
	r = strings.NewReader(gridRefFileString)
	gridRef, err := ReadGridRef(r)
	if err != nil {
		t.Fatal(err)
	}
	grid, err := createGrid()
	if err != nil {
		t.Fatal(err)
	}
	sp := NewSpatialProcessor(srgSpecs, []*GridDef{grid}, gridRef, inputSR, true)
	//sp.DiskCachePath = "testcache"

	sourceData := SourceData{
		FIPS:    "",
		SCC:     "",
		Country: USA,
	}
	pointData := PointSourceData{
		Point: geom.Point{X: -73.9712, Y: 40.7831}, // Downtown Manhattan.
		SR:    longlat,
	}

	// Test spatial projection.
	ct, err := pointData.SR.NewTransform(sp.Grids[0].SR)
	if err != nil {
		t.Error(err)
	}
	pointI, err := pointData.Point.Transform(ct)
	if err != nil {
		t.Error(err)
	}
	point := pointI.(geom.Point)
	const xval, yval = 1.9085620728963248e+06, 329746.43597362953
	if math.Abs(point.X-xval) > 1.e-8 {
		t.Errorf("projected X coordinate should equal %g but instead is %g", xval, point.X)
	}
	if math.Abs(point.Y-yval) > 1.e-8 {
		t.Errorf("projected Y coordinate should equal %g but instead is %g", yval, point.Y)
	}

	emis := new(Emissions)
	begin, _ := time.Parse("Jan 2006", "Jan 2005")
	end, _ := time.Parse("Jan 2006", "Jan 2006")
	rate, err := parseEmisRateAnnual("1", "-9", func(v float64) *unit.Unit { return unit.New(v, unit.Kilogram) })
	if err != nil {
		t.Fatal(err)
	}
	emis.Add(begin, end, "testpol", "", rate)

	for fips, covered := range coveredByGrid {
		for _, scc := range []string{"0010200501", "2101006002", "2102001000", "2102001001"} {
			sourceData.FIPS = fips
			sourceData.SCC = scc
			for i, rec := range []Record{
				&PolygonRecord{
					SourceData: sourceData,
					Emissions:  *emis,
				},
				&PointRecord{
					SourceData:      sourceData,
					PointSourceData: pointData,
					Emissions:       *emis,
				},
			} {
				if i == 0 && testing.Short() {
					continue // Skip surrogate creation for polygon record.
				}

				emis, _, err := GriddedEmissions(rec, begin, end, sp, 0)
				if err != nil {
					t.Fatalf("scc: %s, fips: %s, i: %d, err: %v", scc, fips, i, err)
					continue
				}

				if i == 0 { // area record
					sum := emis[Pollutant{Name: "testpol"}].Sum()
					if covered {
						if math.Abs(sum-1) > 0.000001 {
							t.Errorf("%d area gridded emissions should sum to 1 for scc %s and fips %s but "+
								"instead sums to %f", i, scc, fips, sum)
						}
					} else if sum > 1 || sum <= 0 {
						t.Errorf("%d area gridded emissions should sum to between 0 and 1 for scc %s "+
							"and fips %s but instead sums to %f", i, scc, fips, sum)
					}
				} else if i == 1 { // point record
					sum := emis[Pollutant{Name: "testpol"}].Sum()
					if math.Abs(sum-1) > 0.000001 {
						t.Errorf("%d point gridded emissions should sum to 1 for scc %s and fips %s but "+
							"instead sums to %f", i, scc, fips, sum)
					}
				}
			}
		}
	}
}
