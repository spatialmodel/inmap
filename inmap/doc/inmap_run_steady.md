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
      --NumIterations int   
              NumIterations is the number of iterations to calculate. If < 1, convergence
              is automatically calculated.
  -h, --help                help for steady
```

### Options inherited from parent commands

```
      --EmissionUnits string                   
              EmissionUnits gives the units that the input emissions are in.
              Acceptable values are 'tons/year', 'kg/year', 'ug/s', and 'μg/s'. (default "tons/year")
      --EmissionsShapefiles string             
              EmissionsShapefiles are the paths to any emissions shapefiles.
              Can be elevated or ground level; elevated files need to have columns
              labeled "height", "diam", "temp", and "velocity" containing stack
              information in units of m, m, K, and m/s, respectively.
              Emissions will be allocated from the geometries in the shape file
              to the InMAP computational grid, but the mapping projection of the
              shapefile must be the same as the projection InMAP uses.
              Can include environment variables. (default "${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/testEmis.shp")
      --InMAPData string                       
              InMAPData is the path to location of baseline meteorology and pollutant data.
              The path can include environment variables. (default "${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/testInMAPInputData.ncf")
      --LogFile string                         
              LogFile is the path to the desired logfile location. It can include
              environment variables. If LogFile is left blank, the logfile will be saved in
              the same location as the OutputFile.
      --NotMeters                              
              NotMeters should be set to true if the units of the grid are not meters
              (e.g., if the grid is in degrees latitude/longitude).
      --OutputAllLayers                        
              If OutputAllLayers is true, output data for all model layers. If false, only output
              the lowest layer.
      --OutputFile string                      
              OutputFile is the path to the desired output shapefile location. It can
              include environment variables. (default "inmap_output.shp")
      --OutputVariables string                 
              OutputVariables specifies which model variables should be included in the
              output file. It can include environment variables. (default "{\"TotalPM25\":\"PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA\",\"TotalPopD\":\"coxHazard(loglogRR(TotalPM25), TotalPop, allcause)\"}\n")
      --VarGrid.CensusFile string              
              VarGrid.CensusFile is the path to the shapefile holding population information. (default "${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/testPopulation.shp")
      --VarGrid.CensusPopColumns stringSlice   
              VarGrid.CensusPopColumns is a list of the data fields in CensusFile that should
              be included as population estimates in the model. They can be population
              of different demographics or for different population scenarios. (default [TotalPop,WhiteNoLat,Black,Native,Asian,Latino])
      --VarGrid.GridProj string                
              GridProj gives projection info for the CTM grid in Proj4 or WKT format. (default "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1")
      --VarGrid.HiResLayers int                
              HiResLayers is the number of layers, starting at ground level, to do
              nesting in. Layers above this will have all grid cells in the lowest
              spatial resolution. This option is only used with static grids. (default 8)
      --VarGrid.MortalityRateColumns string    
              VarGrid.MortalityRateColumns gives names of fields in MortalityRateFile that
              contain baseline mortality rates (as keys) in units of deaths per year per 100,000 people.
							The values specify the population group that should be used with each mortality rate
							for population-weighted averaging.
               (default "{\"AllCause\":\"TotalPop\",\"AsianMort\":\"Asian\",\"BlackMort\":\"Black\",\"LatinoMort\":\"Latino\",\"NativeMort\":\"Native\",\"WhNoLMort\":\"WhiteNoLat\"}\n")
      --VarGrid.MortalityRateFile string       
              VarGrid.MortalityRateFile is the path to the shapefile containing baseline
              mortality rate data. (default "${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/testMortalityRate.shp")
      --VarGrid.PopConcThreshold float         
              PopConcThreshold is the limit for
              Σ(|ΔConcentration|)*combinedVolume*|ΔPopulation| / {Σ(|totalMass|)*totalPopulation}.
              See the documentation for PopConcMutator for more information. This
              option is only used with dynamic grids. (default 1e-09)
      --VarGrid.PopDensityThreshold float      
              PopDensityThreshold is a limit for people per unit area in a grid cell
              (units will typically be either people / m^2 or people / degree^2,
              depending on the spatial projection of the model grid). If
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
              and longitude. (default 288000)
      --VarGrid.VariableGridDy float           
              VarGrid.VariableGridDy specifies the Y edge lengths of grid
              cells in the outermost nest, in the units of the grid model
              spatial projection--typically meters or degrees latitude
              and longitude. (default 288000)
      --VarGrid.VariableGridXo float           
              VarGrid.VariableGridXo specifies the X coordinate of the
              lower-left corner of the InMAP grid. (default -2.736e+06)
      --VarGrid.VariableGridYo float           
              VarGrid.VariableGridYo specifies the Y coordinate of the
              lower-left corner of the InMAP grid. (default -2.088e+06)
      --VarGrid.Xnests intSlice                
              Xnests specifies nesting multiples in the X direction. (default [18,3,2,2,2,3,2,2])
      --VarGrid.Ynests intSlice                
              Ynests specifies nesting multiples in the Y direction. (default [14,3,2,2,2,3,2,2])
      --VariableGridData string                
              VariableGridData is the path to the location of the variable-resolution gridded
              InMAP data, or the location where it should be created if it doesn't already
              exist. The path can include environment variables. (default "${GOPATH}/src/github.com/spatialmodel/inmap/inmap/testdata/inmapVarGrid.gob")
      --config string                          
              config specifies the configuration file location.
      --creategrid                             
              creategrid specifies whether to create the
              variable-resolution grid as specified in the configuration file before starting
              the simulation instead of reading it from a file. If --static is false, then
              this flag will also be automatically set to false.
  -s, --static                                 
              static specifies whether to run with a static grid that
              is determined before the simulation starts. If false, the
              simulation runs with a dynamic grid that changes resolution
              depending on spatial gradients in population density and
              concentration.
```

### SEE ALSO
* [inmap run](inmap_run.md)	 - Run the model.

