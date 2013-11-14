package main

import (
	"bitbucket.org/ctessum/aim/lib.aim"
	"bitbucket.org/ctessum/sparse"
	"bufio"
	"code.google.com/p/lvd.go/cdf"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var configFile *string = flag.String("config", "none", "Path to configuration file")

type configData struct {
	AIMdata              string // Path to location of baseline meteorology and pollutant data. Can include environment variables.
	NumProcessors        int    // Number of processors to use for calculations
	GroundLevelEmissions string // Path to ground level emissions file. Can include environment variables.
	ElevatedEmissions    string // Path to elevated emissions file. Can include environment variables.
	Output               string // Path to desired output file location. Can include environment variables.
	HTTPport             string // Port for hosting web page.
	// If HTTPport is `8080`, then the GUI would be viewed by visiting `localhost:8080` in a web browser.
}

func main() {
	flag.Parse()
	if *configFile == "" {
		fmt.Println("Need to specify configuration file as in " +
			"`aim -config=configFile.json`")
		os.Exit(1)
	}
	config := ReadConfigFile(*configFile)

	fmt.Println("\n",
		"-------------------------------------\n",
		"             Welcome!\n",
		"  (A)irshed (I)ntervention (M)odel\n",
		"   Copyright 2013 Chris Tessum\n",
		"-------------------------------------\n")

	runtime.GOMAXPROCS(config.NumProcessors)

	fmt.Println("Reading input data...")
	d := aim.InitAIMdata(config.AIMdata, config.HTTPport)
	fmt.Println("Reading plume rise information...")
	p := aim.GetPlumeRiseInfo(config.AIMdata)

	const (
		height   = 75. * 0.3048             // m
		diam     = 11.28 * 0.3048           // m
		temp     = (377.-32)*5./9. + 273.15 // K
		velocity = 61.94 * 1097. / 3600.    // m/hr
	)

	emissions := make(map[string]*sparse.DenseArray)
	if config.GroundLevelEmissions != "" {
		emissions = getEmissionsNCF(config.GroundLevelEmissions, d)
	}

	if config.ElevatedEmissions != "" {
		elevatedEmis := getEmissionsNCF(config.ElevatedEmissions, d)
		// apply plume rise
		for pol, elev := range elevatedEmis {
			if _, ok := emissions[pol]; !ok {
				emissions[pol] = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
			}
			for j := 0; j < elev.Shape[1]; j++ {
				for i := 0; i < elev.Shape[2]; i++ {
					k := p.CalcPlumeRise(height, diam, temp, velocity, j, i)
					emissions[pol].AddVal(elev.Get(0, j, i), k, j, i)
				}
			}
		}
	}

	// Run model
	finalConc := d.Run(emissions)

	writeOutput(finalConc, d, config.Output)

	fmt.Println("\n",
		"------------------------------------\n",
		"           AIM Completed!\n",
		"------------------------------------\n")
}

// Get the emissions from a NetCDF file
func getEmissionsNCF(filename string, d *aim.AIMdata) (
	emissions map[string]*sparse.DenseArray) {

	const massConv = 907184740000.       // μg per short ton
	const timeConv = 3600. * 8760.       // seconds per year
	const emisConv = massConv / timeConv // convert tons/year to μg/s

	emissions = make(map[string]*sparse.DenseArray)
	ff, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	f, err := cdf.Open(ff)
	if err != nil {
		panic(err)
	}
	defer ff.Close()
	for _, Var := range f.Header.Variables() {
		if Var == "CO" || Var == "PM10" || Var == "CH4" {
			continue
		}
		emissions[polTrans(Var)] = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
		dims := f.Header.Lengths(Var)
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
			emissions[polTrans(Var)].Elements[i] = float64(val) * emisConv
		}
	}
	return
}

func polTrans(pol string) string {
	switch pol {
	case "PM2.5":
		return "PM2_5"
	default:
		return pol
	}
}

// write data out to netcdf
func writeOutput(finalConc map[string]*sparse.DenseArray, d *aim.AIMdata, outfile string) {
	h := cdf.NewHeader(
		[]string{"nx", "ny", "nz"},
		[]int{d.Nx, d.Ny, d.Nz})
	for pol, _ := range finalConc {
		h.AddVariable(pol, []string{"nz", "ny", "nx"}, []float32{0})
		h.AddAttribute(pol, "units", "ug m-3")
	}
	h.Define()
	ff, err := os.Create(outfile)
	if err != nil {
		panic(err)
	}
	f, err := cdf.Create(ff, h) // writes the header to ff
	if err != nil {
		panic(err)
	}
	for pol, arr := range finalConc {
		writeNCF(f, pol, arr)
	}
	ff.Close()
}

func getEmissions(filename string, d *aim.AIMdata) (
	emissions map[string]*sparse.DenseArray) {

	const massConv = 907184740000.       // μg per short ton
	const timeConv = 3600. * 8760.       // seconds per year
	const emisConv = massConv / timeConv // convert tons/year to μg/s

	emissions = make(map[string]*sparse.DenseArray)
	emissions["VOC"] = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	emissions["PM2_5"] = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	emissions["NH3"] = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	emissions["SOx"] = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	emissions["NOx"] = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)

	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)
	firstLine := true
	polCols := make(map[string]int)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), ",")
		if firstLine {
			for i, pol := range line {
				polCols[pol] = i
			}
			firstLine = false
			continue
		}
		row, col := s2i(line[polCols["row"]])-1, s2i(line[polCols["col"]])-1
		SOx := s2f(line[polCols["SOx"]])
		VOC := s2f(line[polCols["VOC"]])
		PM2_5 := s2f(line[polCols["PM2.5"]])
		NH3 := s2f(line[polCols["NH3"]])
		NOx := s2f(line[polCols["NOx"]])

		emissions["SOx"].Set(SOx*emisConv, 0, row, col)
		emissions["VOC"].Set(VOC*emisConv, 0, row, col)
		emissions["PM2_5"].Set(PM2_5*emisConv, 0, row, col)
		emissions["NH3"].Set(NH3*emisConv, 0, row, col)
		emissions["NOx"].Set(NOx*emisConv, 0, row, col)
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return
}

func s2i(s string) int {
	i, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		panic(err)
	}
	return int(i)
}
func s2f(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return f
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

// Reads and parse a json configuration file.
// See below for the required variables.
func ReadConfigFile(filename string) (config *configData) {
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

	config = new(configData)
	err = json.Unmarshal(bytes, config)
	if err != nil {
		fmt.Printf(
			"There has been an error parsing the configuration file.\n"+
				"Please ensure that the file is in valid JSON format\n"+
				"(you can check for errors at http://jsonlint.com/)\n"+
				"and try again!\n\n%v\n\n", err.Error())
		os.Exit(1)
	}

	config.AIMdata = os.ExpandEnv(config.AIMdata)
	config.GroundLevelEmissions = os.ExpandEnv(config.GroundLevelEmissions)
	config.ElevatedEmissions = os.ExpandEnv(config.ElevatedEmissions)
	config.Output = os.ExpandEnv(config.Output)

	outdir := filepath.Dir(config.Output)
	err = os.MkdirAll(outdir, os.ModePerm)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return
}
