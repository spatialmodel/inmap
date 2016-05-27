package inmap

import (
	"fmt"
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

	ctmGridXo float64 // lower left of Chemical Transport Model (CTM) grid, x
	ctmGridYo float64 // lower left of grid, y
	ctmGridDx float64 // m
	ctmGridDy float64 // m
	ctmGridNx int
	ctmGridNy int

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
	gridTree *rtree.Rtree
	data     map[string]ctmData
}

type ctmData struct {
	description string
	units       string
	data        *sparse.DenseArray
}

// LoadCTMData loads CTM data from a netcdf file.
func (config *VarGridConfig) LoadCTMData(rw cdf.ReaderWriterAt) (*CTMData, error) {
	f, err := cdf.Open(rw)
	if err != nil {
		return nil, fmt.Errorf("inmap.LoadCTMData: %v", err)
	}
	o := new(CTMData)
	nz := f.Header.Lengths("UAvg")[0]

	// Get CTM grid attributes
	config.ctmGridDx = f.Header.GetAttribute("", "dx").([]float64)[0]
	config.ctmGridDy = f.Header.GetAttribute("", "dy").([]float64)[0]
	config.ctmGridNx = int(f.Header.GetAttribute("", "nx").([]int32)[0])
	config.ctmGridNy = int(f.Header.GetAttribute("", "ny").([]int32)[0])
	config.ctmGridXo = f.Header.GetAttribute("", "x0").([]float64)[0]
	config.ctmGridYo = f.Header.GetAttribute("", "y0").([]float64)[0]

	o.gridTree = config.makeCTMgrid(nz)

	od := make(map[string]ctmData)
	for _, v := range f.Header.Variables() {
		d := ctmData{}
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
		od[v] = d
	}
	o.data = od
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

func (config *VarGridConfig) loadPopMort(gridSR *proj.SR) (*rtree.Rtree, *rtree.Rtree, error) {
	pop, err := config.loadPopulation(gridSR)
	if err != nil {
		return nil, nil, fmt.Errorf("inmap: while loading population: %v", err)
	}
	mort, err := config.loadMortality(gridSR)
	if err != nil {
		return nil, nil, fmt.Errorf("inmap: while loading mortality rate: %v", err)
	}
	return pop, mort, nil
}

// NewInMAPData initializes the model where `data` is preprocessed
// output data from a chemical transport model,
// and `numIterations` is the number of iterations to calculate.
// If `numIterations` < 1, convergence is calculated automatically.
func (config *VarGridConfig) NewInMAPData(data *CTMData, numIterations int) (*InMAPdata, error) {
	d := new(InMAPdata)
	d.NumIterations = numIterations

	var err error
	gridSR, err := proj.Parse(config.GridProj)
	if err != nil {
		return nil, fmt.Errorf("inmap: while parsing GridProj: %v", err)
	}

	pop, mort, err := config.loadPopMort(gridSR)
	if err != nil {
		return nil, err
	}

	err = config.RegularGrid(data, pop, mort)(d)
	if err != nil {
		return nil, err
	}
	err = config.StaticVariableGrid(data, pop, mort)(d)
	if err != nil {
		return nil, err
	}

	sortCells(d.Cells)
	d.setupTree()
	for i, c := range d.Cells {
		c.Row = i
		d.setNeighbors(c, config.BboxOffset)
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

func (d *InMAPdata) setNeighbors(c *Cell, bboxOffset float64) {
	d.neighbors(c, bboxOffset)

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

func (d *InMAPdata) neighbors(c *Cell, bboxOffset float64) {
	b := c.Bounds()

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

// RegularGrid returns a function that creates a new regular
// (i.e., not variable resolution) grid
// as specified by the information in c.
func (config *VarGridConfig) RegularGrid(data *CTMData, pop, mort *rtree.Rtree) DomainManipulator {
	return func(d *InMAPdata) error {

		nz := data.data["UAvg"].data.Shape[0]
		d.Nlayers = nz

		nx := config.Xnests[0]
		ny := config.Ynests[0]
		d.Cells = make([]*Cell, nx*ny*nz)
		// Iterate through indices and create the cells in the outermost nest.
		ii := 0
		for k := 0; k < nz; k++ {
			for j := 0; j < ny; j++ {
				for i := 0; i < nx; i++ {
					index := [][2]int{{i, j}}
					// Create the cell
					d.Cells[ii] = config.createCell(data, pop, mort, index, k)
					ii++
				}
			}
		}
		return nil
	}
}

// StaticVariableGrid returns a function that creates a static variable
// resolution grid (i.e., one that does not change during the simulation)
// by dividing cells in the previously created grid
// based on the population and population density cutoffs in config.
func (config *VarGridConfig) StaticVariableGrid(data *CTMData, pop, mort *rtree.Rtree) DomainManipulator {
	return func(d *InMAPdata) error {

		continueSplitting := true
		for continueSplitting {
			continueSplitting = false
			var newCells []*Cell
			var indicesToDelete []int
			for i, cell := range d.Cells {
				if len(cell.index) < len(config.Xnests) {
					// Check if this grid cell is above the population threshold
					// or the population density threshold.
					if cell.Layer < config.HiResLayers &&
						(cell.aboveDensityThreshold ||
							cell.PopData[config.PopGridColumn] > config.PopCutoff) {

						continueSplitting = true
						indicesToDelete = append(indicesToDelete, i)

						// If this cell is above a threshold, create inner
						// nested cells instead of using this one.
						for ii := 0; ii < config.Xnests[len(cell.index)]; ii++ {
							for jj := 0; jj < config.Ynests[len(cell.index)]; jj++ {
								newIndex := append(cell.index, [2]int{ii, jj})
								newCells = append(newCells, config.createCell(data, pop, mort, newIndex, cell.Layer))
							}
						}
					}
				}
			}
			// Delete the cells that were split.
			indexToSubtract := 0
			for _, ii := range indicesToDelete {
				i := ii - indexToSubtract
				copy(d.Cells[i:], d.Cells[i+1:])
				d.Cells[len(d.Cells)-1] = nil
				d.Cells = d.Cells[:len(d.Cells)-1]
				indexToSubtract++
			}
			// Add the new cells.
			d.Cells = append(d.Cells, newCells...)
		}
		return nil
	}
}

// createCell creates a new grid cell. If any of the census shapes
// that intersect the cell are above the population density threshold,
// then the grid cell is also set to being above the density threshold.
func (config *VarGridConfig) createCell(data *CTMData, pop, mort *rtree.Rtree, index [][2]int, layer int) *Cell {

	xResFac, yResFac := 1., 1.
	l := config.VariableGridXo
	b := config.VariableGridYo
	for i, ii := range index {
		if i > 0 {
			xResFac *= float64(config.Xnests[i])
			yResFac *= float64(config.Ynests[i])
		}
		l += float64(ii[0]) * config.VariableGridDx / xResFac
		b += float64(ii[1]) * config.VariableGridDy / yResFac
	}
	r := l + config.VariableGridDx/xResFac
	u := b + config.VariableGridDy/yResFac

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
		pDensity := p.PopData[config.PopGridColumn] / area2
		if pDensity > config.PopDensityCutoff {
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

	cell.loadData(data, layer)
	cell.prepare()

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

func (config *VarGridConfig) loadPopulation(sr *proj.SR) (*rtree.Rtree, error) {
	var err error
	popshp, err := shp.NewDecoder(config.CensusFile)
	if err != nil {
		return nil, err
	}
	popsr, err := popshp.SR()
	if err != nil {
		return nil, err
	}
	trans, err := popsr.NewTransform(sr)
	if err != nil {
		return nil, err
	}

	pop := rtree.NewTree(25, 50)
	for {
		g, fields, more := popshp.DecodeRowFields(config.CensusPopColumns...)
		if !more {
			break
		}
		p := new(population)
		p.PopData = make(map[string]float64)
		for _, pop := range config.CensusPopColumns {
			p.PopData[pop], err = s2f(fields[pop])
			if err != nil {
				return nil, err
			}
			if math.IsNaN(p.PopData[pop]) {
				panic("NaN!")
			}
		}
		if p.PopData[config.PopGridColumn] == 0. {
			continue
		}
		gg, err := g.Transform(trans)
		if err != nil {
			return nil, err
		}
		p.Polygonal = gg.(geom.Polygonal)
		pop.Insert(p)
	}
	if err := popshp.Error(); err != nil {
		return nil, err
	}

	popshp.Close()
	return pop, nil
}

func (config *VarGridConfig) loadMortality(sr *proj.SR) (*rtree.Rtree, error) {
	mortshp, err := shp.NewDecoder(config.MortalityRateFile)
	if err != nil {
		return nil, err
	}

	mortshpSR, err := mortshp.SR()
	if err != nil {
		return nil, err
	}
	trans, err := mortshpSR.NewTransform(sr)
	if err != nil {
		return nil, err
	}

	mortalityrate := rtree.NewTree(25, 50)
	for {
		g, fields, more := mortshp.DecodeRowFields(config.MortalityRateColumn)
		if !more {
			break
		}
		m := new(mortality)
		m.AllCause, err = s2f(fields[config.MortalityRateColumn])
		if err != nil {
			return nil, err
		}
		if math.IsNaN(m.AllCause) {
			return nil, fmt.Errorf("NaN mortality rate")
		}
		gg, err := g.Transform(trans)
		if err != nil {
			return nil, err
		}
		m.Polygonal = gg.(geom.Polygonal)
		mortalityrate.Insert(m)
	}
	if err := mortshp.Error(); err != nil {
		return nil, err
	}
	mortshp.Close()
	return mortalityrate, nil
}

func (c *Cell) loadData(data *CTMData, k int) {
	c.Layer = k
	ctmcellsAllLayers := data.gridTree.SearchIntersect(c.Bounds())
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
		c.UAvg += data.data["UAvg"].data.Get(k, ctmrow, ctmcol) / ncells
		c.VAvg += data.data["VAvg"].data.Get(k, ctmrow, ctmcol) / ncells
		c.WAvg += data.data["WAvg"].data.Get(k, ctmrow, ctmcol) / ncells

		c.UDeviation += data.data["UDeviation"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.VDeviation += data.data["VDeviation"].data.Get(
			k, ctmrow, ctmcol) / ncells

		c.AOrgPartitioning += data.data["aOrgPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.BOrgPartitioning += data.data["bOrgPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NOPartitioning += data.data["NOPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SPartitioning += data.data["SPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NHPartitioning += data.data["NHPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SO2oxidation += data.data["SO2oxidation"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.ParticleDryDep += data.data["ParticleDryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SO2DryDep += data.data["SO2DryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NOxDryDep += data.data["NOxDryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NH3DryDep += data.data["NH3DryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.VOCDryDep += data.data["VOCDryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.Kxxyy += data.data["Kxxyy"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.LayerHeight += data.data["LayerHeights"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.Dz += data.data["Dz"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.ParticleWetDep += data.data["ParticleWetDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SO2WetDep += data.data["SO2WetDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.OtherGasWetDep += data.data["OtherGasWetDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.Kzz += data.data["Kzz"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.M2u += data.data["M2u"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.M2d += data.data["M2d"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeed += data.data["WindSpeed"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeedInverse += data.data["WindSpeedInverse"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeedMinusThird += data.data["WindSpeedMinusThird"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeedMinusOnePointFour +=
			data.data["WindSpeedMinusOnePointFour"].data.Get(
				k, ctmrow, ctmcol) / ncells
		c.Temperature += data.data["Temperature"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.S1 += data.data["S1"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SClass += data.data["Sclass"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.TotalPM25 += data.data["TotalPM25"].data.Get(
			k, ctmrow, ctmcol) / ncells

	}
}

// make a vector representation of the chemical transport model grid
func (config *VarGridConfig) makeCTMgrid(nlayers int) *rtree.Rtree {
	tree := rtree.NewTree(25, 50)
	for k := 0; k < nlayers; k++ {
		for ix := 0; ix < config.ctmGridNx; ix++ {
			for iy := 0; iy < config.ctmGridNy; iy++ {
				cell := new(gridCellLight)
				x0 := config.ctmGridXo + config.ctmGridDx*float64(ix)
				x1 := config.ctmGridXo + config.ctmGridDx*float64(ix+1)
				y0 := config.ctmGridYo + config.ctmGridDy*float64(iy)
				y1 := config.ctmGridYo + config.ctmGridDy*float64(iy+1)
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
