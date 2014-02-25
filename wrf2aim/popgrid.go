package main

import (
	"bitbucket.org/ctessum/gis"
	"encoding/json"
	"fmt"
	"github.com/dhconnelly/rtreego"
	"github.com/lukeroth/gdal"
	"github.com/paulsmith/gogeos/geos"
	"github.com/twpayne/gogeom/geom"
	"github.com/twpayne/gogeom/geom/encoding/geojson"
	"io"
	"math"
	"os"
	"path/filepath"
)

const (
	variableGrid_x_o = -2736000. // lower left of grid, x
	variableGrid_y_o = -2088000. // lower left of grid, y
	variableGrid_dx  = 36000.    // m
	variableGrid_dy  = 36000.    // m
	ctmGrid_x_o      = -2736000. // lower left of grid, x
	ctmGrid_y_o      = -2088000. // lower left of grid, y
	ctmGrid_dx       = 12000.    // m
	ctmGrid_dy       = 12000.    // m
	ctmGrid_nx       = 444
	ctmGrid_ny       = 336
	proj             = "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1"
	popCutoff        = 50000 // people per grid cell
	bboxOffset       = 1.    // A number significantly less than the smallest grid size but not small enough to be confused with zero.
	censusDir        = "/home/marshall/tessumcm/src/bitbucket.org/ctessum/aim/wrf2aim/census2000"
)

var (
	xNests = []int{148, 3, 3, 4}
	yNests = []int{112, 3, 3, 4}
	//xNests = []int{10, 3, 3, 4}
	//yNests = []int{10, 3, 3, 4}
)

type gridCell struct {
	ggeom                                            *geos.Geometry
	geom                                             geom.T
	bbox                                             *rtreego.Rect
	Row, Col, Layer                                  int
	Dx, Dy, Dz                                       float64
	index                                            [][2]int
	Totalpop, Whitepop, Totalpoor, Whitepoor         float64
	IWest, IEast, INorth, ISouth, IAbove, IBelow     []int
	IGroundLevel                                     []int
	UPlusSpeed, UMinusSpeed, VPlusSpeed, VMinusSpeed float64
	WPlusSpeed, WMinusSpeed                          float64
	OrgPartitioning, NOPartitioning, SPartitioning   float64
	NHPartitioning, FracAmmoniaPoor                  float64
	SO2oxidation                                     float64
	ParticleDryDep, SO2DryDep, NOxDryDep, NH3DryDep  float64
	VOCDryDep, Kyyxx, LayerHeights                   float64
	ParticleWetDep, SO2WetDep, OtherGasWetDep        float64
	Kzz, M2u, M2d, PblTopLayer, Pblh, WindSpeed      float64
	Temperature, S1, Sclass                          float64
}

func (c *gridCell) Bounds() *rtreego.Rect {
	return c.bbox
}

func variableGrid(data map[string]dataHolder) {
	sr := gdal.CreateSpatialReference("")
	err := sr.FromProj4(proj)
	if err.Error() != "No Error" {
		panic(err)
	}
	filePrefix := filepath.Join(outputDir, outputFilePrefix)
	os.Remove(filePrefix + ".shp")
	os.Remove(filePrefix + ".shx")
	os.Remove(filePrefix + ".dbf")
	os.Remove(filePrefix + ".prj")
	fieldNames := []string{"row", "col",
		"TotalPop", "WhitePop", "TotalPoor", "WhitePoor"}
	shp, err := gis.CreateShapefile(outputDir, outputFilePrefix, sr,
		gdal.GT_Polygon, fieldNames, 0, 0, 0., 0., 0., 0.)
	if err != nil {
		panic(err)
	}
	pop := loadPopulation(sr)

	cellChan := make(chan *gridCell)
	go createCells(xNests, yNests, nil, pop, cellChan)
	cells := make([]*gridCell, 0, xNests[0]*yNests[0]*2)
	cellTree := rtreego.NewTree(2, 25, 50)

	id := 0
	for cell := range cellChan {
		cell.Row = id
		cell.geom, err = gis.GEOStoGeom(cell.ggeom)
		if err != nil {
			panic(err)
		}
		cell.bbox, err = gis.GeomToRect(cell.geom)
		if err != nil {
			panic(err)
		}
		writeCell(shp, cell)
		cells = append(cells, cell)
		cellTree.Insert(cell)
		id++
	}
	shp.Close()
	getNeighbors(cells, cellTree)
	kmax := data["UPlusSpeed"].data.Shape[0]
	for k := 0; k < kmax; k++ {
		getData(cells, data, k, kmax)
		writeJson(cells, k)
	}
}

