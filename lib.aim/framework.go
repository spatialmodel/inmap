package aim

import (
	"bitbucket.org/ctessum/sparse"
	"code.google.com/p/lvd.go/cdf"
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
	UbinsEastEdge, UfreqEastEdge    [][][]float64 // Edge of the Arakawa C-grid
	VbinsNorthEdge, VfreqNorthEdge  [][][]float64 // Edge of the Arakawa C-grid
	WbinsTopEdge, WfreqTopEdge      [][][]float64 // Edge of the Arakawa C-grid
	UeastEdge, VnorthEdge, WtopEdge [][]float64   // Edge of the Arakawa C-grid
	xRandom, yRandom, zRandom       float64
}

// Data for a single grid cell
type AIMcell struct {
	UbinsWest, UfreqWest           []float64 // m/s
	VbinsSouth, VfreqSouth         []float64 // m/s
	WbinsBelow, WfreqBelow         []float64 // m/s
	Uwest, Vsouth, Wbelow          float64
	orgPartitioning, SPartitioning float64           // gaseous fraction
	NOPartitioning, NHPartitioning float64           // gaseous fraction
	wdParticle, wdSO2, wdOtherGas  float64           // wet deposition rate, 1/s
	verticalDiffusivity            float64           // vertical diffusivity, m2/s
	Dx, Dy, Dz                     float64           // meters
	k                              int               // k cell index
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
	c.UbinsWest = make([]float64, nbins)
	c.VbinsSouth = make([]float64, nbins)
	c.WbinsBelow = make([]float64, nbins)
	c.UfreqWest = make([]float64, nbins)
	c.VfreqSouth = make([]float64, nbins)
	c.WfreqBelow = make([]float64, nbins)
	c.initialConc = make([]float64, len(polNames))
	c.finalConc = make([]float64, len(polNames))
	c.emisFlux = make([]float64, len(polNames))
	return c
}

