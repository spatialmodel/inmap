---
id: inmap_cloud_start
title: inmap cloud start
sidebar_label: inmap cloud start
---

## inmap cloud start

Start a job on a Kubernetes cluster.

### Synopsis

Start a job on a Kubernetes cluster. Of the flags available to this command, 'cmds', 'storage_gb', and 'memory_gb' relate to the creation of the job. All other flags and configuation file information are used to configure the remote simulation.

```
inmap cloud start [flags]
```

### Options

```
      --InMAPData string                      
                                                            InMAPData is the path to location of baseline meteorology and pollutant data.
                                                            The path can include environment variables. (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/testInMAPInputData.ncf")
      --NumIterations int                     
                                                            NumIterations is the number of iterations to calculate. If < 1, convergence
                                                            is automatically calculated.
      --OutputAllLayers                       
                                                            If OutputAllLayers is true, output data for all model layers. If false, only output
                                                            the lowest layer.
      --OutputVariables string                
                                                            OutputVariables specifies which model variables should be included in the
                                                            output file. It can include environment variables. (default "{\"TotalPM25\":\"PrimaryPM25 + pNH4 + pSO4 + pNO3 + SOA\",\"TotalPopD\":\"(exp(log(1.078)/10 * TotalPM25) - 1) * TotalPop * AllCause / 100000\"}\n")
      --VarGrid.CensusFile string             
                                                            VarGrid.CensusFile is the path to the shapefile holding population information. (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/testPopulation.shp")
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
                                                            mortality rate data. (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/testMortalityRate.shp")
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
      --cmds strings                          
                                              							cmds specifies the inmap subcommands to run. (default [run,steady])
      --creategrid                            
                                                            creategrid specifies whether to create the
                                                            variable-resolution grid as specified in the configuration file before starting
                                                            the simulation instead of reading it from a file. If --static is false, then
                                                            this flag will also be automatically set to false.
  -h, --help                                  help for start
      --memory_gb int                         
                                              							memory_gb specifies the gigabytes of RAM memory required for this job. (default 20)
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

* [inmap cloud](../inmap_cloud)	 - Interact with a Kubernetes cluster.
