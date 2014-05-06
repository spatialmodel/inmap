package main

import (
	"bitbucket.org/ctessum/atmos/acm2"
	"bitbucket.org/ctessum/atmos/emep"
	"bitbucket.org/ctessum/atmos/gocart"
	"bitbucket.org/ctessum/atmos/seinfeld"
	"bitbucket.org/ctessum/sparse"
	"bufio"
	"code.google.com/p/lvd.go/cdf"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
)

type ConfigInfo struct {
	Wrfout            string  // Location of WRF output files. [DATE] is a wild card for the simulation date.
	OutputDir         string  // Directory to put the output files in
	OutputFilePrefix  string  // name for output files
	StartDate         string  // Format = "YYYYMMDD"
	EndDate           string  // Format = "YYYYMMDD"
	Nprocs            int     // number of processors to use
	VariableGrid_x_o  float64 // lower left of output grid, x
	VariableGrid_y_o  float64 // lower left of output grid, y
	VariableGrid_dx   float64 // m
	VariableGrid_dy   float64 // m
	Xnests            []int   // Nesting multiples in the X direction
	Ynests            []int   // Nesting multiples in the Y direction
	CtmGrid_x_o       float64 // lower left of Chemical Transport Model (CTM) grid, x
	CtmGrid_y_o       float64 // lower left of grid, y
	CtmGrid_dx        float64 // m
	CtmGrid_dy        float64 // m
	CtmGrid_nx        int
	CtmGrid_ny        int
	GridProj          string  // projection info for CTM grid; Proj4 format
	PopCutoff         float64 // people per grid cell
	BboxOffset        float64 // A number significantly less than the smallest grid size but not small enough to be confused with zero.
	CensusDir         string  // directory holding census shapefile
	CensusFile        string  // Name of census shapefile
	MortalityRateFile string  // Name of the mortality rate shapefile
}

const (
	wrfFormat    = "2006-01-02_15_04_05"
	inDateFormat = "20060102"
	tolerance    = 1.e-10 // tolerance for comparing floats

	// physical constants
	MWa      = 28.97   // g/mol, molar mass of air
	mwN      = 46.0055 // g/mol, molar mass of nitrogen
	mwS      = 32.0655 // g/mol, molar mass of sulfur
	mwNH4    = 18.03851
	mwSO4    = 96.0632
	mwNO3    = 62.00501
	g        = 9.80665 // m/s2
	κ        = 0.41    // Von Kármán constant
	atmPerPa = 9.86923267e-6
)

