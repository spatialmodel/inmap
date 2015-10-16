package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/geojson"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/op"
	"github.com/ctessum/geom/proj"
	goshp "github.com/jonas-p/go-shp"
)

const WebMapProj = "+proj=merc +a=6378137 +b=6378137 +lat_ts=0.0 +lon_0=0.0 +x_0=0.0 +y_0=0 +k=1.0 +units=m +nadgrids=@null +no_defs"

func init() {
	gob.Register(geom.Polygon{})
}

type gridCell struct {
	geom.T
	WebMapGeom                                       geom.T
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
	Kzz, M2u, M2d                                    float64
	WindSpeed, WindSpeedMinusThird                   float64
	WindSpeedInverse, WindSpeedMinusOnePointFour     float64
	Temperature                                      float64
	S1, Sclass                                       float64
	aboveDensityThreshold                            bool
}

func (c *gridCell) copyCell() *gridCell {
	o := new(gridCell)
	o.T = c.T
	o.WebMapGeom = c.WebMapGeom
	o.PopData = make(map[string]float64)
	for key, val := range c.PopData {
		o.PopData[key] = val
	}
	o.MortalityRate = c.MortalityRate
	o.index = c.index
	o.Dx, o.Dy = c.Dx, c.Dy
	o.MortalityRate = c.MortalityRate
	o.aboveDensityThreshold = c.aboveDensityThreshold
	return o
}

