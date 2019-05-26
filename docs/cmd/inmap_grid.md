---
id: inmap_grid
title: inmap grid
sidebar_label: inmap grid
---

## inmap grid

Create a variable resolution grid

### Synopsis

grid creates and saves a variable resolution grid as specified by the
	information in the configuration file. The saved data can then be loaded
	for future InMAP simulations.

```
inmap grid [flags]
```

### Options

```
      --InMAPData string                      
                                                            InMAPData is the path to location of baseline meteorology and pollutant data.
                                                            The path can include environment variables. (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/testInMAPInputData.ncf")
      --LogFile string                        
                                                            LogFile is the path to the desired logfile location. It can include
                                                            environment variables. If LogFile is left blank, the logfile will be saved in
                                                            the same location as the OutputFile.
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
      --VariableGridData string               
                                                            VariableGridData is the path to the location of the variable-resolution gridded
                                                            InMAP data, or the location where it should be created if it doesn't already
                                                            exist. The path can include environment variables. (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/inmapVarGrid.gob")
  -h, --help                                  help for grid
```

### Options inherited from parent commands

```
      --config string   
                                      config specifies the configuration file location.
```

### SEE ALSO

* [inmap](../inmap)	 - A reduced-form air quality model.