var (
	start      time.Time
	end        time.Time
	current    time.Time
	numTsteps  float64
	configFile *string = flag.String("config", "none", "Path to configuration file")
	config             = new(ConfigInfo)
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

	flag.Parse()
	if *configFile == "" {
		fmt.Println("Need to specify configuration file as in " +
			"`aim -config=configFile.json`")
		os.Exit(1)
	}
	ReadConfigFile(*configFile)

	start, err = time.Parse(inDateFormat, config.StartDate)
	if err != nil {
		panic(err)
	}
	end, err = time.Parse(inDateFormat, config.EndDate)
	if err != nil {
		panic(err)
	}
	end = end.AddDate(0, 0, 1) // add 1 day to the end
	numTsteps = end.Sub(start).Hours()

	runtime.GOMAXPROCS(config.Nprocs)
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

	// calculate wind speed and direction
	windDirectionChanU := make(chan *sparse.DenseArray)
	windDirectionChanV := make(chan *sparse.DenseArray)
	windDirectionChanW := make(chan *sparse.DenseArray)
	go calcWindDirection(windDirectionChanU, windDirectionChanV,
		windDirectionChanW)
	pblhChan := make(chan *sparse.DenseArray)
	go average(pblhChan)
	phChan := make(chan *sparse.DenseArray)
	go average(phChan)
	phbChan := make(chan *sparse.DenseArray)
	go average(phbChan)
	uAvgChan := make(chan *sparse.DenseArray)
	vAvgChan := make(chan *sparse.DenseArray)
	wAvgChan := make(chan *sparse.DenseArray)
	go windSpeed(uAvgChan, vAvgChan, wAvgChan)

	iterateTimeSteps("Reading data--pass 1: ",
		readSingleVar("U", windDirectionChanU, uAvgChan),
		readSingleVar("V", windDirectionChanV, vAvgChan),
		readSingleVar("W", windDirectionChanW, wAvgChan),
		readSingleVar("PBLH", pblhChan),
		readSingleVar("PH", phChan), readSingleVar("PHB", phbChan))

	windDirectionChanU <- nil
	windDirectionChanV <- nil
	windDirectionChanW <- nil
	pblhChan <- nil
	phChan <- nil
	phbChan <- nil
	uAvgChan <- nil
	vAvgChan <- nil
	wAvgChan <- nil
	uPlusSpeed := <-windDirectionChanU
	uMinusSpeed := <-windDirectionChanU
	vPlusSpeed := <-windDirectionChanU
	vMinusSpeed := <-windDirectionChanU
	wPlusSpeed := <-windDirectionChanU
	wMinusSpeed := <-windDirectionChanU
	pblh := <-pblhChan
	ph := <-phChan
	phb := <-phbChan
	windSpeed := <-uAvgChan

	layerHeights, Dz := calcLayerHeights(ph, phb)

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

	NH3chan2 := make(chan *sparse.DenseArray)
	pNHchan2 := make(chan *sparse.DenseArray)
	pSchan2 := make(chan *sparse.DenseArray)
	go ammoniaStatus(NH3chan2, pNHchan2, pSchan2)

	qrainChan := make(chan *sparse.DenseArray)
	cloudFracChan := make(chan *sparse.DenseArray)
	altChanWetDep := make(chan *sparse.DenseArray)
	altChan := make(chan *sparse.DenseArray)
	go average(altChan)
	go calcWetDeposition(layerHeights, qrainChan, cloudFracChan,
		altChanWetDep)

	// Calculate stability for plume rise, vertical mixing,
	// and chemical reaction rates
	Tchan := make(chan *sparse.DenseArray)
	PBchan := make(chan *sparse.DenseArray)
	Pchan := make(chan *sparse.DenseArray)
	surfaceHeatFluxChan := make(chan *sparse.DenseArray)
	hoChan := make(chan *sparse.DenseArray)
	h2o2Chan := make(chan *sparse.DenseArray)
	luIndexChan := make(chan *sparse.DenseArray) // surface skin temp
	ustarChan := make(chan *sparse.DenseArray)
	altChanMixing := make(chan *sparse.DenseArray)
	qCloudChan := make(chan *sparse.DenseArray)
	go StabilityMixingChemistry(layerHeights, pblh, ustarChan, altChanMixing,
		Tchan, PBchan, Pchan, surfaceHeatFluxChan, hoChan, h2o2Chan,
		luIndexChan, qCloudChan)

	iterateTimeSteps("Reading data--pass 2: ",
		readGasGroup(VOC, VOCchan), readParticleGroup(SOA, SOAchan),
		readGasGroup(NOx, NOxchan), readParticleGroup(pNO, pNOchan),
		readGasGroup(SOx, SOxchan), readParticleGroup(pS, pSchan, pSchan2),
		readGasGroup(NH3, NH3chan, NH3chan2), readParticleGroup(pNH, pNHchan, pNHchan2),
		readSingleVar("HFX", surfaceHeatFluxChan),
		readSingleVar("UST", ustarChan),
		readSingleVar("T", Tchan), readSingleVar("PB", PBchan),
		readSingleVar("P", Pchan), readSingleVar("ho", hoChan),
		readSingleVar("h2o2", h2o2Chan),
		readSingleVar("LU_INDEX", luIndexChan),
		readSingleVar("QRAIN", qrainChan),
		readSingleVar("CLDFRA", cloudFracChan),
		readSingleVar("QCLOUD", qCloudChan),
		readSingleVar("ALT", altChanWetDep, altChanMixing, altChan))

	VOCchan <- nil
	NOxchan <- nil
	SOxchan <- nil
	NH3chan <- nil
	NH3chan2 <- nil
	pNHchan2 <- nil
	pSchan2 <- nil
	Tchan <- nil
	PBchan <- nil
	Pchan <- nil
	surfaceHeatFluxChan <- nil
	hoChan <- nil
	h2o2Chan <- nil
	luIndexChan <- nil
	ustarChan <- nil
	qrainChan <- nil
	cloudFracChan <- nil
	altChanWetDep <- nil
	altChanMixing <- nil
	altChan <- nil
	qCloudChan <- nil
	alt := <-altChan
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
	pblTopLayer := <-Tchan
	temperature := <-Tchan
	S1 := <-Tchan
	Sclass := <-Tchan
	Kzz := <-Tchan
	M2u := <-Tchan
	M2d := <-Tchan
	SO2oxidation := <-Tchan
	particleDryDep := <-Tchan
	SO2DryDep := <-Tchan
	NOxDryDep := <-Tchan
	NH3DryDep := <-Tchan
	VOCDryDep := <-Tchan
	Kxxyy := <-Tchan
	particleWetDep := <-qrainChan
	SO2WetDep := <-qrainChan
	otherGasWetDep := <-qrainChan
	fracAmmoniaPoor := <-NH3chan2

	// write out data to file
	outputFile := filepath.Join(config.OutputDir, config.OutputFilePrefix+".ncf")
	fmt.Printf("Writing out data to %v...\n", outputFile)
	h := cdf.NewHeader(
		[]string{"x", "y", "z", "zStagger"},
		[]int{windSpeed.Shape[2], windSpeed.Shape[1], windSpeed.Shape[0],
			windSpeed.Shape[0] + 1})
	h.AddAttribute("", "comment", "Meteorology and baseline chemistry data file")

	data := map[string]dataHolder{
		"UPlusSpeed": dataHolder{[]string{"z", "y", "x"},
			"Average speed of wind going in +U direction", "m/s", uPlusSpeed},
		"UMinusSpeed": dataHolder{[]string{"z", "y", "x"},
			"Average speed of wind going in -U direction", "m/s", uMinusSpeed},
		"VPlusSpeed": dataHolder{[]string{"z", "y", "x"},
			"Average speed of wind going in +V direction", "m/s", vPlusSpeed},
		"VMinusSpeed": dataHolder{[]string{"z", "y", "x"},
			"Average speed of wind going in -V direction", "m/s", vMinusSpeed},
		"WPlusSpeed": dataHolder{[]string{"z", "y", "x"},
			"Average speed of wind going in +W direction", "m/s", wPlusSpeed},
		"WMinusSpeed": dataHolder{[]string{"z", "y", "x"},
			"Average speed of wind going in -W direction", "m/s", wMinusSpeed},
		"OrgPartitioning": dataHolder{[]string{"z", "y", "x"},
			"Mass fraction of organic matter in gas {vs. particle} phase",
			"fraction", orgPartitioning},
		"VOC": dataHolder{[]string{"z", "y", "x"},
			"Average VOC concentration", "ug m-3", VOC},
		"SOA": dataHolder{[]string{"z", "y", "x"},
			"Average secondary organic aerosol concentration", "ug m-3", SOA},
		"NOPartitioning": dataHolder{[]string{"z", "y", "x"},
			"Mass fraction of N from NOx in gas {vs. particle} phase", "fraction",
			NOPartitioning},
		"gNO": dataHolder{[]string{"z", "y", "x"},
			"Average concentration of nitrogen fraction of gaseous NOx", "ug m-3",
			gNO},
		"pNO": dataHolder{[]string{"z", "y", "x"},
			"Average concentration of nitrogen fraction of particulate NO3",
			"ug m-3", pNO},
		"SPartitioning": dataHolder{[]string{"z", "y", "x"},
			"Mass fraction of S from SOx in gas {vs. particle} phase", "fraction",
			SPartitioning},
		"gS": dataHolder{[]string{"z", "y", "x"},
			"Average concentration of sulfur fraction of gaseous SOx", "ug m-3",
			gS},
		"pS": dataHolder{[]string{"z", "y", "x"},
			"Average concentration of sulfur fraction of particulate sulfate",
			"ug m-3", pS},
		"NHPartitioning": dataHolder{[]string{"z", "y", "x"},
			"Mass fraction of N from NH3 in gas {vs. particle} phase", "fraction",
			NHPartitioning},
		"gNH": dataHolder{[]string{"z", "y", "x"},
			"Average concentration of nitrogen fraction of gaseous ammonia",
			"ug m-3", gNH},
		"pNH": dataHolder{[]string{"z", "y", "x"},
			"Average concentration of nitrogen fraction of particulate ammonium",
			"ug m-3", pNH},
		"FracAmmoniaPoor": dataHolder{[]string{"z", "y", "x"},
			"Fraction of the time that aerosol chemistry is ammonia poor",
			"fraction", fracAmmoniaPoor},
		"SO2oxidation": dataHolder{[]string{"z", "y", "x"},
			"Rate of SO2 oxidation to SO4 by hydroxyl radical and H2O2",
			"s-1", SO2oxidation},
		"ParticleDryDep": dataHolder{[]string{"y", "x"},
			"Dry deposition velocity for particles", "m s-1", particleDryDep},
		"SO2DryDep": dataHolder{[]string{"y", "x"},
			"Dry deposition velocity for SO2", "m s-1", SO2DryDep},
		"NOxDryDep": dataHolder{[]string{"y", "x"},
			"Dry deposition velocity for NOx", "m s-1", NOxDryDep},
		"NH3DryDep": dataHolder{[]string{"y", "x"},
			"Dry deposition velocity for NH3", "m s-1", NH3DryDep},
		"VOCDryDep": dataHolder{[]string{"y", "x"},
			"Dry deposition velocity for VOCs", "m s-1", VOCDryDep},
		"Kxxyy": dataHolder{[]string{"z", "y", "x"},
			"Horizontal eddy diffusion coefficient", "m2 s-1", Kxxyy},
		"LayerHeights": dataHolder{[]string{"zStagger", "y", "x"},
			"Height at edge of layer", "m", layerHeights},
		"Dz": dataHolder{[]string{"z", "y", "x"},
			"Vertical grid size", "m", Dz},
		"ParticleWetDep": dataHolder{[]string{"z", "y", "x"},
			"Wet deposition rate constant for fine particles",
			"s-1", particleWetDep},
		"SO2WetDep": dataHolder{[]string{"z", "y", "x"},
			"Wet deposition rate constant for SO2 gas", "s-1", SO2WetDep},
		"OtherGasWetDep": dataHolder{[]string{"z", "y", "x"},
			"Wet deposition rate constant for other gases", "s-1", otherGasWetDep},
		"Kzz": dataHolder{[]string{"zStagger", "y", "x"},
			"Vertical turbulent diffusivity", "m2 s-1", Kzz},
		"M2u": dataHolder{[]string{"y", "x"},
			"ACM2 nonlocal upward mixing {Pleim 2007}", "s-1", M2u},
		"M2d": dataHolder{[]string{"z", "y", "x"},
			"ACM2 nonlocal downward mixing {Pleim 2007}", "s-1", M2d},
		"PblTopLayer": dataHolder{[]string{"y", "x"},
			"Planetary boundary layer top grid index", "-", pblTopLayer},
		"Pblh": dataHolder{[]string{"y", "x"},
			"Planetary boundary layer height", "m", pblh},
		"WindSpeed": dataHolder{[]string{"z", "y", "x"},
			"RMS wind speed", "m s-1", windSpeed},
		"Temperature": dataHolder{[]string{"z", "y", "x"},
			"Average Temperature", "K", temperature},
		"S1": dataHolder{[]string{"z", "y", "x"},
			"Stability parameter", "?", S1},
		"Sclass": dataHolder{[]string{"z", "y", "x"},
			"Stability parameter", "0=Unstable; 1=Stable", Sclass},
		"alt": dataHolder{[]string{"z", "y", "x"},
			"Inverse density", "m3 kg-1", alt}}

	for name, d := range data {
		h.AddVariable(name, d.dims, []float32{0})
		h.AddAttribute(name, "description", d.Description)
		h.AddAttribute(name, "units", d.Units)
	}
	h.Define()
	ff, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	f, err := cdf.Create(ff, h) // writes the header to ff
	if err != nil {
		panic(err)
	}
	for name, d := range data {
		writeNCF(f, name, d.data)
	}
	err = cdf.UpdateNumRecs(ff)
	if err != nil {
		panic(err)
	}
	ff.Close()
	variableGrid(data)
}

