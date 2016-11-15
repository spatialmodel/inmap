/*
Copyright (C) 2012-2014 Regents of the University of Minnesota.
This file is part of AEP.

AEP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

AEP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with AEP.  If not, see <http://www.gnu.org/licenses/>.
*/

package aep

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ctessum/geom/proj"
)

const (
	months = "jan feb mar apr may jun jul aug sep oct nov dec"
)

const Website = "http://github.com/ctessum/aep/"
const Version = "0.1.0" // versioning scheme at: http://semver.org/

// Reads and parse a json configuration file.
// See below for the required variables.
func ReadConfigFile(filepath *string, testmode *bool, slaves []string, e *ErrCat) (
	config *ConfigData) {
	// Open the configuration file
	var (
		file  *os.File
		bytes []byte
		err   error
	)
	file, err = os.Open(*filepath)
	if err != nil {
		fmt.Printf("The configuration file you have specified, %v, does not "+
			"appear to exist. Please check the file name and location and "+
			"try again.\n", *filepath)
		os.Exit(1)
	}
	reader := bufio.NewReader(file)
	bytes, err = ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	config = new(ConfigData)
	err = json.Unmarshal(bytes, config)
	if err != nil {
		fmt.Printf(
			"There has been an error parsing the configuration file.\n"+
				"Please ensure that the file is in valid JSON format\n"+
				"(you can check for errors at http://jsonlint.com/)\n"+
				"and try again!\n\n%v\n\n", err.Error())
		os.Exit(1)
	}

	config.DefaultSettings.testMode = *testmode
	config.DefaultSettings.slaves = slaves
	// Replace variables in directories with full paths
	config.setup(e)

	return
}

type ConfigData struct {
	Dirs            *DirInfo
	DefaultSettings *Context
	Sectors         map[string]*Context
}

type DirInfo struct {
	Home        string // Home directory
	Input       string // Directory of input files
	Ancilliary  string // Directory for ancilliary information
	Shapefiles  string // Directory where input shapefiles are stored
	GriddedSrgs string // Directory where gridded spatial surrogate shapefiles are to be created. Don't put anything important in here because it may get automatically deleted.
	Output      string // Directory for output data and reports
}

