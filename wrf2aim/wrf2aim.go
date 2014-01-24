package main

import (
	"bitbucket.org/ctessum/atmos/gocart"
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
	//endDate    = "20051231"
	endDate   = "20050101"
	nWindBins = 20 // number of bins for wind speed
	nProcs    = 16 // number of processors to use

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
	κ     = 0.41    // Von Kármán constant
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

	// Calculate stability for plume rise, vertical mixing,
	// and chemical reaction rates
	Tchan := make(chan *sparse.DenseArray)
	PBchan := make(chan *sparse.DenseArray)
	Pchan := make(chan *sparse.DenseArray)
	surfaceHeatFluxChan := make(chan *sparse.DenseArray)
	hoChan := make(chan *sparse.DenseArray)
	luIndexChan := make(chan *sparse.DenseArray) // surface skin temp
	go StabilityMixingChemistry(layerHeights, ustar, pblh, alt,
		Tchan, PBchan, Pchan, surfaceHeatFluxChan, hoChan, luIndexChan)

	iterateTimeSteps("Reading data for concentrations and bin frequencies: ",
		readSingleVar("U", windStatsChanU), readSingleVar("V", windStatsChanV),
		readSingleVar("W", windStatsChanW),
		readGasGroup(VOC, VOCchan), readParticleGroup(SOA, SOAchan),
		readGasGroup(NOx, NOxchan), readParticleGroup(pNO, pNOchan),
		readGasGroup(SOx, SOxchan), readParticleGroup(pS, pSchan),
		readGasGroup(NH3, NH3chan), readParticleGroup(pNH, pNHchan),
		readSingleVar("HFX", surfaceHeatFluxChan),
		readSingleVar("T", Tchan), readSingleVar("PB", PBchan),
		readSingleVar("P", Pchan), readSingleVar("ho", hoChan),
		readSingleVar("LU_INDEX", luIndexChan))

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
	VOC := <-VOCchan
	SOA := <-VOCchan
	NOPartitioning := <-NOxchan
	gNO := <-NOxchan
	pNO := <-NOxchan
	SPartitioning := <-SOxchan
	gS := <-SOxchan
	pS := <-SOxchan
	NHPartitioning := <-NH3chan
	gNH := <-NH3chan
	pNH := <-NH3chan
	Tchan <- nil
	PBchan <- nil
	Pchan <- nil
	surfaceHeatFluxChan <- nil
	hoChan <- nil
	luIndexChan <- nil
	temperature := <-Tchan
	S1 := <-Tchan
	Sclass := <-Tchan
	Kz := <-Tchan
	M2u := <-Tchan
	M2d := <-Tchan
	pblTopLayer := <-Tchan
	SO2oxidation := <-Tchan
	particleDryDep := <-Tchan

	// write out data to file
	fmt.Printf("Writing out data to %v...\n", outputFile)
	h := cdf.NewHeader(
		[]string{"bins",
			"x", "xStagger",
			"y", "yStagger",
			"z", "zStagger"},
		[]int{nWindBins + 1,
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
	h.AddVariable("VOC", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("VOC", "description", "Average VOC concentration")
	h.AddAttribute("VOC", "units", "ug m-3")
	h.AddVariable("SOA", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("SOA", "description", "Average secondary organic aerosol concentration")
	h.AddAttribute("SOA", "units", "ug m-3")

	h.AddVariable("NOPartitioning", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("NOPartitioning", "description", "Mass fraction of N from NOx in gas (vs. particle) phase")
	h.AddAttribute("NOPartitioning", "units", "fraction")
	h.AddVariable("gNO", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("gNO", "description", "Average concentration of nitrogen fraction of gaseous NOx")
	h.AddAttribute("gNO", "units", "ug m-3")
	h.AddVariable("pNO", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("pNO", "description", "Average concentration of nitrogen fraction of particulate NO3")
	h.AddAttribute("pNO", "units", "ug m-3")

	h.AddVariable("SPartitioning", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("SPartitioning", "description", "Mass fraction of S from SOx in gas (vs. particle) phase")
	h.AddAttribute("SPartitioning", "units", "fraction")
	h.AddVariable("gS", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("gS", "description", "Average concentration of sulfur fraction of gaseous SOx")
	h.AddAttribute("gS", "units", "ug m-3")
	h.AddVariable("pS", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("pS", "description", "Average concentration of sulfur fraction of particulate sulfate")
	h.AddAttribute("pS", "units", "ug m-3")

	h.AddVariable("NHPartitioning", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("NHPartitioning", "description", "Mass fraction of N from NH3 in gas (vs. particle) phase")
	h.AddAttribute("NHPartitioning", "units", "fraction")
	h.AddVariable("gNH", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("gNH", "description", "Average concentration of nitrogen fraction of gaseous ammonia")
	h.AddAttribute("gNH", "units", "ug m-3")
	h.AddVariable("pNH", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("pNH", "description", "Average concentration of nitrogen fraction of particulate ammonium")
	h.AddAttribute("pNH", "units", "ug m-3")

	h.AddVariable("SO2oxidation", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("SO2oxidation", "description", "Rate of SO2 oxidation to SO4 by hydroxyl radical")
	h.AddAttribute("SO2oxidation", "units", "s-1")

	h.AddVariable("particleDryDep", []string{"y", "x"}, []float32{0})
	h.AddAttribute("particleDryDep", "description", "Dry deposition velocity for particles")
	h.AddAttribute("particleDryDep", "units", "m s-1")

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

	h.AddVariable("Kz", []string{"zStagger", "y", "x"}, []float32{0})
	h.AddAttribute("Kz", "description", "Vertical turbulent diffusivity")
	h.AddAttribute("Kz", "units", "m2 s-1")

	h.AddVariable("M2u", []string{"y", "x"}, []float32{0})
	h.AddAttribute("M2u", "description", "ACM2 nonlocal upward mixing (Pleim 2007)")
	h.AddAttribute("M2u", "units", "s-1")

	h.AddVariable("M2d", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("M2d", "description", "ACM2 nonlocal downward mixing (Pleim 2007)")
	h.AddAttribute("M2d", "units", "s-1")

	h.AddVariable("pblTopLayer", []string{"y", "x"}, []float32{0})
	h.AddAttribute("pblTopLayer", "description", "Planetary boundary layer top grid index")
	h.AddAttribute("pblTopLayer", "units", "-")

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

	h.AddVariable("alt", []string{"z", "y", "x"}, []float32{0})
	h.AddAttribute("alt", "description", "Inverse density")
	h.AddAttribute("alt", "units", "m3 kg-1")

	h.Define()
	ff, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	f, err := cdf.Create(ff, h) // writes the header to ff
	if err != nil {
		panic(err)
	}
	writeNCF(f, "Ubins", windBinsU)
	writeNCF(f, "Vbins", windBinsV)
	writeNCF(f, "Wbins", windBinsW)
	writeNCF(f, "Ufreq", statsCumulative(windStatsU))
	writeNCF(f, "Vfreq", statsCumulative(windStatsV))
	writeNCF(f, "Wfreq", statsCumulative(windStatsW))
	writeNCF(f, "orgPartitioning", orgPartitioning)
	writeNCF(f, "VOC", VOC)
	writeNCF(f, "SOA", SOA)
	writeNCF(f, "NOPartitioning", NOPartitioning)
	writeNCF(f, "gNO", gNO)
	writeNCF(f, "pNO", pNO)
	writeNCF(f, "SPartitioning", SPartitioning)
	writeNCF(f, "gS", gS)
	writeNCF(f, "pS", pS)
	writeNCF(f, "NHPartitioning", NHPartitioning)
	writeNCF(f, "gNH", gNH)
	writeNCF(f, "pNH", pNH)
	writeNCF(f, "layerHeights", layerHeights)
	writeNCF(f, "wdParticle", wdParticle)
	writeNCF(f, "wdSO2", wdSO2)
	writeNCF(f, "wdOtherGas", wdOtherGas)
	writeNCF(f, "Kz", Kz)
	writeNCF(f, "M2u", M2u)
	writeNCF(f, "M2d", M2d)
	writeNCF(f, "pblTopLayer", pblTopLayer)
	writeNCF(f, "pblh", pblh)
	writeNCF(f, "alt", alt)
	writeNCF(f, "windSpeed", windSpeed)
	writeNCF(f, "temperature", temperature)
	writeNCF(f, "S1", S1)
	writeNCF(f, "Sclass", Sclass)
	writeNCF(f, "SO2oxidation", SO2oxidation)
	writeNCF(f, "particleDryDep", particleDryDep)
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
						maxval := max.Get(i, j, k)
						minval := min.Get(i, j, k)
						for b := 0; b <= nWindBins; b++ {
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
						for b := 0; b < bins.Shape[0]; b++ {
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
			stats = initBins(data, nWindBins+1)
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
						for b := 0; b < bins.Shape[0]; b++ {
							if val <= bins.Get(b, i, j, k) {
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
			partitioning := sparse.ZerosDense(gas.Shape...)
			fmt.Println("Calculating partitioning...")
			for i, gasval := range gas.Elements {
				particleval := particle.Elements[i]
				partitioning.Elements[i] = gasval / (gasval + particleval)
				gas.Elements[i] /= numTsteps
				particle.Elements[i] /= numTsteps
			}
			gaschan <- partitioning
			gaschan <- gas
			gaschan <- particle
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

// Roughness lengths for USGS land classes ([m]), from WRF file
// VEGPARM.TBL.
var USGSz0 = []float64{.50, .1, .06, .1, 0.095, .20, .11,
	.03, .035, .15, .50, .50, .50, .50, .35, 0.0001, .20, .40,
	.01, .10, .30, .15, .075, 0.001, .01, .15, .01}

// Calculates:
// 1) Stability parameters for use in plume rise calculation (ASME, 1973,
// as described in Seinfeld and Pandis, 2006).
// 2) Vertical turbulent diffusivity using a middling value (1 m2/s)
// from Wilson (2004) for grid cells above the planetary boundary layer
// and Pleim (2007) for grid cells within the planetary
// boundary layer.
// 3) SO2 oxidation to SO4 by HO (Stockwell 1997).
// 4) Dry deposition velocity (gocart).
// Inputs are layer heights (m), friction velocity (ustar, m/s),
// planetary boundary layer height (pblh, m), inverse density (m3/kg),
// perturbation potential temperature (Temp,K), Pressure (Pb and P, Pa),
// surface heat flux (W/m2), HO mixing ratio (ppmv), and USGS land use index
// (luIndex).
func StabilityMixingChemistry(LayerHeights, ustar, pblh, alt *sparse.DenseArray,
	Tchan, PBchan, Pchan, surfaceHeatFluxChan, hoChan,
	luIndexChan chan *sparse.DenseArray) {
	const (
		po    = 101300. // Pa, reference pressure
		kappa = 0.2854  // related to von karman's constant
		Cp    = 1006.   // m2/s2-K; specific heat of air
	)

	var Temp *sparse.DenseArray
	var S1 *sparse.DenseArray
	var Sclass *sparse.DenseArray
	var Kz *sparse.DenseArray
	var M2d *sparse.DenseArray
	var M2u *sparse.DenseArray
	var SO2oxidation *sparse.DenseArray
	var particleDryDep *sparse.DenseArray
	// Get Layer index of PBL top (staggered)
	pblTopLayer := sparse.ZerosDense(pblh.Shape...)
	for j := 0; j < LayerHeights.Shape[1]; j++ {
		for i := 0; i < LayerHeights.Shape[2]; i++ {
			for k := 0; k < LayerHeights.Shape[0]; k++ {
				if LayerHeights.Get(k, j, i) >= pblh.Get(j, i) {
					pblTopLayer.Set(float64(k), j, i)
					break
				}
			}
		}
	}
	firstData := true
	for {
		T := <-Tchan                 // K
		PB := <-PBchan               // Pa
		P := <-Pchan                 // Pa
		hfx := <-surfaceHeatFluxChan // W/m2
		ho := <-hoChan               // ppmv
		luIndex := <-luIndexChan
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
			for i, val := range Kz.Elements {
				Kz.Elements[i] = val / numTsteps
			}
			for i, val := range M2u.Elements {
				M2u.Elements[i] = val / numTsteps
			}
			for i, val := range M2d.Elements {
				M2d.Elements[i] = val / numTsteps
			}
			for i, val := range SO2oxidation.Elements {
				SO2oxidation.Elements[i] = val / numTsteps
			}
			for i, val := range particleDryDep.Elements {
				particleDryDep.Elements[i] = val / numTsteps
			}
			Tchan <- Temp
			Tchan <- S1
			Tchan <- Sclass
			Tchan <- Kz
			Tchan <- M2u
			Tchan <- M2d
			Tchan <- pblTopLayer
			Tchan <- SO2oxidation
			Tchan <- particleDryDep
			return
		}
		if firstData {
			Temp = sparse.ZerosDense(T.Shape...) // units = K
			S1 = sparse.ZerosDense(T.Shape...)
			Sclass = sparse.ZerosDense(T.Shape...)
			Kz = sparse.ZerosDense(LayerHeights.Shape...)     // units = m2/s
			M2u = sparse.ZerosDense(pblh.Shape...)            // units = 1/s
			M2d = sparse.ZerosDense(T.Shape...)               // units = 1/s
			SO2oxidation = sparse.ZerosDense(T.Shape...)      // units = 1/s
			particleDryDep = sparse.ZerosDense(pblh.Shape...) // units = m/s
			firstData = false
		}
		type empty struct{}
		sem := make(chan empty, T.Shape[1]) // semaphore pattern
		for j := 0; j < T.Shape[1]; j++ {
			go func(j int) { // concurrent processing
				for i := 0; i < T.Shape[2]; i++ {
					// Calculate boundary layer average temperature (K)
					// and index for top layer of the PBL
					kPblTop := f2i(pblTopLayer.Get(j, i))
					To := 0.
					for k := 0; k < LayerHeights.Shape[0]; k++ {
						if k == kPblTop {
							To /= float64(k)
							break
						}
						To += T.Get(k, j, i) + 300.
					}
					u := ustar.Get(j, i)
					h := LayerHeights.Get(kPblTop, j, i)
					// Potential temperature flux = surfaceHeatFlux / Cp /  ρ
					// θf (K m / s) = hfx (W / m2) / Cp (J / kg-K) * alt (m3 / kg)
					θf := hfx.Get(j, i) / Cp * alt.Get(0, j, i)
					// L=Monin-Obukhov length, Pleim et al., 2007 equation 14.
					L := To * math.Pow(u, 3) / g / κ / θf

					fconv := max(0., 1/(1+math.Pow(κ, -2./3.)/.72*
						math.Pow(-h/L, -1./3.))) // Pleim 2007, Eq. 19

					z1plushalf := LayerHeights.Get(1, j, i)
					Δz1plushalf := LayerHeights.Get(2, j, i) / 2.
					kh := calculateKh(z1plushalf, h, L, u)
					// Pleim 2007, Eq. 9
					m2u := fconv * kh / Δz1plushalf / max(1., h-z1plushalf)
					// Pleim 2007, Eq 11a
					M2u.AddVal(m2u, j, i)

					vd := gocart.DryDeposition(hfx.Get(j, i), 1./alt.Get(0, j, i),
						ustar.Get(j, i), To, h, USGSz0[f2i(luIndex.Get(j, i))])
					particleDryDep.AddVal(vd, j, i)

					for k := 0; k < T.Shape[0]; k++ {
						Tval := T.Get(k, j, i)
						var dtheta_dz = 0. // potential temperature gradient
						if k > 0 {
							dtheta_dz = (Tval - T.Get(k-1, j, i)) /
								(LayerHeights.Get(k, j, i) -
									LayerHeights.Get(k-1, j, i)) // K/m
						}

						pressureCorrection := math.Pow(
							(P.Get(k, j, i)+PB.Get(k, j, i))/po, kappa)

						// potential temperature, K
						θ := Tval + 300.
						// Ambient temperature, K
						t := θ * pressureCorrection
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

						// Mixing
						z := LayerHeights.Get(k, j, i)
						zabove := LayerHeights.Get(k+1, j, i)
						Δz := zabove - z
						km := calculateKm(z, h, L, u)
						if k >= kPblTop-1 { // free atmosphere (unstaggered grid)
							Kz.AddVal(1., k, j, i)
							if k == T.Shape[0]-1 { // Top Layer
								Kz.AddVal(1., k+1, j, i)
							}
						} else { // Boundary layer (unstaggered grid)
							// Pleim 2007, Eq. 11b
							Kz.AddVal(km*(1-fconv), k, j, i)
							// Pleim 2007, Eq. 4
							M2d.AddVal(m2u*(h-z)/Δz, k, j, i)
						}

						// Chemistry
						const Na = 6.02214129e23 // molec./mol (Avogadro's constant)
						const cm3perm3 = 100. * 100. * 100.
						const molarMassAir = 28.97 / 1000.             // kg/mol
						const airFactor = molarMassAir / Na * cm3perm3 // kg/molec.* cm3/m3
						M := 1. / (alt.Get(k, j, i) * airFactor)       // molec. air / cm3
						hoConc := ho.Get(k, j, i) * 1.e-6 * M          // molec. HO / cm3
						// SO2 oxidation rate (Stockwell 1997, Table 2d)
						const kinf = 1.5e-12
						ko := 3.e-31 * math.Pow(t/300., -3.3)
						kSO2 := (ko * M / (1 + ko*M/kinf)) * math.Pow(0.6,
							1./(1+math.Pow(math.Log10(ko*M/kinf), 2.))) // cm3/molec/s
						SO2oxidation.AddVal(kSO2*hoConc, k, j, i) // 1/s
					}
				}
				sem <- empty{}
			}(j)
		}
		for j := 0; j < T.Shape[1]; j++ { // wait for routines to finish
			<-sem
		}
	}
	return
}

// Calculate heat diffusivity
func calculateKh(z, h, L, ustar float64) (kh float64) {
	var zs, ϕ_h float64
	if L < 0. { // Unstable conditions
		// Pleim 2007, equation 12.5
		zs = min(z, 0.1*h)
		// Pleim Eq. 13
		ϕ_h = math.Pow(1.-16.*zs/L, -0.5)
	} else { // Stable conditions
		zs = z
		// Dyer, 1974 (Concluding Remarks)
		ϕ_h = 1. + 5.*zs/L
	}
	// Pleim Eq. 12; units = m2/s
	kh = κ * ustar / ϕ_h * z * math.Pow(1-z/h, 2)
	return
}

// Calculate mass diffusivity
func calculateKm(z, h, L, ustar float64) (km float64) {
	var zs, ϕ_m float64
	if L < 0. { // Unstable conditions
		// Pleim 2007, equation 12.5
		zs = min(z, 0.1*h)
		// Pleim Eq. 13
		ϕ_m = math.Pow(1.-16.*zs/L, -0.25)
	} else { // Stable conditions
		zs = z
		// Dyer, 1974 (Concluding Remarks)
		ϕ_m = 1. + 5.*zs/L
	}
	// Pleim Eq. 12; units = m2/s
	km = κ * ustar / ϕ_m * z * math.Pow(1-z/h, 2)
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

func min(vals ...float64) float64 {
	minval := vals[0]
	for _, val := range vals {
		if val < minval {
			minval = val
		}
	}
	return minval
}
func max(vals ...float64) float64 {
	maxval := vals[0]
	for _, val := range vals {
		if val > maxval {
			maxval = val
		}
	}
	return maxval
}

// convert float to int (rounding)
func f2i(f float64) int {
	return int(f + 0.5)
}
