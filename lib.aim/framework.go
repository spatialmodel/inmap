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
	Data             []*AIMcell   // One data holder for each grid cell
	Nx, Ny, Nz       int          // number of meteorology bins
	arrayLock        sync.RWMutex // Avoid concentration arrays being written by one subroutine and read by another at the same time.
	Dt               float64      // seconds
	vs               float64      // Settling velocity [m/s]
	VOCoxidationRate float64      // VOC oxidation rate constant
	westBoundary     []*AIMcell   // boundary cells
	eastBoundary     []*AIMcell   // boundary cells
	northBoundary    []*AIMcell   // boundary cells
	southBoundary    []*AIMcell   // boundary cells
	topBoundary      []*AIMcell   // boundary cells; assume bottom boundary is the same as lowest layer
}

// Data for a single grid cell
type AIMcell struct {
	uPlusSpeed, uMinusSpeed        float64   // [m/s]
	vPlusSpeed, vMinusSpeed        float64   // [m/s]
	wPlusSpeed, wMinusSpeed        float64   // [m/s]
	orgPartitioning, SPartitioning float64   // gaseous fraction
	NOPartitioning, NHPartitioning float64   // gaseous fraction
	wdParticle, wdSO2, wdOtherGas  float64   // wet deposition rate [1/s]
	particleDryDep                 float64   // aerosol dry deposition velocity [m/s]
	SO2oxidation                   float64   // SO2 oxidation to SO4 by HO [1/s]
	Kz                             float64   // vertical diffusivity [m2/s]
	KyySouth                       float64   // horizontal diffusivity at south edge [m2/s] (staggered grid)
	KxxWest                        float64   // horizontal diffusivity at west edge [m2/s]
	M2u                            float64   // ACM2 upward mixing (Pleim 2007) [1/s]
	M2d                            float64   // ACM2 downward mixing (Pleim 2007) [1/s]
	kPblTop                        float64   // k index of boundary layer top
	Dx, Dy, Dz                     float64   // grid size [meters]
	Volume                         float64   // [cubic meters]
	k, j, i                        int       // cell indicies
	ii                             int       // master cell index
	Ci                             []float64 // concentrations at beginning of time step [μg/m3]
	Cˣ, Cˣˣ                        []float64 // concentrations after first and second Runge-Kutta passes [μg/m3]
	Cf                             []float64 // concentrations at end of time step [μg/m3]
	Csum                           []float64 // sum of concentrations over time for later averaging [μg/m3]
	Cbackground                    []float64 // background pollutant concentrations (not associated with the current simulation) [μg/m3]
	emisFlux                       []float64 //  emissions [μg/m3/s]
	West                           *AIMcell  // Neighbor to the East
	East                           *AIMcell  // Neighbor to the West
	South                          *AIMcell  // Neighbor to the South
	North                          *AIMcell  // Neighbor to the North
	Below                          *AIMcell  // Neighbor below
	Above                          *AIMcell  // Neighbor above
	GroundLevel                    *AIMcell  // Neighbor at ground level
	dzPlusHalf                     float64   // Distance between centers of cell and Above [m]
	dzMinusHalf                    float64   // Distance between centers of cell and Below [m]
	nextToEdge                     bool      // Is the grid cell next to the edge?
	twoFromEdge                    bool      // Is the grid cell 2 cells from the edge?
}

func newAIMcell(dx, dy, dz float64) *AIMcell {
	c := new(AIMcell)
	c.Dx, c.Dy, c.Dz = dx, dy, dz
	c.Volume = dx * dy * dz
	c.Ci = make([]float64, len(polNames))
	c.Cf = make([]float64, len(polNames))
	c.Cˣ = make([]float64, len(polNames))
	c.Cˣˣ = make([]float64, len(polNames))
	c.Csum = make([]float64, len(polNames))
	c.Cbackground = make([]float64, len(polNames))
	c.emisFlux = make([]float64, len(polNames))
	return c
}

