/*
Package wesely1989 implements an algorithm for surface resistance to dry deposition.

Citation for the original article, followed by citation for an article with some corrections which have been incorporated here:

M. L. Wesely, Parameterization of surface resistances to gaseous dry deposition in regional-scale numerical models, Atmos. Environ. 23, 1293–1304 (1989), http:dx.doi.org/10.1016/0004-6981(89)90153-4.

J. Walmsley, and M. L. Wesely, Modification of coded parametrizations of surface resistances to gaseous dry deposition, Atmos. Environ. 30(7), 1181–1188 (1996), http://dx.doi.org/10.1016/1352-2310(95)00403-3.

The abstract of the original article:

Methods for estimating the dry deposition velocities of atmospheric gases in the U.S. and surrounding areas have been improved and incorporated into a revised computer code module for use in numerical models of atmospheric transport and deposition of pollutants over regional scales. The key improvement is the computation of bulk surface resistances along three distinct pathways of mass transfer to sites of deposition at the upper portions of vegetative canopies or structures, the lower portions, and the ground (or water surface). This approach replaces the previous technique of providing simple look-up tables of bulk surface resistances. With the surface resistances divided explicitly into distinct pathways, the bulk surface resistances for a large number of gases in addition IO those usually addressed in acid deposition models (SO2, O3, NOx, and HNO3) can be computed, if estimates of the effective Henry’s Law constants and appropriate measures of the chemical reactiiity of the various substances are known. This has been accomnlished successfullv for H2O2, HCHO, CH3CHO (to represent other aldehydes CH3O2H (to represent organic peroxides), CH3C(O)O2H, HCOOH (to represent organic acids), NH3, CH3C(O)O2NO2, and HNO2. Other factors considered include surface temperature, stomatal response to environmental parameters, the wetting of surfaces by dew and rain, and the covering of surfaces by snow. Surface emission of gases and variations of uptake characteristics by individual plant species within the landuse types are not considered explicitly.
*/
package wesely1989
