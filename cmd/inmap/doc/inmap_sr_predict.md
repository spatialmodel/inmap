## inmap sr predict

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
inmap sr predict [flags]
```

### Options

```
      --EmissionUnits string   
              EmissionUnits gives the units that the input emissions are in.
              Acceptable values are 'tons/year', 'kg/year', 'ug/s', and 'μg/s'. (default "tons/year")
      --OutputFile string      
              OutputFile is the path to the desired output shapefile location. It can
              include environment variables. (default "inmap_output.shp")
      --SR.OutputFile string   
              SR.OutputFile is the path where the output file is or should be created
               when creating a source-receptor matrix. It can contain environment variables. (default "${GOPATH}/src/github.com/spatialmodel/inmap/cmd/inmap/testdata/output_${InMAPRunType}.shp")
  -h, --help                   help for predict
```

### Options inherited from parent commands

```
      --config string   
              config specifies the configuration file location.
```

### SEE ALSO
* [inmap sr](inmap_sr.md)	 - Create an SR matrix.