func InitAIMData(filename string) *AIMdata {
	d := new(AIMdata)
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
	Ubins := sparse.ZerosDense(d.Nz, d.Ny, d.Nx+1)
	go readNCF(filename, &wg, "Ubins", Ubins)
	Vbins := sparse.ZerosDense(d.Nz, d.Ny+1, d.Nx)
	go readNCF(filename, &wg, "Vbins", Vbins)
	Wbins := sparse.ZerosDense(d.Nz+1, d.Ny, d.Nx)
	go readNCF(filename, &wg, "Wbins", Wbins)
	Ufreq := sparse.ZerosDense(d.nbins, d.Nz, d.Ny, d.Nx+1)
	go readNCF(filename, &wg, "Ufreq", Ufreq)
	Vfreq := sparse.ZerosDense(d.nbins, d.Nz, d.Ny+1, d.Nx)
	go readNCF(filename, &wg, "Vfreq", Vfreq)
	Wfreq := sparse.ZerosDense(d.nbins, d.Nz+1, d.Ny, d.Nx)
	go readNCF(filename, &wg, "Wfreq", Wfreq)
	orgPartitioning := sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "orgPartitioning", orgPartitioning)
	SPartitioning := sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "SPartitioning", SPartitioning)
	NOPartitioning := sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "NOPartitioning", NOPartitioning)
	NHPartitioning := sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "NHPartitioning", NHPartitioning)
	wdParticle := sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "wdParticle", wdParticle)
	wdSO2 := sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "wdSO2", wdSO2)
	wdOtherGas := sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	go readNCF(filename, &wg, "wdOtherGas", wdOtherGas)
	layerHeights := sparse.ZerosDense(d.Nz+1, d.Ny, d.Nx)
	go readNCF(filename, &wg, "layerHeights", layerHeights)
	wg.Wait()

	// set up data holders
	d.Data = make([]*AIMcell, d.Nz*d.Ny*d.Nx)
	ii := 0
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			for i := 0; i < d.Nx; i++ {
				// calculate Dz (varies by layer)
				dz := layerHeights.Get(k+1, j, i) - layerHeights.Get(k, j, i)
				d.Data[ii] = newAIMcell(d.nbins, 12000., 12000., dz)
				ii++
			}
		}
	}
	ii = 0
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			for i := 0; i < d.Nx; i++ {
				for b := 0; b < d.nbins; b++ {
					d.Data[ii].UbinsWest[b] = Ubins.Get(k, j, i)
					d.Data[ii].VbinsSouth[b] = Vbins.Get(k, j, i)
					d.Data[ii].WbinsBelow[b] = Wbins.Get(k, j, i)
					d.Data[ii].UfreqWest[b] = Ufreq.Get(k, j, i)
					d.Data[ii].VfreqSouth[b] = Vfreq.Get(k, j, i)
					d.Data[ii].WfreqBelow[b] = Wfreq.Get(k, j, i)
				}
				d.Data[ii].orgPartitioning = orgPartitioning.Get(k, j, i)
				d.Data[ii].SPartitioning = SPartitioning.Get(k, j, i)
				d.Data[ii].NOPartitioning = NOPartitioning.Get(k, j, i)
				d.Data[ii].NHPartitioning = NHPartitioning.Get(k, j, i)
				d.Data[ii].wdParticle = wdParticle.Get(k, j, i)
				d.Data[ii].wdSO2 = wdSO2.Get(k, j, i)
				d.Data[ii].wdOtherGas = wdOtherGas.Get(k, j, i)
				d.Data[ii].getUeast = d.getUeastFunc(k, j, i)
				d.Data[ii].getVnorth = d.getVnorthFunc(k, j, i)
				d.Data[ii].getWabove = d.getWaboveFunc(k, j, i)
				d.Data[ii].getWestNeighbor = d.getWestNeighborFunc(k, j, i)
				d.Data[ii].getEastNeighbor = d.getEastNeighborFunc(k, j, i)
				d.Data[ii].getSouthNeighbor = d.getSouthNeighborFunc(k, j, i)
				d.Data[ii].getNorthNeighbor = d.getNorthNeighborFunc(k, j, i)
				d.Data[ii].getBelowNeighbor = d.getBelowNeighborFunc(k, j, i)
				d.Data[ii].getAboveNeighbor = d.getAboveNeighborFunc(k, j, i)
				d.Data[ii].k = k
				ii++
			}
		}
	}
	// Set North, East, and Top edge velocity bins for Arakawa C-grid.
	d.UbinsEastEdge = make([][]float64, d.Nz)
	d.UfreqEastEdge = make([][]float64, d.Nz)
	i := d.Nx - 1
	for k := 0; k < d.Nz; k++ {
		d.UbinsEastEdge[k] = make([]float64, d.Ny)
		d.UfreqEastEdge[k] = make([]float64, d.Ny)
		for j := 0; j < d.Ny; j++ {
			d.UbinsEastEdge[k][j] = Ubins.Get(k, j, i)
			d.UfreqEastEdge[k][j] = Ufreq.Get(k, j, i)
		}
	}
	d.VbinsNorthEdge = make([][]float64, d.Nz)
	d.VfreqNorthEdge = make([][]float64, d.Nz)
	j := d.Ny - 1
	for k := 0; k < d.Nz; k++ {
		d.VbinsNorthEdge[k] = make([]float64, d.Nx)
		d.VfreqNorthEdge[k] = make([]float64, d.Nx)
		for i := 0; i < d.Nx; i++ {
			d.VbinsNorthEdge[k][i] = Vbins.Get(k, j, i)
			d.VfreqNorthEdge[k][i] = Vfreq.Get(k, j, i)
		}
	}
	d.WbinsTopEdge = make([][]float64, d.Ny)
	d.WfreqTopEdge = make([][]float64, d.Ny)
	k := d.Nz - 1
	for j := 0; j < d.Ny; j++ {
		d.WbinsTopEdge[j] = make([]float64, d.Nx)
		d.WfreqTopEdge[j] = make([]float64, d.Nx)
		for i := 0; i < d.Nx; i++ {
			d.WbinsTopEdge[j][i] = Wbins.Get(k, j, i)
			d.WfreqTopEdge[j][i] = Wfreq.Get(k, j, i)
		}
	}
	d.SettlingVelocity()
	return d
}

// convert 3d index to 1d index
func (d *AIMdata) getIndex(k, j, i int) int {
	return k*d.Ny + j*d.Nx + i
}

