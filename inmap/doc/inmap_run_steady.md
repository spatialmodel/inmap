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
      --creategrid   Create the variable-resolution grid as specified in the configuration file before starting the simulation instead of reading it from a file. If --static is false, then this flag will also be automatically set to false.
  -s, --static       Run with a static grid that is determined before the simulation starts. If false, run with a dynamic grid that changes resolution depending on spatial gradients in population density and concentration.
```

### Options inherited from parent commands

```
      --config string   configuration file location (default "./inmap.toml")
```

### SEE ALSO
* [inmap run](inmap_run.md)	 - Run the model.

