package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	PopData                                          map[string]float64 // Population for multiple demographic types
	MortalityRate                                    float64            // mortalities per year per 100,000
	IWest, IEast, INorth, ISouth, IAbove, IBelow     []int
	IGroundLevel                                     []int
	UPlusSpeed, UMinusSpeed, VPlusSpeed, VMinusSpeed float64
	WPlusSpeed, WMinusSpeed                          float64
	AOrgPartitioning, BOrgPartitioning               float64
	NOPartitioning, SPartitioning                    float64
	NHPartitioning                                   float64
	SO2oxidation                                     float64
	ParticleDryDep, SO2DryDep, NOxDryDep, NH3DryDep  float64
	VOCDryDep, Kxxyy, LayerHeights                   float64
	ParticleWetDep, SO2WetDep, OtherGasWetDep        float64
	Kzz, M2u, M2d, WindSpeed                         float64
	Temperature, S1, Sclass                          float64
}

func (c gridCell) Bounds() *rtreego.Rect {
	return c.bbox
}

func (c *gridCell) copyCell() *gridCell {
	o := new(gridCell)
	o.Geom = c.Geom
	o.WebMapGeom = c.WebMapGeom
	o.PopData = make(map[string]float64)
	for key, val := range c.PopData {
		o.PopData[key] = val
	}
	o.MortalityRate = c.MortalityRate
	o.index = c.index
	o.Dx, o.Dy = c.Dx, c.Dy
	o.bbox = c.bbox
	o.MortalityRate = c.MortalityRate
	return o
}

