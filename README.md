# (In)tervention (M)odel for (A)ir (P)ollution

This program is still being developed and tested. As such, features and functionality may change without warning. Please do not distribute the program in its current form or assume any results that it gives are correct.

## Installation

This program should work on most types of computers. Refer [here](http://golang.org/doc/install#requirements) for a list of theoretically supported systems.

1. Install the [Go compiler](http://golang.org/doc/install). Make sure you install the correct version (32 or 64 bit) for your system. Also make sure to set the [`$GOPATH`](http://golang.org/doc/code.html#GOPATH) environment variable to a different directory than the `$GOROOT` environment variable. It may be useful to go through one of the tutorials to make sure the compiler is correctly installed.

2. Install the [git](http://git-scm.com/) and [mercurial](http://mercurial.selenic.com/) version control programs, if they are not already installed.

3. Download and install the main program:

		go get bitbucket.org/ctessum/inmap
	The Go language has an automatic system for finding and installing library dependencies; you may want to refer [here](http://golang.org/doc/code.html) to understand how it works.

## Running InMAP

1. Download the general input files from [here](https://bitbucket.org/ctessum/inmap/downloads/InMAPdata_1km_50000.zip) and the example emissions files from [here](https://bitbucket.org/ctessum/inmap/downloads/exampleEmissions.zip).

1. Make a copy of the [configuration file template](src/default/configExample.json) and edit it so that the `InMAPdataTemplate`, `GroundLevelEmissions`, and `ElevatedEmissions` variables point to the locations where you downloaded the general input and emissions files to, and so the `OutputTemplate` variable points to the desired location for the output files. The wildcard `[layer]` is a place holder for the vertical layer number. (Input and output data are separated into individual files by model layer). Refer directly to the source code ([here](src/default/inmap.go#cl-22)) for information about other configuration options.

2. Run the program:

		inmap -config=/path/to/configfile.json 
	While the program is running, you can open a web browser and navigate to `localhost:8080` to view status and diagnostic information.

3. View the output program output. The output files are in [GeoJSON](http://en.wikipedia.org/wiki/GeoJSON) format which can be viewed in GIS programs including [QGIS](http://www.qgis.org/). Output from each model layer is put into a separate file. Layer 0 is the one closest to the ground and will probably be of the most interest.

3. Create your own emissions scenarios:

	Emissions files are .csv files, where the rows correspond to the grid cells in [this](https://bitbucket.org/ctessum/inmap/downloads/gridShape_1km_50000.zip) shapefile, and the columns correspond to the names of emitted pollutants. Refer [here](src/default/lib.inmap/run.go#cl-38) (the `EmisNames` variable) for acceptable pollutant names. Emissions should be in units of short tons per year. Currently, the program accepts two emissions files: one for ground-level emissions and one for elevated emissions. The program currently assumes elevated emissions are emitted from stacks with the stack parameters shown [here](src/default/inmap.go#cl-56). 

	Emissions should be in units of short tons per year.

	Future versions of the program may accept emissions in the form of a shape file, and then allocate the emissions to the grid automatically, and may allow elevated emissions with varying stack parameters.
	

## API

The InMAP package is split into an executable program and an application programming interface (API). To see what capabilities are available in the API, you can start a `godoc` server:
	
	godoc -http=:8080

and then open a web browser and navigate to 

	http://localhost:8080/pkg/bitbucket.org/ctessum/inmap/lib.inmap/
to see the functions available in the API and investigate how they work.