// Context is a container for the configuration and report info
type Context struct {
	outputDir              string // Output directory
	ShapefileDir           string // Directory where input shapefiles are stored
	GriddedSrgs            string // Directory where gridded spatial surrogate shapefiles are to be created
	Sector                 string // Name of the sector
	SectorType             string // Type of sector (point, area)
	RunSpeciate            bool   // Whether to speciate data
	RunSpatialize          bool   // Whether to spatialize data
	RunTemporal            bool   // Whether to temporalize data
	StartDate              string // Date for which to begin processing emissions
	startDate              time.Time
	EndDate                string // Date for which to end processing emissions
	endDate                time.Time
	Tstep                  string // Timestep between emissions outputs. Currently, it is recommended to set this to "1h".
	tStep                  time.Duration
	TstepsPerFile          int    // Number of output timesteps in each file. Currently, it is recommended to set this to "24".
	OutputType             string // Type of output file. Currently the only available option is "WRF".
	SccDesc                string // name of file with SCC code descriptions
	SicDesc                string // name of file with SIC code descriptions
	NaicsDesc              string // name of file with NAICS code descriptions
	SpecRefFile            string // Location of the speciation code reference file.
	SpecRefComboFile       string // Location of the county specific speciation code reference file, if any.
	SpecProFile            string // Location of the SPECIATE database in sqlite format.
	SpecType               string // Type of speciation to perform. Either "mol" or "mass".
	ChemicalMechanism      string // name for chemical species grouping scheme (needs to be present in SPECIATE database as columns with "_GROUP" and "_FACTOR" appended)
	MechAssignmentFile     string // Path to the mechanism assignment file. Can be downloaded from http://www.cert.ucr.edu/~carter/emitdb/
	MechanismMWFile        string // Path to the mechanism molecular weight file. Can be compiled from information from http://www.cert.ucr.edu/~carter/emitdb/
	SpeciesInfoFile        string // Path to the VOC species info file. Can be downloaded from http://www.cert.ucr.edu/~carter/emitdb/
	specFrac               map[string]map[string]map[string]SpecHolder
	PolsToKeep             map[string]*PolHolder // List and characteristics of pollutants to extract from the inventory and process
	InputProj4             string                // Proj4 specification of the spatial projection of the input emissions data.
	inputSr                *proj.SR
	GridRefFile            string             // Location of the gridding reference file
	SrgSpecFile            string             // Location of the surrogate specification file
	TemporalRefFile        string             // Location of the temporal reference file
	TemporalProFile        string             // Location of the temporal profile file
	HolidayFile            string             // Location of the file specifying which days are holidays
	InventoryFreq          InventoryFrequency // The temporal frequency of the inventory data files. Currently the options are "annual", "monthly", and "cem".
	RunPeriods             []Period           // Periods (annual, or individual months, etc.) to be run for this sector
	MatchFullSCC           bool               // Whether to only match codes which are identical, or to accept partial matches.
	DebugLevel             int                // Sets the volume of output printed to the screen. Set to 0 for least output, 3 for most output. Also, if DebugLevel > 0, any errors encountered will cause the entire program to crash with a stack trace, rather than just printing an error message and continuing.
	Ncpus                  int                // Number of processors available for use
	InputUnits             string             // Units of emissions in input file
	InputConv              float64
	ForceWesternHemisphere bool     // If all data is in the western hemisphere, fix any errors where the minus sign was left out of the longitude.
	InvFileNames           []string // List of input files. "[month]" can be used as a wildcard for the month name.
	CEMFileNames           []string // List of input files with CEM data (needs to be a whole year's worth of data
	EarthRadius            float64  // in meters
	WPSnamelist            string   // Path to WPS namelist file
	WRFnamelist            string   // Path to WPS namelist file
	OldWRFout              string   // Path to old WRF output files for plume rise calculations.
	//[DOMAIN] and [DATE] can be used as wildcards. If kemit > 1 in the WRF namelist.input file, these files
	// will be needed to calculate plume rise for point sources. Otherwise, these files will not be necessary.
	RegenerateSpatialData bool   // Whether or not to delete surrogates and ShapefileDir generated by previous run and start over. If the model has not been previously run with the current "SimuationName", this does nothing. If the model did not sucessfully complete the last time it was run, the model may crash again if this is set to `false`.
	SimulationName        string // Name for the simulation: user choice
	currentTime           time.Time
	ErrorFlag             bool // whether this sector has already encountered an error
	testMode              bool // testMode is set by a command line flag and is used for testing the program. In ensures that SpecType is set to "mass" and turns off the conversion from VOCs to TOGs so that speciated emissions totals can be compared to total emissions in the inventory
	msgchan               chan string
	slaves                []string // Addresses for available slaves when in distributed mode.

	CheckSrgs bool // Should we check the surrogate shapefiles before running?
}

// PolHolder allows the configuration of chemical speciation settings for individual pollutants.
// Only on of the fields below should be used for each pollutant.
type PolHolder struct {
	SpecType  string                 // The type of speciation that will be applied. Options are "VOC","PM2.5","NOx", and "SOx". If empty, the pollutant will be carried through to the output without speciation, or grouped as if it were the pollutants in "SpecNames".
	SpecNames []string               // Names of pollutants in the SPECIATE database which are equivalent to this pollutant. For records containing this pollutant, the pollutants included in "SpecNames" will be left out of any speciation that occurs to avoid double counting.
	SpecProf  map[string]*SpecHolder // Use this field to directly specify the speciation factors and units.
}

// Period specifies the time period of the emissions data.
type Period int

// The Periods are the months of the year, annual, or Cem which is hourly
// continuous emissions monitoring data.
const (
	Jan Period = iota + 1
	Feb
	Mar
	Apr
	May
	Jun
	Jul
	Aug
	Sep
	Oct
	Nov
	Dec
	Annual
	Cem
)

func (p Period) String() string {
	switch p {
	case Jan:
		return "Jan"
	case Feb:
		return "Feb"
	case Mar:
		return "Mar"
	case Apr:
		return "Apr"
	case May:
		return "May"
	case Jun:
		return "Jun"
	case Jul:
		return "Jul"
	case Aug:
		return "Aug"
	case Sep:
		return "Sep"
	case Oct:
		return "Oct"
	case Nov:
		return "Nov"
	case Dec:
		return "Dec"
	case Annual:
		return "Annual"
	default:
		panic(fmt.Sprintf("unknown period: %d", int(p)))
	}
}

