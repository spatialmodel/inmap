package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"

	"bitbucket.org/ctessum/gis"
	"bitbucket.org/ctessum/gisconversions"
	"github.com/ctessum/geomop"
	"github.com/ctessum/projgeom"
	"github.com/dhconnelly/rtreego"
	"github.com/lukeroth/gdal"
	"github.com/pebbe/go-proj-4/proj"
	"github.com/twpayne/gogeom/geom"
	"github.com/twpayne/gogeom/geom/encoding/geojson"
)

const WebMapProj = "+proj=merc +a=6378137 +b=6378137 +lat_ts=0.0 +lon_0=0.0 +x_0=0.0 +y_0=0 +k=1.0 +units=m +nadgrids=@null +no_defs"

func init() {
	gob.Register(geom.Polygon{})
}

type gridCell struct {
	Geom                                             geom.T
	WebMapGeom                                       geom.T
	bbox                                             *rtreego.Rect
	Row, Col, Layer                                  int
	Dx, Dy, Dz                                       float64
	index                                            [][2]int
	TotalPop, WhitePop, TotalPoor, WhitePoor         float64
	AllCauseMortality, RespiratoryMortality          float64 // mortalities per year per 100,000
	IWest, IEast, INorth, ISouth, IAbove, IBelow     []int
	IGroundLevel                                     []int
	UPlusSpeed, UMinusSpeed, VPlusSpeed, VMinusSpeed float64
	WPlusSpeed, WMinusSpeed                          float64
	OrgPartitioning, NOPartitioning, SPartitioning   float64
	NHPartitioning                                   float64
	SO2oxidation                                     float64
	ParticleDryDep, SO2DryDep, NOxDryDep, NH3DryDep  float64
	VOCDryDep, Kxxyy, LayerHeights                   float64
	ParticleWetDep, SO2WetDep, OtherGasWetDep        float64
	Kzz, M2u, M2d, WindSpeed                         float64
	Temperature, S1, Sclass                          float64
}

func (c *gridCell) Bounds() *rtreego.Rect {
	return c.bbox
}

func variableGrid(data map[string]dataHolder) {
	sr := gdal.CreateSpatialReference("")
	err := sr.FromProj4(config.GridProj)
	if err.Error() != "No Error" {
		panic(err)
	}
	pop := loadPopulation(sr)
	mort := loadMortality(sr)
	filePrefix := filepath.Join(config.OutputDir, config.OutputFilePrefix)
	kmax := data["UPlusSpeed"].data.Shape[0]
	var cellsBelow []*gridCell
	var cellTreeBelow *rtreego.Rtree
	var cellTreeGroundLevel *rtreego.Rtree
	id := 0
	for k := 0; k < kmax; k++ {
		os.Remove(fmt.Sprintf("%v_%v.shp", filePrefix, k))
		os.Remove(fmt.Sprintf("%v_%v.shx", filePrefix, k))
		os.Remove(fmt.Sprintf("%v_%v.dbf", filePrefix, k))
		os.Remove(fmt.Sprintf("%v_%v.prj", filePrefix, k))
		fieldNames := []string{"row", "col",
			"TotalPop", "WhitePop", "TotalPoor", "WhitePoor",
			"AllCause", "Respirator"}
		shp, err := gis.CreateShapefile(config.OutputDir,
			fmt.Sprintf("%v_%v", config.OutputFilePrefix, k), sr,
			gdal.GT_Polygon, fieldNames, 0, 0, 0., 0., 0., 0., 0., 0.)
		if err != nil {
			panic(err)
		}

		cellChan := make(chan *gridCell)
		if k < config.HiResLayers {
			go createCells(config.Xnests, config.Ynests, nil, pop, mort, cellChan)
		} else { // no nested grids above the boundary layer
			go createCells(config.Xnests[0:1], config.Ynests[0:1],
				nil, pop, mort, cellChan)
		}
		cells := make([]*gridCell, 0, config.Xnests[0]*config.Ynests[0]*2)
		cellTree := rtreego.NewTree(2, 25, 50)

		for cell := range cellChan {
			cell.bbox, err = gisconversions.GeomToRect(cell.Geom)
			if err != nil {
				panic(err)
			}
			cells = append(cells, cell)
			cellTree.Insert(cell)
		}
		sortCells(cells)
		for _, cell := range cells {
			cell.Row = id
			writeCell(shp, cell)
			id++
		}
		shp.Close()
		getNeighborsHorizontal(cells, cellTree)
		if k != 0 {
			getNeighborsAbove(cellsBelow, cellTree)
			getNeighborsBelow(cells, cellTreeBelow)
			getNeighborsGroundLevel(cells, cellTreeGroundLevel)
		} else {
			cellTreeGroundLevel = cellTree
		}
		if k != 0 {
			getData(cellsBelow, data, k-1)
			writeJsonAndGob(cellsBelow, k-1)
		}

		if k == kmax-1 {
			getData(cells, data, k)
			writeJsonAndGob(cells, k)
		}
		cellsBelow = cells
		cellTreeBelow = cellTree
	}
}

