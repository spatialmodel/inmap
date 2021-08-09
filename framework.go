/*
Copyright © 2013 the InMAP authors.
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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmap

import (
	"fmt"
	"math"
	"sync"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
)

const (
	// Version gives the version number.
	Version = "1.9.0"

	// VarGridDataVersion gives the version of the variable grid data reuquired by
	// this version of the software.
	VarGridDataVersion = "1.6.1"

	// InMAPDataVersion is the version of the InMAP data required by this version
	// of the software.
	InMAPDataVersion = "1.2.0"
)

// InMAP holds the current state of the model.
type InMAP struct {

	// InitFuncs are functions to be called in the given order
	//  at the beginning of the simulation.
	InitFuncs []DomainManipulator

	// RunFuncs are functions to be called in the given order repeatedly
	// until "Done" is true. Therefore, the simulation will not end until
	// one of RunFuncs sets "Done" to true.
	RunFuncs []DomainManipulator

	// CleanupFuncs are functions to be run in the given order after the
	// simulation has completed.
	CleanupFuncs []DomainManipulator

	cells   *cellList // One data holder for each grid cell
	Dt      float64   // seconds
	nlayers int       // number of model layers

	// Done specifies whether the simulation is finished.
	Done bool

	// VariableDescriptions gives descriptions of the model variables.
	VariableDescriptions map[string]string
	// VariableUnits gives the units of the model variables.
	VariableUnits map[string]string

	westBoundary  *cellList // boundary cells
	eastBoundary  *cellList // boundary cells
	northBoundary *cellList // boundary cells
	southBoundary *cellList // boundary cells

	// boundary cells; assume bottom boundary is the same as lowest layer
	topBoundary *cellList

	// PopIndices gives the array index of each population type in the PopData
	// field in each Cell.
	PopIndices map[string]int

	// mortIndices gives the array index of each mortality rate in the mortData
	// field in each Cell.
	mortIndices map[string]int

	// index is a spatial index of Cells.
	index *rtree.Rtree

	cellLock sync.Mutex
}

// Init initializes the simulation by running d.InitFuncs.
func (d *InMAP) Init() error {
	d.init()
	for _, f := range d.InitFuncs {
		if err := f(d); err != nil {
			return err
		}
	}
	return nil
}

func (d *InMAP) init() {
	d.cells = new(cellList)
	d.westBoundary = new(cellList)
	d.eastBoundary = new(cellList)
	d.northBoundary = new(cellList)
	d.southBoundary = new(cellList)
	d.topBoundary = new(cellList)
	d.index = rtree.NewTree(25, 50)
}

// Run carries out the simulation by running d.RunFuncs until d.Done is true.
func (d *InMAP) Run() error {
	for !d.Done {
		for _, f := range d.RunFuncs {
			if err := f(d); err != nil {
				return err
			}
		}
	}
	return nil
}

// Cleanup finishes the simulation by running d.CleanupFuncs.
func (d *InMAP) Cleanup() error {
	for _, f := range d.CleanupFuncs {
		if err := f(d); err != nil {
			return err
		}
	}
	return nil
}

// Cell holds the state of a single grid cell.
type Cell struct {
	geom.Polygonal                // Cell geometry
	WebMapGeom     geom.Polygonal // Cell geometry in web map (mercator) coordinate system

	UAvg       float64 `desc:"Average East-West wind speed" units:"m/s"`
	VAvg       float64 `desc:"Average North-South wind speed" units:"m/s"`
	WAvg       float64 `desc:"Average up-down wind speed" units:"m/s"`
	UDeviation float64 `desc:"Average deviation from East-West velocity" units:"m/s"`
	VDeviation float64 `desc:"Average deviation from North-South velocity" units:"m/s"`

	AOrgPartitioning float64 `desc:"Organic particle partitioning" units:"fraction particles"`
	BOrgPartitioning float64 // particle fraction
	SPartitioning    float64 `desc:"Sulfur particle partitioning" units:"fraction particles"`
	NOPartitioning   float64 `desc:"Nitrate particle partitioning" units:"fraction particles"`
	NHPartitioning   float64 `desc:"Ammonium particle partitioning" units:"fraction particles"`
	SO2oxidation     float64 `desc:"SO2 oxidation to SO4 by HO and H2O2" units:"1/s"`

	ParticleWetDep float64 `desc:"Particle wet deposition" units:"1/s"`
	SO2WetDep      float64 `desc:"SO2 wet deposition" units:"1/s"`
	OtherGasWetDep float64 `desc:"Wet deposition: other gases" units:"1/s"`
	ParticleDryDep float64 `desc:"Particle dry deposition" units:"m/s"`

	NH3DryDep float64 `desc:"Ammonia dry deposition" units:"m/s"`
	SO2DryDep float64 `desc:"SO2 dry deposition" units:"m/s"`
	VOCDryDep float64 `desc:"VOC dry deposition" units:"m/s"`
	NOxDryDep float64 `desc:"NOx dry deposition" units:"m/s"`

	Kzz   float64 `desc:"Grid center vertical diffusivity after applying convective fraction" units:"m²/s"`
	Kxxyy float64 `desc:"Grid center horizontal diffusivity" units:"m²/s"`

	M2u float64 `desc:"ACM2 upward mixing (Pleim 2007)" units:"1/s"`
	M2d float64 `desc:"ACM2 downward mixing (Pleim 2007)" units:"1/s"`

	PopData  []float64 // Population for multiple demographics [people/grid cell]
	MortData []float64 // Baseline mortality rates for multiple demographics [Deaths per 100,000 people per year/grid cell]

	Dx     float64 `desc:"Cell x length" units:"m"`
	Dy     float64 `desc:"Cell y length" units:"m"`
	Dz     float64 `desc:"Cell z length" units:"m"`
	Volume float64 `desc:"Cell volume" units:"m³"`

	Ci        []float64 // concentrations at beginning of time step [μg/m³]
	Cf        []float64 // concentrations at end of time step [μg/m³]
	EmisFlux  []float64 // emissions [μg/m³/s]
	CBaseline []float64 // Total baseline PM2.5 concentration.

	west        *cellList // Neighbors to the East
	east        *cellList // Neighbors to the West
	south       *cellList // Neighbors to the South
	north       *cellList // Neighbors to the North
	below       *cellList // Neighbors below
	above       *cellList // Neighbors above
	groundLevel *cellList // Neighbors at ground level
	boundary    bool      // Does this cell represent a boundary condition?

	Layer       int     `desc:"Vertical layer index" units:"-"`
	LayerHeight float64 `desc:"Height at layer bottom" units:"m"`

	Temperature                float64 `desc:"Average temperature" units:"K"`
	WindSpeed                  float64 `desc:"RMS wind speed" units:"m/s"`
	WindSpeedInverse           float64 `desc:"RMS wind speed inverse" units:"(m/s)^(-1)"`
	WindSpeedMinusThird        float64 `desc:"RMS wind speed^(-1/3)" units:"(m/s)^(-1/3)"`
	WindSpeedMinusOnePointFour float64 `desc:"RMS wind speed^(-1.4)" units:"(m/s)^(-1.4)"`
	S1                         float64 `desc:"Stability parameter" units:"?"`
	SClass                     float64 `desc:"Stability class" units:"0=Unstable; 1=Stable"`

	mutex sync.RWMutex // Avoid cell being written by one subroutine and read by another at the same time.

	Index                 [][2]int // Index gives this cell's place in the nest structure.
	AboveDensityThreshold bool
}

func (c *Cell) String() string {
	b := c.Bounds()
	return fmt.Sprintf("{min=%+v, max=%+v, layer=%d, boundary=%v}", b.Min, b.Max, c.Layer, c.boundary)
}

// neighborInfo holds information about the relationship between a cell and
// its neighbor.
type neighborInfo struct {
	// coverFrac is the fration of the cell covered by
	// this neighbor. It adds up to 1 for all neighbors.
	coverFrac float64

	// centerDistance is the distance between the
	// center of this cell the neighbor [m].
	centerDistance float64

	// diff is the staggered grid diffusivity between this
	// cell and the neighbor [m2/s].
	diff float64
}

// Cells returns the InMAP grid cells as an array.
func (d *InMAP) Cells() []*Cell {
	return d.cells.array()
}

// DomainManipulator is a class of functions that operate on the entire InMAP
// domain.
type DomainManipulator func(d *InMAP) error

// CellManipulator is a class of functions that operate on a single grid cell,
// using the given timestep Dt [seconds].
type CellManipulator func(c *Cell, Dt float64)

func (c *Cell) make(m Mechanism) {
	c.Ci = make([]float64, m.Len())
	c.Cf = make([]float64, m.Len())
	c.CBaseline = make([]float64, len(PolNames))
	c.west = new(cellList)
	c.east = new(cellList)
	c.south = new(cellList)
	c.north = new(cellList)
	c.below = new(cellList)
	c.above = new(cellList)
	c.groundLevel = new(cellList)
}

func (c *Cell) boundaryCopy(m Mechanism) *Cell {
	c2 := new(Cell)
	c2.Polygonal = c.Polygonal
	c2.Dx, c2.Dy, c2.Dz = c.Dx, c.Dy, c.Dz
	c2.UAvg, c2.VAvg, c2.WAvg = c.UAvg, c.VAvg, c.WAvg
	c2.UDeviation, c2.VDeviation = c.UDeviation, c.VDeviation
	c2.Kxxyy, c2.Kzz = c.Kxxyy, c.Kzz
	c2.M2u, c2.M2d = c.M2u, c.M2d
	c2.Layer, c2.LayerHeight = c.Layer, c.LayerHeight
	c2.boundary = true
	c2.make(m)
	c2.Volume = c2.Dx * c2.Dy * c2.Dz
	c2.PopData = c.PopData
	c2.MortData = c.MortData
	return c2
}

// addWestBoundary adds a cell to the western boundary of the domain.
func (d *InMAP) addWestBoundary(cell *Cell, m Mechanism) {
	c := cell.boundaryCopy(m)
	ref := cell.west.add(c)
	d.westBoundary.add(c)
	neighborInfoBoundaryEastWest(ref)
}

// addEastBoundary adds a cell to the eastern boundary of the domain.
func (d *InMAP) addEastBoundary(cell *Cell, m Mechanism) {
	c := cell.boundaryCopy(m)
	ref := cell.east.add(c)
	d.eastBoundary.add(c)
	neighborInfoBoundaryEastWest(ref)
}

// addSouthBoundary adds a cell to the southern boundary of the domain.
func (d *InMAP) addSouthBoundary(cell *Cell, m Mechanism) {
	c := cell.boundaryCopy(m)
	ref := cell.south.add(c)
	d.southBoundary.add(c)
	neighborInfoBoundarySouthNorth(ref)
}

// addNorthBoundary adds a cell to the northern boundary of the domain.
func (d *InMAP) addNorthBoundary(cell *Cell, m Mechanism) {
	c := cell.boundaryCopy(m)
	ref := cell.north.add(c)
	d.northBoundary.add(c)
	neighborInfoBoundarySouthNorth(ref)
}

// addTopBoundary adds a cell to the top boundary of the domain.
func (d *InMAP) addTopBoundary(cell *Cell, m Mechanism) {
	c := cell.boundaryCopy(m)
	ref := cell.above.add(c)
	d.topBoundary.add(c)
	neighborInfoBoundaryTopBottom(ref)
}

// SetTimestepCFL returns a function that sets the time step using the
// Courant–Friedrichs–Lewy (CFL) condition
// for advection or Von Neumann stability analysis
// (http://en.wikipedia.org/wiki/Von_Neumann_stability_analysis) for
// diffusion, whichever one yields a smaller time step.
func SetTimestepCFL() DomainManipulator {
	sqrt3 := math.Pow(3., 0.5)
	return func(d *InMAP) error {
		const (
			// Cmax is the maximum CFL value allowed.
			CMax = 0.75
		)
		d.Dt = math.Inf(1)
		for _, c := range *d.cells {
			// Advection time step
			cUadv := (math.Abs(c.UAvg) + c.UDeviation*2) / c.Dx
			cVadv := (math.Abs(c.VAvg) + c.VDeviation*2) / c.Dy
			cWadv := math.Abs(c.WAvg) / c.Dz
			// horizontal diffusion time step
			cXdiff := 2. * c.Kxxyy / (c.Dx * c.Dx)
			cYdiff := 2. * c.Kxxyy / (c.Dy * c.Dy)
			// vertical diffusion time step
			cZdiff := 2. * c.Kzz / (c.Dz * c.Dz)

			dt1 := CMax / sqrt3 / max(cUadv+cXdiff, cVadv+cYdiff,
				cWadv+cZdiff+c.M2d+c.M2u)

			d.Dt = amin(d.Dt, dt1) // seconds
		}
		if !(d.Dt > 0) {
			return fmt.Errorf("invalid timestep %g; check InMAP input data", d.Dt)
		}
		return nil
	}
}

func harmonicMean(a, b float64) float64 {
	return 2. * a * b / (a + b)
}

// GetGeometry returns the cell geometry for the given layer.
// if WebMap is true, it returns the geometry in web mercator projection,
// otherwise it returns the native grid projection.
func (d *InMAP) GetGeometry(layer int, webMap bool) []geom.Polygonal {
	o := make([]geom.Polygonal, 0, d.cells.len())
	cells := d.cells.array()
	for _, c := range cells {
		c.mutex.RLock()
		if c.Layer > layer {
			// The cells should be sorted with the lower layers first, so we
			// should be done here.
			c.mutex.RUnlock()
			return o
		}
		if c.Layer == layer {
			if webMap {
				o = append(o, c.WebMapGeom)
			} else {
				o = append(o, c.Polygonal)
			}
		}
		c.mutex.RUnlock()
	}
	return o
}

// Regrid regrids concentration data from one spatial grid to a different one.
func Regrid(oldGeom, newGeom []geom.Polygonal, oldData []float64) (newData []float64, err error) {
	type data struct {
		geom.Polygonal
		data float64
	}
	if len(oldGeom) != len(oldData) {
		return nil, fmt.Errorf("oldGeom and oldData have different lengths: %d!=%d", len(oldGeom), len(oldData))
	}
	index := rtree.NewTree(25, 50)
	for i, g := range oldGeom {
		index.Insert(&data{
			Polygonal: g,
			data:      oldData[i],
		})
	}
	newData = make([]float64, len(newGeom))
	for i, g := range newGeom {
		for _, dI := range index.SearchIntersect(g.Bounds()) {
			d := dI.(*data)
			isect := g.Intersection(d.Polygonal)
			if isect == nil {
				continue
			}
			a := isect.Area()
			frac := a / g.Area()
			newData[i] += d.data * frac
		}
	}
	return newData, nil
}

// CellIntersections returns an array of all of the grid cells (on all vertical levels)
// that intersect g, and an array of the fraction of g that intersects with each
// cell.
func (d *InMAP) CellIntersections(g geom.Geom) (cells []*Cell, fractions []float64) {
	cellIs := d.index.SearchIntersect(g.Bounds())
	cells = make([]*Cell, 0, len(cellIs))
	fractions = make([]float64, 0, len(cellIs))
	for _, cellI := range cellIs {
		cell := cellI.(*Cell)
		if fraction := calcWeightFactor(g, cell); fraction != 0 {
			cells = append(cells, cell)
			fractions = append(fractions, fraction)
		}
	}
	return cells, fractions
}

// VerticalProfile retrieves the vertical profile for a given
// variable at the given location p in the native grid projection.
func (d *InMAP) VerticalProfile(variable string, p geom.Point, m Mechanism) (height, vals []float64, err error) {
	if err := d.checkModelVars(m, variable); err != nil {
		return nil, nil, err
	}

	height = make([]float64, d.nlayers)
	vals = make([]float64, d.nlayers)
	cells := d.index.SearchIntersect(p.Bounds())
	if len(cells) == 0 {
		return nil, nil, fmt.Errorf("inmap.VerticalProfile: location %+v not in grid", p)
	}
	var c *Cell
	for _, cI := range cells {
		c = cI.(*Cell)
		if c.Layer == 0 {
			break
		}
	}
	if c.Layer != 0 {
		panic("couldn't find a ground level cell.")
	}
	i := 0
	for !c.boundary {
		vals[i] = c.getValue(variable, d.PopIndices, d.mortIndices, m)
		height[i] = c.LayerHeight + c.Dz/2.
		c = (*c.above)[0].Cell
		i++
	}
	return
}
