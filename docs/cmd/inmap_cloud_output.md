---
id: inmap_cloud_output
title: inmap cloud output
sidebar_label: inmap cloud output
---

## inmap cloud output

Retrieve and save the output of a job on a Kubernetes cluster.

### Synopsis

The files will be saved in 'current_dir/job_name', where current_dir is the directory the command is run in.

```
inmap cloud output [flags]
```

### Options

```
  -h, --help   help for output
```

### Options inherited from parent commands

```
      --addr string       
                          							addr specifies the URL to connect to for running cloud jobs (default "inmap.run:443")
      --config string     
                                        config specifies the configuration file location.
      --job_name string   
                          							job_name specifies the name of a cloud job (default "test_job")
```

### SEE ALSO

* [inmap cloud](../inmap_cloud)	 - Interact with a Kubernetes cluster.
