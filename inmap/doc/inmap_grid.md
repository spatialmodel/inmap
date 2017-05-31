## inmap grid

Create a variable resolution grid

### Synopsis


grid creates and saves a variable resolution grid as specified by the
information in the configuration file. The saved data can then be loaded
for future InMAP simulations.

```
inmap grid
```

### Options

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
```

### Options inherited from parent commands

```
      --config string   
              config specifies the configuration file location.
```

### SEE ALSO
* [inmap](inmap.md)	 - A reduced-form air quality model.

