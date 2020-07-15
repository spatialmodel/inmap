---
id: emissions
title: Emissions
sidebar_label: Emissions
---

## InMAP emissions

Any InMAP simulation will require input emissions, which will need to be specified by the user.
InMAP is a steady-state, annual average model, so it requires annual-total or annual-average emissions to be input.
There are currently three ways to specify emissions, which we will describe here.
Users can use any combination of these three methods to specify emissions for a simulation, and all specified emissions will be combined together.
The examples below all assume that configuration is specified in a configuration file, but command line arguments or environment variables can be used with minimal changes as described [here](run_config.html).

Fully-working example configuration files with different emissions types are available [here](https://github.com/spatialmodel/inmap/tree/master/cmd/inmap)

### Emissions shapefiles

One way to specify emissions is to use [shapefiles](https://en.wikipedia.org/wiki/Shapefile).
Within the shapefiles, emissions can be expressed as point, line, or polygon geometries.
Users can specify a list of more than one shapefile, and emissions from all specified shapefiles will be included together in a single simulation (rather than running separate simulations for each shapefile).
An example of how to specify shapefile emissions is below.

``` toml
EmissionsShapefiles = [
  "emissions/emis1.shp",
  "emissions/emis2.shp"
]
EmissionUnits = "tons/year"
```

Within the shapefiles, emissions of different pollutants are specified using attribute columns with names `VOC`, `NOx`, `NH3`, `SOx`, and `PM2_5`.
Files with elevated emissions need to have attribute columns labeled `height`, `diam`, `temp`, and `velocity` containing stack information in units of m, m, K, and m/s, respectively. (Shapefiles without these attribute columns will be assumed to contain ground-level emissions only.)
Emissions will be allocated from the geometries in the shapefile to the InMAP computational grid, so users do not need ensure that emissions geometries or spatial projections match that of the InMAP grid.
`EmissionUnits` gives the units that the input emissions are in.
Acceptable values are 'tons/year', 'kg/year', 'ug/s', and 'μg/s'.

### SMOKE-formatted emissions

A second way of specifying emissions is using [SMOKE](https://www.cmascenter.org/smoke/)-formatted emissions files.
Most formats included in the [SMOKE user manual](https://www.cmascenter.org/smoke/documentation/4.6/html/ch08s02.html) are supported, including IDA, ORL, and FF10.
As with emissions shapefiles, InMAP will only use emissions records with pollutants named `VOC`, `NOx`, `NH3`, `SOx`, or `PM2_5`, so in many cases emissions files obtained from govenment agencies may need to be modified to change pollutant names before they are used in InMAP.
An example of how to specify SMOKE-formatted emissions is below.

``` toml
[aep]
SrgSpecSMOKE = "srgspec.txt"
GridRef = ["gridref.txt"]

[aep.InventoryConfig]
  InputUnits = "tons"
[aep.InventoryConfig.NEIFiles]
  onroad = ["file1.orl.txt", "file2.orl.txt"]
  egu = ["file3.ff10.txt", "file4.orl.txt"]

[aep.SpatialConfig]
  InputSR = "+proj=longlat"
```

`aep.InventoryConfig.NEIFiles` specifies the emissions files that should be read in.
Users can specify multiple sectors ("onroad" and "egu" above) and multiple emissions files for each sector.
All emissions files will be combined in a single simulation.
`InputUnits` specifies the emissions units.
Acceptable values are 'tons', 'tonnes', 'kg', 'g', and 'lbs'.
Emissions are assumed to be per year or per month, depending on the type of SMOKE emissions file that is used.

`GridRef` specifies a path to a file that matches emissions records to spatial surrogates. The GridRef file is specified in the [SMOKE manual](https://www.cmascenter.org/smoke/documentation/4.6/html/ch08s04s03.html).

`SrgSpecSMOKE` and `SrgSpecOSM` specify paths to files that match spatial surrogate codes to the spatial information used to create spatial surrogates. Two file formats can be used either `SrgSpecSMOKE` for a [SMOKE-formatted surrogate specification file](https://raw.githubusercontent.com/spatialmodel/inmap/master/emissions/aep/data/nei2014/surrogate_specification_2014.csv) or `SrgSpecOSM` for a surrogate specification file matching [this format](https://github.com/spatialmodel/inmap/blob/master/emissions/aep/testdata/srgspec_osm.json) that uses [OpenStreetMap](https://www.openstreetmap.org/) data to create spatial surrogates.

If the fields for `GridRef` and `SrgSpec` are left empty, then emissions records will be allocated directly to the InMAP grid: no spatial surrogates will be used.
Here is an example of the same configuration above, but changed so that no spatial surrogates are used:

``` toml
[aep]
SrgSpecSMOKE = ""
GridRef = []

[aep.InventoryConfig]
  InputUnits = "tons"
[aep.InventoryConfig.NEIFiles]
  onroad = ["file1.orl.txt", "file2.orl.txt"]
  egu = ["file3.ff10.txt", "file4.orl.txt"]

[aep.SpatialConfig]
  InputSR = "+proj=longlat"
```

`InputSR` specifies the spatial reference for the input emissions in [PROJ4](https://proj.org/) format. `InputSR = "+proj=longlat"` means that emissions with point coordinates are in latitude/longitude format.

### COARDS-formatted emissions

The third option for inputting emissions is to used [COARDS](https://ferret.pmel.noaa.gov/Ferret/documentation/coards-netcdf-conventions)-formatted [NetCDF](https://www.unidata.ucar.edu/software/netcdf/) files.
Specifying COARDS emissions files is almost the same as specifying SMOKE-formatted files, but `[aep.InventoryConfig.NEIFiles]` is replaced with `[aep.InventoryConfig.COARDSFiles]` and an additional `COARDSYear` variable (which specifies the emissions year) is required as shown below:

``` toml
[aep]
SrgSpecSMOKE = ""
GridRef = []

[aep.InventoryConfig]
  InputUnits = "tons"
  COARDSYear = 2016
[aep.InventoryConfig.COARDSFiles]
  onroad = ["emis_coards_onroad.nc"]
  egu = ["emis_coards_egu.nc"]

[aep.SpatialConfig]
  InputSR = "+proj=longlat"
```

Spatial surrogates can also be used with COARDS files as described above.

Currently, only NetCDF version 3 files are supported; the more recent version 4 is not supported.

Within each NetCDF file, InMAP will read emissions from all floating point variables that have dimensions `[lat, lon]` and are named one of `VOC`, `NOx`, `NH3`, `SOx`, or `PM2_5`.
Emisssions are assumed to be annual totals.
