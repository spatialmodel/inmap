## inmap cloud start

Start a job on a Kubernetes cluster.

### Synopsis

Start a job on a Kubernetes cluster. Of the flags available to this command, 'cmds', 'storage_gb', and 'memory_gb' relate to the creation of the job. All other flags and configuation file information are used to configure the remote simulation.

```
inmap cloud start [flags]
```

### Options

```
      --EmissionUnits string                         
                                                                   EmissionUnits gives the units that the input emissions are in.
                                                                   Acceptable values are 'tons/year', 'kg/year', 'ug/s', and 'μg/s'. (default "tons/year")
      --EmissionsShapefiles strings                  
                                                                   EmissionsShapefiles are the paths to any emissions shapefiles.
                                                                   Can be elevated or ground level; elevated files need to have columns
                                                                   labeled "height", "diam", "temp", and "velocity" containing stack
                                                                   information in units of m, m, K, and m/s, respectively.
                                                                   Emissions will be allocated from the geometries in the shape file
                                                                   to the InMAP computational grid, but the mapping projection of the
                                                                   shapefile must be the same as the projection InMAP uses.
                                                                   Can include environment variables. (default [${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/testEmis.shp])
      --InMAPData string                             
                                                                   InMAPData is the path to location of baseline meteorology and pollutant data.
                                                                   The path can include environment variables. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/testInMAPInputData.ncf")
      --LogFile string                               
                                                                   LogFile is the path to the desired logfile location. It can include
                                                                   environment variables. If LogFile is left blank, the logfile will be saved in
                                                                   the same location as the OutputFile.
      --NumIterations int                            
                                                                   NumIterations is the number of iterations to calculate. If < 1, convergence
                                                                   is automatically calculated.
      --OutputAllLayers                              
                                                                   If OutputAllLayers is true, output data for all model layers. If false, only output
                                                                   the lowest layer.
      --OutputFile string                            
                                                                   OutputFile is the path to the desired output shapefile location. It can
                                                                   include environment variables. (default "inmap_output.shp")
      --OutputVariables string                       
                                                                   OutputVariables specifies which model variables should be included in the
                                                                   output file. It can include environment variables. (default "{\"TotalPM25\":\"PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA\",\"TotalPopD\":\"(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * AllCause / 100000\"}\n")
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
      --Preproc.GEOSChem.ChemFileInterval string     
                                                                   Preproc.GEOSChem.ChemFileInterval specifies the time duration represented by each GEOS-Chem output file.
                                                                   E.g. "3h" for 3 hours (default "3h")
      --Preproc.GEOSChem.ChemRecordInterval string   
                                                                   Preproc.GEOSChem.ChemRecordInterval specifies the time duration represented by each GEOS-Chem output record.
                                                                   E.g. "3h" for 3 hours (default "3h")
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
      --Preproc.GEOSChem.NoChemHourIndex             
                                                                   If Preproc.GEOSChem.NoChemHourIndex is true, the GEOS-Chem output files will be assumed to not contain a time dimension.
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
      --SR.LogDir string                             
                                                                   LogDir is the directory that log files should be stored in when creating
                                                                   a source-receptor matrix. It can contain environment variables. (default "log")
      --SR.OutputFile string                         
                                                                   SR.OutputFile is the path where the output file is or should be created
                                                                    when creating a source-receptor matrix. It can contain environment variables. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/output_${InMAPRunType}.shp")
      --VarGrid.CensusFile string                    
                                                                   VarGrid.CensusFile is the path to the shapefile holding population information. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/testPopulation.shp")
      --VarGrid.CensusPopColumns strings             
                                                                   VarGrid.CensusPopColumns is a list of the data fields in CensusFile that should
                                                                   be included as population estimates in the model. They can be population
                                                                   of different demographics or for different population scenarios. (default [TotalPop,WhiteNoLat,Black,Native,Asian,Latino])
      --VarGrid.GridProj string                      
                                                                   GridProj gives projection info for the CTM grid in Proj4 or WKT format. (default "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1")
      --VarGrid.HiResLayers int                      
                                                                   HiResLayers is the number of layers, starting at ground level, to do
                                                                   nesting in. Layers above this will have all grid cells in the lowest
                                                                   spatial resolution. This option is only used with static grids. (default 1)
      --VarGrid.MortalityRateColumns string          
                                                                   VarGrid.MortalityRateColumns gives names of fields in MortalityRateFile that
                                                                   contain baseline mortality rates (as keys) in units of deaths per year per 100,000 people.
                                                     							The values specify the population group that should be used with each mortality rate
                                                     							for population-weighted averaging.
                                                                    (default "{\"AllCause\":\"TotalPop\",\"AsianMort\":\"Asian\",\"BlackMort\":\"Black\",\"LatinoMort\":\"Latino\",\"NativeMort\":\"Native\",\"WhNoLMort\":\"WhiteNoLat\"}\n")
      --VarGrid.MortalityRateFile string             
                                                                   VarGrid.MortalityRateFile is the path to the shapefile containing baseline
                                                                   mortality rate data. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/testMortalityRate.shp")
      --VarGrid.PopConcThreshold float               
                                                                   PopConcThreshold is the limit for
                                                                   Σ(|ΔConcentration|)*combinedVolume*|ΔPopulation| / {Σ(|totalMass|)*totalPopulation}.
                                                                   See the documentation for PopConcMutator for more information. This
                                                                   option is only used with dynamic grids. (default 1e-09)
      --VarGrid.PopDensityThreshold float            
                                                                   PopDensityThreshold is a limit for people per unit area in a grid cell
                                                                   in units of people / m². If
                                                                   the population density in a grid cell is above this level, the cell in question
                                                                   is a candidate for splitting into smaller cells. This option is only used with
                                                                   static grids. (default 0.0055)
      --VarGrid.PopGridColumn string                 
                                                                   VarGrid.PopGridColumn is the name of the field in CensusFile that contains the data
                                                                   that should be compared to PopThreshold and PopDensityThreshold when determining
                                                                   if a grid cell should be split. It should be one of the fields
                                                                   in CensusPopColumns. (default "TotalPop")
      --VarGrid.PopThreshold float                   
                                                                   PopThreshold is a limit for the total number of people in a grid cell.
                                                                   If the total population in a grid cell is above this level, the cell in question
                                                                   is a candidate for splitting into smaller cells. This option is only used with
                                                                   static grids. (default 40000)
      --VarGrid.VariableGridDx float                 
                                                                   VarGrid.VariableGridDx specifies the X edge lengths of grid
                                                                   cells in the outermost nest, in the units of the grid model
                                                                   spatial projection--typically meters or degrees latitude
                                                                   and longitude. (default 4000)
      --VarGrid.VariableGridDy float                 
                                                                   VarGrid.VariableGridDy specifies the Y edge lengths of grid
                                                                   cells in the outermost nest, in the units of the grid model
                                                                   spatial projection--typically meters or degrees latitude
                                                                   and longitude. (default 4000)
      --VarGrid.VariableGridXo float                 
                                                                   VarGrid.VariableGridXo specifies the X coordinate of the
                                                                   lower-left corner of the InMAP grid. (default -4000)
      --VarGrid.VariableGridYo float                 
                                                                   VarGrid.VariableGridYo specifies the Y coordinate of the
                                                                   lower-left corner of the InMAP grid. (default -4000)
      --VarGrid.Xnests ints                          
                                                                   Xnests specifies nesting multiples in the X direction. (default [2,2,2])
      --VarGrid.Ynests ints                          
                                                                   Ynests specifies nesting multiples in the Y direction. (default [2,2,2])
      --VariableGridData string                      
                                                                   VariableGridData is the path to the location of the variable-resolution gridded
                                                                   InMAP data, or the location where it should be created if it doesn't already
                                                                   exist. The path can include environment variables. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/inmapVarGrid.gob")
      --begin int                                    
                                                                   begin specifies the beginning grid index (inclusive) for SR
                                                                   matrix generation.
      --cmds strings                                 
                                                     							cmds specifies the inmap subcommands to run. (default [run,steady])
      --creategrid                                   
                                                                   creategrid specifies whether to create the
                                                                   variable-resolution grid as specified in the configuration file before starting
                                                                   the simulation instead of reading it from a file. If --static is false, then
                                                                   this flag will also be automatically set to false.
      --end int                                      
                                                                   end specifies the ending grid index (exclusive) for SR matrix
                                                                   generation. The default is -1 which represents the last row. (default -1)
  -h, --help                                         help for start
      --layers ints                                  
                                                                   layers specifies a list of vertical layer numbers to
                                                                   be included in the SR matrix. (default [0,2,4,6])
      --memory_gb int                                
                                                     							memorgy_gb specifies the gigabytes of RAM memory required for this job. (default 20)
      --rpcport string                               
                                                                   rpcport specifies the port to be used for RPC communication
                                                                   when using distributed computing. (default "6060")
  -s, --static                                       
                                                                   static specifies whether to run with a static grid that
                                                                   is determined before the simulation starts. If false, the
                                                                   simulation runs with a dynamic grid that changes resolution
                                                                   depending on spatial gradients in population density and
                                                                   concentration.
```

### Options inherited from parent commands

```
      --addr string       
                          							addr specifies the URL to connect to for running cloud jobs (default "inmap.run:443")
      --config string     
                                        config specifies the configuration file location.
      --job_name string   
                          							job_name specifies the name of a cloud job (default "test_job")
```

### SEE ALSO

* [inmap cloud](inmap_cloud.md)	 - Interact with a Kubernetes cluster.

