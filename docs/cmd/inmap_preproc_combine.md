---
id: inmap_preproc_combine
title: inmap preproc combine
sidebar_label: inmap preproc combine
---

## inmap preproc combine

Combine preprocessed CTM output from nested grids

### Synopsis

combine combines preprocessed chemical transport model
output from multiple nested grids into a single InMAP input file.
It should be run after independently preprocessing the output of
each nested grid.

```
inmap preproc combine [flags]
```

### Options

```
  -h, --help                          help for combine
      --output_file string            output_file is the location where the combined output file should be written.
                                       (default "inmapdata_combined.ncf")
      --preprocessed_inputs strings   preprocessed_inputs is a list of preprocessed input files to be combined.
```

### Options inherited from parent commands

```
      --config string   config specifies the configuration file location.
```

### SEE ALSO

* [inmap preproc](/docs/cmd/inmap_preproc)	 - Preprocess CTM output
