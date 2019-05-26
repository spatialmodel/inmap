---
id: inmap_sr
title: inmap sr
sidebar_label: inmap sr
---

## inmap sr

Interact with an SR matrix.

### Synopsis

Interact with an SR matrix.

### Options

```
      --addr string       
                          							addr specifies the URL to connect to for running cloud jobs (default "inmap.run:443")
      --begin int         
                                        begin specifies the beginning grid index (inclusive) for SR
                                        matrix generation.
      --end int           
                                        end specifies the ending grid index (exclusive) for SR matrix
                                        generation. The default is -1 which represents the last row. (default -1)
  -h, --help              help for sr
      --job_name string   
                          							job_name specifies the name of a cloud job (default "test_job")
      --layers ints       
                                        layers specifies a list of vertical layer numbers to
                                        be included in the SR matrix. (default [0,2,4,6])
```

### Options inherited from parent commands

```
      --config string   
                                      config specifies the configuration file location.
```

### SEE ALSO

* [inmap](../inmap)	 - A reduced-form air quality model.
* [inmap sr clean](../inmap_sr_clean)	 - clean cleans up temporary simulation output
* [inmap sr save](../inmap_sr_save)	 - Save simulation results to create an SR matrix
* [inmap sr start](../inmap_sr_start)	 - Start simulations to create an SR matrix
