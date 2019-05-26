---
id: inmap_sr_clean
title: inmap sr clean
sidebar_label: inmap sr clean
---

## inmap sr clean

clean cleans up temporary simulation output

### Synopsis

save cleans up the InMAP simulations created using 'start'

```
inmap sr clean [flags]
```

### Options

```
  -h, --help   help for clean
```

### Options inherited from parent commands

```
      --addr string       
                          							addr specifies the URL to connect to for running cloud jobs (default "inmap.run:443")
      --begin int         
                                        begin specifies the beginning grid index (inclusive) for SR
                                        matrix generation.
      --config string     
                                        config specifies the configuration file location.
      --end int           
                                        end specifies the ending grid index (exclusive) for SR matrix
                                        generation. The default is -1 which represents the last row. (default -1)
      --job_name string   
                          							job_name specifies the name of a cloud job (default "test_job")
      --layers ints       
                                        layers specifies a list of vertical layer numbers to
                                        be included in the SR matrix. (default [0,2,4,6])
```

### SEE ALSO

* [inmap sr](../inmap_sr)	 - Interact with an SR matrix.
