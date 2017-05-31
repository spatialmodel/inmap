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
inmap sr predict
```

### Options inherited from parent commands

```
      --config string   
              config specifies the configuration file location.
```

### SEE ALSO
* [inmap sr](inmap_sr.md)	 - Create an SR matrix.

