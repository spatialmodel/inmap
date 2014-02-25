package aim

import (
	"bitbucket.org/ctessum/sparse"
	"code.google.com/p/lvd.go/cdf"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
)

type AIMdata struct {
	Data             []*AIMcell // One data holder for each grid cell
	Dt               float64    // seconds
	vs               float64    // Settling velocity [m/s]
	VOCoxidationRate float64    // VOC oxidation rate constant
	westBoundary     []*AIMcell // boundary cells
	eastBoundary     []*AIMcell // boundary cells
	northBoundary    []*AIMcell // boundary cells
	southBoundary    []*AIMcell // boundary cells
	topBoundary      []*AIMcell // boundary cells; assume bottom boundary is the same as lowest layer
}

// Data for a single grid cell
type AIMcell struct {
	geom                           geom.T       // Cell geometry
	UPlusSpeed, UMinusSpeed        float64      // [m/s]
	VPlusSpeed, VMinusSpeed        float64      // [m/s]
	WPlusSpeed, WMinusSpeed        float64      // [m/s]
	OrgPartitioning, SPartitioning float64      // gaseous fraction
	NOPartitioning, NHPartitioning float64      // gaseous fraction
	ParticleWetDep, SO2WetDep      float64      // wet deposition rate [1/s]
	OtherGasWetDep                 float64      // wet deposition rate [1/s]
	ParticleDryDep, NH3DryDep      float64      // Dry deposition velocities [m/s]
	SO2DryDep, VOCDryDep           float64      // Dry deposition velocities [m/s]
	NOxDryDep                      float64      // Dry deposition velocities [m/s]
	SO2oxidation                   float64      // SO2 oxidation to SO4 by HO [1/s]
	Kzz                            float64      // vertical diffusivity at bottom edge [m2/s]
	Kyyxx                          float64      // unstaggered horizontal diffusivity [m2/s]
	KyySouth                       float64      // horizontal diffusivity at south edge [m2/s] (staggered grid)
	KxxWest                        float64      // horizontal diffusivity at west edge [m2/s]
	M2u                            float64      // ACM2 upward mixing (Pleim 2007) [1/s]
	M2d                            float64      // ACM2 downward mixing (Pleim 2007) [1/s]
	PblTopLayer                    float64      // k index of boundary layer top
	Dx, Dy, Dz                     float64      // grid size [meters]
	Volume                         float64      // [cubic meters]
	Row                            int          // master cell index
	Ci                             []float64    // concentrations at beginning of time step [μg/m3]
	Cˣ, Cˣˣ                        []float64    // concentrations after first and second Runge-Kutta passes [μg/m3]
	Cf                             []float64    // concentrations at end of time step [μg/m3]
	emisFlux                       []float64    //  emissions [μg/m3/s]
	West                           []*AIMcell   // Neighbors to the East
	East                           []*AIMcell   // Neighbors to the West
	South                          []*AIMcell   // Neighbors to the South
	North                          []*AIMcell   // Neighbors to the North
	Below                          []*AIMcell   // Neighbors below
	Above                          []*AIMcell   // Neighbors above
	GroundLevel                    []*AIMcell   // Neighbors at ground level
	iWest                          []int        // Row indexes of neighbors to the East
	iEast                          []int        // Row indexes of neighbors to the West
	iSouth                         []int        // Row indexes of neighbors to the South
	iNorth                         []int        // Row indexes of neighbors to the north
	iBelow                         []int        // Row indexes of neighbors below
	iAbove                         []int        // Row indexes of neighbors above
	iGroundLevel                   []int        // Row indexes of neighbors at ground level
	dxPlusHalf                     []float64    // Distance between centers of cell and East [m]
	dxMinusHalf                    []float64    // Distance between centers of cell and West [m]
	dyPlusHalf                     []float64    // Distance between centers of cell and North [m]
	dyMinusHalf                    []float64    // Distance between centers of cell and South [m]
	dzPlusHalf                     []float64    // Distance between centers of cell and Above [m]
	dzMinusHalf                    []float64    // Distance between centers of cell and Below [m]
	nextToEdge                     bool         // Is the grid cell next to the edge?
	Layer                          int          // layer index of grid cell
	lock                           sync.RWMutex // Avoid cell being written by one subroutine and read by another at the same time.
}

