---
id: inmap
title: inmap
sidebar_label: inmap
---

## inmap

A reduced-form air quality model.

### Synopsis

InMAP is a reduced-form air quality model for fine particulate matter (PM2.5).
	Use the subcommands specified below to access the model functionality.
	Additional information is available at http://inmap.spatialmodel.com.

	Refer to the subcommand documentation for configuration options and default settings.
	Configuration can be changed by using a configuration file (and providing the
	path to the file using the --config flag), by using command-line arguments,
	or by setting environment variables in the format 'INMAP_var' where 'var' is the
	name of the variable to be set. Many configuration variables are additionally
	allowed to contain environment variables within them.
	Refer to https://github.com/spf13/viper for additional configuration information.

### Options

```
      --config string   
                                      config specifies the configuration file location.
  -h, --help            help for inmap
```

### SEE ALSO

* [inmap cloud](../inmap_cloud)	 - Interact with a Kubernetes cluster.
* [inmap grid](../inmap_grid)	 - Create a variable resolution grid
* [inmap preproc](../inmap_preproc)	 - Preprocess CTM output
* [inmap run](../inmap_run)	 - Run the model.
* [inmap sr](../inmap_sr)	 - Interact with an SR matrix.
* [inmap srpredict](../inmap_srpredict)	 - Predict concentrations
* [inmap version](../inmap_version)	 - Print the version number
