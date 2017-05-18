# Release 1.3.0 (?)
* Removed vendored libraries
* A log file containing information about each model run is now automatically created
* Added a GEOS-Chem preprocessor
* Allowed new output variables to be defined as expressions of existing output variables
* Added "Outputter", a holder for output parameters.
* Allowed output variable expressions to now be evaluated at the grid level in addition to the grid cell level
* Incorporated the proprocessor into the main program

# Release 1.2.1 (2016-11-15)
* Changed the time step calculation algorithm to work with larger grid cell sizes
* Changed the "Total PM2.5" and "Primary PM2.5" output variables to "TotalPM25" and "PrimaryPM25" to allow opening in ArcGIS
* Changed SR matrix generator to allow variable startup time
* Removed the population concentration threshold adjuster
* Fixed bug in time step calculation that was causing very occasional crashes
* Fixed bugs in source-receptor matrix reader
* Added additional evaluations and evaluation data

# Release 1.2.0 (2016-8-22)
* Allowed the input emissions data shapefiles to have arbitrary spatial projections instead of requiring them to be the same as the InMAP grid
* Changed the program to be able to create the variable grid at runtime from user supplied population and mortality data
* Fixed a bug involving the the loss of mass conservation in adjacent cells with different heights
* Population in output files is now population per grid cell instead of population per square km
* The user can now specify which variables to output
* Added option to dynamically vary grid resolution during the simulation based on spatial gradients in concentration and population density
* Changed the command line interface for the executable program to be more flexible
* Changed the allocation from CTM cells to InMAP cells so that the InMAP cell sizes no longer have to be multiples of the CTM cell sizes
* Added a source-receptor (SR) matrix generator
* Fixed bug in stability calculation for plume rise
* Temporarily removed the HTML user interface

# Release 1.1.0 (2016-2-12)
* Fixed a bug related to molar mass conversions
* Changed the advection algorithim to use Reynolds averaging instead of an empirical adjustment coefficient
* Removed the empirical correction factor for ammonia chemistry
* Changed the dry-deposition algorithm

# Release 1.0.0 (2015-6-18)
This is the version of the model documented in the journal discussion article:

[C. W. Tessum, J. D. Hill, J. D. Marshall (2015) "InMAP: A New Model for Air Pollution Interventions", Geosci. Model Dev. Discuss., 8, 9281-9321, 2015](http://www.geosci-model-dev-discuss.net/8/9281/2015/gmdd-8-9281-2015.html)
