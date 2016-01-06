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
	"archive/zip"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"

	"bitbucket.org/ctessum/aqhealth"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/op"
)

// InMAPdata is holds the current state of the model.
type InMAPdata struct {
	Data    []*Cell // One data holder for each grid cell
	Dt      float64 // seconds
	Nlayers int     // number of model layers

	// Number of iterations to calculate. If < 1,
	// calculate convergence automatically.
	NumIterations int

	LayerStart    []int   // start index of each layer (inclusive)
	LayerEnd      []int   // end index of each layer (exclusive)
	westBoundary  []*Cell // boundary cells
	eastBoundary  []*Cell // boundary cells
	northBoundary []*Cell // boundary cells
	southBoundary []*Cell // boundary cells

	// boundary cells; assume bottom boundary is the same as lowest layer
	topBoundary []*Cell
}

func init() {
	gob.Register(geom.Polygon{})
}

// Cell holds the state of a single grid cell.
type Cell struct {
	geom.T            // Cell geometry
	WebMapGeom geom.T // Cell geometry in web map (mercator) coordinate system

	UAvg       float64 `desc:"Average East-West wind speed" units:"m/s"`
	VAvg       float64 `desc:"Average North-South wind speed" units:"m/s"`
	WAvg       float64 `desc:"Average up-down wind speed" units:"m/s"`
	UDeviation float64 `desc:"Average deviation from East-West velocity" units:"m/s"`
	VDeviation float64 `desc:"Average deviation from North-South velocity" units:"m/s"`
	UDevLength float64 `desc:"Length of average East-West velocity deviation" units:"m"`
	VDevLength float64 `desc:"Length of average North-South velocity deviation" units:"m"`

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
	Row        int     // master cell index

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

	IWest        []int // Row indexes of neighbors to the East
	IEast        []int // Row indexes of neighbors to the West
	ISouth       []int // Row indexes of neighbors to the South
	INorth       []int // Row indexes of neighbors to the north
	IBelow       []int // Row indexes of neighbors below
	IAbove       []int // Row indexes of neighbors above
	IGroundLevel []int // Row indexes of neighbors at ground level

	NSMeanderCells []*Cell   // Cells for nonlocal advection
	EWMeanderCells []*Cell   // Cells for nonlocal advection
	NSMeanderFrac  []float64 // Volume fractions of each nonlocal cell
	EWMeanderFrac  []float64 // Volume fractions of each nonlocal cell

	DxPlusHalf  []float64 // Distance between centers of cell and East [m]
	DxMinusHalf []float64 // Distance between centers of cell and West [m]
	DyPlusHalf  []float64 // Distance between centers of cell and North [m]
	DyMinusHalf []float64 // Distance between centers of cell and South [m]
	DzPlusHalf  []float64 // Distance between centers of cell and Above [m]
	DzMinusHalf []float64 // Distance between centers of cell and Below [m]

	Layer int // layer index of grid cell

	Temperature                float64 `desc:"Average temperature" units:"K"`
	WindSpeed                  float64 `desc:"RMS wind speed" units:"m/s"`
	WindSpeedInverse           float64 `desc:"RMS wind speed inverse" units:"(m/s)^(-1)"`
	WindSpeedMinusThird        float64 `desc:"RMS wind speed^(-1/3)" units:"(m/s)^(-1/3)"`
	WindSpeedMinusOnePointFour float64 `desc:"RMS wind speed^(-1.4)" units:"(m/s)^(-1.4)"`
	S1                         float64 `desc:"Stability parameter" units:"?"`
	SClass                     float64 `desc:"Stability class" units:"0=Unstable; 1=Stable"`

	sync.RWMutex // Avoid cell being written by one subroutine and read by another at the same time.
}

func (c *Cell) prepare() {
	c.Volume = c.Dx * c.Dy * c.Dz
	c.Ci = make([]float64, len(polNames))
	c.Cf = make([]float64, len(polNames))
	c.emisFlux = make([]float64, len(polNames))
}

func (c *Cell) makecopy() *Cell {
	c2 := new(Cell)
	c2.Dx, c2.Dy, c2.Dz = c.Dx, c.Dy, c.Dz
	c2.Kxxyy = c.Kxxyy
	c2.prepare()
	return c2
}

// InitOption allows options of different ways to initialize
// the model.
type InitOption func(*InMAPdata) error