// periodToTimeInterval calculates the start and the end of the given period
// in the given year.
func periodToTimeInterval(p Period, year string) (start, end time.Time, err error) {
	switch p {
	case Jan, Feb, Mar, Apr, May, Jun, Jul, Aug, Sep, Oct, Nov, Dec:
		start, err = time.Parse("Jan 2006", fmt.Sprintf("%s %s", p, year))
		if err != nil {
			return
		}
		if p == Dec {
			var intYear int64
			intYear, err = strconv.ParseInt(year, 0, 32)
			if err != nil {
				return
			}
			nextYear := fmt.Sprintf("%04d", intYear+1)
			end, err = time.Parse("Jan 2006", fmt.Sprintf("Jan %s", nextYear))
			if err != nil {
				return
			}
		} else {
			end, err = time.Parse("Jan 2006", fmt.Sprintf("%s %s", p+1, year))
			if err != nil {
				return
			}
		}
	case Annual:
		start, err = time.Parse("Jan 2006",
			fmt.Sprintf("Jan %s", year))
		if err != nil {
			return
		}
		var intYear int64
		intYear, err = strconv.ParseInt(year, 0, 32)
		if err != nil {
			return
		}
		nextYear := fmt.Sprintf("%04d", intYear+1)
		end, err = time.Parse("Jan 2006", fmt.Sprintf("Jan %s", nextYear))
		if err != nil {
			return
		}
	default:
		panic(fmt.Sprintf("unknown period: %d", int(p)))
	}
	return
}

func (c *Context) getPeriod(t time.Time) Period {
	switch c.InventoryFreq {
	case "annual":
		return Annual
	case "monthly":
		return Period(t.Month())
	default:
		panic("getPeriod error")
	}
}

func (p *ConfigData) setup(e *ErrCat) {
	c := *p
	if c.DefaultSettings.InventoryFreq == "" {
		c.DefaultSettings.InventoryFreq = "annual"
	}
	c.DefaultSettings.OutputType =
		strings.ToLower(c.DefaultSettings.OutputType)
	c.DefaultSettings.setup(e)
	c.DefaultSettings.catPaths(c.Dirs, e)
	if c.DefaultSettings.testMode {
		c.DefaultSettings.SpecType = "mass"
	}
	for sector, _ := range c.Sectors {
		c.Sectors[sector].Sector = sector
		c.Sectors[sector].FillWithDefaults(c.DefaultSettings, e)
		c.Sectors[sector].setup(e)
		c.Sectors[sector].catPaths(c.Dirs, e)
	}
	*p = c
}

// Fill the configuration settings for an individual sector with
// default settings, where `p` represents the settings for the individual
// sector and `d` represents the default settings.
func (p *Context) FillWithDefaults(d *Context, e *ErrCat) {
	c := *p
	c.RunSpeciate = d.RunSpeciate
	c.RunSpatialize = d.RunSpatialize
	c.RunTemporal = d.RunTemporal
	c.StartDate = d.StartDate
	c.EndDate = d.EndDate
	c.Tstep = d.Tstep
	c.OutputType = d.OutputType
	c.TemporalRefFile = d.TemporalRefFile
	c.TemporalProFile = d.TemporalProFile
	c.HolidayFile = d.HolidayFile
	c.EarthRadius = d.EarthRadius
	c.WPSnamelist = d.WPSnamelist
	c.WRFnamelist = d.WRFnamelist
	c.OldWRFout = d.OldWRFout
	c.ShapefileDir = d.ShapefileDir
	d.SimulationName = os.ExpandEnv(d.SimulationName)
	c.SimulationName = d.SimulationName
	c.ForceWesternHemisphere = d.ForceWesternHemisphere
	c.ChemicalMechanism = d.ChemicalMechanism
	c.MechAssignmentFile = d.MechAssignmentFile
	c.MechanismMWFile = d.MechanismMWFile
	c.SpeciesInfoFile = d.SpeciesInfoFile
	if d.TstepsPerFile == 0 {
		d.TstepsPerFile = 24
	}
	c.TstepsPerFile = d.TstepsPerFile
	c.testMode = d.testMode
	if c.SccDesc == "" {
		c.SccDesc = d.SccDesc
	}
	if c.SicDesc == "" {
		c.SicDesc = d.SicDesc
	}
	if c.NaicsDesc == "" {
		c.NaicsDesc = d.NaicsDesc
	}
	if c.SpecRefFile == "" {
		c.SpecRefFile = d.SpecRefFile
	}
	if c.SpecRefComboFile == "" {
		c.SpecRefComboFile = d.SpecRefComboFile
	}
	if c.SpecProFile == "" {
		c.SpecProFile = d.SpecProFile
	}
	if c.SpecType == "" {
		c.SpecType = d.SpecType
	}
	if c.testMode { // for testing do mass speciation so speciated totals match inventory totals.
		c.SpecType = "mass"
	}
	if c.PolsToKeep == nil {
		c.PolsToKeep = d.PolsToKeep
	}
	if c.GridRefFile == "" {
		c.GridRefFile = d.GridRefFile
	}
	if c.SrgSpecFile == "" {
		c.SrgSpecFile = d.SrgSpecFile
	}
	if c.InputUnits == "" {
		c.InputUnits = d.InputUnits
	}
	if c.InputProj4 == "" {
		c.InputProj4 = d.InputProj4
		if d.InputProj4 == "" {
			e.Add(fmt.Errorf("InputProj4 is unspecified"))
		}
	}
	var err error
	c.inputSr, err = proj.Parse(c.InputProj4)
	e.Add(err)
	c.slaves = d.slaves
	*p = c
}

