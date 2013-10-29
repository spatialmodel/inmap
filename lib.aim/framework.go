package aim

import (
	"bitbucket.org/ctessum/sparse"
	"runtime"
	"time"
	"code.google.com/p/lvd.go/cdf"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
)

type AIMdata struct {
	Data                            []*AIMcell    // One data holder for each grid cell
	nbins, Nx, Ny, Nz               int           // number of meteorology bins
	arrayLock                       sync.RWMutex  // Avoid concentration arrays being written by one subroutine and read by another at the same time.
	Dt                              float64       // seconds
	vs                              float64       // Settling velocity, m/s
	VOCoxidationRate                float64       // VOC oxidation rate constant
	UbinsEastEdge, UfreqEastEdge    [][][]float32 // Edge of the Arakawa C-grid
	VbinsNorthEdge, VfreqNorthEdge  [][][]float32 // Edge of the Arakawa C-grid
	WbinsTopEdge, WfreqTopEdge      [][][]float32 // Edge of the Arakawa C-grid
	UeastEdge, VnorthEdge, WtopEdge [][]float64   // Edge of the Arakawa C-grid
	xRandom, yRandom, zRandom       float32
}

// Data for a single grid cell
type AIMcell struct {
	UbinsWest, UfreqWest           []float32 // m/s
	VbinsSouth, VfreqSouth         []float32 // m/s
	WbinsBelow, WfreqBelow         []float32 // m/s
	Uwest, Vsouth, Wbelow          float64
	orgPartitioning, SPartitioning float64           // gaseous fraction
	NOPartitioning, NHPartitioning float64           // gaseous fraction
	wdParticle, wdSO2, wdOtherGas  float64           // wet deposition rate, 1/s
	verticalDiffusivity            float64           // vertical diffusivity, m2/s
	Dx, Dy, Dz                     float64           // meters
	Volume                         float64           // cubic meters
	k, j, i                        int               // cell indicies
	ii                             int               // master index
	initialConc                    []float64         // concentrations at beginning of time step
	finalConc                      []float64         // concentrations at end of time step
	emisFlux                       []float64         //  emissions (μg/m3/s)
	getWestNeighbor                func(int) float64 // takes pollutant index and returns concentration at neighbor
	getEastNeighbor                func(int) float64 // takes pollutant index and returns concentration at neighbor
	getSouthNeighbor               func(int) float64 // takes pollutant index and returns concentration at neighbor
	getNorthNeighbor               func(int) float64 // takes pollutant index and returns concentration at neighbor
	getBelowNeighbor               func(int) float64 // takes pollutant index and returns concentration at neighbor
	getAboveNeighbor               func(int) float64 // takes pollutant index and returns concentration at neighbor
	getUeast                       func() float64    // gets U velocity at East edge
	getVnorth                      func() float64    // gets U velocity at North edge
	getWabove                      func() float64    // gets U velocity at Top edge
}

func newAIMcell(nbins int, dx, dy, dz float64) *AIMcell {
	c := new(AIMcell)
	c.Dx, c.Dy, c.Dz = dx, dy, dz
	c.Volume = dx * dy * dz
	c.UbinsWest = make([]float32, nbins)
	c.VbinsSouth = make([]float32, nbins)
	c.WbinsBelow = make([]float32, nbins)
	c.UfreqWest = make([]float32, nbins)
	c.VfreqSouth = make([]float32, nbins)
	c.WfreqBelow = make([]float32, nbins)
	c.initialConc = make([]float64, len(polNames))
	c.finalConc = make([]float64, len(polNames))
	c.emisFlux = make([]float64, len(polNames))
	return c
}