// UseFileTemplate initializes the model with data from a local disk,
// where `filetemplate` is the path to
// the Gob files with meteorology and background concentration data
// (where `[layer]` is a stand-in for the layer number), and
// `nLayers` is the number of vertical layers in the model.
func UseFileTemplate(filetemplate string, nLayers int) InitOption {
	return func(d *InMAPdata) error {
		readers := make([]io.ReadCloser, nLayers)
		var err error
		for k := 0; k < nLayers; k++ {
			filename := strings.Replace(filetemplate, "[layer]",
				fmt.Sprintf("%v", k), -1)
			if readers[k], err = os.Open(filename); err != nil {
				return fmt.Errorf("Problem opening InMAP data file: %v",
					err.Error())
			}
		}
		return UseReaders(readers)(d)
	}
}

// UseWebArchive initializes the model with data from a network,
// where `url` is the network address, `fileNameTemplate` is the template for
// the names of the Gob files with meteorology and background concentration data
// (where `[layer]` is a stand-in for the layer number), and
// `nLayers` is the number of vertical layers in the model. It is assumed
// that the input files are contained in a single zip archive.
func UseWebArchive(url, fileNameTemplate string, nLayers int) InitOption {
	return func(d *InMAPdata) error {
		fmt.Printf("Downloading data from %v...\n", url)
		response, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("error while downloading %v: %v", url, err)
		}
		defer response.Body.Close()
		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}
		zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			panic(err)
		}
		readers := make([]io.ReadCloser, nLayers)
		for k := 0; k < nLayers; k++ {
			found := false
			filename := strings.Replace(fileNameTemplate, "[layer]",
				fmt.Sprintf("%v", k), -1)
			for _, f := range zr.File {
				_, file := filepath.Split(f.Name)
				if file == filename {
					found = true
					if readers[k], err = f.Open(); err != nil {
						return fmt.Errorf(
							"error while opening web archive: %v", err)
					}
				}
			}
			if !found {
				return fmt.Errorf("could not file file %v in web archive", filename)

			}
		}
		return UseReaders(readers)(d)
	}
}

// UseCloudStorage initializes the model with data from Google Cloud Storage,
// where ctx is the Context, `bucket` is the name of the bucket,
// `fileNameTemplate` is the template for
// the names of the Gob files with meteorology and background concentration data
// (where `[layer]` is a stand-in for the layer number), and
// `nLayers` is the number of vertical layers in the model. To minimize individual
// download sizes, the input files
// must be enclosed in zip files, with one input file per zip file.
// ctx must include any authentication needed for acessing the files.
func UseCloudStorage(ctx context.Context, bucket string, fileNameTemplate string,
	nLayers int) InitOption {
	return func(d *InMAPdata) error {
		readers := make([]io.ReadCloser, nLayers)
		ts, err := google.DefaultTokenSource(ctx, storage.ScopeReadOnly)
		if err != nil {
			return fmt.Errorf("could not retrieve default token source: %v", err)
		}
		client, err := storage.NewClient(ctx, cloud.WithTokenSource(ts))
		if err != nil {
			return fmt.Errorf("unable to get default client: %v", err)
		}
		bh := client.Bucket(bucket)
		for k := 0; k < nLayers; k++ {
			filename := strings.Replace(fileNameTemplate, "[layer]",
				fmt.Sprintf("%v", k), -1)
			obj := bh.Object(filename)
			rc, err := obj.NewReader(ctx)
			if err != nil {
				log.Printf("In UseCloudStorage, retrieving file "+
					"%v: %v", filename, err)
				return err
			}
			b, err := ioutil.ReadAll(rc)
			if err != nil {
				log.Printf(
					"UseCloudStorage: error while opening zip file: %v", err)
				return err
			}
			zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
			if err != nil {
				panic(err)
			}
			if readers[k], err = zr.File[0].Open(); err != nil {
				log.Printf(
					"UseCloudStorage: error while opening zip file: %v", err)
				return err
			}
		}
		log.Printf("got readers")
		return UseReaders(readers)(d)
	}
}

// UseReaders initializes the model with data from `readers`,
// with one reader for the data for each model layer. The
// readers must be input in order with the ground-level
// data first.
func UseReaders(readers []io.ReadCloser) InitOption {
	return func(d *InMAPdata) error {
		d.Nlayers = len(readers)
		inputData := make([][]*Cell, d.Nlayers)
		d.LayerStart = make([]int, d.Nlayers)
		d.LayerEnd = make([]int, d.Nlayers)
		var wg sync.WaitGroup
		wg.Add(d.Nlayers)
		for k := 0; k < d.Nlayers; k++ {
			go func(k int) {
				f := readers[k]
				g := gob.NewDecoder(f)
				if err := g.Decode(&inputData[k]); err != nil {
					panic(err)
				}
				d.LayerStart[k] = 0
				d.LayerEnd[k] = len(inputData[k])
				f.Close()
				wg.Done()
			}(k)
		}
		wg.Wait()
		ncells := 0
		// Adjust so beginning of layer is at end of previous layer
		for k := 0; k < d.Nlayers; k++ {
			d.LayerStart[k] += ncells
			d.LayerEnd[k] += ncells
			ncells += len(inputData[k])
		}
		// set up data holders
		d.Data = make([]*Cell, ncells)
		for _, indata := range inputData {
			for _, c := range indata {
				c.prepare()
				d.Data[c.Row] = c
			}
		}
		return nil
	}
}

