## inmap preproc

Preprocess CTM output

### Synopsis


preproc preprocesses chemical transport model
output as specified by information in the configuration
file and saves the result for use in future InMAP simulations.

```
inmap preproc [flags]
```

### Options

```
      --InMAPData string                        
              InMAPData is the path to location of baseline meteorology and pollutant data.
              The path can include environment variables. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/testInMAPInputData.ncf")
      --Preproc.CTMType string                  
              Preproc.CTMType specifies what type of chemical transport
              model we are going to be reading data from. Valid
              options are "GEOS-Chem" and "WRF-Chem". (default "WRF-Chem")
      --Preproc.CtmGridDx float                 
              Preproc.CtmGridDx is the grid cell length in x direction [m] (default 1000)
      --Preproc.CtmGridDy float                 
              Preproc.CtmGridDy is the grid cell length in y direction [m] (default 1000)
      --Preproc.CtmGridXo float                 
              Preproc.CtmGridXo is the lower left of Chemical Transport Model (CTM) grid, x
      --Preproc.CtmGridYo float                 
              Preproc.CtmGridYo is the lower left of grid, y
      --Preproc.EndDate string                  
              Preproc.EndDate is the date of the end of the simulation.
              Format = "YYYYMMDD". (default "No Default")
      --Preproc.GEOSChem.Dash                   
              Preproc.GEOSChem.Dash indicates whether GEOS-Chem chemical variable
              names should be assumed to be in the form 'IJ-AVG-S__xxx' vs.
              the form 'IJ_AVG_S__xxx'.
      --Preproc.GEOSChem.GEOSA1 string          
              Preproc.GEOSChem.GEOSA1 is the location of the GEOS 1-hour time average files.
              [DATE] should be used as a wild card for the simulation date. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/preproc/GEOSFP.[DATE].A1.2x25.nc")
      --Preproc.GEOSChem.GEOSA3Cld string       
              Preproc.GEOSChem.GEOSA3Cld is the location of the GEOS 3-hour average cloud
              parameter files. [DATE] should be used as a wild card for
              the simulation date. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3cld.2x25.nc")
      --Preproc.GEOSChem.GEOSA3Dyn string       
              Preproc.GEOSChem.GEOSA3Dyn is the location of the GEOS 3-hour average dynamical
              parameter files. [DATE] should be used as a wild card for
              the simulation date. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3dyn.2x25.nc")
      --Preproc.GEOSChem.GEOSA3MstE string      
              Preproc.GEOSChem.GEOSA3MstE is the location of the GEOS 3-hour average moist parameters
              on level edges files. [DATE] should be used as a wild card for
              the simulation date. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/preproc/GEOSFP.[DATE].A3mstE.2x25.nc")
      --Preproc.GEOSChem.GEOSApBp string        
              Preproc.GEOSChem.GEOSApBp is the location of the constant GEOS pressure level
              variable file. It is optional; if it is not specified the Ap and Bp information
              will be extracted from the GEOSChem files.
      --Preproc.GEOSChem.GEOSChem string        
              Preproc.GEOSChem.GEOSChem is the location of GEOS-Chem output files.
              [DATE] should be used as a wild card for the simulation date. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/preproc/gc_output.[DATE].nc")
      --Preproc.GEOSChem.GEOSI3 string          
              Preproc.GEOSChem.GEOSI3 is the location of the GEOS 3-hour instantaneous parameter
              files. [DATE] should be used as a wild card for
              the simulation date. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/preproc/GEOSFP.[DATE].I3.2x25.nc")
      --Preproc.GEOSChem.VegTypeGlobal string   
              Preproc.GEOSChem.VegTypeGlobal is the location of the GEOS-Chem vegtype.global file,
              which is described here:
              http://wiki.seas.harvard.edu/geos-chem/index.php/Olson_land_map#Structure_of_the_vegtype.global_file (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/preproc/vegtype.global.txt")
      --Preproc.StartDate string                
              Preproc.StartDate is the date of the beginning of the simulation.
              Format = "YYYYMMDD". (default "No Default")
      --Preproc.WRFChem.WRFOut string           
              Preproc.WRFChem.WRFOut is the location of WRF-Chem output files.
              [DATE] should be used as a wild card for the simulation date. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/preproc/wrfout_d01_[DATE]")
  -h, --help                                    help for preproc
```

### Options inherited from parent commands

```
      --config string   
              config specifies the configuration file location.
```

### SEE ALSO
* [inmap](inmap.md)	 - A reduced-form air quality model.

