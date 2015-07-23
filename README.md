# (In)tervention (M)odel for (A)ir (P)ollution

[![Build Status](https://travis-ci.org/ctessum/inmap.svg?branch=master)](https://travis-ci.org/ctessum/inmap) [![Coverage Status](https://img.shields.io/coveralls/ctessum/inmap.svg)](https://coveralls.io/r/ctessum/inmap) [![GoDoc](http://godoc.org/github.com/ctessum/inmap/lib.inmap?status.svg)](http://godoc.org/github.com/ctessum/inmap/lib.inmap)

This program is still being developed and tested. As such, features and functionality may change without warning.

## About InMAP

InMAP is a multi-scale emissions-to-health impact model for fine particulate matter (PM<sub>2.5</sub>) that mechanistically evaluates air quality and health benefits of perturbations to baseline emissions. The main simplification of InMAP compared to a comprehensive chemical transport model is that it does so on an annual-average basis rather than the highly time-resolved performance of a full CTM. The model incorporates annual-average parameters (e.g. transport, deposition, and reaction rates) from the WRF/Chem chemical transport model. Grid-cell size varies as shown in Figure 1, ranging from smaller grid cells in urban areas to larger grid cells in rural areas; any grid cell above a specified population threshold is subdivided until no grid larger than 1 km has >10,000 people. This variable resolution grid is used to simulate population exposures to PM<sub>2.5</sub> with high spatial resolution while minimizing computational expense.

![alt tag](fig1.png?raw=true)
Figure 1: InMAP spatial discretization of the model domain into variable resolution grid cells. Left panel: full domain; right panel: a small section of the domain centered on the city of Los Angeles.

## Installation

This program should work on most types of computers. Refer [here](http://golang.org/doc/install#requirements) for a list of theoretically supported systems.

1. Install the [Go compiler](http://golang.org/doc/install). Make sure you install the correct version (32 or 64 bit) for your system. Also make sure to set the [`$GOPATH`](http://golang.org/doc/code.html#GOPATH) environment variable to a *different directory* than the `$GOROOT` environment variable (it can also not be a subdirectory of `$GOROOT`). It may be useful to go through one of the tutorials to make sure the compiler is correctly installed.

2. Make sure your `$PATH` environment variable includes the directories `$GOROOT/bin` and `$GOPATH/bin`. On Linux or Macintosh systems, this can be done using the command `export PATH=$PATH:$GOROOT/bin:$GOPATH/bin`. On Windows systems, you can follow [these](http://www.computerhope.com/issues/ch000549.htm) directions.

3. Install the [git](http://git-scm.com/) and [mercurial](http://mercurial.selenic.com/) version control programs, if they are not already installed. If you are using a shared system or cluster, you may just need to load them with the commands `module load git` and `module load hg`.

4. Download and install the main program:

		go get github.com/ctessum/inmap/inmap
	The Go language has an automatic system for finding and installing library dependencies; you may want to refer [here](http://golang.org/doc/code.html) to understand how it works.

## Running InMAP

1. Download the general input files (`InMAPData`) from and example emissions files from [here](https://github.com/ctessum/inmap/releases/download/v1.0.0/inmapData_48_24_12_4_2_1_40000.zip). For the InMAPData files, there are several options of computational grids varying numbers of high- and low-resolution grid cells. The example emissions data is a scenario where all of the pollutant emissions in the U.S., southern Canada, and Northern Mexico increase by 1% over their 2005 levels.

1. Make a copy of the [configuration file template](inmap/configExample.json) and edit it so that the `InMAPdataTemplate` and `EmissionsShapefiles` variables point to the locations where you downloaded the general input and emissions files to, and so the `OutputTemplate` variable points to the desired location for the output files. The wildcard `[layer]` is a place holder for the vertical layer number. (Input and output data are separated into individual files by model layer). Refer directly to the source code ([here](inmap/inmap.go#cl-22)) for information about other configuration options.

2. Run the program:

		inmap -config=/path/to/configfile.json
	While the program is running, you can open a web browser and navigate to `localhost:8080` to view status and diagnostic information.

3. View the output program output. The output files are in [shapefile](http://en.wikipedia.org/wiki/Shapefile) format which can be viewed in most GIS programs. One free GIS program is [QGIS](http://www.qgis.org/). Output from each model layer is put into a separate file. Layer 0 is the one closest to the ground and will probably be of the most interest. By default, the InMAP only outputs results from layer zero, but this can be changed using the configuration file.
  The output shapefiles have a number of attribute columns. They include:
    * Pollutant concentrations in units of μg m<sup>-3</sup>:
      * VOC (`VOC`)
      * NO<sub>x</sub> (`NOx`)
      * NH<sub>3</sub> (`NH3`)
      * SO<sub>x</sub> (`SOx`)
      * Total PM<sub>2.5</sub> (`TotalPM2_5`; The sum of all PM<sub>2.5</sub> components)
      * Particulate sulfate (`pSO4`)
      * Particulate nitrate (`pNO3`)
      * Particulate ammonium (`pNH4`)
      * Secondary organic aerosol (`SOA`)
    * Populations of different demographic subgroups in units of people per grid cell. The included populations may vary but in the default dataset as of this writing the groups included are:
      * total population (`TotalPop`)
      * people identifying as black (`Black`), asian  (`Asian`), latino (`Latino`), native american or american indian (`Native`), non-latino white (`WhiteNoLat`) and everyone else (`Other`).
      * People living below the poverty time (`Poverty`) and people living at more than twice the poverty line (`TwoXPov`).
    * Numbers of deaths attributable to PM<sub>2.5</sub> in each of the populations in units of deaths/year. Attribute names in shapfiles are limited to 11 characters, so, for example, deaths in the `TotalPop` population would be labeled `TotalPop de`, deaths in the `Black` population would be labeled `Black death`, and—interestingly—deaths in the `WhiteNoLat` population would be labeled `WhiteNoLat_1`.
    * Baseline mortality rate in units of deaths per year per 100,000 people, which can be used for performing alternative health impact calculations.

3. Create your own emissions scenarios. Emissions files can be in [shapefile](http://en.wikipedia.org/wiki/Shapefile) format where the attribute columns correspond to the names of emitted pollutants. Refer [here](http://godoc.org/github.com/ctessum/inmap#pkg-variables) (the `EmisNames` variable) for acceptable pollutant names. Emissions should be in units of short tons per year. The model can handle multiple input emissions files, and emissions can be either elevated or ground level. Files with elevated emissions need to have attribute columns labeled "height", "diam", "temp", and "velocity" containing stack information in units of m, m, K, and m/s, respectively.

	Emissions will be allocated from the geometries in the shape file to the InMAP computational grid, but currently the mapping projection of the shapefile must be the same as the projection InMAP uses. In ESRI format, this projection is:

		PROJCS["Lambert_Conformal_Conic_2SP",GEOGCS["GCS_unnamed ellipse",
		DATUM["D_unknown",SPHEROID["Unknown",6370997,0]],PRIMEM["Greenwich",0],
		UNIT["Degree",0.017453292519943295]],PROJECTION["Lambert_Conformal_Conic_2SP"],
		PARAMETER["standard_parallel_1",33],PARAMETER["standard_parallel_2",45],
		PARAMETER["latitude_of_origin",40],PARAMETER["central_meridian",-97],
		PARAMETER["false_easting",0],PARAMETER["false_northing",0],UNIT["Meter",1]]



## API

The InMAP package is split into an executable program and an application programming interface (API). The documentation [here](http://godoc.org/github.com/ctessum/inmap) shows the functions available in the API and how they work.
