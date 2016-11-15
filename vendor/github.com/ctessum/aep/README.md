# **A**ir **E**missions **P**rocessor program

AEP is a program designed to ingest data collected during national emissions inventories and process it for use in air quality models by breaking up emissions into detailed chemical groups, spatially allocating the emissions to a grid or other set of shapes, and then temporally allocating the emissions to specific times of the year.

The program is designed to more or less reproduce the functionality of the [SMOKE](http://www.cmascenter.org/smoke/) model, but with a focus on usability, flexibility, and expandability. This model differs from the SMOKE model in several ways:

* AEP is a single self-contained model, rather than a set of executables linked by shell scripts. This makes it much easier to use.
* AEP can process all of the sectors of a national emissions inventory simultaneously, based on a single configuration file, instead of requiring a custom set of shell scripts for each sector. This makes the program much easier to use and reduces the opportunity for configuration errors.
* AEP's spatial surrogate generator is integrated into the program and generates surrogates automatically, instead of requiring a completely separate program to generate spatial surrogates. This greatly reduces the time and effort required to produce emissions for a new model domain.
* In AEP, the spatial domain is set up automatically based on WRF `namelist.input` and `namelist.wps` files, and meteorology information for plume rise is read directly from WRF output files from a previous run. This avoids the need for a seperate meteorology preprocesser and a multiple spatial domain configuration files in different formats.
* AEP extracts chemical speciation information directly from the [SPECIATE](http://www.epa.gov/ttnchie1/software/speciate/) database, eliminating the need for a separate program to create speciation files and greatly reducing the effort required to change the chemical speciation mechanism used when processing emissions.
* AEP outputs emissions information directly to the WRF/Chem file format; other file formats can be added.
* AEP is designed to take advantage of multiprocessor computers, with automatic concurrancy on a single machine and the option of using a distributed cluster of computers to generate spatial surrogates.

## Installation

1. Install the [Go compiler](http://golang.org/doc/install). Make sure you install the correct version (32 or 64 bit) for your system. It is recommended to install the compiler to the default location; if you experience problems try seeing if they are resolved by installing the compiler to the default location. Also make sure to set the [`$GOPATH`](http://golang.org/doc/code.html#GOPATH) environment variable.

2. Download and install the main program:

		go get github.com/ctessum/aep
	The Go language has an automatic system for finding and installing library dependencies; you may want to refer [here](http://golang.org/doc/code.html) to understand how it works.

## Use

1. Obtain the necessary shapefiles for spatial allocation. For the 2005 NEI, most of them can be obtained by running the script at [`$GOPATH/src/github.com/ctessum/aep/scripts/downloadShapefiles.sh`](src/default/scripts/downloadShapefiles.sh). The default download location is `$GOPATH/src/github.com/ctessum/aep/test/inputdata/shapefiles`. Otherwise, most of them are [here](ftp://ftp.epa.gov/EmisInventory/emiss_shp2003/us/). The script at  will download these files automatically, plus add the missing .prj files.
	* You may additionally need the shapefiles [here](https://bitbucket.org/ctessum/aep/downloads/additionalShapefiles.tar.gz). These will need to be downloaded manually.

2. Obtain the input emissions data. For the 2005 NEI, you can download them from [here](ftp://ftp.epa.gov/EmisInventory/2005v4_2/2005emis) and extract them into `$GOPATH/src/github.com/ctessum/aep/test/inputdata/emissions`.
	The file locations will need to match the locations specified in the configuration file described below.


2. Run the program:

		aep -config=$GOPATH/src/github.com/ctessum/aep/scripts/config2005nei.json
	While the program is running, you can open a web browser and navigate to `localhost:6060` to view status and diagnostic information.
	After the program finishes running, output can be found in the directory

		$GOPATH/src/github.com/ctessum/aep/test/Output
	After running the default scenario, you can edit the configuration file to your specific needs by setting it to point to the `namelist.input` and `namelist.wps` files for your WRF domain, and changing the starting and ending dates. Additionally, the WRF output files from a previous simulation which are required to calculate emissions plume rise, which is not calculated in the default scenario.


## Additional information

### Configuration file

The configuration file needs to be a valid [json](http://en.wikipedia.org/wiki/JSON) file with the following format:

	{
		"Dirs": {
			DirInfo
		},
		"DefaultSettings": {
			RunData
		},
		"sectors": {
			"sectorname1": {
				RunData
			},
			"sectorname2": {
				RunData
			}
		}
	}
Refer directly to the source code documentation for the fields that make up the
[DirInfo](http://godoc.org/bitbucket.org/ctessum/aep/lib.aep#DirInfo)
and [RunData](http://godoc.org/bitbucket.org/ctessum/aep/lib.aep#RunData) data holders.

Within the configuration file, you can use the variables `[Home]`, `[Input]`, and `[Ancilliary]` to refer to the directories specified in the `Dirs` section of the file.You can also use environment variables such as `$GOPATH`. When specifiying the locations of the `OldWRFout` files, you can use the variables `[DATE]` and `[DOMAIN]` which will be replaced with the relevant dates and domains while the program is being run. In the emissions files, you can use the variable [month] to represent the current month.

Some of the fields in the configuration file have automatic default values associated with them. Additionally, some can only be specified in the `DefaultSettings` section of the file; for these variables, settings specified for individual sectors will be ignored. Refer to the [source code](aep/src/default/lib.aep/configure.go#cl-177) to further understand this behavior.

### API

The AEP package is split into an executable program and an application programming interface (API). To see what capabilities are available in the API refer to the online [source code documentation](http://godoc.org/github.com/ctessum/aep).


to see the functions available in the API and investigate how they work.


### TODO (Things that SMOKE can do that AEP cannot)

* Add capability to process meteorology-dependent emissions (e.g., vehicle tailpipe, wood smoke, road dust)

* Add capability to integrate with the MOVES vehicle emissions model.
