package main

import (
	"bitbucket.org/ctessum/sparse"
	"code.google.com/p/lvd.go/cdf"
	"flag"
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
)

const (
	// user settings
	wrfout     = "/home/marshall/tessumcm/WRFchem_output/WRF.2005_nei.na12.chem.3.4/output/wrfout_d01_[DATE]"
	outputFile = "aimData.ncf"
	startDate  = "20050101"
	endDate    = "20051231"
	nWindBins  = 20 // number of bins for wind speed
	nProcs     = 16 // number of processors to use

	// non-user settings
	wrfFormat    = "2006-01-02_15_04_05"
	inDateFormat = "20060102"
	tolerance    = 1.e-10 // tolerance for comparing floats

	// physical constants
	MWa   = 28.97   // g/mol, molar mass of air
	mwN   = 46.0055 // g/mol, molar mass of nitrogen
	mwS   = 32.0655 // g/mol, molar mass of sulfur
	mwNH4 = 18.03851
	mwSO4 = 96.0632
	mwNO3 = 62.00501
	g     = 9.80665 // m/s2
)

var (
	start     time.Time
	end       time.Time
	current   time.Time
	numTsteps float64
)

var (
	// RACM VOC species and molecular weights (g/mol);
	// assume condensable vapor from SOA has molar mass of 70
	VOC = map[string]float64{"eth": 30, "hc3": 44, "hc5": 72, "hc8": 114,
		"ete": 28, "olt": 42, "oli": 68, "dien": 54, "iso": 68, "api": 136,
		"lim": 136, "tol": 92, "xyl": 106, "csl": 108, "hcho": 30, "ald": 44,
		"ket": 72, "gly": 58, "mgly": 72, "dcb": 87, "macr": 70, "udd": 119,
		"hket": 74, "onit": 119, "pan": 121, "tpan": 147, "op1": 48, "op2": 62,
		"paa": 76, "ora1": 46, "ora2": 60, "cvasoa1": 70, "cvasoa2": 70,
		"cvasoa3": 70, "cvasoa4": 70, "cvbsoa1": 70, "cvbsoa2": 70,
		"cvbsoa3": 70, "cvbsoa4": 70}
	// VBS SOA species (both anthropogenic and biogenic)
	SOA = map[string]float64{"asoa1i": 1, "asoa1j": 1, "asoa2i": 1,
		"asoa2j": 1, "asoa3i": 1, "asoa3j": 1, "asoa4i": 1, "asoa4j": 1,
		"bsoa1i": 1, "bsoa1j": 1, "bsoa2i": 1, "bsoa2j": 1, "bsoa3i": 1,
		"bsoa3j": 1, "bsoa4i": 1, "bsoa4j": 1}
	// RACM NOx species and molecular weights, multiplyied by their
	// nitrogen fractions
	NOx = map[string]float64{"no": 30 * 30 / mwN, "no2": 46 * 46 / mwN}
	// MADE particulate NO species, nitrogen fraction
	pNO = map[string]float64{"no3ai": mwNO3 / mwN, "no3aj": mwNO3 / mwN}
	// RACM SOx species and molecular weights
	SOx = map[string]float64{"so2": 64 * 64 / mwS, "sulf": 98 * 98 / mwS}
	// MADE particulate Sulfur species; sulfur fraction
	pS  = map[string]float64{"so4ai": mwSO4 / mwS, "so4aj": mwSO4 / mwS}
	NH3 = map[string]float64{"nh3": 17.03056 * 17.03056 / mwN}
	// MADE particulate ammonia species, nitrogen fraction
	pNH = map[string]float64{"nh4ai": mwNH4 / mwN, "nh4aj": mwNH4 / mwN}
)