func (c *AIMcell) prepare() {
	c := new(AIMcell)
	c.Volume = c.Dx * c.Dy * c.Dz
	c.Ci = make([]float64, len(polNames))
	c.Cf = make([]float64, len(polNames))
	c.Cˣ = make([]float64, len(polNames))
	c.Cˣˣ = make([]float64, len(polNames))
	c.emisFlux = make([]float64, len(polNames))
}

func (c *AIMcell) makecopy() *AIMcell {
	c2 := new(AIMcell)
	c2.Dx, c2.Dy, cd.Dz = c.Dx, c.Dy, c.Dz
	c2.prepare()
	return c2
}

// Initialize the model, where `filename` is the path to
// the GeoJSON files with meteorology and background concentration data
// (where `[layer]` is a stand-in for the layer number),
// `nLayers` is the number of vertical layers in the model,
// and `httpPort` is the port number for hosting the html GUI.
func InitAIMdata(filename string, nLayers int, httpPort string) *AIMdata {
	go d.WebServer(httpPort)

	type dataHolder struct {
		Type       string
		Geometry   *geojson.Geometry
		Properties *AIMcell
	}
	type dataHolderHolder struct {
		Proj4, Type string
		Features    []*dataHolder
	}
	inputData := make([]*dataHolderHolder, nLayers)
	ncells := 0
	for k := 0; k < nLayers; k++ {
		f, err := os.Open(filename)
		if err != nil {
			panic(err)
		}
		var d dataHolderHolder
		err = json.Unmarshal(f, &dataHolderHolder)
		inputData[k] = &d
		ncells += len(d.Features)
		f.Close()
	}
	// set up data holders
	d := new(AIMdata)
	d.Data = make([]*AIMcell, ncells)
	for _, indata := range inputData {
		for _, c := range indata {
			c.prepare()
			d.Data[c.Row] = c
		}
	}
	d.westBoundary = make([]*AIMcell, 0)
	d.eastBoundary = make([]*AIMcell, 0)
	d.southBoundary = make([]*AIMcell, 0)
	d.northBoundary = make([]*AIMcell, 0)
	d.topBoundary = make([]*AIMcell, 0)
	for _, cell := range d.Data {
		if len(cell.iWest) == 0 {
			c := cell.makecopy()
			c.nextToEdge = true
			cell.nextToEdge = true
			cell.West = []*AIMcell{c}
			d.westBoundary = append(d.westBoundary, c)
		} else {
			cell.West = make([]*AIMcell, len(cell.iWest))
			for i, row := range cell.iWest {
				cell.West[i] = d.Data[row]
			}
			cell.iWest = nil
		}
		if len(cell.iEast) == 0 {
			c := cell.makecopy()
			c.nextToEdge = true
			// Since we have converted from unstaggered to staggered
			// grid for Kxx, fill in final value for Kxx.
			c.KxxWest = cell.Kyyxx
			cell.nextToEdge = true
			cell.East = []*AIMcell{c}
			d.eastBoundary = append(d.eastBoundary, c)
		} else {
			cell.East = make([]*AIMcell, len(cell.iEast))
			for i, row := range cell.iEast {
				cell.East[i] = d.Data[row]
			}
			cell.iEast = nil
		}
		if len(cell.iSouth) == 0 {
			c := cell.makecopy()
			c.nextToEdge = true
			cell.nextToEdge = true
			cell.South = []*AIMcell{c}
			d.southBoundary = append(d.southBoundary, c)
		} else {
			cell.South = make([]*AIMcell, len(cell.iSouth))
			for i, row := range cell.iSouth {
				cell.South[i] = d.Data[row]
			}
			cell.iSouth = nil
		}
		if len(cell.iNorth) == 0 {
			c := cell.makecopy()
			c.nextToEdge = true
			// Since we have converted from unstaggered to staggered
			// grid for Kyy, fill in final value for Kyy.
			c.KyySouth = cell.Kyyxx
			cell.nextToEdge = true
			cell.North = []*AIMcell{c}
			d.northBoundary = append(d.northBoundary, c)
		} else {
			cell.North = make([]*AIMcell, len(cell.iNorth))
			for i, row := range cell.iNorth {
				cell.North[i] = d.Data[row]
			}
			cell.iNorth = nil
		}
		if len(cell.iAbove) == 0 {
			c := cell.makecopy()
			c.nextToEdge = true
			cell.nextToEdge = true
			cell.Above = []*AIMcell{c}
			d.topBoundary = append(d.topBoundary, c)
		} else {
			cell.Above = make([]*AIMcell, len(cell.iAbove))
			for i, row := range cell.iAbove {
				cell.Above[i] = d.Data[row]
			}
			cell.iAbove = nil
		}
		if cell.Layer != 0 {
			cell.Below = make([]*AIMcell, len(cell.iBelow))
			cell.GroundLevel = make([]*AIMcell, len(cell.iGroundLevel))
			for i, row := range cell.iBelow {
				cell.Below[i] = d.Data[row]
			}
			for i, row := range cell.iGroundLevel {
				cell.GroundLevel[i] = d.Data[row]
			}
			cell.iBelow = nil
			cell.iGroundLevel = nil
		} else { // assume bottom boundary is the same as lowest layer.
			cell.Below = []*AIMcell{d.Data[cell.Row]}
			cell.GroundLevel = []*AIMcell{d.Data[cell.Row]}
		}

		// Calculate center-to-center cell distance
		cell.dxPlusHalf = make([]float64, len(cell.East))
		for i, c := range cell.East {
			cell.dxPlusHalf[i] = (cell.Dx + c.Dx) / 2.
		}
		cell.dxMinusHalf = make([]float64, len(cell.West))
		for i, c := range cell.West {
			cell.dxMinusHalf[i] = (cell.Dx + c.Dx) / 2.
		}
		cell.dyPlusHalf = make([]float64, len(cell.North))
		for i, c := range cell.Above {
			cell.dyPlusHalf[i] = (cell.Dy + c.Dy) / 2.
		}
		cell.dyMinusHalf = make([]float64, len(cell.South))
		for i, c := range cell.Below {
			cell.dyMinusHalf[i] = (cell.Dy + c.Dy) / 2.
		}
		cell.dzPlusHalf = make([]float64, len(cell.Above))
		for i, c := range cell.Above {
			cell.dzPlusHalf[i] = (cell.Dz + c.Dz) / 2.
		}
		cell.dzMinusHalf = make([]float64, len(cell.Below))
		for i, c := range cell.Below {
			cell.dzMinusHalf[i] = (cell.Dz + c.Dz) / 2.
		}
	}
	return d
}

