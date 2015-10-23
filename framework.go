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
	"runtime"
	"strings"
	"sync"

	"google.golang.org/cloud/storage"

	"golang.org/x/net/context"

	"bitbucket.org/ctessum/aqhealth"
	"github.com/ctessum/geom"
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
	Geom                       geom.T             // Cell geometry
	WebMapGeom                 geom.T             // Cell geometry in web map (mercator) coordinate system
	UPlusSpeed                 float64            `desc:"Westerly wind speed" units:"m/s"`
	UMinusSpeed                float64            `desc:"Easterly wind speed" units:"m/s"`
	VPlusSpeed                 float64            `desc:"Southerly wind speed" units:"m/s"`
	VMinusSpeed                float64            `desc:"Northerly wind speed" units:"m/s"`
	WPlusSpeed, WMinusSpeed    float64            `desc:"Upwardly wind speed" units:"m/s"`
	AOrgPartitioning           float64            `desc:"Organic particle partitioning" units:"fraction particles"`
	BOrgPartitioning           float64            // particle fraction
	SPartitioning              float64            `desc:"Sulfur particle partitioning" units:"fraction particles"`
	NOPartitioning             float64            `desc:"Nitrate particle partitioning" units:"fraction particles"`
	NHPartitioning             float64            `desc:"Ammonium particle partitioning" units:"fraction particles"`
	ParticleWetDep             float64            `desc:"Particle wet deposition" units:"1/s"`
	SO2WetDep                  float64            `desc:"SO2 wet deposition" units:"1/s"`
	OtherGasWetDep             float64            `desc:"Wet deposition: other gases" units:"1/s"`
	ParticleDryDep             float64            `desc:"Particle dry deposition" units:"m/s"`
	NH3DryDep                  float64            `desc:"Ammonia dry deposition" units:"m/s"`
	SO2DryDep                  float64            `desc:"SO2 dry deposition" units:"m/s"`
	VOCDryDep                  float64            `desc:"VOC dry deposition" units:"m/s"`
	NOxDryDep                  float64            `desc:"NOx dry deposition" units:"m/s"`
	SO2oxidation               float64            `desc:"SO2 oxidation to SO4 by HO and H2O2" units:"1/s"`
	Kzz                        float64            `desc:"Grid center vertical diffusivity after applying convective fraction" units:"m²/s"`
	KzzAbove, KzzBelow         []float64          // horizontal diffusivity [m2/s] (staggered grid)
	Kxxyy                      float64            `desc:"Grid center horizontal diffusivity" units:"m²/s"`
	KyySouth, KyyNorth         []float64          // horizontal diffusivity [m2/s] (staggered grid)
	KxxWest, KxxEast           []float64          // horizontal diffusivity at [m2/s] (staggered grid)
	M2u                        float64            `desc:"ACM2 upward mixing (Pleim 2007)" units:"1/s"`
	M2d                        float64            `desc:"ACM2 downward mixing (Pleim 2007)" units:"1/s"`
	PopData                    map[string]float64 // Population for multiple demographics [people/grid cell]
	MortalityRate              float64            `desc:"Baseline mortalities rate" units:"Deaths per 100,000 people per year"`
	Dx, Dy, Dz                 float64            // grid size [meters]
	Volume                     float64            `desc:"Cell volume" units:"m³"`
	Row                        int                // master cell index
	Ci                         []float64          // concentrations at beginning of time step [μg/m³]
	Cf                         []float64          // concentrations at end of time step [μg/m³]
	emisFlux                   []float64          // emissions [μg/m³/s]
	West                       []*Cell            // Neighbors to the East
	East                       []*Cell            // Neighbors to the West
	South                      []*Cell            // Neighbors to the South
	North                      []*Cell            // Neighbors to the North
	Below                      []*Cell            // Neighbors below
	Above                      []*Cell            // Neighbors above
	GroundLevel                []*Cell            // Neighbors at ground level
	WestFrac, EastFrac         []float64          // Fraction of cell covered by each neighbor (adds up to 1).
	NorthFrac, SouthFrac       []float64          // Fraction of cell covered by each neighbor (adds up to 1).
	AboveFrac, BelowFrac       []float64          // Fraction of cell covered by each neighbor (adds up to 1).
	GroundLevelFrac            []float64          // Fraction of cell above to each ground level cell (adds up to 1).
	IWest                      []int              // Row indexes of neighbors to the East
	IEast                      []int              // Row indexes of neighbors to the West
	ISouth                     []int              // Row indexes of neighbors to the South
	INorth                     []int              // Row indexes of neighbors to the north
	IBelow                     []int              // Row indexes of neighbors below
	IAbove                     []int              // Row indexes of neighbors above
	IGroundLevel               []int              // Row indexes of neighbors at ground level
	DxPlusHalf                 []float64          // Distance between centers of cell and East [m]
	DxMinusHalf                []float64          // Distance between centers of cell and West [m]
	DyPlusHalf                 []float64          // Distance between centers of cell and North [m]
	DyMinusHalf                []float64          // Distance between centers of cell and South [m]
	DzPlusHalf                 []float64          // Distance between centers of cell and Above [m]
	DzMinusHalf                []float64          // Distance between centers of cell and Below [m]
	Layer                      int                // layer index of grid cell
	Temperature                float64            `desc:"Average temperature" units:"K"`
	WindSpeed                  float64            `desc:"RMS wind speed" units:"m/s"`
	WindSpeedInverse           float64            `desc:"RMS wind speed inverse" units:"(m/s)^(-1)"`
	WindSpeedMinusThird        float64            `desc:"RMS wind speed^(-1/3)" units:"(m/s)^(-1/3)"`
	WindSpeedMinusOnePointFour float64            `desc:"RMS wind speed^(-1.4)" units:"(m/s)^(-1.4)"`
	S1                         float64            `desc:"Stability parameter" units:"?"`
	SClass                     float64            `desc:"Stability class" units:"0=Unstable; 1=Stable"`
	sync.RWMutex                                  // Avoid cell being written by one subroutine and read by another at the same time.
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
		for k := 0; k < nLayers; k++ {
			filename := strings.Replace(fileNameTemplate, "[layer]",
				fmt.Sprintf("%v", k), -1)
			rc, err := storage.NewReader(ctx, bucket, filename)
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
func InitInMAPdata(option InitOption, numIterations int,
	httpPort string) (*InMAPdata, error) {
	d := new(InMAPdata)
	d.NumIterations = numIterations
	err := option(d)
	if err != nil {
		return nil, err
	}
	d.westBoundary = make([]*Cell, 0, 200)
	d.eastBoundary = make([]*Cell, 0, 200)
	d.southBoundary = make([]*Cell, 0, 200)
	d.northBoundary = make([]*Cell, 0, 200)
	d.topBoundary = make([]*Cell, 0, 200)
	nprocs := runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	wg.Add(nprocs)
	for procNum := 0; procNum < nprocs; procNum++ {
		go func(procNum int) {
			for ii := procNum; ii < len(d.Data); ii += nprocs {
				cell := d.Data[ii]
				// Link cells to neighbors and/or boundaries.
				if len(cell.IWest) == 0 {
					c := cell.makecopy()
					cell.West = []*Cell{c}
					d.westBoundary = append(d.westBoundary, c)
				} else {
					cell.West = make([]*Cell, len(cell.IWest))
					for i, row := range cell.IWest {
						cell.West[i] = d.Data[row]
					}
					cell.IWest = nil
				}
				if len(cell.IEast) == 0 {
					c := cell.makecopy()
					cell.East = []*Cell{c}
					d.eastBoundary = append(d.eastBoundary, c)
				} else {
					cell.East = make([]*Cell, len(cell.IEast))
					for i, row := range cell.IEast {
						cell.East[i] = d.Data[row]
					}
					cell.IEast = nil
				}
				if len(cell.ISouth) == 0 {
					c := cell.makecopy()
					cell.South = []*Cell{c}
					d.southBoundary = append(d.southBoundary, c)
				} else {
					cell.South = make([]*Cell, len(cell.ISouth))
					for i, row := range cell.ISouth {
						cell.South[i] = d.Data[row]
					}
					cell.ISouth = nil
				}
				if len(cell.INorth) == 0 {
					c := cell.makecopy()
					cell.North = []*Cell{c}
					d.northBoundary = append(d.northBoundary, c)
				} else {
					cell.North = make([]*Cell, len(cell.INorth))
					for i, row := range cell.INorth {
						cell.North[i] = d.Data[row]
					}
					cell.INorth = nil
				}
				if len(cell.IAbove) == 0 || cell.Layer == d.Nlayers-1 {
					c := cell.makecopy()
					cell.Above = []*Cell{c}
					d.topBoundary = append(d.topBoundary, c)
				} else {
					cell.Above = make([]*Cell, len(cell.IAbove))
					for i, row := range cell.IAbove {
						cell.Above[i] = d.Data[row]
					}
					cell.IAbove = nil
				}
				if cell.Layer != 0 {
					cell.Below = make([]*Cell, len(cell.IBelow))
					cell.GroundLevel = make([]*Cell, len(cell.IGroundLevel))
					for i, row := range cell.IBelow {
						cell.Below[i] = d.Data[row]
					}
					for i, row := range cell.IGroundLevel {
						cell.GroundLevel[i] = d.Data[row]
					}
					cell.IBelow = nil
					cell.IGroundLevel = nil
				} else { // assume bottom boundary is the same as lowest layer.
					cell.Below = []*Cell{d.Data[cell.Row]}
					cell.GroundLevel = []*Cell{d.Data[cell.Row]}
				}
				cell.neighborInfo()
			}
			wg.Done()
		}(procNum)
	}
	wg.Wait()
	d.setTstepCFL() // Set time step
	//d.setTstepRuleOfThumb() // Set time step
	if httpPort != "" {
		go d.WebServer(httpPort)
	}
	return d, nil
}

// Calculate center-to-center cell distance,
// fractions of grid cell covered by each neighbor
// and harmonic mean staggered-grid diffusivities.
func (cell *Cell) neighborInfo() {
	cell.DxPlusHalf = make([]float64, len(cell.East))
	cell.EastFrac = make([]float64, len(cell.East))
	cell.KxxEast = make([]float64, len(cell.East))
	for i, c := range cell.East {
		cell.DxPlusHalf[i] = (cell.Dx + c.Dx) / 2.
		cell.EastFrac[i] = min(c.Dy/cell.Dy, 1.)
		cell.KxxEast[i] = harmonicMean(cell.Kxxyy, c.Kxxyy)
	}
	cell.DxMinusHalf = make([]float64, len(cell.West))
	cell.WestFrac = make([]float64, len(cell.West))
	cell.KxxWest = make([]float64, len(cell.West))
	for i, c := range cell.West {
		cell.DxMinusHalf[i] = (cell.Dx + c.Dx) / 2.
		cell.WestFrac[i] = min(c.Dy/cell.Dy, 1.)
		cell.KxxWest[i] = harmonicMean(cell.Kxxyy, c.Kxxyy)
	}
	cell.DyPlusHalf = make([]float64, len(cell.North))
	cell.NorthFrac = make([]float64, len(cell.North))
	cell.KyyNorth = make([]float64, len(cell.North))
	for i, c := range cell.North {
		cell.DyPlusHalf[i] = (cell.Dy + c.Dy) / 2.
		cell.NorthFrac[i] = min(c.Dx/cell.Dx, 1.)
		cell.KyyNorth[i] = harmonicMean(cell.Kxxyy, c.Kxxyy)
	}
	cell.DyMinusHalf = make([]float64, len(cell.South))
	cell.SouthFrac = make([]float64, len(cell.South))
	cell.KyySouth = make([]float64, len(cell.South))
	for i, c := range cell.South {
		cell.DyMinusHalf[i] = (cell.Dy + c.Dy) / 2.
		cell.SouthFrac[i] = min(c.Dx/cell.Dx, 1.)
		cell.KyySouth[i] = harmonicMean(cell.Kxxyy, c.Kxxyy)
	}
	cell.DzPlusHalf = make([]float64, len(cell.Above))
	cell.AboveFrac = make([]float64, len(cell.Above))
	cell.KzzAbove = make([]float64, len(cell.Above))
	for i, c := range cell.Above {
		cell.DzPlusHalf[i] = (cell.Dz + c.Dz) / 2.
		cell.AboveFrac[i] = min((c.Dx*c.Dy)/(cell.Dx*cell.Dy), 1.)
		cell.KzzAbove[i] = harmonicMean(cell.Kzz, c.Kzz)
	}
	cell.DzMinusHalf = make([]float64, len(cell.Below))
	cell.BelowFrac = make([]float64, len(cell.Below))
	cell.KzzBelow = make([]float64, len(cell.Below))
	for i, c := range cell.Below {
		cell.DzMinusHalf[i] = (cell.Dz + c.Dz) / 2.
		cell.BelowFrac[i] = min((c.Dx*c.Dy)/(cell.Dx*cell.Dy), 1.)
		cell.KzzBelow[i] = harmonicMean(cell.Kzz, c.Kzz)
	}
	cell.GroundLevelFrac = make([]float64, len(cell.GroundLevel))
	for i, c := range cell.GroundLevel {
		cell.GroundLevelFrac[i] = min((c.Dx*c.Dy)/(cell.Dx*cell.Dy), 1.)
	}
}

// Add in emissions flux to each cell at every time step, also
// set initial concentrations to final concentrations from previous
// time step, and set old velocities to velocities from previous time
// step.
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
			max(c.UPlusSpeed/c.Dx, c.UMinusSpeed/c.Dx,
				c.VPlusSpeed/c.Dy, c.VMinusSpeed/c.Dy,
				c.WPlusSpeed/c.Dz, c.WMinusSpeed/c.Dz)
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
	d.Dt /= advectionFactor
}

//  Set the time step using the WRF rule of thumb.
func (d *InMAPdata) setTstepRuleOfThumb() {
	d.Dt = d.Data[0].Dx / 1000. * 6
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