// sort the cells so that the order doesn't change between program runs.
func sortCells(cells []*gridCell) {
	sc := &cellsSorter{
		cells: cells,
	}
	sort.Sort(sc)
}

type cellsSorter struct {
	cells []*gridCell
}

// Len is part of sort.Interface.
func (c *cellsSorter) Len() int {
	return len(c.cells)
}

// Swap is part of sort.Interface.
func (c *cellsSorter) Swap(i, j int) {
	c.cells[i], c.cells[j] = c.cells[j], c.cells[i]
}

func (c *cellsSorter) Less(i, j int) bool {
	iindex := c.cells[i].index
	jindex := c.cells[j].index
	for q, _ := range iindex {
		if iindex[q][0] < jindex[q][0] {
			return true
		} else if iindex[q][0] > jindex[q][0] {
			return false
		}
		if iindex[q][1] < jindex[q][1] {
			return true
		} else if iindex[q][1] > jindex[q][1] {
			return false
		}
	}
	panic("Problem sorting")
	return false
}

func getNeighborsHorizontal(cells []*gridCell, cellTree *rtreego.Rtree) {
	for _, cell := range cells {
		b := geom.NewBounds()
		b = cell.Geom.Bounds(b)
		bboxOffset := config.BboxOffset
		westbox := newRect(b.Min.X-2*bboxOffset, b.Min.Y+bboxOffset,
			b.Min.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.IWest = getIndexes(cellTree, westbox)
		eastbox := newRect(b.Max.X+bboxOffset, b.Min.Y+bboxOffset,
			b.Max.X+2*bboxOffset, b.Max.Y-bboxOffset)
		cell.IEast = getIndexes(cellTree, eastbox)
		southbox := newRect(b.Min.X+bboxOffset, b.Min.Y-2*bboxOffset,
			b.Max.X-bboxOffset, b.Min.Y-bboxOffset)
		cell.ISouth = getIndexes(cellTree, southbox)
		northbox := newRect(b.Min.X+bboxOffset, b.Max.Y+bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y+2*bboxOffset)
		cell.INorth = getIndexes(cellTree, northbox)
	}
}

func getNeighborsAbove(cells []*gridCell, aboveCellTree *rtreego.Rtree) {
	for _, cell := range cells {
		b := geom.NewBounds()
		b = cell.Geom.Bounds(b)
		bboxOffset := config.BboxOffset
		abovebox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.IAbove = getIndexes(aboveCellTree, abovebox)
	}
}
func getNeighborsBelow(cells []*gridCell, belowCellTree *rtreego.Rtree) {
	for _, cell := range cells {
		b := geom.NewBounds()
		b = cell.Geom.Bounds(b)
		bboxOffset := config.BboxOffset
		belowbox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.IBelow = getIndexes(belowCellTree, belowbox)
	}
}
func getNeighborsGroundLevel(cells []*gridCell, groundlevelCellTree *rtreego.Rtree) {
	for _, cell := range cells {
		b := geom.NewBounds()
		b = cell.Geom.Bounds(b)
		bboxOffset := config.BboxOffset
		groundlevelbox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.IGroundLevel = getIndexes(groundlevelCellTree, groundlevelbox)
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

func writeJsonAndGob(cells []*gridCell, k int) {
	var err error
	outData := new(JsonHolderHolder)
	outData.Proj4 = config.GridProj
	outData.Type = "FeatureCollection"
	outData.Features = make([]*JsonHolder, len(cells))
	for i, cell := range cells {
		x := new(JsonHolder)
		x.Type = "Feature"
		x.Geometry, err = geojson.ToGeoJSON(cell.Geom)
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
	fname := fmt.Sprintf("%v_%v.geojson", config.OutputFilePrefix, k)
	f, err := os.Create(filepath.Join(config.OutputDir, fname))
	if err != nil {
		panic(err)
	}
	_, err = f.Write(b)
	if err != nil {
		panic(err)
	}
	f.Close()

	// Convert to google maps projection
	src, err := proj.NewProj(config.GridProj)
	if err != nil {
		panic(err)
	}
	dst, err := proj.NewProj(WebMapProj)
	if err != nil {
		panic(err)
	}
	for _, cell := range cells {
		cell.WebMapGeom, err = projgeom.Project(
			cell.Geom, src, dst, false, false)
		if err != nil {
			panic(err)
		}
	}

	fname = fmt.Sprintf("%v_%v.gob", config.OutputFilePrefix, k)
	f, err = os.Create(filepath.Join(config.OutputDir, fname))
	if err != nil {
		panic(err)
	}
	g := gob.NewEncoder(f)
	err = g.Encode(cells)
	if err != nil {
		panic(err)
	}
	f.Close()
}

func createCells(localxNests, localyNests []int, index [][2]int,
	pop, mort *rtreego.Rtree, cellChan chan *gridCell) {
	for j := 0; j < localyNests[0]; j++ {
		for i := 0; i < localxNests[0]; i++ {
			newIndex := make([][2]int, 0, len(config.Xnests))
			for _, i := range index {
				newIndex = append(newIndex, i)
			}
			newIndex = append(newIndex, [2]int{i, j})
			cell := CreateCell(pop, mort, newIndex)
			if cell.TotalPop < config.PopCutoff {
				cellChan <- cell
			} else if len(localxNests) > 1 {
				go createCells(localxNests[1:], localyNests[1:], newIndex,
					pop, mort, cellChan)
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

func CreateCell(pop, mort *rtreego.Rtree, index [][2]int) (
	cell *gridCell) {
	var err error
	xResFac, yResFac := 1., 1.
	l := config.VariableGrid_x_o
	b := config.VariableGrid_y_o
	for i, ii := range index {
		if i > 0 {
			xResFac *= float64(config.Xnests[i])
			yResFac *= float64(config.Ynests[i])
		}
		l += float64(ii[0]) * config.VariableGrid_dx / xResFac
		b += float64(ii[1]) * config.VariableGrid_dy / yResFac
	}
	r := l + config.VariableGrid_dx/xResFac
	u := b + config.VariableGrid_dy/yResFac

	cell = new(gridCell)
	cell.index = index
	// Polygon must go counter-clockwise
	cell.Geom = geom.Polygon{[][]geom.Point{{{l, b}, {r, b}, {r, u}, {l, u}, {l, b}}}}
	cellBounds, err := gisconversions.GeomToRect(cell.Geom)
	if err != nil {
		panic(err)
	}
	for _, pInterface := range pop.SearchIntersect(cellBounds) {
		p := pInterface.(*population)
		intersection := geomop.Construct(
			cell.Geom, p.Geom, geomop.INTERSECTION)
		area1 := geomop.Area(intersection)
		area2 := geomop.Area(p.Geom) // we want to conserve the total population
		if err != nil {
			panic(err)
		}
		if area2 == 0. {
			panic("divide by zero")
		}
		areaFrac := area1 / area2
		cell.TotalPop += p.totalpop * areaFrac
		cell.WhitePop += p.whitepop * areaFrac
		cell.TotalPoor += p.totalpoor * areaFrac
		cell.WhitePoor += p.whitepoor * areaFrac
	}
	for _, mInterface := range mort.SearchIntersect(cellBounds) {
		m := mInterface.(*mortality)
		intersection := geomop.Construct(
			cell.Geom, m.Geom, geomop.INTERSECTION)
		area1 := geomop.Area(intersection)
		area2 := geomop.Area(cell.Geom) // we want to conserve the average rate here, not the total
		if area2 == 0. {
			panic("divide by zero")
		}
		areaFrac := area1 / area2
		cell.AllCauseMortality += m.AllCause * areaFrac
		cell.RespiratoryMortality += m.Respiratory * areaFrac
	}
	cell.Dx = r - l
	cell.Dy = u - b
	//fmt.Println(index, cell.Totalpop, cell.Dx, cell.Dy)
	return
}

func writeCell(shp *gis.Shapefile, cell *gridCell) {
	fieldIDs := []int{0, 1, 2, 3, 4, 5, 6, 7}
	err := shp.WriteFeature(cell.Row, cell.Geom, fieldIDs,
		cell.Row, cell.Col, cell.TotalPop, cell.WhitePop,
		cell.TotalPoor, cell.WhitePoor, cell.AllCauseMortality,
		cell.RespiratoryMortality)
	if err != nil {
		panic(err)
	}
}

type population struct {
	bounds                                   *rtreego.Rect
	Geom                                     geom.T
	totalpop, whitepop, totalpoor, whitepoor float64
}

func (p *population) Bounds() *rtreego.Rect {
	return p.bounds
}

type mortality struct {
	bounds                *rtreego.Rect
	Geom                  geom.T
	AllCause, Respiratory float64 // Deaths per 100,000 people per year
}

func (m *mortality) Bounds() *rtreego.Rect {
	return m.bounds
}

func loadPopulation(sr gdal.SpatialReference) (
	pop *rtreego.Rtree) {
	var err error
	f := filepath.Join(config.CensusDir, config.CensusFile)
	popshp, err := gis.OpenShapefile(f, true)
	if err != nil {
		panic(err)
	}
	ct, err := gisconversions.NewCoordinateTransform(popshp.Sr, sr)
	if err != nil {
		panic(err)
	}
	iTotalPop, err := popshp.GetColumnIndex("TotalPop")
	if err != nil {
		panic(err)
	}
	iWhitePop, err := popshp.GetColumnIndex("TotalWhite")
	if err != nil {
		panic(err)
	}
	iTotalPoor, err := popshp.GetColumnIndex("TotalPoor")
	if err != nil {
		panic(err)
	}
	iWhitePoor, err := popshp.GetColumnIndex("WhitePoor")
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
		case float32:
			p.totalpop = float64(fieldVals[0].(float32))
			p.whitepop = float64(fieldVals[1].(float32))
			p.totalpoor = float64(fieldVals[2].(float32))
			p.whitepoor = float64(fieldVals[3].(float32))
		case int:
			p.totalpop = float64(fieldVals[0].(int))
			p.whitepop = float64(fieldVals[1].(int))
			p.totalpoor = float64(fieldVals[2].(int))
			p.whitepoor = float64(fieldVals[3].(int))
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
		p.Geom, err = ct.Reproject(g)
		if err != nil {
			panic(err)
		}
		p.bounds, err = gisconversions.GeomToRect(p.Geom)
		if err != nil {
			panic(err)
		}
		pop.Insert(p)
	}
	return
}

func loadMortality(sr gdal.SpatialReference) (
	mort *rtreego.Rtree) {
	var err error
	f := filepath.Join(config.CensusDir, config.MortalityRateFile)
	mortshp, err := gis.OpenShapefile(f, true)
	if err != nil {
		panic(err)
	}
	ct, err := gisconversions.NewCoordinateTransform(mortshp.Sr, sr)
	if err != nil {
		panic(err)
	}
	iAllCause, err := mortshp.GetColumnIndex("AllCause")
	if err != nil {
		panic(err)
	}
	iRespiratory, err := mortshp.GetColumnIndex("Respirator")
	if err != nil {
		panic(err)
	}

	mort = rtreego.NewTree(2, 25, 50)
	for {
		g, fieldVals, err := mortshp.ReadNextFeature(
			iAllCause, iRespiratory)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		m := new(mortality)
		switch fieldVals[0].(type) {
		case float64:
			m.AllCause = fieldVals[0].(float64)
			m.Respiratory = fieldVals[1].(float64)
		case error:
			if err != nil {
				panic(err)
			}
		default:
			panic("Unknown type")
		}
		if math.IsNaN(m.AllCause) || math.IsNaN(m.Respiratory) {
			panic("NaN!")
		}
		m.Geom, err = ct.Reproject(g)
		if err != nil {
			panic(err)
		}
		m.bounds, err = gisconversions.GeomToRect(m.Geom)
		if err != nil {
			panic(err)
		}
		mort.Insert(m)
	}
	return
}

func getData(cells []*gridCell, data map[string]dataHolder, k int) {
	ctmtree := makeCTMgrid()
	for _, cell := range cells {
		cell.Layer = k
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
			cell.SO2oxidation += data["SO2oxidation"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.ParticleDryDep += data["ParticleDryDep"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.SO2DryDep += data["SO2DryDep"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.NOxDryDep += data["NOxDryDep"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.NH3DryDep += data["NH3DryDep"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.VOCDryDep += data["VOCDryDep"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.Kxxyy += data["Kxxyy"].data.Get(
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
				k, ctmrow, ctmcol) / ncells
			cell.M2d += data["M2d"].data.Get(
				k, ctmrow, ctmcol) / ncells
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
	for ix := 0; ix < config.CtmGrid_nx; ix++ {
		for iy := 0; iy < config.CtmGrid_ny; iy++ {
			cell := new(gridCellLight)
			p := rtreego.Point{config.CtmGrid_x_o + config.CtmGrid_dx*float64(ix),
				config.CtmGrid_y_o + config.CtmGrid_dy*float64(iy)}
			lengths := []float64{config.CtmGrid_dx, config.CtmGrid_dy}
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