func variableGrid(data map[string]dataHolder) {
	sr, err := proj.FromProj4(config.GridProj)
	handle(err)
	log.Println("Loading population")
	pop := loadPopulation(sr)
	log.Println("Loading mortality")
	mort := loadMortality(sr)
	log.Println("Loaded mortality")
	filePrefix := filepath.Join(config.OutputDir, config.OutputFilePrefix)
	kmax := data["UPlusSpeed"].data.Shape[0]
	var cellsBelow []*gridCell
	var cellTreeBelow *rtree.Rtree
	var cellTreeGroundLevel *rtree.Rtree
	id := 0
	for k := 0; k < kmax; k++ {
		log.Println("Creating variable grid for layer ", k)
		os.Remove(fmt.Sprintf("%v_%v.shp", filePrefix, k))
		os.Remove(fmt.Sprintf("%v_%v.shx", filePrefix, k))
		os.Remove(fmt.Sprintf("%v_%v.dbf", filePrefix, k))
		os.Remove(fmt.Sprintf("%v_%v.prj", filePrefix, k))
		var fields []goshp.Field
		const intSize = 6        // length of integers in shapefile
		const floatLen = 15      // length of floats
		const floatPrecision = 7 // number of decimal places in floats
		fields = append(fields, goshp.NumberField("row", intSize))
		fields = append(fields, goshp.NumberField("col", intSize))
		for _, pop := range config.CensusPopColumns {
			fields = append(fields, goshp.FloatField(pop, floatLen, floatPrecision))
		}
		fields = append(fields, goshp.FloatField(config.MortalityRateColumn,
			floatLen, floatPrecision))
		fname := filepath.Join(config.OutputDir,
			fmt.Sprintf("%v_%v.shp", config.OutputFilePrefix, k))
		shpf, err := shp.NewEncoderFromFields(fname, goshp.POLYGON, fields...)
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
		cellTree := rtree.NewTree(25, 50)

		log.Printf("%v grid cells.", len(cells))
		for _, cell := range cells {
			cellTree.Insert(cell)
		}
		sortCells(cells)
		for _, cell := range cells {
			cell.Row = id
			writeCell(shpf, cell)
			id++
		}
		shpf.Close()
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

func getNeighborsHorizontal(cells []*gridCell, cellTree *rtree.Rtree) {
	for _, cell := range cells {
		b := cell.Bounds(nil)
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

func getNeighborsAbove(cells []*gridCell, aboveCellTree *rtree.Rtree) {
	for _, cell := range cells {
		b := cell.Bounds(nil)
		bboxOffset := config.BboxOffset
		abovebox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.IAbove = getIndexes(aboveCellTree, abovebox)
	}
}
func getNeighborsBelow(cells []*gridCell, belowCellTree *rtree.Rtree) {
	for _, cell := range cells {
		b := cell.Bounds(nil)
		bboxOffset := config.BboxOffset
		belowbox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.IBelow = getIndexes(belowCellTree, belowbox)
	}
}
func getNeighborsGroundLevel(cells []*gridCell, groundlevelCellTree *rtree.Rtree) {
	for _, cell := range cells {
		b := cell.Bounds(nil)
		bboxOffset := config.BboxOffset
		groundlevelbox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
			b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
		cell.IGroundLevel = getIndexes(groundlevelCellTree, groundlevelbox)
	}
}

func getIndexes(cellTree *rtree.Rtree, box *geom.Bounds) []int {
	x := cellTree.SearchIntersect(box)
	indexes := make([]int, len(x))
	for i, xx := range x {
		indexes[i] = xx.(*gridCell).Row
	}
	return indexes
}

func newRect(xmin, ymin, xmax, ymax float64) *geom.Bounds {
	p := geom.NewBounds()
	p.ExtendPoint(geom.Point{xmin, ymin})
	p.ExtendPoint(geom.Point{xmax, ymax})
	return p
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
		x.Geometry, err = geojson.ToGeoJSON(cell.T)
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
	src, err := proj.FromProj4(config.GridProj)
	if err != nil {
		panic(err)
	}
	dst, err := proj.FromProj4(WebMapProj)
	if err != nil {
		panic(err)
	}
	ct, err := proj.NewCoordinateTransform(src, dst)
	if err != nil {
		panic(err)
	}
	for _, cell := range cells {
		cell.WebMapGeom, err = ct.Reproject(cell.T)
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
// Create the grid cell for each index, If the grid
// cell is below both population thresholds
// (for both total population and population density),
// keep it. Otherwise, recursively generate inner nested cells and
// keep those instead.
func createCells(localxNests, localyNests []int, index [][2]int,
	pop, mort *rtree.Rtree) []*gridCell {

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
			cell := CreateCell(pop, mort, newIndex)
			if len(localxNests) > 1 {
				// Check if this grid cell is above the population threshold
				// or the population density threshold.
				if cell.aboveDensityThreshold ||
					cell.PopData[config.PopGridColumn] > config.PopCutoff {

					// If this cell is above a threshold, create inner
					// nested cells instead of using this one.
					cells = append(cells, createCells(localxNests[1:],
						localyNests[1:], newIndex, pop, mort)...)
				} else {
					// If this cell is not above the threshold, keep it.
					cells = append(cells, cell)
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

// CreateCell creates a new grid cell. If any of the census shapes
// that intersect the cell are above the population density threshold,
// then the grid cell is also set to being above the density threshold.
func CreateCell(pop, mort *rtree.Rtree, index [][2]int) *gridCell {
	// first, see if the cell is already in the cache.
	cacheKey := ""
	for _, v := range index {
		cacheKey += fmt.Sprintf("(%v,%v)", v[0], v[1])
	}
	if tempCell, ok := cellCache[cacheKey]; ok {
		return tempCell.copyCell()
	}

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
	cell.T = geom.Polygon([][]geom.Point{{{l, b}, {r, b}, {r, u}, {l, u}, {l, b}}})
	for _, pInterface := range pop.SearchIntersect(cell.Bounds(nil)) {
		p := pInterface.(*population)
		intersection, err := op.Construct(
			cell.T, p.T, op.INTERSECTION)
		if err != nil {
			panic(err)
		}
		area1 := op.Area(intersection)
		area2 := op.Area(p.T) // we want to conserve the total population
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

		// Check if this census shape is above the density threshold
		pDensity := p.PopData[config.PopGridColumn] / area2
		if pDensity > config.PopDensityCutoff {
			cell.aboveDensityThreshold = true
		}
	}
	for _, mInterface := range mort.SearchIntersect(cell.Bounds(nil)) {
		m := mInterface.(*mortality)
		intersection, err := op.Construct(
			cell.T, m.T, op.INTERSECTION)
		if err != nil {
			panic(err)
		}
		area1 := op.Area(intersection)
		area2 := op.Area(cell.T) // we want to conserve the average rate here, not the total
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

func writeCell(shpf *shp.Encoder, cell *gridCell) {
	outData := make([]interface{}, len(config.CensusPopColumns)+3)
	outData[0] = cell.Row
	outData[1] = cell.Col
	for i, col := range config.CensusPopColumns {
		outData[i+2] = cell.PopData[col]
	}
	outData[len(config.CensusPopColumns)+2] = cell.MortalityRate
	err := shpf.EncodeFields(cell.T, outData...)
	handle(err)
}

type population struct {
	geom.T
	PopData map[string]float64
}

type mortality struct {
	geom.T
	AllCause float64 // Deaths per 100,000 people per year
}

func loadPopulation(sr proj.SR) (
	pop *rtree.Rtree) {
	var err error
	popshp, err := shp.NewDecoder(config.CensusFile)
	handle(err)
	extension := filepath.Ext(config.CensusFile)
	prjf, err := os.Open(config.CensusFile[0:len(config.CensusFile)-
		len(extension)] + ".prj")
	popsr, err := proj.ReadPrj(prjf)
	handle(err)
	ct, err := proj.NewCoordinateTransform(popsr, sr)
	handle(err)

	pop = rtree.NewTree(25, 50)
	for {
		g, fields, more := popshp.DecodeRowFields(config.CensusPopColumns...)
		if !more {
			break
		}
		p := new(population)
		p.PopData = make(map[string]float64)
		for _, pop := range config.CensusPopColumns {
			p.PopData[pop] = s2f(fields[pop])
			if math.IsNaN(p.PopData[pop]) {
				panic("NaN!")
			}
		}
		if p.PopData[config.PopGridColumn] == 0. {
			continue
		}
		p.T, err = ct.Reproject(g)
		handle(err)
		pop.Insert(p)
	}
	handle(popshp.Error())
	popshp.Close()
	return
}

func loadMortality(sr proj.SR) (
	mort *rtree.Rtree) {
	mortshp, err := shp.NewDecoder(config.MortalityRateFile)
	handle(err)
	extension := filepath.Ext(config.MortalityRateFile)
	prjf, err := os.Open(config.MortalityRateFile[0:len(config.MortalityRateFile)-
		len(extension)] + ".prj")
	mortshpSR, err := proj.ReadPrj(prjf)
	handle(err)
	ct, err := proj.NewCoordinateTransform(mortshpSR, sr)
	handle(err)

	mort = rtree.NewTree(25, 50)
	for {
		g, fields, more := mortshp.DecodeRowFields(config.MortalityRateColumn)
		if !more {
			break
		}
		m := new(mortality)
		m.AllCause = s2f(fields[config.MortalityRateColumn])
		if math.IsNaN(m.AllCause) {
			panic("NaN!")
		}
		m.T, err = ct.Reproject(g)
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
		ctmcells := ctmtree.SearchIntersect(cell.Bounds(nil))
		ncells := float64(len(ctmcells))
		if len(ctmcells) == 0. {
			fmt.Println("geom", cell.T)
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
			cell.WindSpeedInverse += data["WindSpeedInverse"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.WindSpeedMinusThird += data["WindSpeedMinusThird"].data.Get(
				k, ctmrow, ctmcol) / ncells
			cell.WindSpeedMinusOnePointFour +=
				data["WindSpeedMinusOnePointFour"].data.Get(
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
func makeCTMgrid() *rtree.Rtree {
	tree := rtree.NewTree(25, 50)
	for ix := 0; ix < config.CtmGrid_nx; ix++ {
		for iy := 0; iy < config.CtmGrid_ny; iy++ {
			cell := new(gridCellLight)
			x0 := config.CtmGrid_x_o + config.CtmGrid_dx*float64(ix)
			x1 := config.CtmGrid_x_o + config.CtmGrid_dx*float64(ix+1)
			y0 := config.CtmGrid_y_o + config.CtmGrid_dy*float64(iy)
			y1 := config.CtmGrid_y_o + config.CtmGrid_dy*float64(iy+1)
			cell.T = geom.Polygon{[]geom.Point{
				geom.Point{x0, y0},
				geom.Point{x1, y0},
				geom.Point{x1, y1},
				geom.Point{x0, y1},
				geom.Point{x0, y0},
			}}
			cell.Row = iy
			cell.Col = ix
			tree.Insert(cell)
		}
	}
	return tree
}

type gridCellLight struct {
	geom.T
	Row, Col int
}

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

func s2f(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	handle(err)
	return f
}
