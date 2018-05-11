# **A**ir **E**missions **P**rocessor program

AEP is a program designed to ingest data collected during national emissions inventories and process it for use in air quality models by breaking up emissions into detailed chemical groups, spatially allocating the emissions to a grid or other set of shapes, and then temporally allocating the emissions to specific times of the year.

The program is designed to more or less reproduce the functionality of the [SMOKE](http://www.cmascenter.org/smoke/) model, but with a focus on usability, flexibility, and expandability. This model differs from the SMOKE model in several ways:

* AEP is a single self-contained model, rather than a set of executables linked by shell scripts. This makes it much easier to use.
* AEP can process all of the sectors of a national emissions inventory simultaneously, based on a single configuration file, instead of requiring a custom set of shell scripts for each sector. This makes the program much easier to use and reduces the opportunity for configuration errors.
* AEP's spatial surrogate generator is integrated into the program and generates surrogates automatically, instead of requiring a completely separate program to generate spatial surrogates. This greatly reduces the time and effort required to produce emissions for a new model domain.
* In AEP, the spatial domain is set up automatically based on WRF `namelist.input` and `namelist.wps` files, and meteorology information for plume rise is read directly from WRF output files from a previous run. This avoids the need for a seperate meteorology preprocesser and a multiple spatial domain configuration files in different formats.
* AEP extracts chemical speciation information directly from the [SPECIATE](http://www.epa.gov/ttnchie1/software/speciate/) database, eliminating the need for a separate program to create speciation files and greatly reducing the effort required to change the chemical speciation mechanism used when processing emissions.
* AEP outputs emissions information directly to the WRF/Chem file format; other file formats can be added.
* AEP is designed to take advantage of multiprocessor computers, with automatic shared-memory concurrancy.

## Installation

1. Install the [Go compiler](http://golang.org/doc/install). Make sure you install the correct version (32 or 64 bit) for your system. It is recommended to install the compiler to the default location; if you experience problems try seeing if they are resolved by installing the compiler to the default location.

2. Download and install the software library and utilities:

		go get github.com/spatialmodel/inmap/emissions/aep/...
	The Go language has an automatic system for finding and installing library dependencies; you may want to refer [here](http://golang.org/doc/code.html) to understand how it works.

## Use

1. Obtain the necessary emissions data and ancilliary information. Information for obtaining 2014 US National Emissions Inventory data is available [here](data/nei2014). In addition to the changes to the data suggested in the README file in that directory, the road shapefile for spatial surrogates is misaligned and emissions from commercial cooking in New York State appear to be unreasonably high.

2. Process the data. Although AEP is not currently available as an executable program, the full API is described [here](https://godoc.org/github.com/spatialmodel/inmap/emissions/aep) and a simplified API for common tasks is described [here](https://godoc.org/github.com/ctessum/spatialmodel/inmap/emissions/aeputil). More extensive documentation is not yet available, but an example of spatially processing annual total emissions is available [here](aeputil/scale_test.go).


### TODO (Things that SMOKE can do that AEP cannot)

* Add capability to process meteorology-dependent emissions (e.g., vehicle tailpipe, wood smoke, road dust)

* Add capability to integrate with the MOVES vehicle emissions model.
