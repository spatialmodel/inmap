/*
Copyright (C) 2013-2014 Regents of the University of Minnesota.
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
	"reflect"
	"strings"
	"sync"

	"bitbucket.org/ctessum/aqhealth"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
)

// InMAPdata holds the current state of the model.
type InMAPdata struct {

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

	Cells   []*Cell // One data holder for each grid cell
	Dt      float64 // seconds
	nlayers int     // number of model layers

	// Done specifies whether the simulation is finished.
	Done bool

	// VariableDescriptions gives descriptions of the model variables.
	VariableDescriptions map[string]string
	// VariableUnits gives the units of the model variables.
	VariableUnits map[string]string

	westBoundary  []*Cell // boundary cells
	eastBoundary  []*Cell // boundary cells
	northBoundary []*Cell // boundary cells
	southBoundary []*Cell // boundary cells

	// boundary cells; assume bottom boundary is the same as lowest layer
	topBoundary []*Cell

	// index is a spatial index of Cells.
	index *rtree.Rtree
}

// Init initializes the simulation by running d.InitFuncs.
func (d *InMAPdata) Init() error {
	for _, f := range d.InitFuncs {
		if err := f(d); err != nil {
			return err
		}
	}
	return nil
}

// Run carries out the simulation by running d.RunFuncs until d.Done is true.
func (d *InMAPdata) Run() error {
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
func (d *InMAPdata) Cleanup() error {
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

	Kzz                float64   `desc:"Grid center vertical diffusivity after applying convective fraction" units:"m²/s"`
	KzzAbove, KzzBelow []float64 // horizontal diffusivity [m2/s] (staggered grid)
	Kxxyy              float64   `desc:"Grid center horizontal diffusivity" units:"m²/s"`
	KyySouth, KyyNorth []float64 // horizontal diffusivity [m2/s] (staggered grid)
	KxxWest, KxxEast   []float64 // horizontal diffusivity at [m2/s] (staggered grid)

	M2u float64 `desc:"ACM2 upward mixing (Pleim 2007)" units:"1/s"`
	M2d float64 `desc:"ACM2 downward mixing (Pleim 2007)" units:"1/s"`

	PopData       map[string]float64 // Population for multiple demographics [people/grid cell]
	MortalityRate float64            `desc:"Baseline mortalities rate" units:"Deaths per 100,000 people per year"`

	Dx, Dy, Dz float64 // grid size [meters]
	Volume     float64 `desc:"Cell volume" units:"m³"`

	Ci       []float64 // concentrations at beginning of time step [μg/m³]
	Cf       []float64 // concentrations at end of time step [μg/m³]
	emisFlux []float64 // emissions [μg/m³/s]

	West        []*Cell // Neighbors to the East
	East        []*Cell // Neighbors to the West
	South       []*Cell // Neighbors to the South
	North       []*Cell // Neighbors to the North
	Below       []*Cell // Neighbors below
	Above       []*Cell // Neighbors above
	GroundLevel []*Cell // Neighbors at ground level
	Boundary    bool    // Does this cell represent a boundary condition?

	WestFrac, EastFrac   []float64 // Fraction of cell covered by each neighbor (adds up to 1).
	NorthFrac, SouthFrac []float64 // Fraction of cell covered by each neighbor (adds up to 1).
	AboveFrac, BelowFrac []float64 // Fraction of cell covered by each neighbor (adds up to 1).
	GroundLevelFrac      []float64 // Fraction of cell above to each ground level cell (adds up to 1).

	DxPlusHalf  []float64 // Distance between centers of cell and East [m]
	DxMinusHalf []float64 // Distance between centers of cell and West [m]
	DyPlusHalf  []float64 // Distance between centers of cell and North [m]
	DyMinusHalf []float64 // Distance between centers of cell and South [m]
	DzPlusHalf  []float64 // Distance between centers of cell and Above [m]
	DzMinusHalf []float64 // Distance between centers of cell and Below [m]

	Layer       int     // layer index of grid cell
	LayerHeight float64 // The height at the edge of this layer

	Temperature                float64 `desc:"Average temperature" units:"K"`
	WindSpeed                  float64 `desc:"RMS wind speed" units:"m/s"`
	WindSpeedInverse           float64 `desc:"RMS wind speed inverse" units:"(m/s)^(-1)"`
	WindSpeedMinusThird        float64 `desc:"RMS wind speed^(-1/3)" units:"(m/s)^(-1/3)"`
	WindSpeedMinusOnePointFour float64 `desc:"RMS wind speed^(-1.4)" units:"(m/s)^(-1.4)"`
	S1                         float64 `desc:"Stability parameter" units:"?"`
	SClass                     float64 `desc:"Stability class" units:"0=Unstable; 1=Stable"`

	TotalPM25 float64 // Total baseline PM2.5 concentration.

	sync.RWMutex // Avoid cell being written by one subroutine and read by another at the same time.

	index                 [][2]int
	aboveDensityThreshold bool
}

// DomainManipulator is a class of functions that operate on the entire InMAP
// domain.
type DomainManipulator func(d *InMAPdata) error

// CellManipulator is a class of functions that operate on a single grid cell,
// using the given timestep Dt.
type CellManipulator func(c *Cell, Dt float64)

func (c *Cell) prepare() {
	c.Volume = c.Dx * c.Dy * c.Dz
	c.Ci = make([]float64, len(polNames))
	c.Cf = make([]float64, len(polNames))
	c.emisFlux = make([]float64, len(polNames))
}

func (c *Cell) boundaryCopy() *Cell {
	c2 := new(Cell)
	c2.Polygonal = c.Polygonal
	c2.Dx, c2.Dy, c2.Dz = c.Dx, c.Dy, c.Dz
	c2.UAvg, c2.VAvg, c2.WAvg = c.UAvg, c.VAvg, c.WAvg
	c2.UDeviation, c2.VDeviation = c.UDeviation, c.VDeviation
	c2.Kxxyy, c2.Kzz = c.Kxxyy, c.Kzz
	c2.M2u, c2.M2d = c.M2u, c.M2d
	c2.Layer, c2.LayerHeight = c.Layer, c.LayerHeight
	c2.Boundary = true
	c2.prepare()
	return c2
}

// addWestBoundary adds a cell to the western boundary of the domain.
func (d *InMAPdata) addWestBoundary(cell *Cell) {
	c := cell.boundaryCopy()
	cell.West = []*Cell{c}
	d.westBoundary = append(d.westBoundary, c)
}

// addEastBoundary adds a cell to the eastern boundary of the domain.
func (d *InMAPdata) addEastBoundary(cell *Cell) {
	c := cell.boundaryCopy()
	cell.East = []*Cell{c}
	d.eastBoundary = append(d.eastBoundary, c)
}

// addSouthBoundary adds a cell to the southern boundary of the domain.
func (d *InMAPdata) addSouthBoundary(cell *Cell) {
	c := cell.boundaryCopy()
	cell.South = []*Cell{c}
	d.southBoundary = append(d.southBoundary, c)
}

// addNorthBoundary adds a cell to the northern boundary of the domain.
func (d *InMAPdata) addNorthBoundary(cell *Cell) {
	c := cell.boundaryCopy()
	cell.North = []*Cell{c}
	d.northBoundary = append(d.northBoundary, c)
}

// addTopBoundary adds a cell to the top boundary of the domain.
func (d *InMAPdata) addTopBoundary(cell *Cell) {
	c := cell.boundaryCopy()
	cell.Above = []*Cell{c}
	d.topBoundary = append(d.topBoundary, c)
}

// SetTimestepCFL returns a function that sets the time step using the
// Courant–Friedrichs–Lewy (CFL) condition
// for advection or Von Neumann stability analysis
// (http://en.wikipedia.org/wiki/Von_Neumann_stability_analysis) for
// diffusion, whichever one yields a smaller time step.
func SetTimestepCFL() DomainManipulator {
	return func(d *InMAPdata) error {
		const Cmax = 1.
		sqrt3 := math.Pow(3., 0.5)
		for i, c := range d.Cells {
			// Advection time step
			dt1 := Cmax / sqrt3 /
				max((math.Abs(c.UAvg)+c.UDeviation*2)/c.Dx,
					(math.Abs(c.VAvg)+c.VDeviation*2)/c.Dy,
					math.Abs(c.WAvg)/c.Dz)
			// vertical diffusion time step
			dt2 := Cmax * c.Dz * c.Dz / 2. / c.Kzz
			// horizontal diffusion time step
			dt3 := Cmax * c.Dx * c.Dx / 2. / c.Kxxyy
			dt4 := Cmax * c.Dy * c.Dy / 2. / c.Kxxyy
			if i == 0 {
				d.Dt = amin(dt1, dt2, dt3, dt4) // seconds
			} else {
				d.Dt = amin(d.Dt, dt1, dt2, dt3, dt4) // seconds
			}
		}
		return nil
	}
}

func harmonicMean(a, b float64) float64 {
	return 2. * a * b / (a + b)
}

// Convert cell data into a regular array
func (d *InMAPdata) toArray(pol string, layer int) []float64 {
	o := make([]float64, 0, len(d.Cells))
	for _, c := range d.Cells {
		c.RLock()
		if c.Layer > layer {
			// The cells should be sorted with the lower layers first, so we
			// should be done here.
			return o
		}
		if c.Layer == layer {
			o = append(o, c.getValue(pol))
		}
		c.RUnlock()
	}
	return o
}

// Get the value in the current cell of the specified variable.
func (c *Cell) getValue(varName string) float64 {
	if index, ok := emisLabels[varName]; ok { // Emissions
		return c.emisFlux[index]

	} else if polConv, ok := polLabels[varName]; ok { // Concentrations
		var o float64
		for i, ii := range polConv.index {
			o += c.Cf[ii] * polConv.conversion[i]
		}
		return o

	} else if _, ok := popNames[varName]; ok { // Population
		return c.PopData[varName] / c.Dx / c.Dy // divide by cell area

	} else if _, ok := popNames[strings.Replace(varName, " deaths", "", 1)]; ok {
		// Mortalities
		v := strings.Replace(varName, " deaths", "", 1)
		rr := aqhealth.RRpm25Linear(c.getValue("TotalPM2_5"))
		return aqhealth.Deaths(rr, c.PopData[v], c.MortalityRate)

	} else { // Everything else
		val := reflect.Indirect(reflect.ValueOf(c))
		return val.FieldByName(varName).Float()
	}
}

// Get the units of a variable
func (d *InMAPdata) getUnits(varName string) string {
	if _, ok := emisLabels[varName]; ok { // Emissions
		return "μg/m³/s"
	} else if _, ok := polLabels[varName]; ok { // Concentrations
		return "μg/m³"
	} else if _, ok := popNames[varName]; ok { // Population
		return "people/m²"
	} else if _, ok := popNames[strings.Replace(varName, " deaths", "", 1)]; ok {
		// Mortalities
		return "deaths/grid cell"
	} else { // Everything else
		t := reflect.TypeOf(*d.Cells[0])
		ftype, ok := t.FieldByName(varName)
		if ok {
			return ftype.Tag.Get("units")
		}
		panic(fmt.Sprintf("Unknown variable %v.", varName))
	}
}

// GetGeometry returns the cell geometry for the given layer.
func (d *InMAPdata) GetGeometry(layer int) []geom.Geom {
	o := make([]geom.Geom, 0, len(d.Cells))
	for _, c := range d.Cells {
		c.RLock()
		if c.Layer > layer {
			// The cells should be sorted with the lower layers first, so we
			// should be done here.
			return o
		}
		if c.Layer == layer {
			o = append(o, c.WebMapGeom)
		}
		c.RUnlock()
	}
	return o
}
