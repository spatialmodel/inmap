Column description of 'GREET processes matched with EPA-SCC FINAL'.

Columns A-E are from GREET.
A: This gives the process type. All of the processes here are of the type 'StationaryProcess'.
B: This gives the name of the process in GREET.
C: This gives the unique identification code for the corresponding process in GREET. It is numerical and up to 8 digit (with no leading zeros).
D: This gives the useful direct output of the process, which may be treated as an end product in its own right or may feed into another process.
E: This gives a list of the pathways in which the process features.

The non-trivial values of the following cells are either:
-the SCC linked to the processes, or
-exclusion and surrogate codes (given below).

SCC: these are 8 digit numerical codes, and should be thought of as X-XX-XXX-XX.

Exclusion and surrogate codes: these are codes for processes which do not link well with the SCC for various reasons. These reasons are outlined in the codes in a similar way to SCC,
except not necessarily X-XX-XXX-XX. The sets of SCC, exclusion codes and surrogates are mutually exclusive.

I have not found the need to use the surrogates, but I have included them in other versions of this spreadsheet, and attach it here for completeness.

The NOTES column is for notes relating to the SCC link with the processes, including assumptions, sources and general thoughts. Some assumptions appear in many rows, e.g. the assumption that
a fuel is stored near a refinery where it is produced.

----------------------------------------------------------------

************************EXCLUSION CODES*************************

----------------------------------------------------------------
Those which do not happen yet:				99910000
	Algae pathways:					99910100
	Pyrolysis and Fischer-Tropsch:			99910200
	AFEX:						99910300
	Dimethyl ether:					99910400
	Biomass oil derived jet fuel:			99910500
	Corn butanol:					99910600
	E-Diesel					99910700
	
Those which we can get better maps for, code		99920000
	Lithium batteries and related:			99920100
	NiMH batteries:					99920200
	Chemicals:					99920300
	Oil Sands/bitumen:				99920400
	Shale Oil production:				99920500
	Carbon fiber:					99920600
	Uranium conversion, fabrication, waste, nuclear	99920700
	Liquid H:					99920800

Those which aren't real processes, code			99930000
	Conversion factors:				99930100
	Mixing processes (no emissions,unknown location)99930200
	Electricity transmission and distribution	99930300

Those which are area sources, code			99940000
	Forestry and Logging Operations:		99941000
	Crop and Tree Farming:				99942000
		Camelina:				99942010
		Canola:					99942020
		Corn:					99942030
		Corn stover:				99942040
		Jatropha:				99942050
		Miscanthus:				99942060
		Soybean:				99942070
		Sugarcane:				99942080
		Sorghum (forage, grain, sweet):		99942090
		Switchgrass:				99942100
		Tree Farming for Ethanol:		99942110
		Willow Farming:				99942120
		Poplar Farming:				99942130
		Palm FFB Farming:			99942140
		
Those which are international processes, code		99950000
		Brazil:					99950100
		Chile:					99950200
		Alberta:				99950300
		Finland:				99950400
		Russia:					99950500
		Japan:					99950600
		Norway:					99950700
		New Caledonia:				99950800
		China:					99950900
		Canada:					99951000
		South Africa:				99951100
		Australia:				99951200
		NNA (Non-North American):		99951300
		DR Congo:				99951400
Those which are not listed, and are assumed to have negligible or no emissions, code '99960000'.
		Hydroelectric electricity generation:	99960100
		Nuclear power generation:		99960200
		Wind power generation:			99960300
Codes which are used for their spatial data but are not suitable for emissions or temporal usage
data (i.e. the emissions don't match up with the process, only the locations do) may be marked in future.



----------------------------------------------------------------

**************************SURROGATES****************************

----------------------------------------------------------------

Distribution of process is assumed to be
Uniform across the US:					8881
Uniform across the following states:			[XX]
Uniform across agricultural land:			8882
Proportional to population density:			8883
Uniform across woodland:				8884
[coastal, road, etc maybe]

CT Notes:
Deleted CA_950_NOFILL.txt
Replaced "OR" with ";" between SCCs
Replaced 30100705 with 30100701;30100702;30100705;30100706;30100709;30100799 because 30100705 doesn't have any emissions.