type dataHolder struct {
	dims        []string
	Description string
	Units       string
	data        *sparse.DenseArray
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
		file := strings.Replace(config.Wrfout, "[DATE]", d, -1)
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

func readGasGroup(Vars map[string]float64, datachans ...chan *sparse.DenseArray) cdfReaderFunc {
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
		for _, datachan := range datachans {
			datachan <- out
		}
	}
}

func readParticleGroup(Vars map[string]float64, datachans ...chan *sparse.DenseArray) cdfReaderFunc {
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
		for _, datachan := range datachans {
			datachan <- out
		}
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

// Calculate fraction of the time that the atmosphere is ammonia-poor, where
// total ammonia [moles] < 2 * total sulfur VI [moles].
func ammoniaStatus(NH3chan, pNHchan, pSchan chan *sparse.DenseArray) {
	var fracAmmoniaPoor *sparse.DenseArray
	firstData := true
	for {
		nh3array := <-NH3chan // μg/m3 N
		pnharray := <-pNHchan // μg/m3 N
		psarray := <-pSchan   // μg/m3 S
		if nh3array == nil {
			NH3chan <- arrayAverage(fracAmmoniaPoor)
			return
		}
		if firstData {
			fracAmmoniaPoor = sparse.ZerosDense(nh3array.Shape...)
			firstData = false
		}
		for i, nh3 := range nh3array.Elements {
			pnh := pnharray.Elements[i]
			ps := psarray.Elements[i]
			var poor float64
			if (nh3+pnh)/mwN < 2.*ps/mwS {
				poor = 1.
			} else {
				poor = 0.
			}
			fracAmmoniaPoor.Elements[i] += poor
		}
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
	layerHeights, Dz *sparse.DenseArray) {
	layerHeights = sparse.ZerosDense(ph.Shape...)
	Dz = sparse.ZerosDense(ph.Shape[0]-1, ph.Shape[1], ph.Shape[2])
	for k := 0; k < ph.Shape[0]; k++ {
		for j := 0; j < ph.Shape[1]; j++ {
			for i := 0; i < ph.Shape[2]; i++ {
				h := (ph.Get(k, j, i) + phb.Get(k, j, i) -
					ph.Get(0, j, i) - phb.Get(0, j, i)) / g // m
				layerHeights.Set(h, k, j, i)
				if k > 0 {
					Dz.Set(h-layerHeights.Get(k-1, j, i), k-1, j, i)
				}
			}
		}
	}
	return
}

// calculate wet deposition
func calcWetDeposition(layerHeights *sparse.DenseArray, qrainChan,
	cloudFracChan, altChan chan *sparse.DenseArray) {
	var wdParticle, wdSO2, wdOtherGas *sparse.DenseArray

	Δz := sparse.ZerosDense(layerHeights.Shape[0]-1, layerHeights.Shape[1],
		layerHeights.Shape[2]) // grid cell height, m
	for k := 0; k < layerHeights.Shape[0]-1; k++ {
		for j := 0; j < layerHeights.Shape[1]; j++ {
			for i := 0; i < layerHeights.Shape[2]; i++ {
				Δz.Set(layerHeights.Get(k+1, j, i)-layerHeights.Get(k, j, i),
					k, j, i)
			}
		}
	}

	firstData := true
	for {
		qrain := <-qrainChan         // mass frac
		cloudFrac := <-cloudFracChan // frac
		alt := <-altChan             // m3/kg
		if qrain == nil {
			qrainChan <- arrayAverage(wdParticle)
			qrainChan <- arrayAverage(wdSO2)
			qrainChan <- arrayAverage(wdOtherGas)
			return
		}
		if firstData {
			wdParticle = sparse.ZerosDense(qrain.Shape...) // units = 1/s
			wdSO2 = sparse.ZerosDense(qrain.Shape...)      // units = 1/s
			wdOtherGas = sparse.ZerosDense(qrain.Shape...) // units = 1/s
			firstData = false
		}
		for i := 0; i < len(qrain.Elements); i++ {
			wdp, wds, wdo := emep.WetDeposition(cloudFrac.Elements[i],
				qrain.Elements[i], 1/alt.Elements[i], Δz.Elements[i])
			wdParticle.Elements[i] += wdp
			wdSO2.Elements[i] += wds
			wdOtherGas.Elements[i] += wdo
		}
	}
}

// Calculate average wind directions and speeds
func calcWindDirection(uChan, vChan, wChan chan *sparse.DenseArray) {
	var uPlusSpeed *sparse.DenseArray
	var uMinusSpeed *sparse.DenseArray
	var vPlusSpeed *sparse.DenseArray
	var vMinusSpeed *sparse.DenseArray
	var wPlusSpeed *sparse.DenseArray
	var wMinusSpeed *sparse.DenseArray
	firstData := true
	var dims []int
	for {
		u := <-uChan
		v := <-vChan
		w := <-wChan
		if u == nil {
			uChan <- arrayAverage(uPlusSpeed)
			uChan <- arrayAverage(uMinusSpeed)
			uChan <- arrayAverage(vPlusSpeed)
			uChan <- arrayAverage(vMinusSpeed)
			uChan <- arrayAverage(wPlusSpeed)
			uChan <- arrayAverage(wMinusSpeed)
			return
		}
		if firstData {
			// get unstaggered grid sizes
			dims = make([]int, len(u.Shape))
			for i, ulen := range u.Shape {
				vlen := v.Shape[i]
				dims[i] = minInt(ulen, vlen)
			}
			uPlusSpeed = sparse.ZerosDense(dims...)
			uMinusSpeed = sparse.ZerosDense(dims...)
			vPlusSpeed = sparse.ZerosDense(dims...)
			vMinusSpeed = sparse.ZerosDense(dims...)
			wPlusSpeed = sparse.ZerosDense(dims...)
			wMinusSpeed = sparse.ZerosDense(dims...)
			firstData = false
		}
		for k := 0; k < dims[0]; k++ {
			for j := 0; j < dims[1]; j++ {
				for i := 0; i < dims[2]; i++ {
					ucenter := (u.Get(k, j, i) + u.Get(k, j, i+1)) / 2.
					vcenter := (v.Get(k, j, i) + v.Get(k, j+1, i)) / 2.
					wcenter := (w.Get(k, j, i) + w.Get(k+1, j, i)) / 2.
					if ucenter > 0 {
						uPlusSpeed.AddVal(ucenter, k, j, i)
					} else {
						uMinusSpeed.AddVal(-ucenter, k, j, i)
					}
					if vcenter > 0 {
						vPlusSpeed.AddVal(vcenter, k, j, i)
					} else {
						vMinusSpeed.AddVal(-vcenter, k, j, i)
					}
					if wcenter > 0 {
						wPlusSpeed.AddVal(wcenter, k, j, i)
					} else {
						wMinusSpeed.AddVal(-wcenter, k, j, i)
					}
				}
			}
		}
	}
	return
}

func arrayAverage(s *sparse.DenseArray) *sparse.DenseArray {
	for i, val := range s.Elements {
		s.Elements[i] = val / numTsteps
	}
	return s
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
// 5) Horizontal eddy diffusion coefficient (Kyy, [m2/s]) assumed to be the
// same as vertical eddy diffusivity.
//
// Inputs are layer heights (m), friction velocity (ustar, m/s),
// planetary boundary layer height (pblh, m), inverse density (m3/kg),
// perturbation potential temperature (Temp,K), Pressure (Pb and P, Pa),
// surface heat flux (W/m2), HO mixing ratio (ppmv), and USGS land use index
// (luIndex).
func StabilityMixingChemistry(LayerHeights, pblh *sparse.DenseArray,
	ustarChan, altChan, Tchan, PBchan, Pchan, surfaceHeatFluxChan,
	hoChan, h2o2Chan, luIndexChan,
	qCloudChan chan *sparse.DenseArray) {
	const (
		po    = 101300. // Pa, reference pressure
		kappa = 0.2854  // related to von karman's constant
		Cp    = 1006.   // m2/s2-K; specific heat of air
	)

	var Temp *sparse.DenseArray
	var S1 *sparse.DenseArray
	var Sclass *sparse.DenseArray
	var Kzz *sparse.DenseArray
	var M2d *sparse.DenseArray
	var M2u *sparse.DenseArray
	var SO2oxidation *sparse.DenseArray
	var particleDryDep *sparse.DenseArray
	var SO2DryDep *sparse.DenseArray
	var NOxDryDep *sparse.DenseArray
	var NH3DryDep *sparse.DenseArray
	var VOCDryDep *sparse.DenseArray
	var Kyy *sparse.DenseArray
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
		h2o2 := <-h2o2Chan           // ppmv
		luIndex := <-luIndexChan     // land use index
		ustar := <-ustarChan         // friction velocity (m/s)
		alt := <-altChan             // inverse density (m3/kg)
		qCloud := <-qCloudChan       // cloud water mixing ratio (kg/kg)
		if T == nil {
			// convert Kzz to unstaggered grid
			KzzUnstaggered := sparse.ZerosDense(Temp.Shape...)
			for j := 0; j < KzzUnstaggered.Shape[1]; j++ {
				for i := 0; i < KzzUnstaggered.Shape[2]; i++ {
					for k := 0; k < KzzUnstaggered.Shape[0]; k++ {
						KzzUnstaggered.Set(
							(Kzz.Get(k, j, i)+Kzz.Get(k+1, j, i))/2.,
							k, j, i)
					}
				}
			}
			Tchan <- pblTopLayer
			for _, arr := range []*sparse.DenseArray{Temp, S1, Sclass,
				KzzUnstaggered, M2u, M2d, SO2oxidation, particleDryDep,
				SO2DryDep, NOxDryDep, NH3DryDep, VOCDryDep, Kyy} {
				Tchan <- arrayAverage(arr)
			}
			return
		}
		if firstData {
			Temp = sparse.ZerosDense(T.Shape...) // units = K
			S1 = sparse.ZerosDense(T.Shape...)
			Sclass = sparse.ZerosDense(T.Shape...)
			Kzz = sparse.ZerosDense(LayerHeights.Shape...)    // units = m2/s
			M2u = sparse.ZerosDense(pblh.Shape...)            // units = 1/s
			M2d = sparse.ZerosDense(T.Shape...)               // units = 1/s
			SO2oxidation = sparse.ZerosDense(T.Shape...)      // units = 1/s
			particleDryDep = sparse.ZerosDense(pblh.Shape...) // units = m/s
			SO2DryDep = sparse.ZerosDense(pblh.Shape...)      // units = m/s
			NOxDryDep = sparse.ZerosDense(pblh.Shape...)      // units = m/s
			NH3DryDep = sparse.ZerosDense(pblh.Shape...)      // units = m/s
			VOCDryDep = sparse.ZerosDense(pblh.Shape...)      // units = m/s
			Kyy = sparse.ZerosDense(LayerHeights.Shape...)    // units = m2/s
			firstData = false
		}
		type empty struct{}
		sem := make(chan empty, T.Shape[1]) // semaphore pattern
		for j := 0; j < T.Shape[1]; j++ {
			go func(j int) { // concurrent processing
				for i := 0; i < T.Shape[2]; i++ {
					kPblTop := f2i(pblTopLayer.Get(j, i))
					// Calculate boundary layer average temperature (K)
					To := 0.
					for k := 0; k < LayerHeights.Shape[0]; k++ {
						if k == kPblTop {
							To /= float64(k)
							break
						}
						To += T.Get(k, j, i) + 300.
					}
					// Calculate convective mixing rate
					u := ustar.Get(j, i) // friction velocity
					h := LayerHeights.Get(kPblTop, j, i)
					hflux := hfx.Get(j, i)                // heat flux [W m-2]
					ρ := 1 / alt.Get(0, j, i)             // density [kg/m3]
					L := acm2.ObukhovLen(hflux, ρ, To, u) // Monin-Obukhov length [m]
					fconv := acm2.ConvectiveFraction(L, h)
					m2u := acm2.M2u(LayerHeights.Get(1, j, i),
						LayerHeights.Get(2, j, i), h, L, u, fconv)
					M2u.AddVal(m2u, j, i)

					// Calculate dry deposition
					gocartObk := gocart.ObhukovLen(hflux, ρ, To, u)
					p := (P.Get(0, j, i) + PB.Get(0, j, i))
					zo := USGSz0[f2i(luIndex.Get(j, i))]
					const rParticle = 0.15e-6 // [m], Seinfeld & Pandis fig 8.11
					const ρparticle = 1830.   // [kg/m3] Jacobson (2005) Ex. 13.5
					particleDryDep.AddVal(
						gocart.ParticleDryDep(gocartObk, u, To, h,
							zo, rParticle, ρparticle, p), j, i)
					SO2DryDep.AddVal(
						gocart.GasDryDep(gocartObk, u, h,
							zo, gocart.DratioForRb["SO2"]), j, i)
					NOxDryDep.AddVal(
						gocart.GasDryDep(gocartObk, u, h,
							zo, gocart.DratioForRb["NOx"]), j, i)
					NH3DryDep.AddVal(
						gocart.GasDryDep(gocartObk, u, h,
							zo, gocart.DratioForRb["NH3"]), j, i)
					VOCDryDep.AddVal(
						gocart.GasDryDep(gocartObk, u, h,
							zo, gocart.DratioForRb["HCHO"]), j, i)

					for k := 0; k < T.Shape[0]; k++ {
						Tval := T.Get(k, j, i)
						var dtheta_dz = 0. // potential temperature gradient
						if k > 0 {
							dtheta_dz = (Tval - T.Get(k-1, j, i)) /
								(LayerHeights.Get(k, j, i) -
									LayerHeights.Get(k-1, j, i)) // K/m
						}

						p := P.Get(k, j, i) + PB.Get(k, j, i) // Pa
						pressureCorrection := math.Pow(p/po, kappa)

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
						zcenter := (LayerHeights.Get(k, j, i) +
							LayerHeights.Get(k+1, j, i)) / 2
						Δz := zabove - z

						const freeAtmKzz = 3. // [m2 s-1]
						if k >= kPblTop-1 {   // free atmosphere (unstaggered grid)
							Kzz.AddVal(freeAtmKzz, k, j, i)
							Kyy.AddVal(freeAtmKzz, k, j, i)
							if k == T.Shape[0]-1 { // Top Layer
								Kzz.AddVal(freeAtmKzz, k+1, j, i)
							}
						} else { // Boundary layer (unstaggered grid)
							Kzz.AddVal(acm2.Kzz(z, h, L, u, fconv), k, j, i)
							M2d.AddVal(acm2.M2d(m2u, z, Δz, h), k, j, i)
							kmyy := acm2.CalculateKm(zcenter, h, L, u)
							Kyy.AddVal(kmyy, k, j, i)
						}

						// Gas phase sulfur chemistry
						const Na = 6.02214129e23 // molec./mol (Avogadro's constant)
						const cm3perm3 = 100. * 100. * 100.
						const molarMassAir = 28.97 / 1000.             // kg/mol
						const airFactor = molarMassAir / Na * cm3perm3 // kg/molec.* cm3/m3
						M := 1. / (alt.Get(k, j, i) * airFactor)       // molec. air / cm3
						hoConc := ho.Get(k, j, i) * 1.e-6 * M          // molec. HO / cm3
						// SO2 oxidation rate (Stockwell 1997, Table 2d)
						const kinf = 1.5e-12
						ko := 3.e-31 * math.Pow(t/300., -3.3)
						SO2rate := (ko * M / (1 + ko*M/kinf)) * math.Pow(0.6,
							1./(1+math.Pow(math.Log10(ko*M/kinf), 2.))) // cm3/molec/s
						kSO2 := SO2rate * hoConc

						// Aqueous phase sulfur chemistry
						qCloudVal := qCloud.Get(k, j, i)
						if qCloudVal > 0. {
							const pH = 3.5 // doesn't really matter for SO2
							qCloudVal /=
								alt.Get(k, j, i) * 1000. // convert to volume frac.
							kSO2 += seinfeld.SulfurH2O2aqueousOxidationRate(
								h2o2.Get(k, j, i)*1000., pH, t, p*atmPerPa,
								qCloudVal)
						}
						SO2oxidation.AddVal(kSO2, k, j, i) // 1/s
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

func minInt(vals ...int) int {
	minval := vals[0]
	for _, val := range vals {
		if val < minval {
			minval = val
		}
	}
	return minval
}

// convert float to int (rounding)
func f2i(f float64) int {
	return int(f + 0.5)
}

// Reads and parse a json configuration file.
func ReadConfigFile(filename string) {
	// Open the configuration file
	var (
		file  *os.File
		bytes []byte
		err   error
	)
	file, err = os.Open(filename)
	if err != nil {
		fmt.Printf("The configuration file you have specified, %v, does not "+
			"appear to exist. Please check the file name and location and "+
			"try again.\n", filename)
		os.Exit(1)
	}
	reader := bufio.NewReader(file)
	bytes, err = ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		fmt.Printf(
			"There has been an error parsing the configuration file.\n"+
				"Please ensure that the file is in valid JSON format\n"+
				"(you can check for errors at http://jsonlint.com/)\n"+
				"and try again!\n\n%v\n\n", err.Error())
		os.Exit(1)
	}

	err = os.MkdirAll(config.OutputDir, os.ModePerm)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return
}
