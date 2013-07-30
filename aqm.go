package main

import (
	"bitbucket.org/ctessum/aep/gis"
	"bitbucket.org/ctessum/aep/sparse"
	"bitbucket.org/ctessum/aqm/lib.aqm"
	"bufio"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const (
	nx    = 150    // number of x cells
	ny    = 150    // number of y cells
	nz    = 5      // number of z cells
	dt    = 40.    // s
	dx    = 1000.  // m
	dy    = 1000.  // m
	dz    = 50.    // m
	U     = 3.53   // m/s
	V     = -3.53  // m/s
	W     = 0.     // m/s
	E_i   = 8      // x index of emissions
	E_j   = 142    // y index of emissions
	E_k   = 1      // z index of emissions (stack height = 65m)
	dp    = 5.e-6  // m, water droplet size
	rhof  = 1.2466 // kg/m3, air density
	rhop  = 1000.  // kg/m3, density of droplet
	g     = 9.8    // m/s2
	mu    = 1.5e-5 // kg/m/s
	kappa = 0.4    // von karmon's constant
	//		T            = 10. + 273.15 // K, atmospheric temperature
	convergeTolerance = 0.001 // maximum difference between two values to be considered equal
	calcMinimum       = 1.e-7 // minimum value considered a nonzero concentration
)

// weather data: http://www.wcc.nrcs.usda.gov/ftpref/downloads/climate/windrose/minnesota/minneapolis/
func main() {

	runtime.GOMAXPROCS(8)

	VOC_E_rate := 10.  // kg/s (emissions)
	PM25_E_rate := 10. // kg/s (emissions)
	NH3_E_rate := 10.  // kg/s (emissions)
	SOx_E_rate := 10.  // kg/s (emissions)
	NOx_E_rate := 10.  // kg/s (emissions)

	// Settling velocity, m/s
	vs := 2. * (rhop - rhof) * g * math.Pow(dp/2, 2) / 9. / mu
	Wt := W - vs // net velocity, m/s

	fmt.Println("Advection numbers")
	fmt.Println(math.Abs(dx/dt/U), math.Abs(dy/dt/V), math.Abs(dz/dt/Wt))
	if 1. > math.Abs(dx/dt/U) ||
		1. > math.Abs(dy/dt/V) ||
		1. > math.Abs(dz/dt/Wt) {
		panic("ERROR: wind speeds are too high or timestep is too large. Fix one.")
	}

	// Read in Urban Area file
	fid, err := os.Open("urban.gis") // urban areas file
	reader := bufio.NewReader(fid)
	urban := sparse.ZerosDense(nx, ny)
	if err != nil {
		panic(err)
	}
	for i := 0; i < nx; i++ {
		for j := 0; j < ny; j++ {
			str, err := reader.ReadString('\n')
			if err != nil {
				panic(err)
			}
			str = strings.Trim(str, " \n")
			val, err := strconv.ParseFloat(str, 64)
			if err != nil {
				panic(err)
			}
			urban.Set(val, i, j)
		}
	}
	fid.Close()

	// Eddy diffusion coefficients, m2/s
	D := sparse.ZerosDense(nx, ny, nz)
	M := math.Pow(math.Pow(U, 2)+math.Pow(V, 2), 0.5) // Wind Speed, m/s
	zoRural := 0.01                                   // m
	zoUrban := 1.3                                    // m
	var zo float64
	for k := 0; k < nz; k += 1 {
		z := (float64(k) + 0.5) * dz
		for i := 0; i < nx; i += 1 {
			for j := 0; j < ny; j += 1 {
				if urban.Get(i, j) == 1. {
					zo = zoUrban
				} else {
					zo = zoRural
				}
				diffusivity := M * math.Pow(kappa, 2) * z / math.Log(z/zo) // m2/s
				D.Set(diffusivity, i, j, k)
			}
		}
	}

	polNames := []string{"VOC", "PM2_5", "NH3", "SOx", "NOx"}
	initialConc := make(map[string]*sparse.DenseArray)
	finalConc := make(map[string]*sparse.DenseArray)
	oldFinalConc := make(map[string]*sparse.DenseArray)

	for _, pol := range polNames {
		initialConc[pol] = sparse.ZerosDense(nx, ny, nz)
		finalConc[pol] = sparse.ZerosDense(nx, ny, nz)
		oldFinalConc[pol] = sparse.ZerosDense(nx, ny, nz)
	}

	iteration := 0
	for {
		iteration++
		fmt.Printf("马上。。。Iteration %v.\n", iteration)

		// Emissions
		VOCeConc := VOC_E_rate / dx / dy / dz * dt // kg/m3
		initialConc["VOC"].AddVal(VOCeConc, E_i, E_j, E_k)
		PM25eConc := PM25_E_rate / dx / dy / dz * dt // kg/m3
		initialConc["PM2_5"].AddVal(PM25eConc, E_i, E_j, E_k)
		NH3eConc := NH3_E_rate / dx / dy / dz * dt // kg/m3
		initialConc["NH3"].AddVal(NH3eConc, E_i, E_j, E_k)
		SOxeConc := SOx_E_rate / dx / dy / dz * dt // kg/m3
		initialConc["SOx"].AddVal(SOxeConc, E_i, E_j, E_k)
		NOxeConc := NOx_E_rate / dx / dy / dz * dt // kg/m3
		initialConc["NOx"].AddVal(NOxeConc, E_i, E_j, E_k)

		type empty struct{}
		sem := make(chan empty, nz) // semaphore pattern
		for k := 0; k < nz; k += 1 {
			go func(k int) { // concurrent processing
				var xdiff, ydiff, zdiff, xadv, yadv, zadv float64
				for i := 1; i < nx-1; i += 1 {
					for j := 1; j < ny-1; j += 1 {
						tempconc := make(map[string]float64)
						for pol, Carr := range initialConc {
							xdiff, ydiff, zdiff = aqm.DiffusiveFlux(
								D, Carr, i, j, k, dx, dy, dz)
							xadv, yadv, zadv = aqm.AdvectiveFlux(
								Carr, U, V, W, i, j, k, dx, dy, dz)
							tempconc[pol] = Carr.Get(i, j, k) +
								dt*(xdiff+ydiff+zdiff+xadv+yadv+zadv)
						}
						VOCval, PM25val, NH3val, SOxval, NOxval := aqm.ReactiveFlux(
							tempconc["VOC"], tempconc["PM2_5"], tempconc["NH3"],
							tempconc["SOx"], tempconc["NOx"], dt)
						finalConc["VOC"].Set(VOCval, i, j, k)
						finalConc["PM2_5"].Set(PM25val, i, j, k)
						finalConc["NH3"].Set(NH3val, i, j, k)
						finalConc["SOx"].Set(SOxval, i, j, k)
						finalConc["NOx"].Set(NOxval, i, j, k)
					}
				}
				sem <- empty{}
			}(k)
		}
		for k := 0; k < nz; k++ { // wait for routines to finish
			<-sem
		}
		timeToQuit := true
		for q, arr := range finalConc {
			if iteration == 1 || !arraysEqual(oldFinalConc[q], arr) {
				timeToQuit = false
				break
			}
		}
		if timeToQuit {
			break
		}
		for q, _ := range finalConc {
			initialConc[q] = finalConc[q].Copy()
			oldFinalConc[q] = finalConc[q].Copy()
			finalConc[q] = sparse.ZerosDense(nx, ny, nz)
		}
	}
	for pol, Cf := range finalConc {
		createImage(Cf, pol)
	}
}

func arraysEqual(a, b *sparse.DenseArray) bool {
	aval, bval := a.Sum(), b.Sum()
	if math.Abs(aval-bval)/aval > convergeTolerance {
		fmt.Printf("oldSum=%3.2g; newSum=%3.2g, diff=%f\n",
			aval, bval, math.Abs(aval-bval)/aval)
		return false
	}
	fmt.Printf("oldSum=%3.2g; newSum=%3.2g, diff=%f\n",
		aval, bval, math.Abs(aval-bval)/aval)
	return true
}

func createImage(Cf *sparse.DenseArray, pol string) {
	cmap := gis.NewColorMap("Linear")
	cmap.AddArray(Cf.Elements)
	cmap.Set()
	cmap.Legend(pol+"_legend.svg", "concentrations (kg/m3)")
	i := image.NewRGBA(image.Rect(0, 0, nx, ny))
	for x := 0; x < nx; x++ {
		for y := 0; y < ny; y++ {
			i.Set(x, y, cmap.GetColor(Cf.Get(x, y, 0)))
		}
	}
	f, err := os.Create(pol + "_results.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	b := bufio.NewWriter(f)
	err = png.Encode(b, i)
	if err != nil {
		panic(err)
	}
	err = b.Flush()
	if err != nil {
		panic(err)
	}
}