// Initialize the model, where `filename` is the path to
// the NetCDF file with meteorology and background concentration data,
// and `httpPort` is the port number for hosting the html GUI.
func InitAIMdata(filename string, httpPort string) *AIMdata {
	d := new(AIMdata)
	d.arrayLock.Lock()
	go d.WebServer(httpPort)
	ff, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer ff.Close()
	f, err := cdf.Open(ff)
	if err != nil {
		panic(err)
	}
	dims := f.Header.Lengths("orgPartitioning")
	d.Nz = dims[0]
	d.Ny = dims[1]
	d.Nx = dims[2]
	dx, dy := 12000., 12000. // need to make these adjustable
	d.VOCoxidationRate = f.Header.GetAttribute("", "VOCoxidationRate").([]float64)[0]
	var wg sync.WaitGroup
	wg.Add(29) // Number of readNCF functions to run simultaneously
	layerHeights := sparse.ZerosDense(d.Nz+1, d.Ny, d.Nx)
	readNCF(filename, &wg, "layerHeights", layerHeights)
	// set up data holders
	d.Data = make([]*AIMcell, d.Nz*d.Ny*d.Nx)
	ii := 0
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			for i := 0; i < d.Nx; i++ {
				// calculate Dz (varies by layer)
				dz := layerHeights.Get(k+1, j, i) - layerHeights.Get(k, j, i)
				d.Data[ii] = newAIMcell(dx, dy, dz)
				d.Data[ii].k = k
				d.Data[ii].j = j
				d.Data[ii].i = i
				d.Data[ii].ii = ii
				ii++
			}
		}
	}
	d.arrayLock.Unlock()
	// set up boundary data holders
	d.westBoundary = make([]*AIMcell, d.Nz*d.Ny)
	ii = 0
	i := 0
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			d.westBoundary[ii] = newAIMcell(dx, dy, 0.)
			d.westBoundary[ii].k = k
			d.westBoundary[ii].j = j
			d.westBoundary[ii].i = i
			d.westBoundary[ii].ii = ii
			d.westBoundary[ii].nextToEdge = true
			ii++
		}
	}
	d.eastBoundary = make([]*AIMcell, d.Nz*d.Ny)
	ii = 0
	i = d.Nx
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			d.eastBoundary[ii] = newAIMcell(dx, dy, 0.)
			d.eastBoundary[ii].k = k
			d.eastBoundary[ii].j = j
			d.eastBoundary[ii].i = i
			d.eastBoundary[ii].ii = ii
			d.eastBoundary[ii].nextToEdge = true
			ii++
		}
	}
	d.southBoundary = make([]*AIMcell, d.Nz*d.Nx)
	ii = 0
	j := 0
	for k := 0; k < d.Nz; k++ {
		for i := 0; i < d.Nx; i++ {
			d.southBoundary[ii] = newAIMcell(dx, dy, 0.)
			d.southBoundary[ii].k = k
			d.southBoundary[ii].j = j
			d.southBoundary[ii].i = i
			d.southBoundary[ii].ii = ii
			d.southBoundary[ii].nextToEdge = true
			ii++
		}
	}
	d.northBoundary = make([]*AIMcell, d.Nz*d.Nx)
	ii = 0
	j = d.Ny
	for k := 0; k < d.Nz; k++ {
		for i := 0; i < d.Nx; i++ {
			d.northBoundary[ii] = newAIMcell(dx, dy, 0.)
			d.northBoundary[ii].k = k
			d.northBoundary[ii].j = j
			d.northBoundary[ii].i = i
			d.northBoundary[ii].ii = ii
			d.northBoundary[ii].nextToEdge = true
			ii++
		}
	}
	d.topBoundary = make([]*AIMcell, d.Ny*d.Nx)
	ii = 0
	k := d.Nz
	for j := 0; j < d.Ny; j++ {
		for i := 0; i < d.Nx; i++ {
			d.topBoundary[ii] = newAIMcell(dx, dy, 0.)
			d.topBoundary[ii].k = k
			d.topBoundary[ii].j = j
			d.topBoundary[ii].i = i
			d.topBoundary[ii].ii = ii
			d.topBoundary[ii].nextToEdge = true
			ii++
		}
	}

	d.arrayLock.Lock()
	go d.readNCF(filename, &wg, "uPlusSpeed")
	go d.readNCF(filename, &wg, "uMinusSpeed")
	go d.readNCF(filename, &wg, "vPlusSpeed")
	go d.readNCF(filename, &wg, "vMinusSpeed")
	go d.readNCF(filename, &wg, "wPlusSpeed")
	go d.readNCF(filename, &wg, "wMinusSpeed")
	go d.readNCF(filename, &wg, "orgPartitioning")
	go d.readNCF(filename, &wg, "VOC")
	go d.readNCF(filename, &wg, "SOA")
	go d.readNCF(filename, &wg, "SPartitioning")
	go d.readNCF(filename, &wg, "gS")
	go d.readNCF(filename, &wg, "pS")
	go d.readNCF(filename, &wg, "NOPartitioning")
	go d.readNCF(filename, &wg, "gNO")
	go d.readNCF(filename, &wg, "pNO")
	go d.readNCF(filename, &wg, "NHPartitioning")
	go d.readNCF(filename, &wg, "gNH")
	go d.readNCF(filename, &wg, "pNH")
	go d.readNCF(filename, &wg, "wdParticle")
	go d.readNCF(filename, &wg, "wdSO2")
	go d.readNCF(filename, &wg, "wdOtherGas")
	go d.readNCF(filename, &wg, "Kz")
	go d.readNCF(filename, &wg, "M2u")
	go d.readNCF(filename, &wg, "M2d")
	go d.readNCF(filename, &wg, "pblTopLayer")
	go d.readNCF(filename, &wg, "SO2oxidation")
	go d.readNCF(filename, &wg, "particleDryDep")
	go d.readNCF(filename, &wg, "Kyy")
	wg.Wait()
	d.arrayLock.Unlock()

	// Set up links to neighbors
	ii = 0
	var jj int
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			for i := 0; i < d.Nx; i++ {
				if i == 0 {
					d.Data[ii].West = d.westBoundary[k*d.Ny+j]
				} else {
					jj = d.getIndex(k, j, i-1)
					d.Data[jj].checkIndicies(k, j, i-1)
					d.Data[ii].West = d.Data[jj]
				}
				if i == d.Nx-1 {
					d.Data[ii].East = d.eastBoundary[k*d.Ny+j]
					// Since we have converted from unstaggered to staggered
					// grid for Kxx, fill in final value for Kxx.
					d.Data[ii].East.KxxWest = d.Data[ii].KxxWest
				} else {
					jj = d.getIndex(k, j, i+1)
					d.Data[jj].checkIndicies(k, j, i+1)
					d.Data[ii].East = d.Data[jj]
				}
				if j == 0 {
					d.Data[ii].South = d.southBoundary[k*d.Nx+i]
				} else {
					jj = d.getIndex(k, j-1, i)
					d.Data[jj].checkIndicies(k, j-1, i)
					d.Data[ii].South = d.Data[jj]
				}
				if j == d.Ny-1 {
					d.Data[ii].North = d.northBoundary[k*d.Nx+i]
					// Since we have converted from unstaggered to staggered
					// grid for Kxx, fill in final value for Kxx.
					d.Data[ii].North.KyySouth = d.Data[ii].KyySouth
				} else {
					jj = d.getIndex(k, j+1, i)
					d.Data[jj].checkIndicies(k, j+1, i)
					d.Data[ii].North = d.Data[jj]
				}
				if k == 0 {
					d.Data[ii].Below = d.Data[ii] // assume bottom boundary is the same as lowest layer.
				} else {
					jj = d.getIndex(k-1, j, i)
					d.Data[jj].checkIndicies(k-1, j, i)
					d.Data[ii].Below = d.Data[jj]
				}
				if k == d.Nz-1 {
					d.Data[ii].Above = d.topBoundary[j*d.Nx+i]
				} else {
					jj = d.getIndex(k+1, j, i)
					d.Data[jj].checkIndicies(k+1, j, i)
					d.Data[ii].Above = d.Data[jj]
				}
				if k == 0 || k == d.Nz-1 || j == 0 || j == d.Ny-1 ||
					i == 0 || i == d.Nx-1 {
					d.Data[ii].nextToEdge = true
				}
				if k == 1 || k == d.Nz-2 || j == 1 || j == d.Ny-2 ||
					i == 1 || i == d.Nx-2 {
					d.Data[ii].twoFromEdge = true
				}
				jj = d.getIndex(0, j, i)
				d.Data[jj].checkIndicies(0, j, i)
				d.Data[ii].GroundLevel = d.Data[jj]

				d.Data[ii].dzPlusHalf = (d.Data[ii].Dz +
					d.Data[ii].Above.Dz) / 2.
				d.Data[ii].dzMinusHalf = (d.Data[ii].Dz +
					d.Data[ii].Below.Dz) / 2.
				ii++
			}
		}
	}
	d.SettlingVelocity()
	return d
}