func InitAIMdata(filename string) *AIMdata {
	d := new(AIMdata)
	go d.WebServer()
	ff, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer ff.Close()
	f, err := cdf.Open(ff)
	if err != nil {
		panic(err)
	}
	d.nbins = f.Header.Lengths("Ubins")[0]
	dims := f.Header.Lengths("orgPartitioning")
	d.Nz = dims[0]
	d.Ny = dims[1]
	d.Nx = dims[2]
	d.VOCoxidationRate = f.Header.GetAttribute("", "VOCoxidationRate").([]float64)[0]
	var wg sync.WaitGroup
	wg.Add(14)
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
				d.Data[ii] = newAIMcell(d.nbins, 12000., 12000., dz)
				d.Data[ii].k = k
				d.Data[ii].j = j
				d.Data[ii].i = i
				d.Data[ii].ii = ii
				ii++
			}
		}
	}
	d.arrayLock.Lock()
	go d.readNCFbins(filename, &wg, "Ubins")
	go d.readNCFbins(filename, &wg, "Vbins")
	go d.readNCFbins(filename, &wg, "Wbins")
	go d.readNCFbins(filename, &wg, "Ufreq")
	go d.readNCFbins(filename, &wg, "Vfreq")
	go d.readNCFbins(filename, &wg, "Wfreq")
	go d.readNCF(filename, &wg, "orgPartitioning")
	go d.readNCF(filename, &wg, "SPartitioning")
	go d.readNCF(filename, &wg, "NOPartitioning")
	go d.readNCF(filename, &wg, "NHPartitioning")
	go d.readNCF(filename, &wg, "wdParticle")
	go d.readNCF(filename, &wg, "wdSO2")
	go d.readNCF(filename, &wg, "wdOtherGas")
	wg.Wait()
	d.arrayLock.Unlock()

	// Set up functions for getting neighboring values
	ii = 0
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			for i := 0; i < d.Nx; i++ {
				d.Data[ii].getUeast = d.getUeastFunc(k, j, i)
				d.Data[ii].getVnorth = d.getVnorthFunc(k, j, i)
				d.Data[ii].getWabove = d.getWaboveFunc(k, j, i)
				d.Data[ii].getWestNeighbor = d.getWestNeighborFunc(k, j, i)
				d.Data[ii].getEastNeighbor = d.getEastNeighborFunc(k, j, i)
				d.Data[ii].getSouthNeighbor = d.getSouthNeighborFunc(k, j, i)
				d.Data[ii].getNorthNeighbor = d.getNorthNeighborFunc(k, j, i)
				d.Data[ii].getBelowNeighbor = d.getBelowNeighborFunc(k, j, i)
				d.Data[ii].getAboveNeighbor = d.getAboveNeighborFunc(k, j, i)
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

// Create functions to get velocities at the East, North, and Top edges of
// the grid cells. These are the same as the West, South, and Bottom edges
// of adjacent cells.
func (d *AIMdata) getUeastFunc(k, j, i int) func() float64 {
	if i == d.Nx-1 {
		return func() float64 {
			return d.UeastEdge[k][j]
		}
	} else {
		ii := d.getIndex(k, j, i+1)
		d.Data[ii].checkIndicies(k, j, i+1)
		return func() float64 {
			return d.Data[ii].Uwest
		}
	}
}
func (d *AIMdata) getVnorthFunc(k, j, i int) func() float64 {
	if j == d.Ny-1 {
		return func() float64 {
			return d.VnorthEdge[k][i]
		}
	} else {
		ii := d.getIndex(k, j+1, i)
		d.Data[ii].checkIndicies(k, j+1, i)
		return func() float64 {
			return d.Data[ii].Vsouth
		}
	}
}
func (d *AIMdata) getWaboveFunc(k, j, i int) func() float64 {
	if k == d.Nz-1 {
		return func() float64 {
			return d.WtopEdge[j][i]
		}
	} else {
		ii := d.getIndex(k+1, j, i)
		d.Data[ii].checkIndicies(k+1, j, i)
		return func() float64 {
			return d.Data[ii].Wbelow
		}
	}
}

// Lower boundary is same as lowest grid cell value.
// All other boundaries = 0.
func getUpperBoundary(_ int) float64 { return 0. }
func getNorthBoundary(_ int) float64 { return 0. }
func getSouthBoundary(_ int) float64 { return 0. }
func getEastBoundary(_ int) float64  { return 0. }
func getWestBoundary(_ int) float64  { return 0. }

// Create functions to get concentrations at neighboring cells.
func (d *AIMdata) getWestNeighborFunc(k, j, i int) func(int) float64 {
	if i == 0 {
		return getWestBoundary
	} else {
		ii := d.getIndex(k, j, i-1)
		d.Data[ii].checkIndicies(k, j, i-1)
		return func(iPol int) float64 {
			return d.Data[ii].initialConc[iPol]
		}
	}
}
func (d *AIMdata) getEastNeighborFunc(k, j, i int) func(int) float64 {
	if i == d.Nx-1 {
		return getEastBoundary
	} else {
		ii := d.getIndex(k, j, i+1)
		d.Data[ii].checkIndicies(k, j, i+1)
		return func(iPol int) float64 {
			return d.Data[ii].initialConc[iPol]
		}
	}
}
func (d *AIMdata) getSouthNeighborFunc(k, j, i int) func(int) float64 {
	if j == 0 {
		return getSouthBoundary
	} else {
		ii := d.getIndex(k, j-1, i)
		d.Data[ii].checkIndicies(k, j-1, i)
		return func(iPol int) float64 {
			return d.Data[ii].initialConc[iPol]
		}
	}
}
func (d *AIMdata) getNorthNeighborFunc(k, j, i int) func(int) float64 {
	if j == d.Ny-1 {
		return getNorthBoundary
	} else {
		ii := d.getIndex(k, j+1, i)
		d.Data[ii].checkIndicies(k, j+1, i)
		return func(iPol int) float64 {
			return d.Data[ii].initialConc[iPol]
		}
	}
}
func (d *AIMdata) getBelowNeighborFunc(k, j, i int) func(int) float64 {
	if k == 0 {
		// Lower boundary is same as lowest grid cell value.
		return func(iPol int) float64 {
			return d.Data[d.getIndex(k, j, i)].initialConc[iPol]
		}
	} else {
		ii := d.getIndex(k-1, j, i)
		d.Data[ii].checkIndicies(k-1, j, i)
		return func(iPol int) float64 {
			return d.Data[ii].initialConc[iPol]
		}
	}
}
func (d *AIMdata) getAboveNeighborFunc(k, j, i int) func(int) float64 {
	if k == d.Nz-1 {
		return getUpperBoundary
	} else {
		ii := d.getIndex(k+1, j, i)
		d.Data[ii].checkIndicies(k+1, j, i)
		return func(iPol int) float64 {
			return d.Data[ii].initialConc[iPol]
		}
	}
}

// choose a bin using a weighted random method
func getbin(bins, freq []float32, random float32) float64 {
	for b := 0; b < len(bins); b++ {
		if random <= freq[b] {
			return float64(bins[b])
		}
	}
	panic(fmt.Sprintf("Could not choose a bin using seed %v "+
		"(max cumulative frequency=%v).", random,
		freq[len(freq)-1]))
	return 0.
}

// Add in emissions, set wind velocities using the weighted
// random walk method, copy initial concentrations to final concentrations,
// and set time step (dt).
func (d *AIMdata) SetupTimeStep() {
	d.arrayLock.Lock()
	iiChan := make(chan int)
	var wg sync.WaitGroup
	wg.Add(runtime.GOMAXPROCS(0))
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go func() {
			defer wg.Done()
			var c *AIMcell
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for ii := range iiChan {
				c = d.Data[ii]
				c.Uwest = getbin(c.UbinsWest, c.UfreqWest, r.Float32())
				c.Vsouth = getbin(c.VbinsSouth, c.VfreqSouth, r.Float32())
				c.Wbelow = getbin(c.WbinsBelow, c.WfreqBelow, r.Float32())
			}
		}()
	}
	for ii := 0; ii < len(d.Data); ii += 1 {
		iiChan <- ii
	}
	close(iiChan)
	wg.Wait()

			r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			d.UeastEdge[k][j] = getbin(d.UbinsEastEdge[k][j],
				d.UfreqEastEdge[k][j], r.Float32())
		}
	}
	for k := 0; k < d.Nz; k++ {
		for i := 0; i < d.Nx; i++ {
			d.VnorthEdge[k][i] = getbin(d.VbinsNorthEdge[k][i],
				d.VfreqNorthEdge[k][i], r.Float32())
		}
	}
	for j := 0; j < d.Ny; j++ {
		for i := 0; i < d.Nx; i++ {
			d.WtopEdge[j][i] = getbin(d.WbinsTopEdge[j][i],
				d.WfreqTopEdge[j][i], r.Float32())
		}
	}
	d.setTstep()
	// Add in emissions after we know dt.
	var c *AIMcell
	for _, c = range d.Data {
		for i, _ := range c.initialConc {
			c.finalConc[i] += c.emisFlux[i] * d.Dt
			c.initialConc[i] = c.finalConc[i]
		}
	}
	d.arrayLock.Unlock()
}

