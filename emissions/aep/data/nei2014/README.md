# Retrieving the 2014 NEI data

Data for the air quality modeling version of the U.S. EPA's 2014 National emissions inventory is available for download from an [EPA FTP server](ftp://ftp.epa.gov/EmisInventory/2014platform/v1/). A description of the included data is available [here](ftp://ftp.epa.gov/EmisInventory/2014platform/v1/README_2014v1_nata_package.txt).

A version of the 2014 NEI data that has successfully been used with AEP is archived here: https://zenodo.org/record/3237211#.XPSHl7zYqto.
Alternatively, this repository includes a script—```download.go```—that downloads the data and preparing it for use. After [installing the Go language compiler](https://golang.org/doc/install), the script can be run with the command ```go run download.go -dir="/path/to/download"``` where ```/path/to/download``` is the location of the directory where the data should be downloaded to. Downloading the data may take a while.

This repository also includes the additional file `surrogate_specification_2014.csv`. This file is combined and edited version of surrogate specification files that can be downloaded from the FTP site which has been edited to replace missing shapefiles with existing replacements and combine US, Canada, and Mexico surrogates in one place. Improvements to this file or advice regarding the locations of the missing files are welcome.

Finally, this directory in includes a configuration file—```cstref_2014.toml```—that specifies how the files can be used to processed the 2014 NEI. The configuration file assumes that a ```$nei2014Dir``` environment variable has been set to the directory where the data files were downloaded to (```/path/to/download``` in the example above).

## Required manual changes

After running the ```download.go``` script, some additional changes need to be made manually:

* The following line should be added to ```ge_dat/gridding/mgref_onroad_us_2014platform_03oct2016_nf_v2.txt```:
```
000000;2201610080;222
000000;2202420080;222
000000;2202210080;239
000000;2202310080;239
000000;2201320080;239
000000;2205000062;239
000000;2202520080;222
000000;2205320080;241
000000;2205210080;239
000000;2201420080;222
000000;2201510080;242
000000;2202430080;202
000000;2202510080;201
000000;2201520080;222
000000;2201540080;239
000000;2202620080;244
000000;2201430080;201
000000;2202610080;222
000000;2201530080;244
000000;2202320080;244
000000;2201000062;239
000000;2201210080;239
000000;2202410080;244
000000;2205310080;239
000000;2203420080;222
000000;2202530080;244
000000;2202540080;239
000000;2201110080;239
000000;2201310080;239
000000;2202000062;239
```
* Delete the line starting with ```COUNTRY_CD``` from ```SmokeFlatFile_ONROAD_20160910.csv```.

* Delete the leading "1" from each record in the ```PRUID``` attribute column of the shapefile ```$nei2014Dir/Canada_2010_surrogate_v1/Non_NAESI/SHAPEFILE/pr2001ca_regions_simplify.shp```. So ```159000``` should become ```59000```.


* The Canadian census division file that comes with the data (```Canada_2010_surrogate_v1/NAESI/SHAPEFILE/gisnodat.shp```) is unnecessarily large (making it unnecessarily difficult to create surrogates) and the ID codes have the same problem as listed above. To fix this, the download script will download an alternative shapefile: ```lcd_000b16a_e.shp```. Before this shapefile can be used however, you need to add an additional attribute column called ```FIPS``` that consists of the first two characters of the attribute ```CDUID```, then a zero, then the final two characters of ```CDUID```. In QGIS, this can be done in the "field calculator" with the expression: ```concat(substr(CDUID,1,2),'0',substr(CDUID,3,2))```.

* In the directory `$nei2014Dir/2014fa_nata_cb6cmaq_14j/inputs`, run the command `find . -type f -name "*" -print0 | xargs -0 sed -i '' -e 's/PM25-PRI/PM2_5/g'`. This will replace all instances of `PM25-PRI` with `PM2_5`. This is necessary because there are no speciation profiles for `PM25-PRI`.

* Download the SPECIATE database from [here](https://www.epa.gov/air-emissions-modeling/speciate-version-45-through-40) and save the `GAS_PROFILE`, `GAS_SPECIES`, `OTHER_GASES_SPECIES`, `PM_SPECIES`, and `SPECIES_PROPERTIES` tables as CSV files in the `$nei2014Dir/speciation` directory.

* Download [this file](http://www.cert.ucr.edu/~carter/emitdb/SpecDB.zip) from http://www.cert.ucr.edu/~carter/emitdb/ and copy the files `MechAsn.csv` and `SpeciesTable.csv` from the `SpTool` directory of the downloaded file to `$nei2014Dir/speciation`. There is also a file called `MechMW.csv` that lists the molar weights of the chemical mechanism species. This file is required but I am currently unsure of where it can be downloaded from.

* Add the following lines to the end of `$nei2014Dir/speciation/OTHER_GASES_SPECIES.csv`:
```
6755,2606,HONO,,Gas,0.092,,,,,,,
6756,2605,HONO,,Gas,0.9,,,,,,,
6757,2607,HONO,,Gas,0.008,,,,,,,
6758,2606,NHONO,,Gas,0.1,,,,,,,
6759,2605,NHONO,,Gas,0.9,,,,,,,
```