func getNeighbors(cells []*gridCell, cellTree *rtreego.Rtree) {
	for _, cell := range cells {
		b := geom.NewBounds()
		b = cell.geom.Bounds(b)
		westbox := newRect(b.Min.X-2*bboxOffset, b.Min.Y+bboxOffset,
			b.Min.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.IWest = getIndexes(cellTree, westbox)
		eastbox := newRect(b.Max.X+bboxOffset, b.Min.Y+bboxOffset,
			b.Max.X+2*bboxOffset, b.Max.Y-bboxOffset)
		cell.IEast = getIndexes(cellTree, eastbox)
		southbox := newRect(b.Min.X+bboxOffset, b.Min.Y-2*bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.ISouth = getIndexes(cellTree, southbox)
		northbox := newRect(b.Min.X+bboxOffset, b.Max.Y+bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y+2*bboxOffset)
		cell.INorth = getIndexes(cellTree, northbox)
	}
}

func getIndexes(cellTree *rtreego.Rtree, box *rtreego.Rect) []int {
	x := cellTree.SearchIntersect(box)
	indexes := make([]int, len(x))
	for i, xx := range x {
		indexes[i] = xx.(*gridCell).Row
	}
	return indexes
}

func newRect(xmin, ymin, xmax, ymax float64) *rtreego.Rect {
	p := rtreego.Point{xmin, ymin}
	lengths := []float64{xmax - xmin, ymax - ymin}
	r, err := rtreego.NewRect(p, lengths)
	if err != nil {
		panic(err)
	}
	return r
}

type JsonHolder struct {
	Type       string
	Geometry   *geojson.Geometry
	Properties *gridCell
}
type JsonHolderHolder struct {
	Proj4, Type string
	Features    []*JsonHolder
}

func writeJson(cells []*gridCell, k int) {
	var err error
	outData := new(JsonHolderHolder)
	outData.Proj4 = proj
	outData.Type = "FeatureCollection"
	outData.Features = make([]*JsonHolder, len(cells))
	for i, cell := range cells {
		x := new(JsonHolder)
		x.Type = "Feature"
		x.Geometry, err = geojson.ToGeoJSON(cell.geom)
		if err != nil {
			panic(err)
		}
		x.Properties = cell
		outData.Features[i] = x
	}
	b, err := json.Marshal(outData)
	if err != nil {
		panic(err)
	}
	fname := fmt.Sprintf("%v_%v.geojson", outputFilePrefix, k)
	f, err := os.Create(filepath.Join(outputDir, fname))
	if err != nil {
		panic(err)
	}
	_, err = f.Write(b)
	if err != nil {
		panic(err)
	}
	f.Close()
}

func createCells(localxNests, localyNests []int, index [][2]int,
	pop *rtreego.Rtree, cellChan chan *gridCell) {
	for j := 0; j < localyNests[0]; j++ {
		for i := 0; i < localxNests[0]; i++ {
			newIndex := make([][2]int, 0, len(xNests))
			for _, i := range index {
				newIndex = append(newIndex, i)
			}
			newIndex = append(newIndex, [2]int{i, j})
			cell := CreateCell(pop, newIndex)
			if cell.Totalpop < popCutoff {
				cellChan <- cell
			} else if len(localxNests) > 1 {
				go createCells(localxNests[1:], localyNests[1:], newIndex,
					pop, cellChan)
			} else {
				cellChan <- cell
			}
		}
	}
	// close chan if this is the outer function
	if len(index) == 0 {
		close(cellChan)
	}
	return
}

func CreateCell(pop *rtreego.Rtree, index [][2]int) (
	cell *gridCell) {
	var err error
	xResFac, yResFac := 1., 1.
	l := variableGrid_x_o
	b := variableGrid_y_o
	for i, ii := range index {
		if i > 0 {
			xResFac *= float64(xNests[i])
			yResFac *= float64(yNests[i])
		}
		l += float64(ii[0]) * variableGrid_dx / xResFac
		b += float64(ii[1]) * variableGrid_dy / yResFac
	}
	r := l + variableGrid_dx/xResFac
	u := b + variableGrid_dy/yResFac

	cell = new(gridCell)
	cell.index = index
	// Polygon must go counter-clockwise
	wkt := fmt.Sprintf("POLYGON ((%v %v, %v %v, %v %v, %v %v, %v %v))",
		l, b, r, b, r, u, l, u, l, b)
	cell.ggeom, err = geos.FromWKT(wkt)
	if err != nil {
		panic(err)
	}
	cellBounds, err := gis.GeosToRect(cell.ggeom)
	if err != nil {
		panic(err)
	}
	var intersection *geos.Geometry
	var intersects bool
	for pp, pInterface := range pop.SearchIntersect(cellBounds) {
		p := pInterface.(*population)
		intersects, err = cell.ggeom.Intersects(p.geom)
		if err != nil {
			fmt.Println("xxxxxxxx", pp)
			panic(err)
		}
		if intersects {
			intersection, err =
				IntersectionFaultTolerant(cell.ggeom, p.geom)
			area1, err := intersection.Area()
			if err != nil {
				panic(err)
			}
			area2, err := p.geom.Area()
			if err != nil {
				panic(err)
			}
			if area2 == 0. {
				panic("divide by zero")
			}
			areaFrac := area1 / area2
			cell.Totalpop += p.totalpop * areaFrac
			cell.Whitepop += p.whitepop * areaFrac
			cell.Totalpoor += p.totalpoor * areaFrac
			cell.Whitepoor += p.whitepoor * areaFrac
		}
	}
	cell.Dx = r - l
	cell.Dy = u - b
	//fmt.Println(index, cell.Totalpop, cell.Dx, cell.Dy)
	return
}

func writeCell(shp *gis.Shapefile, cell *gridCell) {
	fieldIDs := []int{0, 1, 2, 3, 4, 5}
	err := shp.WriteFeature(cell.Row, cell.ggeom, fieldIDs,
		cell.Row, cell.Col, cell.Totalpop, cell.Whitepop,
		cell.Totalpoor, cell.Whitepoor)
	if err != nil {
		panic(err)
	}
}

type population struct {
	bounds                                   *rtreego.Rect
	geom                                     *geos.Geometry
	totalpop, whitepop, totalpoor, whitepoor float64
}

func (p *population) Bounds() *rtreego.Rect {
	return p.bounds
}

func loadPopulation(sr gdal.SpatialReference) (pop *rtreego.Rtree) {
	var err error
	f := filepath.Join(censusDir, "Census2000.shp")
	popshp, err := gis.OpenShapefile(f, true)
	if err != nil {
		panic(err)
	}
	ct, err := gis.NewCoordinateTransform(popshp.Sr, sr)
	if err != nil {
		panic(err)
	}
	iTotalPop, err := popshp.GetColumnIndex("TOTALPOP")
	if err != nil {
		panic(err)
	}
	iWhitePop, err := popshp.GetColumnIndex("WHITEPOP")
	if err != nil {
		panic(err)
	}
	iTotalPoor, err := popshp.GetColumnIndex("TOTALPOOR")
	if err != nil {
		panic(err)
	}
	iWhitePoor, err := popshp.GetColumnIndex("WHITEPOOR")
	if err != nil {
		panic(err)
	}

	pop = rtreego.NewTree(2, 25, 50)
	for {
		g, fieldVals, err := popshp.ReadNextFeature(
			iTotalPop, iWhitePop, iTotalPoor, iWhitePoor)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		p := new(population)
		switch fieldVals[0].(type) {
		case float64:
			p.totalpop = fieldVals[0].(float64)
			p.whitepop = fieldVals[1].(float64)
			p.totalpoor = fieldVals[2].(float64)
			p.whitepoor = fieldVals[3].(float64)
		case error:
			if err != nil {
				panic(err)
			}
		default:
			panic("Unknown type")
		}
		if math.IsNaN(p.totalpop) || math.IsNaN(p.whitepop) ||
			math.IsNaN(p.totalpoor) || math.IsNaN(p.whitepoor) {
			panic("NaN!")
		}
		if p.totalpop == 0. {
			continue
		}
		g, err = ct.Reproject(g)
		if err != nil {
			panic(err)
		}
		p.bounds, err = gis.GeomToRect(g)
		if err != nil {
			panic(err)
		}
		p.geom, err = gis.GeomToGEOS(g)
		if err != nil {
			panic(err)
		}
		pop.Insert(p)
	}
	return
}

func getData(cells []*gridCell, data map[string]dataHolder, k, kmax int) {
	ctmtree := makeCTMgrid()
	for _, cell := range cells {
		cell.UPlusSpeed = 0.
		cell.UMinusSpeed = 0.
		cell.VPlusSpeed = 0.
		cell.VMinusSpeed = 0.
		cell.WPlusSpeed = 0.
		cell.WMinusSpeed = 0.
		cell.OrgPartitioning = 0.
		cell.NOPartitioning = 0.
		cell.SPartitioning = 0.
		cell.NHPartitioning = 0.
		cell.FracAmmoniaPoor = 0.
		cell.SO2oxidation = 0.
		cell.ParticleDryDep = 0.
		cell.SO2DryDep = 0.
		cell.NOxDryDep = 0.
		cell.NH3DryDep = 0.
		cell.VOCDryDep = 0.
		cell.Kyyxx = 0.
		cell.LayerHeights = 0.
		cell.Dz = 0.
		cell.ParticleWetDep = 0.
		cell.SO2WetDep = 0.
		cell.OtherGasWetDep = 0.
		cell.Kzz = 0.
		cell.M2u = 0.
		cell.M2d = 0.
		cell.PblTopLayer = 0.
		cell.Pblh = 0.
		cell.WindSpeed = 0.
		cell.Temperature = 0.
		cell.S1 = 0.
		cell.Sclass = 0.

		cell.Layer = k
		// Link with cells above and below.
		if k != 0 {
			cell.Row += len(cells)
			cell.IBelow = []int{cell.Row - len(cells)}
		}
		cell.IGroundLevel = []int{cell.Row - k*len(cells)}
		if k != kmax-1 {
			cell.IAbove = []int{cell.Row + len(cells)}
		}

		ctmcells := ctmtree.SearchIntersect(cell.bbox)
		ncells := float64(len(ctmcells))
		if len(ctmcells) == 0. {
			panic("No matching cells!")
		}
		for _, c := range ctmcells {
			ctmrow := c.(*gridCellLight).Row
			ctmcol := c.(*gridCellLight).Col

			cell.UPlusSpeed += data["UPlusSpeed"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.UMinusSpeed += data["UMinusSpeed"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.VPlusSpeed += data["VPlusSpeed"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.VMinusSpeed += data["VMinusSpeed"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.WPlusSpeed += data["WPlusSpeed"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.WMinusSpeed += data["WMinusSpeed"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.OrgPartitioning += data["OrgPartitioning"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.NOPartitioning += data["NOPartitioning"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.SPartitioning += data["SPartitioning"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.NHPartitioning += data["NHPartitioning"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.FracAmmoniaPoor += data["FracAmmoniaPoor"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.SO2oxidation += data["SO2oxidation"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.ParticleDryDep += data["ParticleDryDep"].data.Get(
				ctmrow, ctmcol) / ncells
			cell.SO2DryDep += data["SO2DryDep"].data.Get(
				ctmrow, ctmcol) / ncells
			cell.NOxDryDep += data["NOxDryDep"].data.Get(
				ctmrow, ctmcol) / ncells
			cell.NH3DryDep += data["NH3DryDep"].data.Get(
				ctmrow, ctmcol) / ncells
			cell.VOCDryDep += data["VOCDryDep"].data.Get(
				ctmrow, ctmcol) / ncells
			cell.Kyyxx += data["Kyy"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.LayerHeights += data["LayerHeights"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.Dz += data["Dz"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.ParticleWetDep += data["ParticleWetDep"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.SO2WetDep += data["SO2WetDep"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.OtherGasWetDep += data["OtherGasWetDep"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.Kzz += data["Kzz"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.M2u += data["M2u"].data.Get(
				ctmrow, ctmcol) / ncells
			cell.M2d += data["M2d"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.PblTopLayer += data["PblTopLayer"].data.Get(
				ctmrow, ctmcol) / ncells
			cell.Pblh += data["Pblh"].data.Get(
				ctmrow, ctmcol) / ncells
			cell.WindSpeed += data["WindSpeed"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.Temperature += data["Temperature"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.S1 += data["S1"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.Sclass += data["Sclass"].data.Get(
				k, ctmrow, ctmcol) / ncells
		}
	}
}

// make a vector representation of the chemical transport model grid
func makeCTMgrid() *rtreego.Rtree {
	var err error
	tree := rtreego.NewTree(2, 25, 50)
	for ix := 0; ix < ctmGrid_nx; ix++ {
		for iy := 0; iy < ctmGrid_ny; iy++ {
			cell := new(gridCellLight)
			p := rtreego.Point{ctmGrid_x_o + ctmGrid_dx*float64(ix),
				ctmGrid_y_o + ctmGrid_dy*float64(iy)}
			lengths := []float64{ctmGrid_dx, ctmGrid_dy}
			cell.bbox, err = rtreego.NewRect(p, lengths)
			if err != nil {
				panic(err)
			}
			cell.Row = iy
			cell.Col = ix
			tree.Insert(cell)
		}
	}
	return tree
}

type gridCellLight struct {
	bbox     *rtreego.Rect
	Row, Col int
}

func (c *gridCellLight) Bounds() *rtreego.Rect {
	return c.bbox
}

func IntersectionFaultTolerant(g1, g2 *geos.Geometry) (g3 *geos.Geometry,
	err error) {
	var buf1, buf2 *geos.Geometry
	g3, err = g1.Intersection(g2)
	if err != nil { // If there is a problem, try a 0 buffer
		buf1, err = g1.Buffer(0.)
		if err != nil {
			return
		}
		buf2, err = g2.Buffer(0.)
		if err != nil {
			return
		}
		g3, err = buf1.Intersection(buf2)
		if err != nil {
			return
		}
	}
	return
}
