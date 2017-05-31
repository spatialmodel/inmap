## inmap run steady

Run InMAP in steady-state mode.

### Synopsis


steady runs InMAP in steady-state mode to calculate annual average
concentrations with no temporal variability.

```
inmap run steady
```

### Options

```
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

### Options inherited from parent commands

```
      --Vargrid.VariableGridDx float   
              Vargrid.VariableGridDx specifies the X edge lengths of grid
              cells in the outermost nest, in the units of the grid model
              spatial projection--typically meters or degrees latitude
              and longitude. (default 48000)
      --Vargrid.VariableGridDy float   
              Vargrid.VariableGridDy specifies the Y edge lengths of grid
              cells in the outermost nest, in the units of the grid model
              spatial projection--typically meters or degrees latitude
              and longitude. (default 48000)
      --Vargrid.VariableGridXo float   
              Vargrid.VariableGridXo specifies the X coordinate of the
              lower-left corner of the InMAP grid. (default -2.736e+06)
      --Vargrid.VariableGridYo float   
              Vargrid.VariableGridYo specifies the Y coordinate of the
              lower-left corner of the InMAP grid. (default -2.088e+06)
      --config string                  
              config specifies the configuration file location.
```

### SEE ALSO
* [inmap run](inmap_run.md)	 - Run the model.