// StartTime returns the start time of the simulation as specified in the Context
// field `StartDate`, which should be in the format "2006/01/02" and
// may contain environment variables.
func (c *Context) StartTime() (time.Time, error) {
	return time.Parse("2006/01/02", os.ExpandEnv(c.StartDate))
}

// EndTime returns the end time of the simulation as specified in the Context
// field `EndDate`, which should be in the format "2006/01/02" and
// may contain environment variables.
func (c *Context) EndTime() (time.Time, error) {
	return time.Parse("2006/01/02", os.ExpandEnv(c.EndDate))
}

// TimeStep returns the interval between output emissions records
// as specified in the Context field `Tstep`.
func (c *Context) TimeStep() (time.Duration, error) {
	return time.ParseDuration(c.Tstep)
}

func (p *Context) setup(e *ErrCat) {
	c := *p
	var err error
	c.startDate, err = time.Parse("2006/01/02", os.ExpandEnv(c.StartDate))
	if err != nil {
		panic(err)
	}
	e.Add(err)
	c.currentTime = c.startDate
	c.endDate, err = time.Parse("2006/01/02", os.ExpandEnv(c.EndDate))
	e.Add(err)
	c.tStep, err = time.ParseDuration(c.Tstep)
	e.Add(err)
	if c.InventoryFreq != "annual" && c.InventoryFreq != "monthly" && c.InventoryFreq != "cem" {
		e.Add(fmt.Errorf("In sector " + c.Sector + ", " + string(c.InventoryFreq) +
			" is not a valid value for variable inventoryFreq. Please choose " +
			"either `annual', `monthly', or `cem'."))
	}
	switch c.InventoryFreq {
	case "annual":
		c.RunPeriods = []Period{Annual}
	case "cem":
		c.RunPeriods = []Period{Cem}
	case "monthly":
		c.RunPeriods = make([]Period, 0)
		for m := int(c.startDate.Month()); m <= int(c.endDate.Month()); m++ {
			// period 1 is january
			c.RunPeriods = append(c.RunPeriods, Period(m))
		}
	}
	// Fill in information about ending time (varies by output file format)
	if c.OutputType == "camx" || c.OutputType == "wrf" {
		d, err := time.ParseDuration("24h")
		e.Add(err)
		c.endDate = c.endDate.Add(d)
	} else if c.OutputType == "cmaq" {
		d, err := time.ParseDuration("25h")
		e.Add(err)
		c.endDate = c.endDate.Add(d)
	} else if c.OutputType != "none" {
		e.Add(fmt.Errorf("OutputType " + c.OutputType + " unknown"))
	}
	*p = c
}

