package inmap

import (
	"fmt"
	"log"
	"math"
	"sort"

	"bitbucket.org/ctessum/cdf"
	"bitbucket.org/ctessum/sparse"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
)

// webMapProj is the spatial reference for web mapping.
const webMapProj = "+proj=merc +a=6378137 +b=6378137 +lat_ts=0.0 +lon_0=0.0 +x_0=0.0 +y_0=0 +k=1.0 +units=m +nadgrids=@null +no_defs"

// VarGridConfig is a holder for the configuration information for creating a
// variable-resolution grid.
type VarGridConfig struct {
	VariableGridXo float64 // lower left of output grid, x
	VariableGridYo float64 // lower left of output grid, y
	VariableGridDx float64 // m
	VariableGridDy float64 // m
	Xnests         []int   // Nesting multiples in the X direction
	Ynests         []int   // Nesting multiples in the Y direction
	HiResLayers    int     // number of layers to do in high resolution (layers above this will be lowest resolution.

	CtmGridXo float64 // lower left of Chemical Transport Model (CTM) grid, x
	CtmGridYo float64 // lower left of grid, y
	CtmGridDx float64 // m
	CtmGridDy float64 // m
	CtmGridNx int
	CtmGridNy int

	PopDensityCutoff    float64  // limit for people per unit area in the grid cell
	PopCutoff           float64  // limit for total number of people in the grid cell
	BboxOffset          float64  // A number significantly less than the smallest grid size but not small enough to be confused with zero.
	CensusFile          string   // Path to census shapefile
	CensusPopColumns    []string // Shapefile fields containing populations for multiple demographics
	PopGridColumn       string   // Name of field in shapefile to be used for determining variable grid resolution
	MortalityRateFile   string   // Path to the mortality rate shapefile
	MortalityRateColumn string   // Name of field in mortality rate shapefile containing the mortality rate.

	GridProj string // projection info for CTM grid; Proj4 format
}

// CTMData holds processed data from a chemical transport model
type CTMData struct {
	description string
	units       string
	data        *sparse.DenseArray
}

// LoadCTMData loads CTM data from a netcdf file.
func LoadCTMData(rw cdf.ReaderWriterAt) (map[string]CTMData, error) {
	f, err := cdf.Open(rw)
	if err != nil {
		return nil, fmt.Errorf("inmap.LoadCTMData: %v", err)
	}
	o := make(map[string]CTMData)
	for _, v := range f.Header.Variables() {
		d := CTMData{}
		d.description = f.Header.GetAttribute(v, "description").(string)
		d.units = f.Header.GetAttribute(v, "units").(string)
		dims := f.Header.Lengths(v)
		r := f.Reader(v, nil, nil)
		d.data = sparse.ZerosDense(dims...)
		tmp := make([]float32, len(d.data.Elements))
		_, err = r.Read(tmp)
		if err != nil {
			return nil, fmt.Errorf("inmap.LoadCTMData: %v", err)
		}
		for i, v := range tmp {
			d.data.Elements[i] = float64(v)
		}
		o[v] = d
	}
	return o, nil
}

