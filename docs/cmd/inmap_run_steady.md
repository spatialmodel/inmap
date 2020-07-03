---
id: inmap_run_steady
title: inmap run steady
sidebar_label: inmap run steady
---

## inmap run steady

Run InMAP in steady-state mode.

### Synopsis

steady runs InMAP in steady-state mode to calculate annual average
concentrations with no temporal variability.

```
inmap run steady [flags]
```

### Options

```
      --NumIterations int                        NumIterations is the number of iterations to calculate. If < 1, convergence is automatically calculated.
                                                 
      --aep.GridRef strings                      GridRef specifies the locations of the spatial surrogate gridding reference files used for processing emissions. It is used for assigning spatial locations to emissions records.
                                                  (default [no_default])
      --aep.InventoryConfig.COARDSFiles string   COARDSFiles lists COARDS-compliant NetCDF emission files (NetCDF 4 and greater not supported). Information regarding the COARDS NetCDF conventions are available here: https://ferret.pmel.noaa.gov/Ferret/documentation/coards-netcdf-conventions. The file names can include environment variables. The format is map[sector name][list of files]. For COARDS files, the sector name will also be used as the SCC code.
                                                  (default "{}\n")
      --aep.InventoryConfig.COARDSYear int       COARDSYear specifies the year of emissions for COARDS emissions files. COARDS emissions are assumed to be in units of mass of emissions per year. The year will not be used for NEI emissions files.
                                                 
      --aep.InventoryConfig.InputUnits string    InputUnits specifies the units of input data. Acceptable values are 'tons', 'tonnes', 'kg', 'g', and 'lbs'. This value will be used for AEP emissions only, not for shapefiles. (default "no_default")
      --aep.InventoryConfig.NEIFiles string      NEIFiles lists National Emissions Inventory emissions files. The file names can include environment variables. The format is map[sector name][list of files].
                                                  (default "{}\n")
      --aep.SCCExactMatch                        SCCExactMatch specifies whether SCC codes must match exactly when processing emissions.
                                                  (default true)
      --aep.SpatialConfig.GridName string        GridName specifies a name for the grid which is used in the names of intermediate and output files. Changes to the geometry of the grid must be accompanied by either a a change in GridName or the deletion of all the files in the SpatialCache directory.
                                                  (default "inmap")
      --aep.SpatialConfig.InputSR string         InputSR specifies the input emissions spatial reference in Proj4 format.
                                                  (default "+proj=longlat")
      --aep.SpatialConfig.MaxCacheEntries int    MaxCacheEntries specifies the maximum number of emissions and concentrations surrogates to hold in a memory cache. Larger numbers can result in faster processing but increased memory usage.
                                                  (default 10)
      --aep.SpatialConfig.SpatialCache string    SpatialCache specifies the location for storing spatial emissions data for quick access. If this is left empty, no cache will be used.
                                                 
      --aep.SrgShapefileDirectory string         SrgShapefileDirectory gives the location of the directory holding the shapefiles used for creating spatial surrogates. It is used for assigning spatial locations to emissions records. It is only used when SrgSpecType == "SMOKE".
                                                  (default "no_default")
      --aep.SrgSpec string                       SrgSpec gives the location of the surrogate specification file. It is used for assigning spatial locations to emissions records.
                                                  (default "no_default")
      --aep.SrgSpecType string                   SrgSpecType specifies the type of data the gridding surrogates are being created from. It can be "SMOKE" or "OSM".
                                                  (default "no_default")
  -h, --help                                     help for steady
```

### Options inherited from parent commands