// InitInMAPdata initializes the model where
// `option` is the selected option for retrieving the input data,
// `numIterations` is the number of iterations to calculate
// (if `numIterations` < 1, convergence is calculated automatically),
// and `httpPort` is the port number for hosting the html GUI
// (if `httpPort` is "", then the GUI doesn't run).
func InitInMAPdata(option InitOption, numIterations int, httpPort string) (*InMAPdata, error) {
	d := new(InMAPdata)
	d.NumIterations = numIterations
	err := option(d)
	if err != nil {
		return nil, err
	}
	for _, cell := range d.Data {
		cell.setup(d)
	}
	/*nprocs := runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	wg.Add(nprocs)
	for procNum := 0; procNum < nprocs; procNum++ {
		go func(procNum int) {
			for ii := procNum; ii < len(d.Data); ii += nprocs {
				cell := d.Data[ii]
				cell.setup(d)
			}
			wg.Done()
		}(procNum)
	}*/
	//wg.Wait()
	d.setTstepCFL() // Set time step
	if httpPort != "" {
		go d.WebServer(httpPort)
	}
	return d, nil
}

func (c *Cell) setup(d *InMAPdata) {
	// Link cells to neighbors or boundaries.
	if len(c.IWest) == 0 {
		d.addWestBoundary(c)
	} else {
		c.West = make([]*Cell, len(c.IWest))
		for i, row := range c.IWest {
			c.West[i] = d.Data[row]
		}
		c.IWest = nil
	}
	if len(c.IEast) == 0 {
		d.addEastBoundary(c)
	} else {
		c.East = make([]*Cell, len(c.IEast))
		for i, row := range c.IEast {
			c.East[i] = d.Data[row]
		}
		c.IEast = nil
	}
	if len(c.ISouth) == 0 {
		d.addSouthBoundary(c)
	} else {
		c.South = make([]*Cell, len(c.ISouth))
		for i, row := range c.ISouth {
			c.South[i] = d.Data[row]
		}
		c.ISouth = nil
	}
	if len(c.INorth) == 0 {
		d.addNorthBoundary(c)
	} else {
		c.North = make([]*Cell, len(c.INorth))
		for i, row := range c.INorth {
			c.North[i] = d.Data[row]
		}
		c.INorth = nil
	}
	if len(c.IAbove) == 0 || c.Layer == d.Nlayers-1 {
		d.addTopBoundary(c)
	} else {
		c.Above = make([]*Cell, len(c.IAbove))
		for i, row := range c.IAbove {
			c.Above[i] = d.Data[row]
		}
		c.IAbove = nil
	}
	if c.Layer != 0 {
		c.Below = make([]*Cell, len(c.IBelow))
		c.GroundLevel = make([]*Cell, len(c.IGroundLevel))
		for i, row := range c.IBelow {
			c.Below[i] = d.Data[row]
		}
		for i, row := range c.IGroundLevel {
			c.GroundLevel[i] = d.Data[row]
		}
		c.IBelow = nil
		c.IGroundLevel = nil
	} else { // assume bottom boundary is the same as lowest layer.
		c.Below = []*Cell{d.Data[c.Row]}
		c.GroundLevel = []*Cell{d.Data[c.Row]}
	}
	c.neighborInfo()
}

// addWestBoundary adds a cell to the western boundary of the domain.
func (d *InMAPdata) addWestBoundary(cell *Cell) {
	c := cell.makecopy()
	cell.West = []*Cell{c}
	c.West, c.East = []*Cell{c}, []*Cell{c} // boundary cells are adjacent to themselves.
	c.Boundary = true
	c.UAvg = cell.UAvg
	d.westBoundary = append(d.westBoundary, c)
}

// addEastBoundary adds a cell to the eastern boundary of the domain.
func (d *InMAPdata) addEastBoundary(cell *Cell) {
	c := cell.makecopy()
	cell.East = []*Cell{c}
	c.West, c.East = []*Cell{c}, []*Cell{c} // boundary cells are adjacent to themselves.
	c.Boundary = true
	c.UAvg = cell.UAvg
	d.eastBoundary = append(d.eastBoundary, c)
}

