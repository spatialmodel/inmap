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
* Temporally removed the HTML user interface

# Release 1.1.0 (2016-2-12)
* Fixed a bug related to molar mass conversions
* Changed the advection algorithim to use Reynolds averaging instead of an empirical adjustment coefficient
* Removed the empirical correction factor for ammonia chemistry
* Changed the dry-deposition algorithm

# Release 1.0.0 (2015-6-18)
This is the version of the model documented in the journal discussion article:

[C. W. Tessum, J. D. Hill, J. D. Marshall (2015) "InMAP: A New Model for Air Pollution Interventions", Geosci. Model Dev. Discuss., 8, 9281-9321, 2015](http://www.geosci-model-dev-discuss.net/8/9281/2015/gmdd-8-9281-2015.html)
