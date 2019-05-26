---
id: inmap_sr_save
title: inmap sr save
sidebar_label: inmap sr save
---

## inmap sr save

Save simulation results to create an SR matrix

### Synopsis

save saves the results of InMAP simulations created using 'start'

```
inmap sr save [flags]
```

### Options

```
      --SR.OutputFile string   
                                             SR.OutputFile is the path where the output file is or should be created
                                              when creating a source-receptor matrix. It can contain environment variables. (default "${INMAP_ROOT_DIR}/cmd/inmap/testdata/output_${InMAPRunType}.shp")
  -h, --help                   help for save
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