//  Set the time step using the Courant–Friedrichs–Lewy (CFL) condition.
func (d *AIMdata) setTstep() {
	const Cmax = 1
	val := 0.
	// don't worry about the edges of the staggered grids.
	var uval, vval, wval, thisval float64
	var c *AIMcell
	for _, c = range d.Data {
		uval = math.Abs(c.Uwest) / c.Dx
		vval = math.Abs(c.Vsouth) / c.Dy
		wval = math.Abs(c.Wbelow) / c.Dz
		thisval = max(uval, vval, wval)
		if thisval > val {
			val = thisval
		}
	}
	d.Dt = Cmax / math.Pow(3., 0.5) / val // seconds
}

// Read variable which includes random walk bins from NetCDF file.
func (d *AIMdata) readNCFbins(filename string, wg *sync.WaitGroup, Var string) {
	defer wg.Done()
	dat := getNCFbuffer(filename, Var)
	var bstride, kstride, jstride int
	switch Var {
	case "Ubins", "Ufreq":
		bstride = d.Nz * d.Ny * (d.Nx + 1)
		kstride = d.Ny * (d.Nx + 1)
		jstride = d.Nx + 1
	case "Vbins", "Vfreq":
		bstride = d.Nz * (d.Ny + 1) * d.Nx
		kstride = (d.Ny + 1) * d.Nx
		jstride = d.Nx
	case "Wbins", "Wfreq":
		bstride = (d.Nz + 1) * d.Ny * d.Nx
		kstride = d.Ny * d.Nx
		jstride = d.Nx
	default:
		panic("Unexpected error!")
	}
	ii := 0
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			for i := 0; i < d.Nx; i++ {
				for b := 0; b < d.nbins; b++ {
					index := b*bstride + k*kstride + j*jstride + i
					switch Var {
					case "Ubins":
						d.Data[ii].UbinsWest[b] = dat[index]
					case "Ufreq":
						d.Data[ii].UfreqWest[b] = dat[index]
					case "Vbins":
						d.Data[ii].VbinsSouth[b] = dat[index]
					case "Vfreq":
						d.Data[ii].VfreqSouth[b] = dat[index]
					case "Wbins":
						d.Data[ii].WbinsBelow[b] = dat[index]
					case "Wfreq":
						d.Data[ii].WfreqBelow[b] = dat[index]
					default:
						panic(fmt.Sprintf("Variable %v unknown.\n", Var))
					}
				}
				ii++
			}
		}
	}
	// Set North, East, and Top edge velocity bins for Arakawa C-grid.
	switch Var {
	case "Ubins":
		d.UbinsEastEdge = make([][][]float32, d.Nz)
		d.UeastEdge = make([][]float64, d.Nz)
		i := d.Nx - 1
		for k := 0; k < d.Nz; k++ {
			d.UbinsEastEdge[k] = make([][]float32, d.Ny)
			d.UeastEdge[k] = make([]float64, d.Ny)
			for j := 0; j < d.Ny; j++ {
				d.UbinsEastEdge[k][j] = make([]float32, d.nbins)
				for b := 0; b < d.nbins; b++ {
					index := b*bstride + k*kstride + j*jstride + i
					d.UbinsEastEdge[k][j][b] = dat[index]
				}
			}
		}
	case "Ufreq":
		d.UfreqEastEdge = make([][][]float32, d.Nz)
		i := d.Nx
		for k := 0; k < d.Nz; k++ {
			d.UfreqEastEdge[k] = make([][]float32, d.Ny)
			for j := 0; j < d.Ny; j++ {
				d.UfreqEastEdge[k][j] = make([]float32, d.nbins)
				for b := 0; b < d.nbins; b++ {
					index := b*bstride + k*kstride + j*jstride + i
					d.UfreqEastEdge[k][j][b] = dat[index]
				}
			}
		}
	case "Vbins":
		d.VbinsNorthEdge = make([][][]float32, d.Nz)
		d.VnorthEdge = make([][]float64, d.Nz)
		j := d.Ny
		for k := 0; k < d.Nz; k++ {
			d.VbinsNorthEdge[k] = make([][]float32, d.Nx)
			d.VnorthEdge[k] = make([]float64, d.Nx)
			for i := 0; i < d.Nx; i++ {
				d.VbinsNorthEdge[k][i] = make([]float32, d.nbins)
				for b := 0; b < d.nbins; b++ {
					index := b*bstride + k*kstride + j*jstride + i
					d.VbinsNorthEdge[k][i][b] = dat[index]
				}
			}
		}
	case "Vfreq":
		d.VfreqNorthEdge = make([][][]float32, d.Nz)
		j := d.Ny
		for k := 0; k < d.Nz; k++ {
			d.VfreqNorthEdge[k] = make([][]float32, d.Nx)
			for i := 0; i < d.Nx; i++ {
				d.VfreqNorthEdge[k][i] = make([]float32, d.nbins)
				for b := 0; b < d.nbins; b++ {
					index := b*bstride + k*kstride + j*jstride + i
					d.VfreqNorthEdge[k][i][b] = dat[index]
				}
			}
		}
	case "Wbins":
		d.WbinsTopEdge = make([][][]float32, d.Ny)
		d.WtopEdge = make([][]float64, d.Ny)
		k := d.Nz
		for j := 0; j < d.Ny; j++ {
			d.WbinsTopEdge[j] = make([][]float32, d.Nx)
			d.WtopEdge[j] = make([]float64, d.Nx)
			for i := 0; i < d.Nx; i++ {
				d.WbinsTopEdge[j][i] = make([]float32, d.nbins)
				for b := 0; b < d.nbins; b++ {
					index := b*bstride + k*kstride + j*jstride + i
					d.WbinsTopEdge[j][i][b] = dat[index]
				}
			}
		}
	case "Wfreq":
		d.WfreqTopEdge = make([][][]float32, d.Ny)
		k := d.Nz
		for j := 0; j < d.Ny; j++ {
			d.WfreqTopEdge[j] = make([][]float32, d.Nx)
			for i := 0; i < d.Nx; i++ {
				d.WfreqTopEdge[j][i] = make([]float32, d.nbins)
				for b := 0; b < d.nbins; b++ {
					index := b*bstride + k*kstride + j*jstride + i
					d.WfreqTopEdge[j][i][b] = dat[index]
				}
			}
		}
	default:
		panic(fmt.Sprintf("Variable %v unknown.\n", Var))
	}
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
				case "orgPartitioning":
					d.Data[ii].orgPartitioning = float64(dat[index])
				case "SPartitioning":
					d.Data[ii].SPartitioning = float64(dat[index])
				case "NOPartitioning":
					d.Data[ii].NOPartitioning = float64(dat[index])
				case "NHPartitioning":
					d.Data[ii].NHPartitioning = float64(dat[index])
				case "wdParticle":
					d.Data[ii].wdParticle = float64(dat[index])
				case "wdSO2":
					d.Data[ii].wdSO2 = float64(dat[index])
				case "wdOtherGas":
					d.Data[ii].wdOtherGas = float64(dat[index])
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