// Add in emissions flux to each cell at every time step, also
// set initial concentrations to final concentrations from previous
// time step, and set old velocities to velocities from previous time
// step.
func (c *AIMcell) addEmissionsFlux(d *AIMdata) {
	for i, _ := range polNames {
		c.Cf[i] += c.emisFlux[i] * d.Dt
		c.Ci[i] = c.Cf[i]
	}
}

//  Set the time step using the Courant–Friedrichs–Lewy (CFL) condition.
func (d *AIMdata) setTstepCFL(nprocs int) {
	const Cmax = 1.5 // From Wicker and Skamarock (2002) Table 1.
	valChan := make(chan float64)
	calcCFL := func(procNum int) {
		var thisval, val float64
		var c *AIMcell
		for ii := procNum; ii < len(d.Data); ii += nprocs {
			c = d.Data[ii]
			thisval = max(c.uPlusSpeed/c.Dx, c.uMinusSpeed/c.Dx,
				c.vPlusSpeed/c.Dy, c.vMinusSpeed/c.Dy,
				c.wPlusSpeed/c.Dz, c.wMinusSpeed/c.Dz)
			if thisval > val {
				val = thisval
			}
		}
		valChan <- val
	}
	for procNum := 0; procNum < nprocs; procNum++ {
		go calcCFL(procNum)
	}
	val := 0.
	for i := 0; i < nprocs; i++ { // get max value from each processor
		procval := <-valChan
		if procval > val {
			val = procval
		}
	}
	d.Dt = Cmax / math.Pow(3., 0.5) / val // seconds
}

//  Set the time step using the WRF rule of thumb.
func (d *AIMdata) setTstepRuleOfThumb() {
	d.Dt = d.Data[0].Dx / 1000. * 6
}

