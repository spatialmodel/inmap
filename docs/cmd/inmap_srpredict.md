---
id: inmap_srpredict
title: inmap srpredict
sidebar_label: inmap srpredict
---

## inmap srpredict

Predict concentrations

### Synopsis

predict uses the SR matrix specified in the configuration file
	field SR.OutputFile to predict concentrations resulting
	from the emissions specified in the EmissionsShapefiles field in the configuration
	file, outputting the results in the shapefile specified in OutputFile field.
	of the configuration file. The EmissionUnits field in the configuration
	file specifies the units of the emissions. Output units are μg particulate
	matter per m³ air.

		Output variables:
		PNH4: Particulate ammonium
		PNO3: Particulate nitrate
		PSO4: Particulate sulfate
		SOA: Secondary organic aerosol
		PrimaryPM25: Primarily emitted PM2.5
		TotalPM25: The sum of the above components

```
inmap srpredict [flags]
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
                                                    Can include environment variables. (default [${INMAP_ROOT_DIR}/cmd/inmap/testdata/testEmis.shp])
      --OutputFile string             
                                                    OutputFile is the path to the desired output shapefile location. It can
                                                    include environment variables. (default "inmap_output.shp")
      --SR.OutputFile string          
                                                    SR.OutputFile is the path where the output file is or should be created
                                                     when creating a source-receptor matrix. It can contain environment variables. (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/output_${InMAPRunType}.shp")
      --VarGrid.GridProj string       
                                                    GridProj gives projection info for the CTM grid in Proj4 or WKT format. (default "+proj=lcc +lat_1=33.000000 +lat_2=45.000000 +lat_0=40.000000 +lon_0=-97.000000 +x_0=0 +y_0=0 +a=6370997.000000 +b=6370997.000000 +to_meter=1")
  -h, --help                          help for srpredict
```

### Options inherited from parent commands

```
      --config string   
                                      config specifies the configuration file location.
```

### SEE ALSO

* [inmap](inmap)	 - A reduced-form air quality model.