func (c *Cell) clonePartial() *Cell {
	o := new(Cell)
	o.Polygonal = c.Polygonal
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

func (d *InMAPdata) loadPopMort() error {
	log.Println("Loading population")
	if err := d.loadPopulation(d.sr); err != nil {
		return fmt.Errorf("inmap: while loading population: %v", err)
	}
	log.Println("Loading mortality")
	if err := d.loadMortality(d.sr); err != nil {
		return fmt.Errorf("inmap: while loading mortality rate: %v", err)
	}
	return nil
}

// NewInMAPData initializes the model where `data` is preprocessed
// output data from a chemical transport model,
// and `numIterations` is the number of iterations to calculate.
// If `numIterations` < 1, convergence is calculated automatically.
func NewInMAPData(config VarGridConfig, data map[string]CTMData, numIterations int) (*InMAPdata, error) {
	d := new(InMAPdata)
	d.VarGridConfig = config
	d.NumIterations = numIterations

	var err error
	d.sr, err = proj.Parse(config.GridProj)
	if err != nil {
		return nil, fmt.Errorf("inmap: while parsing GridProj: %v", err)
	}

	if err := d.loadPopMort(); err != nil {
		return nil, err
	}

	kmax := data["UAvg"].data.Shape[0]
	d.LayerStart = make([]int, kmax)
	d.LayerEnd = make([]int, kmax)
	d.Nlayers = kmax

	ctmtree := config.makeCTMgrid(kmax)

	d.cellCache = make(map[string]*Cell)

	var layerDivisionIndex int
	for k := 0; k < kmax; k++ {
		log.Println("Creating variable grid for layer ", k)

		var layerCells []*Cell
		if k < config.HiResLayers {
			layerCells = d.createCells(config.Xnests, config.Ynests, nil, d.population, d.mortalityrate)
		} else { // no nested grids above the boundary layer
			layerCells = d.createCells(config.Xnests[0:1], config.Ynests[0:1],
				nil, d.population, d.mortalityrate)
		}

		d.LayerStart[k] = layerDivisionIndex
		layerDivisionIndex += len(layerCells)
		d.LayerEnd[k] = layerDivisionIndex

		for _, c := range layerCells {
			c.loadData(ctmtree, data, k)
			c.prepare()
			d.Cells = append(d.Cells, c)
		}
	}
	sortCells(d.Cells)
	d.setupTree()
	for i, c := range d.Cells {
		c.Row = i
		d.setNeighbors(c)
	}

	d.setTstepCFL() // Set time step

	return d, nil
}

// sort the cells so that the order doesn't change between program runs.
func sortCells(cells []*Cell) {
	sc := &cellsSorter{
		cells: cells,
	}
	sort.Sort(sc)
}

type cellsSorter struct {
	cells []*Cell
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
	ci := c.cells[i]
	cj := c.cells[j]
	if ci.Layer != cj.Layer {
		return ci.Layer < cj.Layer
	}

	icent := ci.Polygonal.Centroid()
	jcent := cj.Polygonal.Centroid()

	if icent.X != jcent.X {
		return icent.X < jcent.X
	}
	if icent.Y != jcent.Y {
		return icent.Y < jcent.Y
	}
	// We apparently have concentric or identical cells if we get to here.
	panic(fmt.Errorf("problem sorting: i: %v, i layer: %d, j: %v, j layer: %d",
		ci.Polygonal, ci.Layer, cj.Polygonal, cj.Layer))
}

func (d *InMAPdata) setupTree() {
	d.index = rtree.NewTree(25, 50)
	for _, c := range d.Cells {
		d.index.Insert(c)
	}
}

func (d *InMAPdata) setNeighbors(c *Cell) {
	d.neighbors(c)

	if len(c.West) == 0 {
		d.addWestBoundary(c)
	}
	if len(c.East) == 0 {
		d.addEastBoundary(c)
	}
	if len(c.North) == 0 {
		d.addNorthBoundary(c)
	}
	if len(c.South) == 0 {
		d.addSouthBoundary(c)
	}
	if len(c.Above) == 0 {
		d.addTopBoundary(c)
	}
	if c.Layer == 0 {
		c.Below = []*Cell{c}
		c.GroundLevel = []*Cell{c}
	}

	c.neighborInfo()
}

func (d *InMAPdata) neighbors(c *Cell) {
	b := c.Bounds()
	bboxOffset := d.VarGridConfig.BboxOffset

	// Horizontal
	westbox := newRect(b.Min.X-2*bboxOffset, b.Min.Y+bboxOffset,
		b.Min.X-bboxOffset, b.Max.Y-bboxOffset)
	c.West = getCells(d.index, westbox, c.Layer)
	eastbox := newRect(b.Max.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X+2*bboxOffset, b.Max.Y-bboxOffset)
	c.East = getCells(d.index, eastbox, c.Layer)
	southbox := newRect(b.Min.X+bboxOffset, b.Min.Y-2*bboxOffset,
		b.Max.X-bboxOffset, b.Min.Y-bboxOffset)
	c.South = getCells(d.index, southbox, c.Layer)
	northbox := newRect(b.Min.X+bboxOffset, b.Max.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y+2*bboxOffset)
	c.North = getCells(d.index, northbox, c.Layer)

	// Above
	abovebox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
	c.Above = getCells(d.index, abovebox, c.Layer+1)

	// Below
	belowbox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
	c.Below = getCells(d.index, belowbox, c.Layer-1)

	// Ground level.
	groundlevelbox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
	c.GroundLevel = getCells(d.index, groundlevelbox, 0)
}

// getCells returns all the grid cells in cellTree that are within box
// and at vertical layer layer.
func getCells(cellTree *rtree.Rtree, box *geom.Bounds, layer int) []*Cell {
	x := cellTree.SearchIntersect(box)
	cells := make([]*Cell, 0, len(x))
	for _, xx := range x {
		c := xx.(*Cell)
		if c.Layer == layer {
			cells = append(cells, c)
		}
	}
	return cells
}

func newRect(xmin, ymin, xmax, ymax float64) *geom.Bounds {
	return &geom.Bounds{
		Min: geom.Point{X: xmin, Y: ymin},
		Max: geom.Point{X: xmax, Y: ymax},
	}
}

// createCells cycles through all of the indicies in the given nest.
// Create the grid cell for each index, If the grid
// cell is below both population thresholds
// (for both total population and population density),
// keep it. Otherwise, recursively generate inner nested cells and
// keep those instead.
func (d *InMAPdata) createCells(localxNests, localyNests []int, index [][2]int,
	pop, mort *rtree.Rtree) []*Cell {

	arrayCap := 0
	for ii := 0; ii < len(localyNests); ii++ {
		arrayCap += localyNests[ii] * localxNests[ii]
	}
	cells := make([]*Cell, 0, arrayCap)
	// Iterate through indices and send them to the concurrent cell generator
	for j := 0; j < localyNests[0]; j++ {
		for i := 0; i < localxNests[0]; i++ {
			newIndex := make([][2]int, 0, len(d.VarGridConfig.Xnests))
			for _, i := range index {
				newIndex = append(newIndex, i)
			}
			newIndex = append(newIndex, [2]int{i, j})
			// Create the cell in this nest
			cell := d.createCell(pop, mort, newIndex)
			if len(localxNests) > 1 {
				// Check if this grid cell is above the population threshold
				// or the population density threshold.
				if cell.aboveDensityThreshold ||
					cell.PopData[d.VarGridConfig.PopGridColumn] > d.VarGridConfig.PopCutoff {

					// If this cell is above a threshold, create inner
					// nested cells instead of using this one.
					cells = append(cells, d.createCells(localxNests[1:],
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

// createCell creates a new grid cell. If any of the census shapes
// that intersect the cell are above the population density threshold,
// then the grid cell is also set to being above the density threshold.
func (d *InMAPdata) createCell(pop, mort *rtree.Rtree, index [][2]int) *Cell {
	// first, see if the cell is already in the cache.
	cacheKey := ""
	for _, v := range index {
		cacheKey += fmt.Sprintf("(%v,%v)", v[0], v[1])
	}
	if tempCell, ok := d.cellCache[cacheKey]; ok {
		return tempCell.clonePartial()
	}

	xResFac, yResFac := 1., 1.
	l := d.VarGridConfig.VariableGridXo
	b := d.VarGridConfig.VariableGridYo
	for i, ii := range index {
		if i > 0 {
			xResFac *= float64(d.VarGridConfig.Xnests[i])
			yResFac *= float64(d.VarGridConfig.Ynests[i])
		}
		l += float64(ii[0]) * d.VarGridConfig.VariableGridDx / xResFac
		b += float64(ii[1]) * d.VarGridConfig.VariableGridDy / yResFac
	}
	r := l + d.VarGridConfig.VariableGridDx/xResFac
	u := b + d.VarGridConfig.VariableGridDy/yResFac

	cell := new(Cell)
	cell.PopData = make(map[string]float64)
	cell.index = index
	// Polygon must go counter-clockwise
	cell.Polygonal = geom.Polygon([][]geom.Point{{{l, b}, {r, b}, {r, u}, {l, u}, {l, b}}})
	for _, pInterface := range pop.SearchIntersect(cell.Bounds()) {
		p := pInterface.(*population)
		intersection := cell.Intersection(p)
		area1 := intersection.Area()
		area2 := p.Area() // we want to conserve the total population
		if area2 == 0. {
			panic("divide by zero")
		}
		areaFrac := area1 / area2
		for popType, pop := range p.PopData {
			cell.PopData[popType] += pop * areaFrac
		}

		// Check if this census shape is above the density threshold
		pDensity := p.PopData[d.VarGridConfig.PopGridColumn] / area2
		if pDensity > d.VarGridConfig.PopDensityCutoff {
			cell.aboveDensityThreshold = true
		}
	}
	for _, mInterface := range mort.SearchIntersect(cell.Bounds()) {
		m := mInterface.(*mortality)
		intersection := cell.Intersection(m)
		area1 := intersection.Area()
		area2 := cell.Area() // we want to conserve the average rate here, not the total
		if area2 == 0. {
			panic("divide by zero")
		}
		areaFrac := area1 / area2
		cell.MortalityRate += m.AllCause * areaFrac
	}
	cell.Dx = r - l
	cell.Dy = u - b

	// store a copy of the cell in the cache for later use.
	d.cellCache[cacheKey] = cell.clonePartial()

	return cell
}

type population struct {
	geom.Polygonal
	PopData map[string]float64
}

type mortality struct {
	geom.Polygonal
	AllCause float64 // Deaths per 100,000 people per year
}

func (d *InMAPdata) loadPopulation(sr *proj.SR) error {
	var err error
	popshp, err := shp.NewDecoder(d.VarGridConfig.CensusFile)
	if err != nil {
		return err
	}
	popsr, err := popshp.SR()
	if err != nil {
		return err
	}
	trans, err := popsr.NewTransform(sr)
	if err != nil {
		return err
	}

	d.population = rtree.NewTree(25, 50)
	for {
		g, fields, more := popshp.DecodeRowFields(d.VarGridConfig.CensusPopColumns...)
		if !more {
			break
		}
		p := new(population)
		p.PopData = make(map[string]float64)
		for _, pop := range d.VarGridConfig.CensusPopColumns {
			p.PopData[pop], err = s2f(fields[pop])
			if err != nil {
				return err
			}
			if math.IsNaN(p.PopData[pop]) {
				panic("NaN!")
			}
		}
		if p.PopData[d.VarGridConfig.PopGridColumn] == 0. {
			continue
		}
		gg, err := g.Transform(trans)
		if err != nil {
			return err
		}
		p.Polygonal = gg.(geom.Polygonal)
		d.population.Insert(p)
	}
	if err := popshp.Error(); err != nil {
		return err
	}

	popshp.Close()
	return nil
}

func (d *InMAPdata) loadMortality(sr *proj.SR) error {
	mortshp, err := shp.NewDecoder(d.VarGridConfig.MortalityRateFile)
	if err != nil {
		return err
	}

	mortshpSR, err := mortshp.SR()
	if err != nil {
		return err
	}
	trans, err := mortshpSR.NewTransform(sr)
	if err != nil {
		return err
	}

	d.mortalityrate = rtree.NewTree(25, 50)
	for {
		g, fields, more := mortshp.DecodeRowFields(d.VarGridConfig.MortalityRateColumn)
		if !more {
			break
		}
		m := new(mortality)
		m.AllCause, err = s2f(fields[d.VarGridConfig.MortalityRateColumn])
		if err != nil {
			return err
		}
		if math.IsNaN(m.AllCause) {
			return fmt.Errorf("NaN mortality rate")
		}
		gg, err := g.Transform(trans)
		if err != nil {
			return err
		}
		m.Polygonal = gg.(geom.Polygonal)
		d.mortalityrate.Insert(m)
	}
	if err := mortshp.Error(); err != nil {
		return err
	}
	mortshp.Close()
	return nil
}

func (c *Cell) loadData(ctmtree *rtree.Rtree, data map[string]CTMData, k int) {
	c.Layer = k
	ctmcellsAllLayers := ctmtree.SearchIntersect(c.Bounds())
	var ctmcells []*gridCellLight
	for _, cc := range ctmcellsAllLayers {
		// we only want grid cells that match our layer.
		ccc := cc.(*gridCellLight)
		if ccc.layer == k {
			ctmcells = append(ctmcells, ccc)
		}
	}
	ncells := float64(len(ctmcells))
	if len(ctmcells) == 0. {
		fmt.Println("geom", c.Polygonal)
		fmt.Println("index", c.index)
		panic("No matching cells!")
	}
	for _, ctmcell := range ctmcells {
		ctmrow := ctmcell.Row
		ctmcol := ctmcell.Col

		// TODO: Average velocity is on a staggered grid, so we should
		// do some sort of interpolation here.
		c.UAvg += data["UAvg"].data.Get(k, ctmrow, ctmcol) / ncells
		c.VAvg += data["VAvg"].data.Get(k, ctmrow, ctmcol) / ncells
		c.WAvg += data["WAvg"].data.Get(k, ctmrow, ctmcol) / ncells

		c.UDeviation += data["UDeviation"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.VDeviation += data["VDeviation"].data.Get(
			k, ctmrow, ctmcol) / ncells

		c.AOrgPartitioning += data["aOrgPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.BOrgPartitioning += data["bOrgPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NOPartitioning += data["NOPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SPartitioning += data["SPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NHPartitioning += data["NHPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SO2oxidation += data["SO2oxidation"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.ParticleDryDep += data["ParticleDryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SO2DryDep += data["SO2DryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NOxDryDep += data["NOxDryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NH3DryDep += data["NH3DryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.VOCDryDep += data["VOCDryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.Kxxyy += data["Kxxyy"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.LayerHeight += data["LayerHeights"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.Dz += data["Dz"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.ParticleWetDep += data["ParticleWetDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SO2WetDep += data["SO2WetDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.OtherGasWetDep += data["OtherGasWetDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.Kzz += data["Kzz"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.M2u += data["M2u"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.M2d += data["M2d"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeed += data["WindSpeed"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeedInverse += data["WindSpeedInverse"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeedMinusThird += data["WindSpeedMinusThird"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeedMinusOnePointFour +=
			data["WindSpeedMinusOnePointFour"].data.Get(
				k, ctmrow, ctmcol) / ncells
		c.Temperature += data["Temperature"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.S1 += data["S1"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SClass += data["Sclass"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.TotalPM25 += data["TotalPM25"].data.Get(
			k, ctmrow, ctmcol) / ncells

	}
}

// make a vector representation of the chemical transport model grid
func (config *VarGridConfig) makeCTMgrid(nlayers int) *rtree.Rtree {
	tree := rtree.NewTree(25, 50)
	for k := 0; k < nlayers; k++ {
		for ix := 0; ix < config.CtmGridNx; ix++ {
			for iy := 0; iy < config.CtmGridNy; iy++ {
				cell := new(gridCellLight)
				x0 := config.CtmGridXo + config.CtmGridDx*float64(ix)
				x1 := config.CtmGridXo + config.CtmGridDx*float64(ix+1)
				y0 := config.CtmGridYo + config.CtmGridDy*float64(iy)
				y1 := config.CtmGridYo + config.CtmGridDy*float64(iy+1)
				cell.Polygonal = geom.Polygon{[]geom.Point{
					geom.Point{X: x0, Y: y0},
					geom.Point{X: x1, Y: y0},
					geom.Point{X: x1, Y: y1},
					geom.Point{X: x0, Y: y1},
					geom.Point{X: x0, Y: y0},
				}}
				cell.Row = iy
				cell.Col = ix
				cell.layer = k
				tree.Insert(cell)
			}
		}
	}
	return tree
}

type gridCellLight struct {
	geom.Polygonal
	Row, Col, layer int
}

func handle(err error) {
	if err != nil {
		panic(err)
	}
}