// convert 3d index to 1d index
func (d *AIMdata) getIndex(k, j, i int) int {
	return k*d.Ny*d.Nx + j*d.Nx + i
}
func (c *AIMcell) checkIndicies(k, j, i int) {
	if k != c.k || j != c.j || k != c.k {
		panic(fmt.Sprintf("Expected indicies (%v,%v,%v) do not match actual "+
			"indicies (%v,%v,%v). Master index=%v.\n", k, j, i, c.k, c.j, c.i, c.ii))
	}
}

func interpolate(random float32, freqs, bins []float32, b int) (val float64) {
	x := freqs[b+1] - freqs[b]
	if x == 0. {
		val = float64(bins[b])
	} else {
		frac := (random - freqs[b]) / (x)
		val = float64(bins[b] + (bins[b+1]-bins[b])*frac)
	}
	return
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

// Add current concentration to sum for later averaging
var addtosum = func(c *AIMcell, d *AIMdata) {
	for i, _ := range polNames {
		c.Csum[i] += c.Cf[i]
	}
}

//  Set the time step using the Courant–Friedrichs–Lewy (CFL) condition.
func (d *AIMdata) setTstepCFL(nprocs int) {
	const Cmax = 1
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
				case "wdParticle":
					d.Data[ii].wdParticle = float64(dat[index])
				case "wdSO2":
					d.Data[ii].wdSO2 = float64(dat[index])
				case "wdOtherGas":
					d.Data[ii].wdOtherGas = float64(dat[index])
				case "Kz":
					d.Data[ii].Kz = float64(dat[index])
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

func getNCFbuffer(filename string, Var string) []float32 {
	ff, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	f, err := cdf.Open(ff)
	if err != nil {
		panic(err)
	}
	vars := f.Header.Variables()
	sort.Strings(vars)
	if i := sort.SearchStrings(vars, Var); vars[i] != Var {
		panic(fmt.Sprintf("Variable %v is not in input data file", Var))
	}
	dims := f.Header.Lengths(Var)
	defer ff.Close()
	nread := 1
	for _, dim := range dims {
		nread *= dim
	}
	r := f.Reader(Var, nil, nil)
	buf := r.Zero(nread)
	_, err = r.Read(buf)
	if err != nil {
		panic(err)
	}
	return buf.([]float32)
}