```
      --EmissionMaskGeoJSON string            EmissionMaskGeoJSON is an optional GeoJSON-formatted polygon string that specifies the area outside of which emissions will be ignored. The mask is assumed to  use the same spatial reference as VarGrid.GridProj. Example="{\"type\": \"Polygon\",\"coordinates\": [ [ [-4000, -4000], [4000, -4000], [4000, 4000], [-4000, 4000] ] ] }"
                                              
      --EmissionUnits string                  EmissionUnits gives the units that the input emissions are in. Acceptable values are 'tons/year', 'kg/year', 'ug/s', and 'μg/s'.
                                               (default "tons/year")
      --EmissionsShapefiles strings           EmissionsShapefiles are the paths to any emissions shapefiles. Can be elevated or ground level; elevated files need to have columns labeled "height", "diam", "temp", and "velocity" containing stack information in units of m, m, K, and m/s, respectively. Emissions will be allocated from the geometries in the shape file to the InMAP computational grid, but the mapping projection of the shapefile must be the same as the projection InMAP uses. Can include environment variables.
                                               (default [${INMAP_ROOT_DIR}/cmd/inmap/testdata/testEmis.shp])
      --InMAPData string                      InMAPData is the path to location of baseline meteorology and pollutant data. The path can include environment variables.
                                               (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/testInMAPInputData.ncf")
      --LogFile string                        LogFile is the path to the desired logfile location. It can include environment variables. If LogFile is left blank, the logfile will be saved in the same location as the OutputFile.
                                              
      --OutputAllLayers                       If OutputAllLayers is true, output data for all model layers. If false, only output the lowest layer.
                                              
      --OutputFile string                     OutputFile is the path to the desired output shapefile location. It can include environment variables.
                                               (default "inmap_output.shp")
      --OutputVariables string                OutputVariables specifies which model variables should be included in the output file. It can include environment variables.
                                               (default "{\"TotalPM25\":\"PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA\",\"TotalPopD\":\"(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * AllCause / 100000\"}\n")
      --VarGrid.CensusFile string             VarGrid.CensusFile is the path to the shapefile or COARDs-compliant NetCDF file holding population information.
                                               (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/testPopulation.shp")
      --VarGrid.CensusPopColumns strings      VarGrid.CensusPopColumns is a list of the data fields in CensusFile that should be included as population estimates in the model. They can be population of different demographics or for different population scenarios.
                                               (default [TotalPop,WhiteNoLat,Black,Native,Asian,Latino])
      --VarGrid.GridProj string               GridProj gives projection info for the CTM grid in Proj4 or WKT format. (default "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1")
      --VarGrid.HiResLayers int               HiResLayers is the number of layers, starting at ground level, to do nesting in. Layers above this will have all grid cells in the lowest spatial resolution. This option is only used with static grids.
                                               (default 1)
      --VarGrid.MortalityRateColumns string   VarGrid.MortalityRateColumns gives names of fields in MortalityRateFile that contain baseline mortality rates (as keys) in units of deaths per year per 100,000 people. The values specify the population group that should be used with each mortality rate for population-weighted averaging.
                                               (default "{\"AllCause\":\"TotalPop\",\"AsianMort\":\"Asian\",\"BlackMort\":\"Black\",\"LatinoMort\":\"Latino\",\"NativeMort\":\"Native\",\"WhNoLMort\":\"WhiteNoLat\"}\n")
      --VarGrid.MortalityRateFile string      VarGrid.MortalityRateFile is the path to the shapefile containing baseline mortality rate data.
                                               (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/testMortalityRate.shp")
      --VarGrid.PopConcThreshold float        PopConcThreshold is the limit for Σ(|ΔConcentration|)*combinedVolume*|ΔPopulation| / {Σ(|totalMass|)*totalPopulation}. See the documentation for PopConcMutator for more information. This option is only used with dynamic grids.
                                               (default 1e-09)
      --VarGrid.PopDensityThreshold float     PopDensityThreshold is a limit for people per unit area in a grid cell in units of people / m². If the population density in a grid cell is above this level, the cell in question is a candidate for splitting into smaller cells. This option is only used with static grids.
                                               (default 0.0055)
      --VarGrid.PopGridColumn string          VarGrid.PopGridColumn is the name of the field in CensusFile that contains the data that should be compared to PopThreshold and PopDensityThreshold when determining if a grid cell should be split. It should be one of the fields in CensusPopColumns.
                                               (default "TotalPop")
      --VarGrid.PopThreshold float            PopThreshold is a limit for the total number of people in a grid cell. If the total population in a grid cell is above this level, the cell in question is a candidate for splitting into smaller cells. This option is only used with static grids.
                                               (default 40000)
      --VarGrid.VariableGridDx float          VarGrid.VariableGridDx specifies the X edge lengths of grid cells in the outermost nest, in the units of the grid model spatial projection--typically meters or degrees latitude and longitude.
                                               (default 4000)
      --VarGrid.VariableGridDy float          VarGrid.VariableGridDy specifies the Y edge lengths of grid cells in the outermost nest, in the units of the grid model spatial projection--typically meters or degrees latitude and longitude.
                                               (default 4000)
      --VarGrid.VariableGridXo float          VarGrid.VariableGridXo specifies the X coordinate of the lower-left corner of the InMAP grid.
                                               (default -4000)
      --VarGrid.VariableGridYo float          VarGrid.VariableGridYo specifies the Y coordinate of the lower-left corner of the InMAP grid. (default -4000)
      --VarGrid.Xnests ints                   Xnests specifies nesting multiples in the X direction. (default [2,2,2])
      --VarGrid.Ynests ints                   Ynests specifies nesting multiples in the Y direction. (default [2,2,2])
      --VariableGridData string               VariableGridData is the path to the location of the variable-resolution gridded InMAP data, or the location where it should be created if it doesn't already exist. The path can include environment variables.
                                               (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/inmapVarGrid.gob")
      --config string                         config specifies the configuration file location.
      --creategrid                            creategrid specifies whether to create the variable-resolution grid as specified in the configuration file before starting the simulation instead of reading it from a file. If --static is false, then this flag will also be automatically set to false.
                                              
  -s, --static                                static specifies whether to run with a static grid that is determined before the simulation starts. If false, the simulation runs with a dynamic grid that changes resolution depending on spatial gradients in population density and concentration.
                                              
```

### SEE ALSO

* [inmap run](/docs/cmd/inmap_run)	 - Run the model.