// Read variable from NetCDF file.
func (d *AIMdata) readNCF(filename string, wg *sync.WaitGroup, Var string) {
	defer wg.Done()
	dat := getNCFbuffer(filename, Var)
	kstride := d.Ny * d.Nx
	jstride := d.Nx
	ii := 0
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			for i := 0; i < d.Nx; i++ {
				index := k*kstride + j*jstride + i
				switch Var {
				case "uPlusSpeed":
					d.Data[ii].uPlusSpeed = float64(dat[index])
				case "uMinusSpeed":
					d.Data[ii].uMinusSpeed = float64(dat[index])
				case "vPlusSpeed":
					d.Data[ii].vPlusSpeed = float64(dat[index])
				case "vMinusSpeed":
					d.Data[ii].vMinusSpeed = float64(dat[index])
				case "wPlusSpeed":
					d.Data[ii].wPlusSpeed = float64(dat[index])
				case "wMinusSpeed":
					d.Data[ii].wMinusSpeed = float64(dat[index])
				case "orgPartitioning":
					d.Data[ii].orgPartitioning = float64(dat[index])
				case "SPartitioning":
					d.Data[ii].SPartitioning = float64(dat[index])
				case "NOPartitioning":
					d.Data[ii].NOPartitioning = float64(dat[index])
				case "NHPartitioning":
					d.Data[ii].NHPartitioning = float64(dat[index])
				case "VOC":
					d.Data[ii].Cbackground[igOrg] = float64(dat[index])
				case "SOA":
					d.Data[ii].Cbackground[ipOrg] = float64(dat[index])
				case "gNO":
					d.Data[ii].Cbackground[igNO] = float64(dat[index])
				case "pNO":
					d.Data[ii].Cbackground[ipNO] = float64(dat[index])
				case "gNH":
					d.Data[ii].Cbackground[igNH] = float64(dat[index])
				case "pNH":
					d.Data[ii].Cbackground[ipNH] = float64(dat[index])
				case "gS":
					d.Data[ii].Cbackground[igS] = float64(dat[index])
				case "pS":
					d.Data[ii].Cbackground[ipS] = float64(dat[index])
				case "particleWetDep":
					d.Data[ii].particleWetDep = float64(dat[index])
				case "SO2WetDep":
					d.Data[ii].SO2WetDep = float64(dat[index])
				case "otherGasWetDep":
					d.Data[ii].otherGasWetDep = float64(dat[index])
				case "Kzz":
					d.Data[ii].Kzz = float64(dat[index])
				case "M2u": // 2d variable
					index = j*jstride + i
					d.Data[ii].M2u = float64(dat[index])
				case "M2d":
					d.Data[ii].M2d = float64(dat[index])
				case "pblTopLayer": // 2d variable
					index = j*jstride + i
					d.Data[ii].kPblTop = float64(dat[index])
				case "SO2oxidation":
					d.Data[ii].SO2oxidation = float64(dat[index])
				case "particleDryDep": // 2d variable
					index = j*jstride + i
					d.Data[ii].particleDryDep = float64(dat[index])
				case "NH3DryDep": // 2d variable
					index = j*jstride + i
					d.Data[ii].NH3DryDep = float64(dat[index])
				case "NOxDryDep": // 2d variable
					index = j*jstride + i
					d.Data[ii].NOxDryDep = float64(dat[index])
				case "VOCDryDep": // 2d variable
					index = j*jstride + i
					d.Data[ii].VOCDryDep = float64(dat[index])
				case "SO2DryDep": // 2d variable
					index = j*jstride + i
					d.Data[ii].SO2DryDep = float64(dat[index])
				case "Kyy": // convert from unstaggered to staggered
					jminusIndex := k*kstride + (j-1)*jstride + i
					iminusIndex := k*kstride + j*jstride + i - 1
					val := float64(dat[index])
					if iminusIndex >= 0 {
						iminus := float64(dat[iminusIndex])
						if val == 0. || iminus == 0. {
							d.Data[ii].KxxWest = 0.
						} else {
							// calculate harmonic mean between center and west
							// values to get Kxx at grid edge
							d.Data[ii].KxxWest = 2 * val * iminus / (val + iminus)
						}
					} else {
						d.Data[ii].KxxWest = val
					}
					if jminusIndex >= 0 {
						jminus := float64(dat[jminusIndex])
						if val == 0. || jminus == 0. {
							d.Data[ii].KyySouth = 0.
						} else {
							// calculate harmonic mean between center and south
							// values to get Kyy at grid edge
							d.Data[ii].KyySouth = 2 * val * jminus / (val + jminus)
						}
					} else {
						d.Data[ii].KyySouth = val
					}

				default:
					panic(fmt.Sprintf("Variable %v unknown.\n", Var))
				}
				ii++
			}
		}
	}
}