// Create functions to get velocities at the East, North, and Top edges of
// the grid cells. These are the same as the West, South, and Bottom edges
// of adjacent cells.
func (d *AIMdata) getUeastFunc(k, j, i int) func() float64 {
	if i == d.Nx-1 {
		return func() {
			return d.UeastEdge[k][j]
		}
	} else {
		return func() {
			return d.Data[d.getIndex(k, j, i+1)].Uwest
		}
	}
}
func (d *AIMdata) getVnorthFunc(k, j, i int) func() float64 {
	if j == d.Ny-1 {
		return func() {
			return d.VnorthEdge[k][i]
		}
	} else {
		return func() {
			return d.Data[d.getIndex(k, j+1, i)].Vsouth
		}
	}
}
func (d *AIMdata) getWaboveFunc(k, j, i int) func() float64 {
	if k == d.Nz-1 {
		return func() {
			return d.WtopEdge[j][i]
		}
	} else {
		return func() {
			return d.Data[d.getIndex(k+1, j, i)].Wbelow
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
		return func(iPol int) float64 {
			return d.Data[d.getIndex(k, j, i-1)].initialConc[iPol]
		}
	}
}
func (d *AIMdata) getEastNeighborFunc(k, j, i int) func(int) float64 {
	if i == d.Nx-1 {
		return getEastBoundary
	} else {
		return func(iPol int) float64 {
			return d.Data[d.getIndex(k, j, i+1)].initialConc[iPol]
		}
	}
}
func (d *AIMdata) getSouthNeighborFunc(k, j, i int) func(int) float64 {
	if j == 0 {
		return getSouthBoundary
	} else {
		return func(iPol int) float64 {
			return d.Data[d.getIndex(k, j-1, i)].initialConc[iPol]
		}
	}
}
func (d *AIMdata) getNorthNeighborFunc(k, j, i int) func(int) float64 {
	if j == d.Ny-1 {
		return getNorthBoundary
	} else {
		return func(iPol int) float64 {
			return d.Data[d.getIndex(k, j+1, i)].initialConc[iPol]
		}
	}
}
func (d *AIMdata) getBelowNeighborFunc(k, j, i int) func(int) float64 {
	if k == 0 {
		// Lower boundary is same as lowest grid cell value.
		return d.Data[d.getIndex(k, j, i)].initialConc[iPol]
	} else {
		return func(iPol int) float64 {
			return d.Data[d.getIndex(k-1, j, i)].initialConc[iPol]
		}
	}
}
func (d *AIMdata) getAboveNeighborFunc(k, j, i int) func(int) float64 {
	if k == d.Nz-1 {
		return getUpperBoundary
	} else {
		return func(iPol int) float64 {
			return d.Data[d.getIndex(k+1, j, i)].initialConc[iPol]
		}
	}
}

// set random numbers for weighted random walk.
func (d *AIMdata) newRand() {
	d.xRandom = rand.Float64()
	d.yRandom = rand.Float64()
	d.zRandom = rand.Float64()
}

// choose a bin using a weighted random method
func getbin(bins, freq []float64, random float64) float64 {
	for b := 0; b < len(c.bins); b++ {
		if random <= c.freq[b] {
			return c.bins[b]
		}
	}
	panic(fmt.Sprintf("Could not choose a bin using seed %v "+
		"(max cumulative frequency=%v).", random,
		c.freq[len(freq)-1]))
	return 0.
}

// Add in emissions, set wind velocities using the weighted
// random walk method, copy initial concentrations to final concentrations,
// and set time step (dt).
func (d *AIMdata) SetupTimeStep() {
	d.newRand()
	for _, c := range d.Data {
		c.Uwest = getbin(c.UbinsWest, c.UfreqWest, d.xRandom)
		c.VSouth = getbin(c.VbinsSouth, c.VfreqSouth, d.yRandom)
		c.Wbelow = getbin(c.WbinsBelow, c.WfreqBelow, d.yRandom)
	}
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			d.UeastEdge[k][j] = getbin(d.UbinsEastEdge[k][j],
				d.UfreqEastEdge[k][j], d.xRandom)
		}
	}
	for k := 0; k < d.Nz; k++ {
		for i := 0; i < d.Nx; i++ {
			d.VnorthEdge[k][i] = getbin(d.VbinsNorthEdge[k][i],
				d.VfreqNorthEdge[k][i], d.yRandom)
		}
	}
	for j := 0; j < d.Ny; j++ {
		for i := 0; i < d.Nx; i++ {
			d.WtopEdge[j][i] = getbin(d.WbinsTopEdge[j][i],
				d.WfreqTopEdge[j][i], d.zRandom)
		}
	}
	d.setTstep()
	// Add in emissions after we know dt.
	d.arrayLock.Lock()
	for _, c := range d.Data {
		for i, _ := range c.initialConc {
			c.finalConc[i] += c.emisFlux[i] * d.Dt
			c.initialConc[i] = c.finalConc[i]
		}
	}
	d.arrayLock.Unlock()
}

//  Set the time step using the Courant–Friedrichs–Lewy (CFL) condition.
func (d *AIMdata) setTstep() {
	//m.Dt = m.Dx * 3 / 1000.
	d.Dt = 30.
	//	const Cmax = 1.26 * 0.75
	//	val := 0.
	//	// don't worry about the edges of the staggered grids.
	//	for k := 0; k < m.Nz; k++ {
	//		for j := 0; j < m.Ny; j++ {
	//			for i := 0; i < m.Nx; i++ {
	//				uval := math.Abs(m.getBin(m.Ufreq, m.Ubins,k, j, i)) / m.Dx
	//				vval := math.Abs(m.getBin(m.Vfreq, m.Vbins,k, j, i)) / m.Dy
	//				wval := math.Abs(m.getBin(m.Wfreq, m.Wbins,k, j, i)) /
	//					m.Dz.Get(k, j, i)
	//				thisval := max(uval,vval,wval)
	//				if thisval > val {
	//					val = thisval
	//				}
	//			}
	//		}
	//	}
	//	m.Dt = Cmax / math.Pow(3., 0.5) / val // seconds
}

// Read variable from NetCDF file.
func readNCF(filename string, wg *sync.WaitGroup, Var string,
	data *sparse.DenseArray) {
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
	defer wg.Done()
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
	dat := buf.([]float32)
	for i, val := range dat {
		data.Elements[i] = float64(val)
	}
}