func init() {
	var err error
	start, err = time.Parse(inDateFormat, startDate)
	if err != nil {
		panic(err)
	}
	end, err = time.Parse(inDateFormat, endDate)
	if err != nil {
		panic(err)
	}
	end = end.AddDate(0, 0, 1) // add 1 day to the end
	numTsteps = end.Sub(start).Hours()

	runtime.GOMAXPROCS(nProcs)
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {

	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// calculate wind speed bins
	windBinsChanU := make(chan *sparse.DenseArray)
	go calcWindBins(windBinsChanU)
	windBinsChanV := make(chan *sparse.DenseArray)
	go calcWindBins(windBinsChanV)
	windBinsChanW := make(chan *sparse.DenseArray)
	go calcWindBins(windBinsChanW)
	ustarChan := make(chan *sparse.DenseArray)
	go average(ustarChan)
	pblhChan := make(chan *sparse.DenseArray)
	go average(pblhChan)
	phChan := make(chan *sparse.DenseArray)
	go average(phChan)
	phbChan := make(chan *sparse.DenseArray)
	go average(phbChan)
	qrainChan := make(chan *sparse.DenseArray)
	go average(qrainChan)
	altChan := make(chan *sparse.DenseArray)
	go average(altChan)
	uAvgChan := make(chan *sparse.DenseArray)
	vAvgChan := make(chan *sparse.DenseArray)
	wAvgChan := make(chan *sparse.DenseArray)
	go windSpeed(uAvgChan, vAvgChan, wAvgChan)
	iterateTimeSteps("Reading data for bin sizes: ",
		readSingleVar("U", windBinsChanU, uAvgChan),
		readSingleVar("V", windBinsChanV, vAvgChan),
		readSingleVar("W", windBinsChanW, wAvgChan),
		readSingleVar("UST", ustarChan),
		readSingleVar("PBLH", pblhChan),
		readSingleVar("PH", phChan), readSingleVar("PHB", phbChan),
		readSingleVar("QRAIN", qrainChan), readSingleVar("ALT", altChan))
	windBinsChanU <- nil
	windBinsChanV <- nil
	windBinsChanW <- nil
	ustarChan <- nil
	pblhChan <- nil
	phChan <- nil
	phbChan <- nil
	qrainChan <- nil
	altChan <- nil
	uAvgChan <- nil
	vAvgChan <- nil
	wAvgChan <- nil
	windBinsU := <-windBinsChanU
	windBinsV := <-windBinsChanV
	windBinsW := <-windBinsChanW
	ustar := <-ustarChan
	pblh := <-pblhChan
	ph := <-phChan
	phb := <-phbChan
	qrain := <-qrainChan
	alt := <-altChan
	windSpeed := <-uAvgChan

	layerHeights := calcLayerHeights(ph, phb)
	verticalDiffusivity := calcMixing(ustar, layerHeights, pblh)
	wdParticle, wdSO2, wdOtherGas := calcWetDeposition(qrain, alt)

	// fill wind speed bins with data
	windStatsChanU := make(chan *sparse.DenseArray)
	go calcWindStats(windBinsU, windStatsChanU)
	windStatsChanV := make(chan *sparse.DenseArray)
	go calcWindStats(windBinsV, windStatsChanV)
	windStatsChanW := make(chan *sparse.DenseArray)
	go calcWindStats(windBinsW, windStatsChanW)

	// calculate gas/particle partitioning
	VOCchan := make(chan *sparse.DenseArray)
	SOAchan := make(chan *sparse.DenseArray)
	go calcPartitioning(VOCchan, SOAchan)
	NOxchan := make(chan *sparse.DenseArray)
	pNOchan := make(chan *sparse.DenseArray)
	go calcPartitioning(NOxchan, pNOchan)
	SOxchan := make(chan *sparse.DenseArray)
	pSchan := make(chan *sparse.DenseArray)
	go calcPartitioning(SOxchan, pSchan)
	NH3chan := make(chan *sparse.DenseArray)
	pNHchan := make(chan *sparse.DenseArray)
	go calcPartitioning(NH3chan, pNHchan)

	// Calculate stability for plume rise
	Tchan := make(chan *sparse.DenseArray)
	PBchan := make(chan *sparse.DenseArray)
	Pchan := make(chan *sparse.DenseArray)
	go Stability(layerHeights, Tchan, PBchan, Pchan)

	iterateTimeSteps("Reading data for concentrations and bin frequencies: ",
		readSingleVar("U", windStatsChanU), readSingleVar("V", windStatsChanV),
		readSingleVar("W", windStatsChanW),
		readGasGroup(VOC, VOCchan), readParticleGroup(SOA, SOAchan),
		readGasGroup(NOx, NOxchan), readParticleGroup(pNO, pNOchan),
		readGasGroup(SOx, SOxchan), readParticleGroup(pS, pSchan),
		readGasGroup(NH3, NH3chan), readParticleGroup(pNH, pNHchan),
		readSingleVar("T", Tchan), readSingleVar("PB", PBchan),
		readSingleVar("P", Pchan))

	windStatsChanU <- nil
	windStatsChanV <- nil
	windStatsChanW <- nil
	windStatsU := <-windStatsChanU
	windStatsV := <-windStatsChanV
	windStatsW := <-windStatsChanW
	VOCchan <- nil
	NOxchan <- nil
	SOxchan <- nil
	NH3chan <- nil
	orgPartitioning := <-VOCchan
	NOPartitioning := <-NOxchan
	SPartitioning := <-SOxchan
	NHPartitioning := <-NH3chan
	Tchan <- nil
	PBchan <- nil
	Pchan <- nil
	temperature := <-Tchan
	S1 := <-Tchan
	Sclass := <-Tchan

	// write out data to file
	fmt.Printf("Writing out data to %v...\n", outputFile)
	h := cdf.NewHeader(
		[]string{"bins",
			"x", "xStagger",
			"y", "yStagger",
			"z", "zStagger"},
		[]int{nWindBins,
			windStatsV.Shape[3], windStatsU.Shape[3],
			windStatsU.Shape[2], windStatsV.Shape[2],
			windStatsU.Shape[1], windStatsW.Shape[1]})
	h.AddAttribute("", "comment", "Meteorology and baseline chemistry data file")
	h.AddVariable("Ubins", []string{"bins", "z", "y", "xStagger"}, []float32{0})
	h.AddAttribute("Ubins", "description", "Centers of U velocity bins")
	h.AddAttribute("Ubins", "units", "m/s")
	h.AddVariable("Vbins", []string{"bins", "z", "yStagger", "x"}, []float32{0})
	h.AddAttribute("Vbins", "description", "Centers of W velocity bins")
	h.AddAttribute("Vbins", "units", "m/s")
	h.AddVariable("Wbins", []string{"bins", "zStagger", "y", "x"}, []float32{0})
	h.AddAttribute("Wbins", "description", "Centers of W velocity bins")
	h.AddAttribute("Wbins", "units", "m/s")

	h.AddVariable("Ufreq", []string{"bins", "z", "y", "xStagger"}, []float32{0})
	h.AddAttribute("Ufreq", "description", "Frequencies U velocity bins")
	h.AddAttribute("Ufreq", "units", "fraction")
	h.AddVariable("Vfreq", []string{"bins", "z", "yStagger", "x"}, []float32{0})
	h.AddAttribute("Vfreq", "description", "Freqencies for W velocity bins")
	h.AddAttribute("Vfreq", "units", "fraction")
	h.AddVariable("Wfreq", []string{"bins", "zStagger", "y", "x"}, []float32{0})
	h.AddAttribute("Wfreq", "description", "Frequencies for W velocity bins")
	h.AddAttribute("Wfreq", "units", "fraction")

	h.AddVariable("orgPartitioning", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("orgPartitioning", "description", "Mass fraction of organic matter in gas (vs. particle) phase")
	h.AddAttribute("orgPartitioning", "units", "fraction")
	h.AddVariable("NOPartitioning", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("NOPartitioning", "description", "Mass fraction of N from NOx in gas (vs. particle) phase")
	h.AddAttribute("NOPartitioning", "units", "fraction")
	h.AddVariable("SPartitioning", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("SPartitioning", "description", "Mass fraction of S from SOx in gas (vs. particle) phase")
	h.AddAttribute("SPartitioning", "units", "fraction")
	h.AddVariable("NHPartitioning", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("NHPartitioning", "description", "Mass fraction of N from NH3 in gas (vs. particle) phase")
	h.AddAttribute("NHPartitioning", "units", "fraction")

	h.AddVariable("layerHeights", []string{"zStagger", "y", "x"}, []float32{0})
	h.AddAttribute("layerHeights", "description", "Height at edge of layer")
	h.AddAttribute("layerHeights", "units", "m")

	h.AddVariable("wdParticle", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("wdParticle", "description", "Wet deposition rate constant for fine particles")
	h.AddAttribute("wdParticle", "units", "s-1")
	h.AddVariable("wdSO2", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("wdSO2", "description", "Wet deposition rate constant for SO2 gas")
	h.AddAttribute("wdSO2", "units", "s-1")
	h.AddVariable("wdOtherGas", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("wdOtherGas", "description", "Wet deposition rate constant for other gases")
	h.AddAttribute("wdOtherGas", "units", "s-1")

	h.AddVariable("verticalDiffusivity", []string{"zStagger", "y", "x"}, []float32{0})
	h.AddAttribute("verticalDiffusivity", "description", "Vertical turbulent diffusivity")
	h.AddAttribute("verticalDiffusivity", "units", "m2 s-1")

	h.AddVariable("pblh", []string{"y", "x"}, []float32{0})
	h.AddAttribute("pblh", "description", "Planetary boundary layer height")
	h.AddAttribute("pblh", "units", "m")

	h.AddAttribute("", "VOCoxidationRate", []float64{1.e-12}) // Estimated from Stockwell et al., A new mechanism for regional atmospheric chemistry modeling, J. Geophys. Res. 1997
	h.AddAttribute("", "VOCoxidationRateUnits", "s-1")

	h.AddVariable("windSpeed", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("windSpeed", "description", "RMS wind speed")
	h.AddAttribute("windSpeed", "units", "m s-1")

	h.AddVariable("temperature", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("temperature", "description", "Average Temperature")
	h.AddAttribute("temperature", "units", "K")
	h.AddVariable("S1", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("S1", "description", "Stability parameter")
	h.AddAttribute("S1", "units", "?")
	h.AddVariable("Sclass", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("Sclass", "description", "Stability parameter")
	h.AddAttribute("Sclass", "units", "0=Unstable; 1=Stable")

	h.Define()
	ff, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	f, err := cdf.Create(ff, h) // writes the header to ff
	if err != nil {
		panic(err)
	}
	writeNCF(f, "Ubins", binEdgeToCenter(windBinsU))
	writeNCF(f, "Vbins", binEdgeToCenter(windBinsV))
	writeNCF(f, "Wbins", binEdgeToCenter(windBinsW))
	writeNCF(f, "Ufreq", statsCumulative(windStatsU))
	writeNCF(f, "Vfreq", statsCumulative(windStatsV))
	writeNCF(f, "Wfreq", statsCumulative(windStatsW))
	writeNCF(f, "orgPartitioning", orgPartitioning)
	writeNCF(f, "NOPartitioning", NOPartitioning)
	writeNCF(f, "SPartitioning", SPartitioning)
	writeNCF(f, "NHPartitioning", NHPartitioning)
	writeNCF(f, "layerHeights", layerHeights)
	writeNCF(f, "wdParticle", wdParticle)
	writeNCF(f, "wdSO2", wdSO2)
	writeNCF(f, "wdOtherGas", wdOtherGas)
	writeNCF(f, "verticalDiffusivity", verticalDiffusivity)
	writeNCF(f, "pblh", pblh)
	writeNCF(f, "windSpeed", windSpeed)
	writeNCF(f, "temperature", temperature)
	writeNCF(f, "S1", S1)
	writeNCF(f, "Sclass", Sclass)
	ff.Close()
}

func iterateTimeSteps(msg string, funcs ...cdfReaderFunc) {
	filechan := make(chan string)
	finishchan := make(chan int)
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go processFile(filechan, finishchan, funcs...)
	}
	delta, _ := time.ParseDuration("24h")
	for now := start; now.Before(end); now = now.Add(delta) {
		d := now.Format(wrfFormat)
		fmt.Println(msg + d + "...")
		file := strings.Replace(wrfout, "[DATE]", d, -1)
		filechan <- file
	}
	close(filechan)
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		<-finishchan
	}
}

func processFile(filechan chan string, finishchan chan int,
	funcs ...cdfReaderFunc) {
	f := new(cdfFile)
	var err error
	for filename := range filechan {
		f.ff, err = os.Open(filename)
		if err != nil {
			panic(err)
		}
		f.f, err = cdf.Open(f.ff)
		if err != nil {
			panic(err)
		}
		for hour := 0; hour < 24; hour++ {
			var wg sync.WaitGroup
			wg.Add(len(funcs))
			for _, fn := range funcs {
				go fn(f, hour, &wg)
			}
			wg.Wait()
		}
		f.ff.Close()
	}
	finishchan <- 0
}

type cdfReaderFunc func(*cdfFile, int, *sync.WaitGroup)

type cdfFile struct {
	f       *cdf.File
	ff      *os.File
	ncfLock sync.Mutex
}

func readNCF(pol string, f *cdfFile, hour int) (data *sparse.DenseArray) {
	dims := f.f.Header.Lengths(pol)
	if len(dims) == 0 {
		panic(fmt.Sprintf("Variable %v not in file.", pol))
	}
	dims = dims[1:]
	nread := 1
	for _, dim := range dims {
		nread *= dim
	}
	start, end := make([]int, len(dims)+1), make([]int, len(dims)+1)
	start[0], end[0] = hour, hour+1
	r := f.f.Reader(pol, start, end)
	buf := r.Zero(nread)
	f.ncfLock.Lock()
	_, err := r.Read(buf)
	f.ncfLock.Unlock()
	if err != nil {
		panic(err)
	}
	data = sparse.ZerosDense(dims...)
	for i, val := range buf.([]float32) {
		data.Elements[i] = float64(val)
	}
	return
}

func readSingleVar(Var string,
	datachans ...chan *sparse.DenseArray) cdfReaderFunc {
	return func(f *cdfFile, hour int, wg *sync.WaitGroup) {
		defer wg.Done()
		data := readNCF(Var, f, hour)
		for _, datachan := range datachans {
			datachan <- data
		}
	}
}

func readGasGroup(Vars map[string]float64, datachan chan *sparse.DenseArray) cdfReaderFunc {
	return func(f *cdfFile, hour int, wg *sync.WaitGroup) {
		defer wg.Done()
		alt := readNCF("ALT", f, hour) // inverse density (m3 kg-1)
		var out *sparse.DenseArray
		firstData := true
		for pol, factor := range Vars {
			data := readNCF(pol, f, hour)
			if firstData {
				out = sparse.ZerosDense(data.Shape...)
				firstData = false
			}
			for i, val := range data.Elements {
				// convert ppm to μg/m3
				out.Elements[i] += val * factor / MWa * 1000. / alt.Elements[i]
			}
		}
		datachan <- out
	}
}

func readParticleGroup(Vars map[string]float64, datachan chan *sparse.DenseArray) cdfReaderFunc {
	return func(f *cdfFile, hour int, wg *sync.WaitGroup) {
		defer wg.Done()
		alt := readNCF("ALT", f, hour) // inverse density (m3 kg-1)
		var out *sparse.DenseArray
		firstData := true
		for pol, factor := range Vars {
			data := readNCF(pol, f, hour)
			if firstData {
				out = sparse.ZerosDense(data.Shape...)
				firstData = false
			}
			for i, val := range data.Elements {
				// convert μg/kg air to μg/m3 air
				out.Elements[i] += val * factor / alt.Elements[i]
			}
		}
		datachan <- out
	}
}

func writeNCF(f *cdf.File, Var string, data *sparse.DenseArray) {
	data32 := make([]float32, len(data.Elements))
	for i, e := range data.Elements {
		data32[i] = float32(e)
	}
	end := f.Header.Lengths(Var)
	start := make([]int, len(end))
	w := f.Writer(Var, start, end)
	_, err := w.Write(data32)
	if err != nil {
		panic(err)
	}
}

func initBins(data *sparse.DenseArray, nbins int) *sparse.DenseArray {
	dims := make([]int, len(data.Shape)+1)
	dims[0] = nbins
	for i, d := range data.Shape {
		dims[i+1] = d
	}
	return sparse.ZerosDense(dims...)
}

func calcWindBins(datachan chan *sparse.DenseArray) {
	var bins, max, min *sparse.DenseArray
	firstData := true
	for {
		data := <-datachan
		if data == nil {
			fmt.Println("Calculating bin edges...")
			for i := 0; i < bins.Shape[1]; i++ {
				for j := 0; j < bins.Shape[2]; j++ {
					for k := 0; k < bins.Shape[3]; k++ {
						for b := 0; b <= nWindBins; b++ {
							maxval := max.Get(i, j, k)
							minval := min.Get(i, j, k)
							edge := minval + float64(b)/float64(nWindBins)*
								(maxval-minval)
							bins.Set(edge, b, i, j, k)
						}
					}
				}
			}
			datachan <- bins
			return
		}
		if firstData {
			bins = initBins(data, nWindBins+1)
			max = sparse.ZerosDense(data.Shape...)
			min = sparse.ZerosDense(data.Shape...)
			firstData = false
		}
		for i, e := range data.Elements {
			if e > max.Elements[i] {
				max.Elements[i] = e
			}
			if e < min.Elements[i] {
				min.Elements[i] = e
			}
		}
	}
}

// calculate the fraction of time steps with wind speeds in each bin
func calcWindStats(bins *sparse.DenseArray, datachan chan *sparse.DenseArray) {
	var stats *sparse.DenseArray
	firstData := true
	for {
		data := <-datachan
		if data == nil {
			// make sure frequencies add up to 1
			for i := 0; i < bins.Shape[1]; i++ {
				for j := 0; j < bins.Shape[2]; j++ {
					for k := 0; k < bins.Shape[3]; k++ {
						total := 0.
						for b := 0; b < nWindBins; b++ {
							total += stats.Get(b, i, j, k)
						}
						if total-1 > tolerance || total-1 < -1.*tolerance {
							panic(fmt.Sprintf("Fractions add up to %v, not 1!",
								total))
						}
					}
				}
			}

			datachan <- stats
			return
		}
		if firstData {
			stats = initBins(data, nWindBins)
			firstData = false
		}
		type empty struct{}
		sem := make(chan empty, bins.Shape[1]) // semaphore pattern
		for i := 0; i < bins.Shape[1]; i++ {
			go func(i int) { // concurrent processing
				for j := 0; j < bins.Shape[2]; j++ {
					for k := 0; k < bins.Shape[3]; k++ {
						val := data.Get(i, j, k)
						if val+tolerance < bins.Get(0, i, j, k) {
							panic(fmt.Sprintf(
								"Value %v is less than minimum bin %v.",
								val, bins.Get(0, i, j, k)))
						}
						if val-tolerance > bins.Get(nWindBins, i, j, k) {
							panic(fmt.Sprintf(
								"Value %v is more than maximum bin %v.\n",
								val, bins.Get(nWindBins, i, j, k)))
						}
						for b := 0; b < nWindBins; b++ {
							if val <= bins.Get(b+1, i, j, k) {
								stats.AddVal(1./numTsteps, b, i, j, k)
								break
							}
						}
					}
				}
				sem <- empty{}
			}(i)
		}
		for i := 0; i < bins.Shape[1]; i++ { // wait for routines to finish
			<-sem
		}
	}
}

func calcPartitioning(gaschan, particlechan chan *sparse.DenseArray) {
	var gas, particle *sparse.DenseArray
	firstData := true
	for {
		gasdata := <-gaschan
		if gasdata == nil {
			fmt.Println("Calculating partitioning...")
			for i, gasval := range gas.Elements {
				particleval := particle.Elements[i]
				gas.Elements[i] = gasval / (gasval + particleval)
			}
			gaschan <- gas
			return
		}
		particledata := <-particlechan
		if firstData {
			gas = sparse.ZerosDense(gasdata.Shape...)
			particle = sparse.ZerosDense(particledata.Shape...)
			firstData = false
		}
		gas.AddDense(gasdata)
		particle.AddDense(particledata)
	}
}

func average(datachan chan *sparse.DenseArray) {
	var avgdata *sparse.DenseArray
	firstData := true
	for {
		data := <-datachan
		if data == nil {
			for i, val := range avgdata.Elements {
				avgdata.Elements[i] = val / numTsteps
			}
			datachan <- avgdata
			return
		}
		if firstData {
			avgdata = sparse.ZerosDense(data.Shape...)
			firstData = false
		}
		avgdata.AddDense(data)
	}
}

func binEdgeToCenter(in *sparse.DenseArray) *sparse.DenseArray {
	outshape := make([]int, len(in.Shape))
	outshape[0] = in.Shape[0] - 1
	outshape[1], outshape[2], outshape[3] = in.Shape[1], in.Shape[2], in.Shape[3]
	out := sparse.ZerosDense(outshape...)
	for b := 0; b < out.Shape[0]; b++ {
		for i := 0; i < out.Shape[1]; i++ {
			for j := 0; j < out.Shape[2]; j++ {
				for k := 0; k < out.Shape[3]; k++ {
					center := (in.Get(b, i, j, k) + in.Get(b+1, i, j, k)) / 2.
					out.Set(center, b, i, j, k)
				}
			}
		}
	}
	return out
}

func statsCumulative(in *sparse.DenseArray) *sparse.DenseArray {
	out := sparse.ZerosDense(in.Shape...)
	for b := 0; b < out.Shape[0]; b++ {
		for i := 0; i < out.Shape[1]; i++ {
			for j := 0; j < out.Shape[2]; j++ {
				for k := 0; k < out.Shape[3]; k++ {
					var cumulativeVal float64
					if b == 0 {
						cumulativeVal = in.Get(b, i, j, k)
					} else {
						cumulativeVal = out.Get(b-1, i, j, k) + in.Get(b, i, j, k)

					}
					out.Set(cumulativeVal, b, i, j, k)
				}
			}
		}
	}
	return out
}

// calcLayerHeights calculates the heights above the ground
// of the layers (in meters).
// For more information, refer to
// http://www.openwfm.org/wiki/How_to_interpret_WRF_variables
func calcLayerHeights(ph, phb *sparse.DenseArray) (
	layerHeights *sparse.DenseArray) {
	layerHeights = sparse.ZerosDense(ph.Shape...)
	for k := 0; k < ph.Shape[0]; k++ {
		for j := 0; j < ph.Shape[1]; j++ {
			for i := 0; i < ph.Shape[2]; i++ {
				h := (ph.Get(k, j, i) + phb.Get(k, j, i) -
					ph.Get(0, j, i) - phb.Get(0, j, i)) / g // m
				layerHeights.Set(h, k, j, i)
			}
		}
	}
	return
}

// Calculate wet deposition using WRF rain mixing ratio (QRAIN) and inverse density (ALT) and
// formulas from http://www.emep.int/UniDoc/node12.html.
// wdParticle = A * P / Vdr * E; P = QRAIN * Vdr * ρgas => wdParticle = A * QRAIN * ρgas * E
// wdGas = wSub * P / Δz / ρwater = wSub * QRAIN * Vdr * ρgas / Δz / ρwater
func calcWetDeposition(qrain, alt *sparse.DenseArray) (
	wdParticle, wdSO2, wdOtherGas *sparse.DenseArray) {

	wdParticle = sparse.ZerosDense(qrain.Shape...) // units = 1/s
	wdSO2 = sparse.ZerosDense(qrain.Shape...)      // units = 1/s
	wdOtherGas = sparse.ZerosDense(qrain.Shape...) // units = 1/s
	const A = 5.2                                  // m3 kg-1 s-1; Empirical coefficient
	const E = 0.1                                  // size-dependent collection efficiency of aerosols by the raindrops
	const wSubSO2 = 0.15                           // sub-cloud scavanging ratio
	const wSubOther = 0.5                          // sub-cloud scavanging ratio
	const ρwater = 1000.                           // kg/m3
	const Vdr = 5.                                 // m/s
	const Δz = 1000.                               // m
	for i, q := range qrain.Elements {
		alti := alt.Elements[i]
		wdParticle.Elements[i] = A * q / alti * E
		wdSO2.Elements[i] = wSubSO2 * q * Vdr / alti / Δz / ρwater
		wdOtherGas.Elements[i] = wSubOther * q * Vdr / alti / Δz / ρwater
	}
	return
}

// calcMixing calculates vertical turbulent diffusivity using a middling value (1 m2/s)
// from R. Wilson: Turbulent diffusivity from MST radar measurements: a review,
// Annales Geophysicae (2004) 22: 3869–3887 for grid cells above the planetary boundary layer
// and equation 5.39 from J. S. Gulliver, Introduction to chemical transport in the
// environment, 2007, Cambridge University Press for grid cells within the planetary
// boundary layer.
// Inputs are friction velocity (ustar, m/s), layer heights (m),
// and planetary boundary layer height (pblh, m).
func calcMixing(ustar, layerHeights, pblh *sparse.DenseArray) (
	verticalDiffusivity *sparse.DenseArray) {
	const κ = 0.41                                                 // Von Kármán constant
	verticalDiffusivity = sparse.ZerosDense(layerHeights.Shape...) // units = m2/s
	for k := 0; k < layerHeights.Shape[0]; k++ {
		for j := 0; j < layerHeights.Shape[1]; j++ {
			for i := 0; i < layerHeights.Shape[2]; i++ {
				h := layerHeights.Get(k, j, i)
				var εz float64
				if h > pblh.Get(j, i) {
					εz = 1. // R. Wilson Table 1
				} else {
					εz = κ * ustar.Get(j, i) * h // Gulliver Eq 5.39
				}
				verticalDiffusivity.Set(εz, k, j, i)
			}
		}
	}
	return
}

// Calculate RMS wind speed
func windSpeed(uChan, vChan, wChan chan *sparse.DenseArray) {
	var speed *sparse.DenseArray
	firstData := true
	var dims []int
	for {
		u := <-uChan
		v := <-vChan
		w := <-wChan
		if u == nil {
			for i, val := range speed.Elements {
				speed.Elements[i] = val / numTsteps
			}
			uChan <- speed
			return
		}
		if firstData {
			// get unstaggered grid sizes
			dims = make([]int, len(u.Shape))
			for i, ulen := range u.Shape {
				vlen := v.Shape[i]
				wlen := w.Shape[i]
				dims[i] = minInt(ulen, vlen, wlen)
			}
			speed = sparse.ZerosDense(dims...)
			firstData = false
		}
		for k := 0; k < dims[0]; k++ {
			for j := 0; j < dims[1]; j++ {
				for i := 0; i < dims[2]; i++ {
					ucenter := (math.Abs(u.Get(k, j, i)) +
						math.Abs(u.Get(k, j, i+1))) / 2.
					vcenter := (math.Abs(v.Get(k, j, i)) +
						math.Abs(v.Get(k, j+1, i))) / 2.
					wcenter := (math.Abs(w.Get(k, j, i)) +
						math.Abs(w.Get(k+1, j, i))) / 2.
					s := math.Pow(math.Pow(ucenter, 2.)+
						math.Pow(vcenter, 2.)+math.Pow(wcenter, 2.), 0.5)
					speed.AddVal(s, k, j, i)
				}
			}
		}
	}
	return
}

func Stability(LayerHeights *sparse.DenseArray,
	Tchan, PBchan, Pchan chan *sparse.DenseArray) {
	const po = 101300.   // Pa, reference pressure
	const kappa = 0.2854 // related to von karman's constant
	var Temp *sparse.DenseArray
	var S1 *sparse.DenseArray
	var Sclass *sparse.DenseArray
	firstData := true
	for {
		T := <-Tchan   // K
		PB := <-PBchan // Pa
		P := <-Pchan   // Pa
		if T == nil {
			for i, val := range Temp.Elements {
				Temp.Elements[i] = val / numTsteps
			}
			for i, val := range S1.Elements {
				S1.Elements[i] = val / numTsteps
			}
			for i, val := range Sclass.Elements {
				Sclass.Elements[i] = val / numTsteps
			}
			Tchan <- Temp
			Tchan <- S1
			Tchan <- Sclass
			return
		}
		if firstData {
			Temp = sparse.ZerosDense(T.Shape...)
			S1 = sparse.ZerosDense(T.Shape...)
			Sclass = sparse.ZerosDense(T.Shape...)
			firstData = false
		}
		type empty struct{}
		sem := make(chan empty, T.Shape[0]) // semaphore pattern
		for k := 0; k < T.Shape[0]; k++ {
			go func(k int) { // concurrent processing
				for j := 0; j < T.Shape[1]; j++ {
					for i := 0; i < T.Shape[2]; i++ {
						Tval := T.Get(k, j, i)
						var dtheta_dz = 0. // potential temperature gradient
						if k > 0 {
							dtheta_dz = (Tval - T.Get(k-1, j, i)) /
								(LayerHeights.Get(k, j, i) -
									LayerHeights.Get(k-1, j, i)) // K/m
						}

						pressureCorrection := math.Pow(
							(P.Get(k, j, i)+PB.Get(k, j, i))/po, kappa)

						// Ambient temperature
						t := (Tval + 300.) * pressureCorrection // K
						Temp.AddVal(t, k, j, i)

						// Stability parameter
						s1 := dtheta_dz / t * pressureCorrection
						S1.AddVal(s1, k, j, i)

						// Stability class
						if dtheta_dz < 0.005 {
							Sclass.AddVal(0., k, j, i)
						} else {
							Sclass.AddVal(1., k, j, i)
						}
					}
				}
				sem <- empty{}
			}(k)
		}
		for k := 0; k < T.Shape[0]; k++ { // wait for routines to finish
			<-sem
		}
	}
	return
}

func minInt(vals ...int) int {
	minval := vals[0]
	for _, val := range vals {
		if val < minval {
			minval = val
		}
	}
	return minval
}