// addSouthBoundary adds a cell to the southern boundary of the domain.
func (d *InMAPdata) addSouthBoundary(cell *Cell) {
	c := cell.makecopy()
	cell.South = []*Cell{c}
	c.South, c.North = []*Cell{c}, []*Cell{c} // boundary cells are adjacent to themselves.
	c.Boundary = true
	c.VAvg = cell.VAvg
	d.southBoundary = append(d.southBoundary, c)
}

// addNorthBoundary adds a cell to the northern boundary of the domain.
func (d *InMAPdata) addNorthBoundary(cell *Cell) {
	c := cell.makecopy()
	cell.North = []*Cell{c}
	c.South, c.North = []*Cell{c}, []*Cell{c} // boundary cells are adjacent to themselves.
	c.Boundary = true
	c.VAvg = cell.VAvg
	d.northBoundary = append(d.northBoundary, c)
}

// addTopBoundary adds a cell to the top boundary of the domain.
func (d *InMAPdata) addTopBoundary(cell *Cell) {
	c := cell.makecopy()
	cell.Above = []*Cell{c}
	c.Below, c.Above = []*Cell{c}, []*Cell{c} // boundary cells are adjacent to themselves.
	c.Boundary = true
	c.WAvg = cell.WAvg
	d.topBoundary = append(d.topBoundary, c)
}

// Calculate center-to-center cell distance,
// fractions of grid cell covered by each neighbor
// and harmonic mean staggered-grid diffusivities.
func (c *Cell) neighborInfo() {
	c.DxPlusHalf = make([]float64, len(c.East))
	c.EastFrac = make([]float64, len(c.East))
	c.KxxEast = make([]float64, len(c.East))
	for i, e := range c.East {
		c.DxPlusHalf[i] = (c.Dx + e.Dx) / 2.
		c.EastFrac[i] = min(e.Dy/c.Dy, 1.)
		c.KxxEast[i] = harmonicMean(c.Kxxyy, e.Kxxyy)
	}
	c.DxMinusHalf = make([]float64, len(c.West))
	c.WestFrac = make([]float64, len(c.West))
	c.KxxWest = make([]float64, len(c.West))
	for i, w := range c.West {
		c.DxMinusHalf[i] = (c.Dx + w.Dx) / 2.
		c.WestFrac[i] = min(w.Dy/c.Dy, 1.)
		c.KxxWest[i] = harmonicMean(c.Kxxyy, w.Kxxyy)
	}
	c.DyPlusHalf = make([]float64, len(c.North))
	c.NorthFrac = make([]float64, len(c.North))
	c.KyyNorth = make([]float64, len(c.North))
	for i, n := range c.North {
		c.DyPlusHalf[i] = (c.Dy + n.Dy) / 2.
		c.NorthFrac[i] = min(n.Dx/c.Dx, 1.)
		c.KyyNorth[i] = harmonicMean(c.Kxxyy, n.Kxxyy)
	}
	c.DyMinusHalf = make([]float64, len(c.South))
	c.SouthFrac = make([]float64, len(c.South))
	c.KyySouth = make([]float64, len(c.South))
	for i, s := range c.South {
		c.DyMinusHalf[i] = (c.Dy + s.Dy) / 2.
		c.SouthFrac[i] = min(s.Dx/c.Dx, 1.)
		c.KyySouth[i] = harmonicMean(c.Kxxyy, s.Kxxyy)
	}
	c.DzPlusHalf = make([]float64, len(c.Above))
	c.AboveFrac = make([]float64, len(c.Above))
	c.KzzAbove = make([]float64, len(c.Above))
	for i, a := range c.Above {
		c.DzPlusHalf[i] = (c.Dz + a.Dz) / 2.
		c.AboveFrac[i] = min((a.Dx*a.Dy)/(c.Dx*c.Dy), 1.)
		c.KzzAbove[i] = harmonicMean(c.Kzz, a.Kzz)
	}
	c.DzMinusHalf = make([]float64, len(c.Below))
	c.BelowFrac = make([]float64, len(c.Below))
	c.KzzBelow = make([]float64, len(c.Below))
	for i, b := range c.Below {
		c.DzMinusHalf[i] = (c.Dz + b.Dz) / 2.
		c.BelowFrac[i] = min((b.Dx*b.Dy)/(c.Dx*c.Dy), 1.)
		c.KzzBelow[i] = harmonicMean(c.Kzz, b.Kzz)
	}
	c.GroundLevelFrac = make([]float64, len(c.GroundLevel))
	for i, g := range c.GroundLevel {
		c.GroundLevelFrac[i] = min((g.Dx*g.Dy)/(c.Dx*c.Dy), 1.)
	}
	c.addMeanderCells()
}