func variableGrid(data map[string]dataHolder) {
	sr := gdal.CreateSpatialReference("")
	err := sr.FromProj4(config.GridProj)
	if err.Error() != "No Error" {
		panic(err)
	}
	log.Println("Loading population")
	pop := loadPopulation(sr)
	log.Println("Loading mortality")
	mort := loadMortality(sr)
	log.Println("Loaded mortality")
	filePrefix := filepath.Join(config.OutputDir, config.OutputFilePrefix)
	kmax := data["UPlusSpeed"].data.Shape[0]
	var cellsBelow []*gridCell
	var cellTreeBelow *rtreego.Rtree
	var cellTreeGroundLevel *rtreego.Rtree
	id := 0
	for k := 0; k < kmax; k++ {
		log.Println("Creating variable grid for layer ", k)
		os.Remove(fmt.Sprintf("%v_%v.shp", filePrefix, k))
		os.Remove(fmt.Sprintf("%v_%v.shx", filePrefix, k))
		os.Remove(fmt.Sprintf("%v_%v.dbf", filePrefix, k))
		os.Remove(fmt.Sprintf("%v_%v.prj", filePrefix, k))
		fieldNames := append([]string{"row", "col"}, config.CensusPopColumns...)
		fieldNames = append(fieldNames, config.MortalityRateColumn)
		outType := make([]interface{}, len(fieldNames))
		outType[0] = 0 // int
		outType[1] = 0 // int
		for i := 2; i < len(fieldNames); i++ {
			outType[i] = 0. // float64
		}
		shp, err := gis.CreateShapefile(config.OutputDir,
			fmt.Sprintf("%v_%v", config.OutputFilePrefix, k), sr,
			gdal.GT_Polygon, fieldNames, outType...)
		if err != nil {
			panic(err)
		}

		var cells []*gridCell
		if k < config.HiResLayers {
			cells = createCells(config.Xnests, config.Ynests, nil, pop, mort)
		} else { // no nested grids above the boundary layer
			cells = createCells(config.Xnests[0:1], config.Ynests[0:1],
				nil, pop, mort)
		}
		cellTree := rtreego.NewTree(2, 25, 50)

		for _, cell := range cells {
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
		log.Println("Created variable grid for layer ", k)
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
	panic(fmt.Errorf("Problem sorting: iindex: %v, jindex: %v",
		iindex, jindex))
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
	src := gdal.CreateSpatialReference("")
	err = src.FromProj4(config.GridProj)
	if err.Error() != "No Error" {
		panic(err)
	}
	dst := gdal.CreateSpatialReference("")
	err = dst.FromProj4(WebMapProj)
	if err.Error() != "No Error" {
		panic(err)
	}
	ct, err := projgeom.NewCoordinateTransform(src, dst)
	if err != nil {
		panic(err)
	}
	for _, cell := range cells {
		cell.WebMapGeom, err = ct.Reproject(cell.Geom)
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

// Cycle through all of the indicies in the given nest.
// For each index first recursively create the grid
// cell in the current nest and all of the grid cells for
// all of the nests inside of this one using the same rules
// described here. If the current grid cell and
// all the gridcells in all of the
// nests inside of this one are below the population thresholds
// (for both total population and population density),
// then discard the inner nests and keep the grid cell in the current
// nest. If the current nest grid cell or at least one of the grid cells
// in the inner nests is
// above the population threshold, then keep the grid cells from
// the inner nests and discard the grid cell in the current nest.
func createCells(localxNests, localyNests []int, index [][2]int,
	pop, mort *rtreego.Rtree) []*gridCell {

	var nextNestCells []*gridCell
	var cell, tempCell *gridCell
	arrayCap := 0
	for ii := 0; ii < len(localyNests); ii++ {
		arrayCap += localyNests[ii] * localxNests[ii]
	}
	cells := make([]*gridCell, 0, arrayCap)
	// Iterate through indices and send them to the concurrent cell generator
	for j := 0; j < localyNests[0]; j++ {
		for i := 0; i < localxNests[0]; i++ {
			newIndex := make([][2]int, 0, len(config.Xnests))
			for _, i := range index {
				newIndex = append(newIndex, i)
			}
			newIndex = append(newIndex, [2]int{i, j})
			// Create the cell in this nest
			cell = CreateCell(pop, mort, newIndex)
			if len(localxNests) > 1 {
				// If this isn't the innermost nest, recursively create all of
				// the cells in all the nests inside of this one.
				nextNestCells = createCells(localxNests[1:],
					localyNests[1:], newIndex, pop, mort)

				// Check whether the cell in this nest and all cells in the
				// inner nests are below the population cutoffs.
				allCellsBelowCutoff := true
				cellPop := cell.PopData[config.PopGridColumn]
				if cellPop/cell.Dx/cell.Dy > config.PopDensityCutoff ||
					cellPop > config.PopCutoff {
					allCellsBelowCutoff = false
				}
				for _, tempCell = range nextNestCells {
					cellPop := tempCell.PopData[config.PopGridColumn]
					if cellPop/tempCell.Dx/tempCell.Dy > config.PopDensityCutoff ||
						cellPop > config.PopCutoff {
						allCellsBelowCutoff = false
					}
				}
				// If all of the cells in this nest and the next nests are below the
				// population cutoffs (for both density and total population),
				// then add the cell in this nest to the array of cells to keep. Discard
				// the cells in the next nest.
				if allCellsBelowCutoff {
					cells = append(cells, cell)
				} else {
					// If at least one of the cells in this nest or the inner nests is
					// above the population cutoffs, then keep the grid
					// cells from the inner nests and discard grid
					// cell for the current nest.
					cells = append(cells, nextNestCells...)
				}
			} else {
				// If this is the innermost nest, just add the
				// current cell to the array of cells to keep.
				cells = append(cells, cell)
			}
		}
	}
	return cells
}

var cellCache map[string]*gridCell

func init() {
	cellCache = make(map[string]*gridCell)
}

func CreateCell(pop, mort *rtreego.Rtree, index [][2]int) *gridCell {
	// first, see if the cell is already in the cache.
	cacheKey := ""
	for _, v := range index {
		cacheKey += fmt.Sprintf("(%v,%v)", v[0], v[1])
	}
	if tempCell, ok := cellCache[cacheKey]; ok {
		return tempCell.copyCell()
	}
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

	cell := new(gridCell)
	cell.PopData = make(map[string]float64)
	cell.index = index
	// Polygon must go counter-clockwise
	cell.Geom = geom.Polygon{[][]geom.Point{{{l, b}, {r, b}, {r, u}, {l, u}, {l, b}}}}
	cell.bbox, err = gisconversions.GeomToRect(cell.Geom)
	if err != nil {
		panic(err)
	}
	for _, pInterface := range pop.SearchIntersect(cell.bbox) {
		p := pInterface.(*population)
		intersection, err := geomop.Construct(
			cell.Geom, p.Geom, geomop.INTERSECTION)
		if err != nil {
			panic(err)
		}
		area1 := geomop.Area(intersection)
		area2 := geomop.Area(p.Geom) // we want to conserve the total population
		if err != nil {
			panic(err)
		}
		if area2 == 0. {
			panic("divide by zero")
		}
		areaFrac := area1 / area2
		for popType, pop := range p.PopData {
			cell.PopData[popType] += pop * areaFrac
		}
	}
	for _, mInterface := range mort.SearchIntersect(cell.bbox) {
		m := mInterface.(*mortality)
		intersection, err := geomop.Construct(
			cell.Geom, m.Geom, geomop.INTERSECTION)
		if err != nil {
			panic(err)
		}
		area1 := geomop.Area(intersection)
		area2 := geomop.Area(cell.Geom) // we want to conserve the average rate here, not the total
		if area2 == 0. {
			panic("divide by zero")
		}
		areaFrac := area1 / area2
		cell.MortalityRate += m.AllCause * areaFrac
	}
	cell.Dx = r - l
	cell.Dy = u - b
	// fmt.Println(index, cell.TotalPop, cell.Dx, cell.Dy)
	// store a copy of the cell in the cache for later use.
	cellCache[cacheKey] = cell.copyCell()
	return cell
}

func writeCell(shp *gis.Shapefile, cell *gridCell) {
	fieldIDs := []int{0, 1}
	outData := make([]interface{}, len(config.CensusPopColumns)+3)
	outData[0] = cell.Row
	outData[1] = cell.Col
	for i, col := range config.CensusPopColumns {
		fieldIDs = append(fieldIDs, i+2)
		outData[i+2] = cell.PopData[col]
	}
	fieldIDs = append(fieldIDs, fieldIDs[len(fieldIDs)-1]+1)
	outData[len(config.CensusPopColumns)+2] = cell.MortalityRate
	err := shp.WriteFeature(cell.Row, cell.Geom, fieldIDs, outData...)
	if err != nil {
		panic(err)
	}
}

type population struct {
	bounds  *rtreego.Rect
	Geom    geom.T
	PopData map[string]float64
}

func (p *population) Bounds() *rtreego.Rect {
	return p.bounds
}

type mortality struct {
	bounds   *rtreego.Rect
	Geom     geom.T
	AllCause float64 // Deaths per 100,000 people per year
}

func (m *mortality) Bounds() *rtreego.Rect {
	return m.bounds
}

func loadPopulation(sr gdal.SpatialReference) (
	pop *rtreego.Rtree) {
	var err error
	popshp, err := gis.OpenShapefile(config.CensusFile, true)
	if err != nil {
		panic(err)
	}
	ct, err := projgeom.NewCoordinateTransform(popshp.Sr, sr)
	if err != nil {
		panic(err)
	}
	indexes := make([]int, len(config.CensusPopColumns))
	for i, col := range config.CensusPopColumns {
		indexes[i], err = popshp.GetColumnIndex(col)
		if err != nil {
			panic(err)
		}
	}

	pop = rtreego.NewTree(2, 25, 50)
	for {
		g, fieldVals, err := popshp.ReadNextFeature(indexes...)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		p := new(population)
		p.PopData = make(map[string]float64)
		for i, col := range config.CensusPopColumns {
			switch fieldVals[i].(type) {
			case float64:
				p.PopData[col] = fieldVals[i].(float64)
			case float32:
				p.PopData[col] = float64(fieldVals[i].(float32))
			case int:
				p.PopData[col] = float64(fieldVals[i].(int))
			case error:
				if err != nil {
					panic(err)
				}
			default:
				panic("Unknown type")
			}
			if math.IsNaN(p.PopData[col]) {
				panic("NaN!")
			}
		}
		if p.PopData[config.PopGridColumn] == 0. {
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
	mortshp, err := gis.OpenShapefile(config.MortalityRateFile, true)
	if err != nil {
		panic(err)
	}
	ct, err := projgeom.NewCoordinateTransform(mortshp.Sr, sr)
	if err != nil {
		panic(err)
	}
	iAllCause, err := mortshp.GetColumnIndex(config.MortalityRateColumn)
	if err != nil {
		panic(err)
	}

	mort = rtreego.NewTree(2, 25, 50)
	for {
		g, fieldVals, err := mortshp.ReadNextFeature(iAllCause)
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
		case error:
			if err != nil {
				panic(err)
			}
		default:
			panic("Unknown type")
		}
		if math.IsNaN(m.AllCause) {
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
			fmt.Println("bbox", cell.bbox)
			fmt.Println("geom", cell.Geom)
			fmt.Println("index", cell.index)
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
			cell.AOrgPartitioning += data["aOrgPartitioning"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.BOrgPartitioning += data["bOrgPartitioning"].data.Get(
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
