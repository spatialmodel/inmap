## inmap worker

Start an InMAP worker.

### Synopsis


worker starts an InMAP worker that listens over RPC for simulation requests,
does the simulations, and returns results.

```
inmap worker
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
      --rpcport string                 
              rpcport specifies the port to be used for RPC communication
              when using distributed computing. (default "6060")
```

### Options inherited from parent commands

```
      --config string   
              config specifies the configuration file location.
```

### SEE ALSO
* [inmap](inmap.md)	 - A reduced-form air quality model.