// meanderCells is a recursive function to find all of the cells within the
// deviation length of c in the direction specified by field.
func meanderCells(c, mCell *Cell, cLoc geom.Point, field string) []*Cell {
	if mCell.Boundary {
		return nil
	}
	var mCells []*Cell
	mcLoc, err := op.Centroid(mCell.T)
	if err != nil {
		panic(err)
	}
	if math.Abs(cLoc.X-mcLoc.X) < c.UDevLength {
		mCells = append(mCells, mCell)
		nextCells := reflect.ValueOf(mCell).Elem().FieldByName(field).Interface().([]*Cell)
		for _, mc := range nextCells {
			mCells = append(mCells, meanderCells(c, mc, cLoc, field)...)
		}
	}
	return mCells
}

func (c *Cell) addMeanderCells() {
	cLoc, err := op.Centroid(c.T)
	if err != nil {
		panic(err)
	}
	// Find cells for east-west nonlocal advection
	for _, w := range c.West {
		c.EWMeanderCells = append(c.EWMeanderCells,
			meanderCells(c, w, cLoc, "West")...)
	}
	for _, e := range c.East {
		c.EWMeanderCells = append(c.EWMeanderCells,
			meanderCells(c, e, cLoc, "East")...)
	}
	// calculate volume fractions for each cell
	c.EWMeanderFrac = make([]float64, len(c.EWMeanderCells))
	v := 0.
	for _, cc := range c.EWMeanderCells {
		v += cc.Volume
	}
	for i, cc := range c.EWMeanderCells {
		c.EWMeanderFrac[i] = cc.Volume / v
	}
	for _, s := range c.South {
		c.NSMeanderCells = append(c.NSMeanderCells,
			meanderCells(c, s, cLoc, "South")...)
	}
	for _, n := range c.North {
		c.NSMeanderCells = append(c.NSMeanderCells,
			meanderCells(c, n, cLoc, "North")...)
	}
	// calculate volume fractions for each cell
	c.NSMeanderFrac = make([]float64, len(c.NSMeanderCells))
	v = 0.
	for _, cc := range c.NSMeanderCells {
		v += cc.Volume
	}
	for i, cc := range c.NSMeanderCells {
		c.NSMeanderFrac[i] = cc.Volume / v
	}
}

// addEmissionsFlux adds emissions to c. It should be run once for each timestep.
func (c *Cell) addEmissionsFlux(d *InMAPdata) {
	for i := range polNames {
		c.Cf[i] += c.emisFlux[i] * d.Dt
		c.Ci[i] = c.Cf[i]
	}
}

// Set the time step using the Courant–Friedrichs–Lewy (CFL) condition.
// for advection or Von Neumann stability analysis
// (http://en.wikipedia.org/wiki/Von_Neumann_stability_analysis) for
// diffusion, whichever one yields a smaller time step.
func (d *InMAPdata) setTstepCFL() {
	const Cmax = 1.
	for i, c := range d.Data {

		// Advection time step
		dt1 := Cmax / math.Pow(3., 0.5) /
			max(math.Abs(c.UAvg)/c.Dx, math.Abs(c.VAvg)/c.Dy, math.Abs(c.WAvg)/c.Dz,
				c.UDeviation/c.Dx, c.VDeviation/c.Dy)
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
}

func harmonicMean(a, b float64) float64 {
	return 2. * a * b / (a + b)
}

// Convert cell data into a regular array
func (d *InMAPdata) toArray(pol string, layer int) []float64 {
	o := make([]float64, d.LayerEnd[layer]-d.LayerStart[layer])
	for i, c := range d.Data[d.LayerStart[layer]:d.LayerEnd[layer]] {
		c.RLock()
		o[i] = c.getValue(pol)
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
		t := reflect.TypeOf(*d.Data[0])
		ftype, ok := t.FieldByName(varName)
		if ok {
			return ftype.Tag.Get("units")
		}
		panic(fmt.Sprintf("Unknown variable %v.", varName))
	}
}

// GetGeometry returns the cell geometry for the given layer.
func (d *InMAPdata) GetGeometry(layer int) []geom.T {
	o := make([]geom.T, d.LayerEnd[layer]-d.LayerStart[layer])
	for i, c := range d.Data[d.LayerStart[layer]:d.LayerEnd[layer]] {
		o[i] = c.WebMapGeom
	}
	return o
}
