# Release 1.2.0 (?)
* Allowed the input emissions data shapefiles to have arbitrary spatial projections instead of requiring them to be the same as the InMAP grid
* Changed the program to create the variable grid at runtime from user supplied population and mortality data
* Fixed a bug involving the the loss of mass conservation in adjacent cells with different heights

# Release 1.1.0 (2016-2-12)
* Fixed a bug related to molar mass conversions
* Changed the advection algorithim to use Reynolds averaging instead of an empirical adjustment coefficient
* Removed the empirical correction factor for ammonia chemistry
* Changed the dry-deposition algorithm

# Release 1.0.0 (2015-6-18)
This is the version of the model documented in the journal discussion article:

[C. W. Tessum, J. D. Hill, J. D. Marshall (2015) "InMAP: A New Model for Air Pollution Interventions", Geosci. Model Dev. Discuss., 8, 9281-9321, 2015](http://www.geosci-model-dev-discuss.net/8/9281/2015/gmdd-8-9281-2015.html)