// Replace directory and file names with full paths
func (p *Context) catPaths(d *DirInfo, e *ErrCat) {
	c := *p

	d.Output = strings.Replace(strings.Replace(strings.Replace(
		d.Output, "[Home]", d.Home, -1),
		"[input]", d.Input, -1),
		"[Ancilliary]", d.Ancilliary, -1)
	d.Output = os.ExpandEnv(d.Output)
	err := os.MkdirAll(d.Output, os.ModePerm)
	if err != nil {
		e.Add(fmt.Errorf("Unable to make output directory `%v'.",
			d.Output))
	}
	c.outputDir = d.Output
	if c.OutputType != "none" {
		path := filepath.Join(d.Output, c.OutputType)
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			e.Add(fmt.Errorf("Unable to make AQM output directory `%v'.",
				d.Output))
		}
	}
	d.GriddedSrgs = strings.Replace(strings.Replace(strings.Replace(
		strings.Replace(d.GriddedSrgs, "[Home]", d.Home, -1),
		"[input]", d.Input, -1),
		"[Ancilliary]", d.Ancilliary, -1),
		"[Output]", d.Output, -1)
	d.GriddedSrgs = os.ExpandEnv(d.GriddedSrgs)
	err = os.MkdirAll(d.GriddedSrgs, os.ModePerm)
	if err != nil {
		e.Add(fmt.Errorf("Unable to make gridded surrogates directory `%v'.",
			d.GriddedSrgs))
	}
	c.GriddedSrgs = d.GriddedSrgs

	paths := []*string{&d.Home, &d.Input, &d.Ancilliary, &d.Shapefiles,
		&c.SpecRefFile, &c.SpecRefComboFile, &c.SpecProFile,
		&c.SccDesc, &c.SicDesc, &c.NaicsDesc,
		&c.GridRefFile, &c.TemporalRefFile, &c.TemporalProFile,
		&c.HolidayFile, &c.SrgSpecFile, &c.WPSnamelist, &c.WRFnamelist,
		&c.OldWRFout, &c.MechAssignmentFile, &c.MechanismMWFile,
		&c.SpeciesInfoFile}
	varnames := []string{"Home", "Input", "Ancilliary", "Shapefiles",
		"SpecRefFile", "SpecRefComboFile", "SpecProFile",
		"SccDesc", "SicDesc", "NaicsDesc",
		"GridRefFile", "TemporalRefFile", "TemporalProFile", "HolidayFile",
		"SrgSpecFile", "WPSnamelist", "WRFnamelist", "OldWRFout",
		"MechAssignmentFile", "MechanismMWFile", "SpeciesInfoFile"}

	for i, path := range paths {
		d.expand(path)
		if strings.Index(*path, "[DOMAIN]") == -1 &&
			strings.Index(*path, "[DATE]") == -1 {
			e.statOS(*path, varnames[i])
		}
	}
	c.ShapefileDir = d.Shapefiles
	if c.Sector != "" {
		for i, _ := range c.InvFileNames {
			d.expand(&c.InvFileNames[i])
			if c.InventoryFreq == "annual" || c.InventoryFreq == "cem" {
				e.statOS(c.InvFileNames[i], c.Sector+" Inventory file")
			} else if c.InventoryFreq == "monthly" {
				for _, month := range strings.Split(months, " ") {
					e.statOS(strings.Replace(
						c.InvFileNames[i], "[month]", month, -1),
						c.Sector+" Inventory file")
				}
			} else {
				panic(fmt.Sprintf("Unknown InventoryFreq %v.", c.InventoryFreq))
			}
		}
		for i, _ := range c.CEMFileNames {
			d.expand(&c.CEMFileNames[i])
			e.statOS(c.CEMFileNames[i], c.Sector+" CEM file")
		}
	}
	*p = c
	return
}

func (d *DirInfo) expand(path *string) {
	*path = strings.Replace(*path, "[Home]", d.Home, -1)
	*path = strings.Replace(*path, "[Output]", d.Output, -1)
	*path = strings.Replace(*path, "[Input]", d.Input, -1)
	*path = strings.Replace(*path, "[Ancilliary]", d.Ancilliary, -1)
	*path = os.ExpandEnv(*path)
}

func (c *Context) MessageChan() chan string {
	if c.msgchan == nil {
		c.msgchan = make(chan string, 1)
	}
	return c.msgchan
}
