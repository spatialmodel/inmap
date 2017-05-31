## inmap sr

Create an SR matrix.

### Synopsis


sr creates a source-receptor matrix from InMAP simulations.
Simulations will be run on the cluster defined by $PBS_NODEFILE.
If $PBS_NODEFILE doesn't exist, the simulations will run on the
local machine.

```
inmap sr
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
      --begin int                      
              begin specifies the beginning grid index (inclusive) for SR
              matrix generation.
      --end int                        
              end specifies the ending grid index (exclusive) for SR matrix
              generation. The default is -1 which represents the last row. (default -1)
      --layers intSlice                
              layers specifies a ist of vertical layer numbers to
              be included in the SR matrix. (default [0,2,4,6])
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
* [inmap sr predict](inmap_sr_predict.md)	 - Predict concentrations

