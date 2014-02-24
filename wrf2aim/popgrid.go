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
	"runtime"
	//"time"
)

const (
	x_o        = -2736000. // lower left of grid, x
	y_o        = -2088000. // lower left of grid, y
	dx         = 36000.    // m
	dy         = 36000.    // m
	proj       = "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1"
	popCutoff  = 50000 // people per grid cell
	bboxOffset = 1.    // A number significantly less than the smallest grid size but not small enough to be confused with zero.
)

var (
	shp      *gis.Shapefile
	sr       gdal.SpatialReference
	numNests = 4
	xNests   = []int{148, 3, 3, 4}
	yNests   = []int{112, 3, 3, 4}
	//xNests = []int{10, 3, 3, 4}
	//yNests = []int{10, 3, 3, 4}
)

type gridCell struct {
	geom                                     *geos.Geometry
	ggeom                                    geom.T
	bbox                                     *rtreego.Rect
	Row, Col                                 int
	index                                    [][2]int
	Totalpop, Whitepop, Totalpoor, Whitepoor float64
	West, East, North, South, Above, Below   []int
}

func (c *gridCell) Bounds() *rtreego.Rect {
	return c.bbox
}

func init() {
	sr = gdal.CreateSpatialReference("")
	err := sr.FromProj4(proj)
	if err.Error() != "No Error" {
		panic(err)
	}
	os.Remove("popgrid.shp")
	os.Remove("popgrid.shx")
	os.Remove("popgrid.dbf")
	os.Remove("popgrid.prj")
	fieldNames := []string{"row", "col",
		"TotalPop", "WhitePop", "TotalPoor", "WhitePoor"}
	shp, err = gis.CreateShapefile(".", "popgrid", sr, gdal.GT_Polygon,
		fieldNames, 0, 0, 0., 0., 0., 0.)
	if err != nil {
		panic(err)
	}
	runtime.GOMAXPROCS(8)
}

func main() {
	var err error
	pop := loadPopulation()

	cellChan := make(chan *gridCell)
	go createCells(xNests, yNests, nil, pop, cellChan)
	cells := make([]*gridCell, 0, xNests[0]*yNests[0]*2)
	cellTree := rtreego.NewTree(2, 25, 50)

	id := 0
	for cell := range cellChan {
		cell.Row = id
		cell.ggeom, err = gis.GEOStoGeom(cell.geom)
		if err != nil {
			panic(err)
		}
		cell.bbox, err = gis.GeomToRect(cell.ggeom)
		if err != nil {
			panic(err)
		}
		writeCell(cell)
		cells = append(cells, cell)
		cellTree.Insert(cell)
		id++
	}
	shp.Close()
	getNeighbors(cells, cellTree)
	writeJson(cells)
}

func getNeighbors(cells []*gridCell, cellTree *rtreego.Rtree) {
	for _, cell := range cells {
		b := geom.NewBounds()
		b = cell.ggeom.Bounds(b)
		westbox := newRect(b.Min.X-2*bboxOffset, b.Min.Y+bboxOffset,
			b.Min.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.West = getIndexes(cellTree, westbox)
		eastbox := newRect(b.Max.X+bboxOffset, b.Min.Y+bboxOffset,
			b.Max.X+2*bboxOffset, b.Max.Y-bboxOffset)
		cell.East = getIndexes(cellTree, eastbox)
		southbox := newRect(b.Min.X+bboxOffset, b.Min.Y-2*bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.South = getIndexes(cellTree, southbox)
		northbox := newRect(b.Min.X+bboxOffset, b.Max.Y+bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y+2*bboxOffset)
		cell.North = getIndexes(cellTree, northbox)
	}
}

func getIndexes(cellTree *rtreego.Rtree, box *rtreego.Rect) []int {
	x := cellTree.SearchIntersect(box)
	//	if len(x) == 0 {
	//		panic(fmt.Errorf("There are no neighbors for cell %v!", cell.index))
	//	}
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

func writeJson(cells []*gridCell) {
	var err error
	outData := new(JsonHolderHolder)
	outData.Proj4 = proj
	outData.Type = "FeatureCollection"
	outData.Features = make([]*JsonHolder, len(cells))
	for i, cell := range cells {
		x := new(JsonHolder)
		x.Type = "Feature"
		x.Geometry, err = geojson.ToGeoJSON(cell.ggeom)
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
	f, err := os.Create("popgrid.geojson")
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
			newIndex := make([][2]int, 0, numNests)
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
	l := x_o
	b := y_o
	for i, ii := range index {
		if i > 0 {
			xResFac *= float64(xNests[i])
			yResFac *= float64(yNests[i])
		}
		l += float64(ii[0]) * dx / xResFac
		b += float64(ii[1]) * dy / yResFac
	}
	r := l + dx/xResFac
	u := b + dy/yResFac

	cell = new(gridCell)
	cell.index = index
	// Polygon must go counter-clockwise
	wkt := fmt.Sprintf("POLYGON ((%v %v, %v %v, %v %v, %v %v, %v %v))",
		l, b, r, b, r, u, l, u, l, b)
	cell.geom, err = geos.FromWKT(wkt)
	if err != nil {
		panic(err)
	}
	cellBounds, err := gis.GeosToRect(cell.geom)
	if err != nil {
		panic(err)
	}
	var intersection, buf1, buf2 *geos.Geometry
	var intersects bool
	for pp, pInterface := range pop.SearchIntersect(cellBounds) {
		p := pInterface.(*population)
		intersects, err = cell.geom.Intersects(p.geom)
		if err != nil {
			fmt.Println("xxxxxxxx", pp)
			panic(err)
		}
		if intersects {
			intersection, err = cell.geom.Intersection(p.geom)
			if err != nil { // If there is a problem, try a 0 buffer
				buf1, err = cell.geom.Buffer(0.)
				if err != nil {
					panic(err)
				}
				buf2, err = p.geom.Buffer(0.)
				if err != nil {
					panic(err)
				}
				intersection, err = buf1.Intersection(buf2)
				if err != nil {
					panic(err)
				}
			}
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
	fmt.Println(index, cell.Totalpop, r-l, u-b)
	return
}

func writeCell(cell *gridCell) {
	fieldIDs := []int{0, 1, 2, 3, 4, 5}
	err := shp.WriteFeature(cell.Row, cell.geom, fieldIDs,
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

func loadPopulation() (pop *rtreego.Rtree) {
	var err error
	f := "/media/chris/data1/Documents/Graduate_School/" +
		"Research/Census/shp/Census2000.shp"
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
