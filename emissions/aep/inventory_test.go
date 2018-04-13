/*
Copyright (C) 2012 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package aep

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func TestInventory(t *testing.T) {

	type testData struct {
		name           string
		fileData       string
		expectedTotals map[string]float64
		freq           InventoryFrequency
		sectorType     string
		period         Period
	}

	table := []testData{
		testData{
			name:   "ag2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL      NONPOINT
#TYPE     NonPoint Inventory for CAPS
#COUNTRY  US
#YEAR     2002
#DESC     ANNUAL
#DESC     US (including AK and HI), PR, VI
#DESC     October 27, 2006 version of NEI
#DESC    ag sector: nonpt_split.sas split out from arinv_non_point_cap2002nei_27oct2006_orl.txt
#DESC     02nov2006: non-point split into oarea minus ag and afdust and removed catastrophic releases (SCC=28300XX000)
#DESC "FIPS","SCC","SIC","MACT","SRCTYPE","NAICS","POLCODE","ANN_EMS","AVD_EMS","CEFF","REFF","RPEN","CPRI","CSEC","DATA_SOURCE","YEAR","TRIBAL_CODE","MACT_FLAG","PROCESS_MACT_COMPLIANCE_STATUS","START_DATE","END_DATE","WINTER_THROUGHPUT_PCT","SPRING_THROUGHPUT_PCT","SUMMER_THROUGHPUT_PCT","FALL_THROUGHPUT_PCT","ANNUAL_AVG_DAYS_PER_WEEK","ANNUAL_AVG_WEEKS_PER_YEAR","ANNUAL_AVG_HOURS_PER_DAY","ANNUAL_AVG_HOURS_PER_YEAR","PERIOD_DAYS_PER_WEEK","PERIOD_WEEKS_PER_PERIOD","PERIOD_HOURS_PER_DAY","PERIOD_HOURS_PER_PERIOD"
#EXPORT_DATE=Tue Dec 16 07:39:31 EST 2008
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"01001","2801700001","","","02","","NH3",1.2305699999999999,,,,,"","","P-02-X","2002","000","","","20020101","20021231",,,,,,,,,,,,,,,,
"01001","2801700003","","","02","","NH3",17.529599999999999,,,,,"","","P-02-X","2002","000","","","20020101","20021231",,,,,,,,,,,,,,,,
`,
		},
		// TODO: Support the ORL FIRE format
		/*		testData{
							name:       "ptfire_ptinv_2005_jan",
							freq:       Annually,
							sectorType: "point",
							period:     Annual,
							fileData: `#ORL  FIRE
				#TYPE Point Source Inventory for FIRES
				#COUNTRY US
				#YEAR 2005
				#DATA ACRESBURNED HFLUX PM2_5 PM10 CO NOX NH3 SO2 VOC  75070 120127 218019 107028 74873 50328 26914181 85018 206440 108883 195197 193395 192972 463581 129000 198550 106990 56553 2381217 65357699 50000 110543 41637905 203338 207089 191242 71432 108383 106423 95476
				#DESC The data are based on Sonoma Access databased named "EmissionsFinal2.mdb"
				#DESC The date of this Access database is 3/12/2008
				#DESC The data were converted by Steve Fudge to ORL format using program reformat_to_orlfire_2005.prg
				#DESC ptfire data re-created on 11/09/08 to fix pollutant codes for methyl benzo pyrene ( from 247 to 65357699), methyl anthracene (from 26714181 to 26914181), and methyl chrysene (from 248 to 41637905)
				#DESC For the ORL field MATBURNED, the values are FCCS codes and not MATBURNED values
				#DESC XYLENE isomers added August 2008 by bkx
				#DESC m-xyl=0.6141*xyl, p-xyl=0.1658*xyl, o-xyl=0.2201*xyl
				#DESC  XYLENE apportioned to isomers based on data in Gaseous and Particulate Emissions from Prescribed Burning in Georgia Sangill Lee, et. al, Environ. Sci. Technol. 2005, 39, 9049-9056, (http://ps.uci.edu/~rowlandblake/publications/lee.pdf)
				#DESC "FIPS", "FIREID", "LOCID", "SCC", "FIRENAME", "LAT", "LON", "NFDRSCODE", "MATBURNED", "HEATCONTENT"
				#EMF_START_DATE="01/01/2005 0:0"
				#EMF_END_DATE="12/31/2005 23:59"
				#EMF_TEMPORAL_RESOLUTION=Daily
				#EMF_SECTOR=ptfire
				#EMF_COUNTRY=US
				#EMF_REGION=US
				#EMF_PROJECT="OAQPS CSC RFS2 Evaluation Case"
				#EXPORT_DATE=Wed May 04 10:28:09 EDT 2011
				#EXPORT_VERSION_NAME=Fix HEATCONTENT
				#EXPORT_VERSION_NUMBER=1
				#REV_HISTORY v1(11/18/2008)  Chris Allen.   Replaced '-9.0' with '8000' for column datavalue  SMOKE doesn't use HEATCONTECT, but if it's missing, Temporal crashes, so something needs to be there; we've used 8000 across the board in the past
				"12051","747604","-9","2810015000","161894",26.571999000000002,-81.353999000000002,"-9",180,8000
				"45089","747652","-9","2810015000","208474",33.780000000000001,-79.710999999999999,"-9",283,8000
				"01021","747634","-9","2810015000","208465",32.893999999999998,-86.748999999999995,"-9",123,8000`,
						},*/
		/*testData{
					name:       "ptfire_ptday_2005_jan",
					freq:       Annually,
					sectorType: "point",
					period:     Annual,
					fileData: `#ORL  FIREEMIS
		#TYPE Point Source Inventory for FIRES
		#COUNTRY US
		#YEAR 2005
		#DESC   January
		#DATA ACRESBURNED HFLUX PM2_5 PM10 CO NOX NH3 SO2 VOC  75070 120127 218019 107028 74873 50328 26914181 85018 206440 108883 195197 193395 192972 463581 129000 198550 106990 56553 2381217 65357699 50000 110543 41637905 203338 207089 191242 71432 108383 106423 95476
		#DESC The data are based on Sonoma Access databased named "EmissionsFinal2.mdb"
		#DESC The date of this Access database is 3/12/2008
		#DESC The data were converted by Steve Fudge to ORL format using program reformat_to_orlfireemis_2005.prg
		#DESC ptfire data re-created on 11/09/08 to fix pollutant codes for methyl benzo pyrene ( from 247 to 65357699), methyl anthracene (from 26714181 to 26914181), and methyl chrysene (from 248 to 41637905)
		#DESC For the ORL field NFDRSCODE, the values are FCCS codes and not NFDRS codes
		#DESC XYLENE isomers added August 2008 by bkx
		#DESC m-xyl=0.6141*xyl, p-xyl=0.1658*xyl, o-xyl=0.2201*xyl
		#DESC  XYLENE apportioned to isomers based on data in Gaseous and Particulate Emissions from Prescribed Burning in Georgia Sangill Lee, et. al, Environ. Sci. Technol. 2005, 39, 9049-9056, (http://ps.uci.edu/~rowlandblake/publications/lee.pdf)
		#DESC "FIPS", "FIREID", "LOCID", "SCC", "DATA", "DATE", "DATAVALUE", "BEGINHOUR", "ENDHOUR"
		#EMF_START_DATE="01/01/2005 0:0"
		#EMF_END_DATE="01/31/2005 23:59"
		#EMF_TEMPORAL_RESOLUTION=Daily
		#EMF_SECTOR=ptfire
		#EMF_COUNTRY=US
		#EMF_REGION=US
		#EMF_PROJECT="OAQPS CSC RFS2 Evaluation Case"
		#EXPORT_DATE=Wed May 04 10:28:21 EDT 2011
		#EXPORT_VERSION_NAME=Initial Version
		#EXPORT_VERSION_NUMBER=0
		"12051","747604","","2810015000","ACRESBURNED","01/01/05",99.9995861158,0,23
		"12051","747604","","2810015000","HFLUX","01/01/05",496055228.16900003,0,23
		`,
				},*/
		testData{
			name:   "ptipm_annual_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL     POINT
#TYPE    Point Source Inventory for CAPS
#COUNTRY US
#YEAR    2005
#DESC    Annual PTIPM
#DESC    US excluding AK, PR, VI and HI. Includes Tribal
#DESC    NEI  2005 Version 2.0
#DESC    Original inventory is EMF dataset ptipm_cap2005v2_revised12mar2009, version 5 (used in 2005cr_05b).
#DESC    Starting from that inventory, the following changes were made for 2005cs_05b:
#DESC    - Added ORIS IDs to_NEI UNIQUE_ID NEI2VA00040 (C. Allen, CSC, 11/30/2010)
#DESC      ORIS_FACILITY_CODE is 7839, ORIS_BOILER_IDs are 1 and 2 (for POINTIDs 1 and 2, respectively)
#DESC    - Applied PM10 and PM2_5 reduction factors for Natural Gas (boilers and turbines),Process Gas,and IGCC Units
#DESC         per Madeleine Strum's spreadsheet 2005cr_adjustmnts3.xlsx (C. Allen, CSC 12/1/2010)
#DESC    - Applied Pechan lat/lon coordinate fixes by ORIS FACILITY ID according to
#DESC      NEEDS410SupplementalFileLatLon111910fromPechanRev120310comparisonwithIPM.xls (C. Allen, CSC 12/7/2010)
#DESC      TD_creating_2005cs_ptipm_and_intermediate_ptnonipm_20DEC10_v4.xlsx changes (J. Beidler, CSC 12/21/2010)
#EMF_START_DATE="1/1/2005 0:0"
#EMF_END_DATE="12/31/2005 0:0"
#EMF_TEMPORAL_RESOLUTION=Annual
#EMF_SECTOR=ptipm
#EMF_COUNTRY="US"
#EMF_REGION="US"
#EMF_PROJECT="Transport Rule 1 Final (/tr1_f)"
#DESC FIPS,PLANTID,POINTID,STACKID,SEGMENT,PLANT,SCC,ERPTYPE,SRCTYPE,STKHGT,STKDIAM,STKTEMP,STKFLOW,STKVEL,SIC,MACT,NAICS,CTYPE,XLOC,YLOC,UTMZ,POLCODE,ANN_EMIS,AVD_EMIS,CEFF,REFF,CPRI,CSEC,NEI_UNIQUE_ID,ORIS_FACILITY_CODE,ORIS_BOILER_ID,IPM_YN,DATA_SOURCE,STACK_DEFAULT_FLAG,LOCATION_DEFAULT_FLAG,YEAR,TRIBAL_CODE,HORIZONTAL_AREA_FUGITIVE,RELEASE_HEIGHT_FUGITIVE,ZIPCODE,NAICS_FLAG,MACT_FLAG,PROCESS_MACT_COMPLIANCE_STATUS,IPM_FACILITY,IPM_UNIT,BART_SOURCE,BART_UNIT,CONTROL_STATUS,START_DATE,END_DATE,WINTER_THROUGHPUT_PCT,SPRING_THROUGHPUT_PCT,SUMMER_THROUGHPUT_PCT,FALL_THROUGHPUT_PCT,ANNUAL_AVG_DAYS_PER_WEEK,ANNUAL_AVG_WEEKS_PER_YEAR,ANNUAL_AVG_HOURS_PER_DAY,ANNUAL_AVG_HOURS_PER_YEAR,PERIOD_DAYS_PER_WEEK,PERIOD_WEEKS_PER_PERIOD,PERIOD_HOURS_PER_DAY,PERIOD_HOURS_PER_PERIOD,DESIGN_CAPACITY
#EXPORT_DATE=Tue May 03 12:44:52 EDT 2011
#EXPORT_VERSION_NAME=Add FAKECEM72 and 82
#EXPORT_VERSION_NUMBER=1
#REV_HISTORY v1(12/29/2010)  James Beidler.   Added FAKECEM72 and FAKECEM82 NOX and SO2  For missing plants
"18043","00004","002","1","1","PSIENERGY-GALLAGHER","10100202","02","01",441.05000000000001,17.201000000000001,303,15176.16,65.307599999999994,"4911","1808-1","221112","L",-85.838099999999997,38.263599999999997,-9,"CO",92.247893099999999,,,,,,"NEI31676","1008","2","Y","E-E","11111"," ","2005","000",,,"47150",,,,"01",,"Y",,,"NA","20050101","20051231",,,,,,,,,,,,,,,,,,,
"18043","00004","002","1","1","PSIENERGY-GALLAGHER","10100202","02","01",441.05000000000001,17.201000000000001,303,15176.16,65.307599999999994,"4911","1808-1","221112","L",-85.838099999999997,38.263599999999997,-9,"NH3",0.1042403001,,,,,,"NEI31676","1008","2","Y","E-E","11111"," ","2005","000",,,"47150",,,,"01",,"Y",,,"NA","20050101","20051231",,,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "avefire_ida_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#IDA
#TYPE     Area Source Emission Inventory
#DESC     Output from SMOKE
#DESC     Average U.S. fires developed taking the following steps:
#DESC        1) Calculate average acres burned for wildfires and Rx burning
#DESC           (separately) by state using 1996 through 2002 data from Pechan
#DESC        2) Calculate 2001->ave year factor by dividing ave acres burned
#DESC           by 2001 acres burned by state for both wildfires and Rx fires.
#DESC           Use resulting factors to create a SMOKE projection packet
#DESC        3) Use SMOKE to apply the projection factors to the 2001 fire
#DESC           inventory.
#DESC   Modified 19jan2007 by ram to include only the following SCCs:
#DESC     2810001000 (wildfires), 2810005000 (Managed Burning/logging), 2810015000 (Rx burning)
#DESC    Original file was arinv.fire_us_ave_1996-2002.ida
#DESC    C. Allen, 21dec2007: Changed VOC across-the-board; now VOC = 0.229*CO
#DESC      Original file: arinv_avefire_2002cc_19jan2007_v0_ida.txt
#POLID    CO NOX VOC NH3 SO2 PM10 PM2_5
#COUNTRY  US
#DESC     2010
#YEAR     2002
#EMF_START_DATE=1/1/2002
#EMF_END_DATE=12/31/2002
#EMF_TEMPORAL_RESOLUTION=Annual
#EMF_SECTOR=avefire
#EMF_REGION=US
#EMF_PROJECT=GHG Final Rule
#EXPORT_DATE=Mon Mar 09 08:45:52 EDT 2009
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
 1  12810001000   1190.36    3.2613     331.48    0.01.0   1.0   25.5372      0.07       7.11    0.01.0   1.0 272.59244 0.7468377       15.6    0.01.0   1.0    5.3593    0.0147       1.49    0.01.0   1.0    6.9965    0.0192       1.95    0.01.0   1.0   115.736    0.3171      32.23    0.01.0   1.0   99.2663     0.272      27.64    0.01.0   1.0
 1  32810001000 2794.4199    7.6559     331.48    0.01.0   1.0    59.946    0.1642       7.11    0.01.0   1.0639.922157 1.7532011       15.6    0.01.0   1.0   12.5657    0.0344       1.49    0.01.0   1.0   16.4418     0.045       1.95    0.01.0   1.0   271.702    0.7444      32.23    0.01.0   1.0   233.025    0.6384      27.64    0.01.0   1.0
`,
		},
		testData{
			name:   "avefire_orl_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL NONPOINT - avefire
#TYPE Nonpoint Inventory - HAPS avefire
#COUNTRY US
#YEAR 2002
#DESC ANNUAL
#DESC US (including AK and HI)
#DESC read pm25 from file
#DESC /orchid/oaqps/2002/smoke/inventory/v3/2002cc/avefire
#DESC 2224479 Mar 22  2007 arinv_avefire_2002cc_19jan2007_v0_ida.txt
#DESC EPA factors times avefire pm2.5 equals estimated HAP
#DESC name, polcode, factor:
#DESC    BUTADIE             106990      0.0147
#DESC    METHYLPYRE1         2381217     0.00033
#DESC    ACETALD             75070       0.0148
#DESC    ACROLEI             107028      0.0154
#DESC    ANTHRACEN           120127      0.00018
#DESC    BENZAANTH           56553       0.00022
#DESC    BENZENE             71432       0.041
#DESC    BNZOAFLURAN         203338      0.00009
#DESC    BNZOCPHENAN         195197      0.00014
#DESC    BENZOAPYR           50328       0.00005
#DESC    BENZOEPYRNE         192972      0.0001
#DESC    BENZOGHIP           191242      0.00018
#DESC    BENZOKFLU           207089      0.00009
#DESC    CARBNYLSUL          463581      0.00002
#DESC    CHRYSENE            218019      0.00022
#DESC    FLUORANTH           206440      0.00024
#DESC    FORMALD             50000       0.0936
#DESC    HEXANE              110543      0.00059
#DESC    INDENO123           193395      0.00012
#DESC    MTHYLCHLRD          74873       0.00464
#DESC    MTHYLANTRAC         26914181    0.0003
#DESC    MTHYLBNZPYR         65357699    0.00011
#DESC    MTHYLCHYSEN         41637905    0.00029
#DESC    OXYL                95476       0.0019325544
#DESC    MXYL                108383      0.0053920652
#DESC    PXYL                106423      0.0014553804
#DESC    PERYLENE            198550      0.00003
#DESC    PHENANTHR           85018       0.00018
#DESC    PYRENE              129000      0.00034
#DESC    TOLUENE             108883      0.0206
#DESC FIPS,SCC,SIC,MACT,SRCTYPE,NAICS,POLCODE,ANN_EMIS,AVD_EMIS,CEFF,REFF,RPEN,PRIMARY_DEVICE_TYPE_CODE,SECONDARY_DEVICE_TYPE_CODE,DATA_SOURCE,YEAR,TRIBAL_CODE,MACT_FLAG,PROCESS_MACT_COMPLIANCE_STATUS,START_DATE,END_DATE,WINTER_THROUGHPUT_PCT,WINTER_THROUGHPUT_PCT,SPRING_THROUGHPUT_PCT,SUMMER_THROUGHPUT_PCT,FALL_THROUGHPUT_PCT,ANNUAL_AVG_DAYS_PER_WEEK,ANNUAL_AVG_WEEKS_PER_YEAR,ANNUAL_AVG_HOURS_PER_DAY,ANNUAL_AVG_HOURS_YEAR,PERIOD_DAYS_PER_WEEK,PERIOD_WEEKS_PER_PERIOD,PERIOD_HOURS_PER_DAY,PERIOD_HOURS_PER_PERIOD
#EXPORT_DATE=Mon Mar 09 08:45:45 EDT 2009
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"01001","2810001000",,,,,"106990",1.4592146100000001,,0,1,1,,,"EPA","2002",,,,,,,,,,,,,,,,,,,,,
"01001","2810001000",,,,,"2381217",0.032757878999999997,,0,1,1,,,"EPA","2002",,,,,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "afdust_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL NONPOINT
#TYPE     Nonpoint Inventory
#DESC        (Output from SMOKE)
#COUNTRY  US
#YEAR     2002
# DESC Transportable fractions applied
# DESC File created on 26sep2007 by M. Houyoux using script:
# DESC   amber:/orchid/oaqps/2002/smoke/subsys/v3/smoke232/scripts/cases/2002ad_xpf/smk_afdust_2002ad_apply_xportfrac.csh
# DESC and original input inventory file:
# DESC   amber:/orchid/oaqps/2002/smoke/inventory/v3/from_NEI_team/arinv_afdust_cap2002nei_27oct2006_orl.txt
# DESC using control package developed by Rich Mason available at:
# DESC   amber:/orchid/oaqps/2002/smoke/inventory/v3/2002ad_xpf/afdust/gcntl.xportfrac.txt
#EXPORT_DATE=Wed Jun 23 08:23:17 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"01001","2294000000","0000","-9","02","-9","PM2_5",3.0557325,0.00837187,0,100,100,,,,,,,,,,,,,,,,,,,,,,,,,
"01001","2294000000","0000","-9","02","-9","PM10",51.677104999999997,0.14158112,0,100,100,,,,,,,,,,,,,,,,,,,,,,,,,`,
		},
		testData{
			name:   "nonroad_CA_jan_2005",
			freq:   Monthly,
			period: Jan,
			fileData: `#ORL      NONROAD
#TYPE     Nonroad Inventory for CAPs and HAPs for California only
#COUNTRY  US
#YEAR     2005
#DESC      jan
#DESC California nonroad CAP and HAP emissions, 2005v2
#DESC Originally created 4/1/2008  by CSC, WO79
#DESC CAP emissions (except NH3) came from annual data provided by Martin Johnson of the California Air Resources Board (CARB), mjohnson@arb.ca.gov  in  March 2007.
#DESC 2007. CAP emissions were created by applying a factor to the 2005 California ORL. Factors were developed by interpolating 2012 and 2020 emissions
#DESC  from CARB data.
#DESC NH3 is from 2015 NMIM runs for California.
#DESC HAP emissions were estimated by applying HAP-to-CAP ratios computed from Calif 2005 NEI submittal (2005 data provided by Laurel Driver 12/2007)
#DESC this was done because the Calif submittal from March 2007 did not include estimates for HAP
#DESC Retained only those HAP that are also estimated by NMIM for nonroad mobile sources; all other HAP dropped
#DESC Revised on 6/23/2010 by CSC to add in missing PM emissions for seven counties. Used the March 2007 version of the Calif 2005 inventory emissions
#EMF_START_DATE=1/1/2005
#EMF_END_DATE=1/31/2005
#EMF_TEMPORAL_RESOLUTION=Monthly
#EMF_SECTOR=nonroad
#EMF_REGION=California
#EMF_PROJECT="Clean Air Transport Rule (/cair_rep)"
#EMF_DESC     FIPS,SCC,POLCODE,ANN_EMIS,AVD_EMIS,CEFF,REFF,RPEN,SRCTYPE,DATA_SOURCE,YEAR,TRIBAL_CODE
#EXPORT_DATE=Thu Jun 24 08:48:20 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"06067","2270002003","EXH__VOC",0,0.025497026700000001,,,,"03","S","2005","000",,,,,,,,,,,,,,,,,,
"06093","2270002045","EXH__VOC",0,0.0016022721999999999,,,,"03","S","2005","000",,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "nonroad_caps_jan_2005",
			freq:   Monthly,
			period: Jan,
			fileData: `#ORL
#NONROAD File for CAPS
#COUNTRY US
#YEAR 2005
#DESC     JANUARY (MONTH=1)
#DESC     US (including AK and HI), PR, VI, excluding California
#DESC   ORL file created August 27, 2008 by CSC, data based solely on NMIM
# NMIM Version: 20071009
# NMIM County Database: NCD20070912 with 2006 meteorological data copied from 2005
# NMIM Nonroad Version: nr05c-BondBase\NR05c.exe
# NMIM Mobile Version: M6203CHC\M6203ChcOxFixNMIM.exe
#DESC  Pollutants in file are CO,NOX,SO2,VOC,NH3,PM10,PM2_5
#DESC  FIPS,SCC,POLCODE,ANN_EMIS,AVD_EMIS,CEFF,REFF,RPEN,SRCTYPE,DATA_SOURCE,YEAR,TRIBAL_CODE
#EMF_START_DATE="1/1/2005 0:0"
#EMF_ENF_DATE="1/31/2005 0:0"
#EMF_TEMPORAL_RESOLUTION=Monthly
#EMF_SECTOR=nonroad
#EMF_COUNTRY=US
#EMF_REGION="US, not Calif"
#EMF_PROJECT="2005-based platform, v2"
#EXPORT_DATE=Wed Apr 28 15:31:04 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"01001","2260002006","EVP__VOC",-9,3.6423011949999997e-05,-9,-9,-9,"03","E","2005"," ",,,,,,,,,,,,,,,,,,
"01001","2260002006","EXH__CO",-9,0.0048233352361100003,-9,-9,-9,"03","E","2005"," ",,,,,,,,,,,,,,,,,,
"01001","2260002006","EXH__NH3",-9,1.9171988418337001e-07,-9,-9,-9,"03","E","2005"," ",,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "onroad_runpm_jan_2005",
			freq:   Monthly,
			period: Jan,
			fileData: `#ORL
#TYPE  Mobile Source (onroad) for MOVES-based Onroad Gasoline PM (including Naphthalene) excluding California, partial monthly file REPLACING a subset of onroad SCCs and pollutants from the 2005v2 NMIM results
#COUNTRY US
#YEAR 2005
#DESC     January (MONTH=1)
#DESC     US excluding California  (including AK and HI but not PR, VI)
#DESC ORL file created 06MAY2010 (from 2005v2 NMIM state:county ratios  and MOVES2010-based data from OTAQ) by Rich
#DESC  MOVES2010-based state inventories provided by Harvey Michaels (OTAQ) in May 2010
#DESC Generated on garnet using: /garnet/home/rhc/2005cr/create_MOVES2005cr.sas
#DESC  NMIM ratios obtained from: /garnet/home/rhc/2005cr/create_2005cr_onroad_NMIM_plus_ST_CTY_ratios.sas
#DESC  The SCCs in this file are all that have the first 7 digits as follows:
#DESC 2201001*, 2201020*, 2201040*, 2201070*, for RUNNING (non-START) SCCs ONLY: *110:20:*330, or, all except *350, and *370
#DESC *** DIESEL Exhaust SCCs for PM and Gasoline non-PM/Naphthalene MOVES emissions provided separately
#DESC Pollutants in file are: PEC_72 POC_72 PNO3 PSO4 OTHER PMFINE_72 PMC_72 NAPHTH_72
#DESC  FIPS,SCC,POLCODE,ANN_EMIS,AVD_EMIS,SRCTYPE,DATA_SOURCE,YEAR
#EMF_START_DATE="1/1/2005 0:0"
#EMF_END_DATE="1/31/2005 0:0"
#EMF_TEMPORAL_RESOLUTION=Monthly
#EMF_SECTOR=on_moves_runpm
#EMF_COUNTRY="US"
#EMF_REGION="US"
#EMF_PROJECT="Clean Air Transport Rule (/cair_rep)"
#EXPORT_DATE=Mon May 10 12:19:55 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"01001","2201001110","NAPHTH_72",,0.00022282850000000001,"04","E","2005",,,,,,,,
"01003","2201001110","NAPHTH_72",,0.00075862830000000001,"04","E","2005",,,,,,,,
`,
		},
		testData{
			name:   "onroad_startpm_jan_2005",
			freq:   Monthly,
			period: Jan,
			fileData: `#ORL
#TYPE  Mobile Source (onroad) for MOVES-based Onroad Gasoline PM (including Naphthalene) excluding California, partial monthly file REPLACING a subset of onroad SCCs and pollutants from the 2005v2 NMIM results
#COUNTRY US
#YEAR 2005
#DESC     January (MONTH=1)
#DESC     US excluding California  (including AK and HI but not PR, VI)
#DESC ORL file created 06MAY2010 (from 2005v2 NMIM state:county ratios  and MOVES2010-based data from OTAQ) by Rich
#DESC  MOVES2010-based state inventories provided by Harvey Michaels (OTAQ) in May 2010
#DESC Generated on garnet using: /garnet/home/rhc/2005cr/create_MOVES2005cr.sas
#DESC  NMIM ratios obtained from: /garnet/home/rhc/2005cr/create_2005cr_onroad_NMIM_plus_ST_CTY_ratios.sas
#DESC  The SCCs in this file are all that have the first 7 digits as follows:
#DESC 2201001*, 2201020*, 2201040*, 2201070*, 2201080* for START SCCs ONLY: *350, and *370
#DESC *** DIESEL Exhaust SCCs for PM and Gasoline non-PM/Naphthalene MOVES emissions provided separately
#DESC   Created URBAN/RURAL parking (SCC= *370/*350 respectively) from broad SCCs based on state-level urban/rural local road (*330/*210 respectively) emissions
#DESC Pollutants in file are: PEC_72 POC_72 PNO3 PSO4 OTHER PMFINE_72 PMC_72 NAPHTH_72
#DESC  FIPS,SCC,POLCODE,ANN_EMIS,AVD_EMIS,SRCTYPE,DATA_SOURCE,YEAR
#EMF_START_DATE="1/1/2005 0:0"
#EMF_END_DATE="1/31/2005 0:0"
#EMF_TEMPORAL_RESOLUTION=Monthly
#EMF_SECTOR=on_moves_startpm
#EMF_COUNTRY="US"
#EMF_REGION="US"
#EMF_PROJECT="Clean Air Transport Rule (/cair_rep)"
#EXPORT_DATE=Mon May 10 13:06:50 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"01001","2201001350","NAPHTH_72",,2.6754199999999999e-05,"04","E","2005",,,,,,,,
"01001","2201001370","NAPHTH_72",,5.4614899999999998e-05,"04","E","2005",,,,,,,,
`,
		},
		testData{
			name:   "onroad_CA_2005",
			freq:   Monthly,
			period: Jan,
			fileData: `#ORL      ONROAD
#TYPE     OnRoad Inventory for CAPs and HAPs for California only; RFL excluded
#COUNTRY  US
#YEAR  2005
#DESC California onroad CAP and HAP emissions, 2002v2
#DESC Created 4/1/2008 by CSC, WO79
#DESC CAP emissions (except NH3) came from annual data provided by Chris Nguyen of the California Air Resources Board (CARB), tnguyen@arb.ca.gov in September 2007.
#DESC SCCs represent 11 vehicle types:  -- Calif data does not have the NMIM vehicle type of 'Heavy Duty Diesel Vehicles (HDDV) Class 6 & 7' (2230073)
#DESC Emissions were  allocated to the NEI road types using NMIM Calif results for 2005
#DESC Monthly emissions computed using NMIM monthly Calif results for 2005
#DESC HAP emissions were estimated by applying HAP-to-CAP ratios computed from Calif 2005 NEI submittal (2005 data provided by Laurel Driver 03/2008)
#DESC Retained only those HAP that are also estimated by NMIM for onroad mobile sources; all other HAP dropped
#DESC Revised from origival 2005v2 by replacing NH3 emissions from the 2005 MOVES data
#DESC     FIPS,SCC,POLCODE,ANN_EMIS,AVD_EMIS,SRCTYPE,DATA_SOURCE,YEAR,TRIBAL_CODE
#EMF_START_DATE=1/1/2005
#EMF_END_DATE=1/31/2005
#EMF_TEMPORAL_RESOLUTION=Monthly
#EMF_SECTOR=onroad
#EMF_REGION=California
#EMF_PROJECT="2005 Platform, v2"
#EXPORT_DATE=Wed Jun 30 08:28:14 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"06001","2201001130","BRK__PM10",0,0.0014111735999999999,"04","S","2005",,,,,,,,
"06001","2201001150","BRK__PM10",0,0.00073439019999999998,"04","S","2005",,,,,,,,
`,
		},
		testData{
			name:   "onroad_not2moves_jan_2005",
			freq:   Monthly,
			period: Jan,
			fileData: `#ORL
#TYPE  Mobile Source (onroad) for HAPs excluding California, partial monthly file containing a subset of onroad SCCs and pollutants from the 2005v2 NMIM results
#COUNTRY US
#YEAR 2005
#DESC     January (MONTH=1)
#DESC     US excluding California  (including AK and HI but does not include PR, VI)
#DESC ORL file created 04MAY2010 by Rich M. using /garnet/home/rhc/2005cr/create_2005cr_onroad_NMIM_plus_ST_CTY_ratios.sas
#DESC  All HAP emissions because CAPs use MOVES emissions -16MAR2010: even for motorcycles (SCC=2201080XXX)...
#DESC   ...all dioxins removed and xylenes split into modes.
#DESC  MOVES2010 replaces all:  CAPs including brake/tire PM and evap VOC and all EXHAUST...
#DESC   106990 (butadiene), 107028 (acrolein), 50000 (formaldehyde), 71432 (benzene), 75070 (acetaldehyde) and all EVAP 71432 (benzene) and 91203 (naphthalene)
#DESC  These HAPs contain all onroad mobile emissions (none are replaced with MOVES): EVP__10041 (ETHYLBENZ), EXH__100414(ETHYLBENZ), EXH__100425(STYRENE),
#DESC   EVP__108883(TOLUENE), EXH__108883(TOLUENE), EVP__110543(HEXANE), EXH__110543(HEXANE), EXH__120127(ANTHRACEN), EXH__123386(PROPIONAL), EXH__129000(PYRENE),
#DESC   EVP__108383 (MXYL as 0.68 of EVP__1330207(XYLS)), EVP__95476 (OXYL as 0.32 of EVP__1330207(XYLS)), EXH__108383 (MXYL as 0.74 of EXH__1330207 (XYLS)),
#DESC   EXH__95476 (OXYL as 0.26 of EXH__1330207(XYLS)), EXH__16065831(CHROMTRI), EXH__1634044(MTBE), EXH__18540299(CHROMHEX), EXH__191242(BENZOGHIP), EXH__193395(INDENO123),
#DESC   EXH__200(HG), EXH__201(HGIIGAS), EXH__202(PHGI), EXH__205992(BENZOBFLU), EXH__206440(FLUORANTH), EXH__207089(BENZOKFLU), EXH__208968(ACENAPHTY),
#DESC   EXH__218019(CHRYSENE), EXH__50328(BENZOAPYR), EXH__53703(DIBENZAHA), EVP__540841(TRMEPN224), EXH__54084(TRMEPN224), EXH__56553(BENZAANTH), EXH__7439965(MANGANESE),
#DESC   EXH__7440020(NICKEL), EXH__83329(ACENAPENE), EXH__85018(PHENANTHR), EXH__86737(FLUORENE), EXH__93(ARSENIC)
# NMIM Version: 20071009
# NMIM County Database: NCD20080522 with 2005 meteorological data
# NMIM Mobile Version: M6203CHC\M6203ChcOxFixNMIM.exe
#DESC  FIPS,SCC,POLCODE,ANN_EMIS,AVD_EMIS,SRCTYPE,DATA_SOURCE,YEAR
#EMF_START_DATE="1/1/2005 0:0"
#EMF_END_DATE="1/31/2005 0:0"
#EMF_TEMPORAL_RESOLUTION=Monthly
#EMF_SECTOR=on_noadj
#EMF_COUNTRY="US"
#EMF_REGION="US, not Calif"
#EMF_PROJECT="Clean Air Transport Rule (/cair_rep)"
#EXPORT_DATE=Mon May 10 08:15:38 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"72001","2230075290","BRK__PM10",,3.2589945999999999e-06,"04","E","2005",,,,,,,,
"72003","2230075230","EXH__PM2_5",,0.0002112626,"04","E","2005",,,,,,,,
`,
		},
		testData{
			name:   "onroad_moves_jan_2005",
			freq:   Monthly,
			period: Jan,
			fileData: `#ORL
#TYPE  Mobile Source (onroad excluding California) for MOVES-based Diesel and Gasoline except PM and Naphthalene gasoline exhaust, partial monthly file REPLACING a subset of onroad SCCs and pollutants from the 2005v2 NMIM results
#COUNTRY US
#YEAR 2005
#DESC     January (MONTH=1)
#DESC     US excluding California  (including AK and HI but not PR, VI)
#DESC ORL file created 06MAY2010 (from 2005v2 NMIM state:county ratios  and MOVES2010-based data from OTAQ) by Rich
#DESC  MOVES2010-based state inventories provided by Harvey Michaels (OTAQ) in May 2010
#DESC Generated on garnet using: /garnet/home/rhc/2005cr/create_MOVES2005cr.sas
#DESC  NMIM ratios obtained from: /garnet/home/rhc/2005cr/create_2005cr_onroad_NMIM_plus_ST_CTY_ratios.sas
#DESC  All onroad SCCs are in this file.  All modes (EXH, EVP, TIR, and BRK) are in this file except PM and Naphthalene exhaust from gasoline engines (provided separately)
#DESC Pollutants in file are: 1) ALL SCCS: BRK__PM10 BRK__PM2_5 TIR__PM10 TIR__PM2_5 EVP__VOC EVP__71432 (benzene), EVP__91203 (naphthalene), EXH__106990 (butadiene), EXH__107028 (acrolein),
#DESC    EXH__50000 (formaldehyde), EXH__71432, EXH__75070 (acetaldehyde), EXH__CO, EXH__NH3, EXH__NOX, EXH__SO2, EXH__VOC
#DESC    2) EXH__91203 (Naphthalene), PEC POC PNO3 PSO4 PMFINE and PMC (exhaust mode) for onroad diesel sources
#DESC  FIPS,SCC,POLCODE,ANN_EMIS,AVD_EMIS,SRCTYPE,DATA_SOURCE,YEAR
#EMF_START_DATE="1/1/2005 0:0"
#EMF_END_DATE="1/31/2005 0:0"
#EMF_TEMPORAL_RESOLUTION=Monthly
#EMF_SECTOR=on_noadj
#EMF_COUNTRY="US"
#EMF_REGION="US"
#EMF_PROJECT="Clean Air Transport Rule (/cair_rep)"
#EXPORT_DATE=Mon May 10 08:12:34 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"01071","2201001170","BRK__PM10",,0.00087661970000000005,"04","E","2005",,,,,,,,
"01001","2201001170","EVP__VOC",,0.0078043662000000001,"04","E","2005",,,,,,,,
`,
		},
		testData{
			name:   "onroad_canada_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL
#TYPE     Onroad Mobile Sources
#COUNTRY  CANADA
#YEAR     2006
#DESC     ANNUAL
#DESC     patricia.nguyen@ec.gc.ca, david.niemi@ec.gc.ca, mourad.sassi@ec.gc.ca
#DESC     Emissions, in short tons, for the on-road Transportation
#DESC     Created: 10 Oct 2008
#DESC
#DESC     Dropped "Region_code","Fuel_type","Annual_VKT","Population","Mobile_class","Trans_category" columns.
#DESC     2/4/2009 James Beidler <beidler.james@epa.gov>
#DESC
#DESC     "FIPS","US_SCC","POLL","ANN_EMISS","AVD_EMISS","SRCTYPE","DATA_SOURCE","RptYear"
#EXPORT_DATE=Mon Apr 26 15:49:16 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"48000","2230070000","CO",1390.23,-9,"4","!","2006",,,,,,,,
"48000","2230060000","VOC",680.94000000000005,-9,"4","!","2006",,,,,,,,
`,
		},
		testData{
			name:   "onroad_mexico_border_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#IDA
#TYPE ONROAD MOBILE SOURCE INVENTORY
#COUNTRY MEXICO
#YEAR 1999
#DESC MEXICO NATIONAL EMISSIONS INVENTORY, VER. 2.2, FOR THE SIX NORTHERN BORDER STATES.
#DESC PREPARED FOR THE WGA, EPA, COMMISSION FOR ENVIRONMENTAL COOPERATION, MEXICO'S
#DESC SECRETARIAT OF NATURAL RESOURCES AND THE ENVIRONMENT (SEMARNAT), AND WRAP.
#DESC PREPARED BY EASTERN RESEARCH GROUP, INC. (ERG), SACRAMENTO, CA.
#DESC DOMAIN: BAJA CALIFORNIA, SONORA, CHIHUAHUA, COAHUILA, NUEVO LEON, AND TAMAULIPAS.
#DESC VEHICLE CLASSIFICATIONS INCLUDE: LDGV, LDGT, HDGV, LDDV, LDDT, HDDV, AND MC.
#DESC SEE "MEXICO NATIONAL EMISSIONS INVENTORY, 1999. DRAFT FINAL." ERG, NOVEMBER 2005.
#DESC IDA CONVERSION BY ERG, MORRISVILLE, NC ON OCTOBER 27, 2006.
#DATA CO NH3 NOX PM10 PM2_5 SO2 VOC
#EXPORT_DATE=Mon Apr 26 15:49:17 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
 2  1          2201001000  7697.581  21.08926  18.363470.05031088  321.4992 0.8808197  20.069510.05498496  18.30982 0.0501639  35.085230.09612392  1053.695  2.886834
 2  1          2201060000  5064.543  13.87546  12.690330.03476802  206.7932 0.5665568  17.694050.04847684  16.153150.04425522  31.003110.08494004  571.3939  1.565463
`,
		},
		testData{
			name:   "onroad_mexico_interior_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#IDA
#TYPE           ONROAD SOURCE INVENTORY
#COUNTRY        MEXICO
#YEAR           1999
#DESC           WESTERN REGIONAL AIR PARTNERSHIP (WRAP)
#DESC           INVENTORY OF LOWER TWENTY-SIX STATES FOR MEXICO
#DESC           VERSION 1 OF 1999 MEXICO NEI - 26 INTERIOR STATES
#DESC           IDA CONVERSION BY MS. STACIE ENOCH, EASTERN RESEARCH GROUP, INC
#DESC           ON August 18, 2006
#DATA           CO NH3 NOX PM10 PM2_5 SO2 VOC
#EXPORT_DATE=Mon Apr 26 15:49:16 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
 1  10         220100100022175.968760.756076831.83475110.08721849773.2713622.1185517351.24645230.1404012446.74902340.1280795189.68405150.245709712726.500977.46986532
 1  10         220106000014671.444340.195735922.00213430.06027982505.9719231.38622438 45.1727180.1237608741.23746490.1129793679.24516290.217110021551.348874.25027084
`,
		},
		testData{
			name:   "canada_point1_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL      POINT
#TYPE     Point Sources
#COUNTRY  CANADA
#YEAR     2006
#DESC     ANNUAL
#DESC     patricia.nguyen@ec.gc.ca, david.niemi@ec.gc.ca, mourad.sassi@ec.gc.ca
#DESC     Emissions, in short tons, for facilities reporting to the NPRI
#DESC     Created: 10 Oct 2008
#DESC
#DESC     Dropped unused columns: CITY, PROVSTATE, CLASS_CAT, CLASS_CODE, SUB_CLASS_CODE, NOTE
#DESC     Moved YEAR from the CC position to the JJ position. Changed SIC to 0000
#DESC     Changed all SCCs to 39999999.
#DESC     2/3/2009 James Beidler <beidler.james@epa.gov>
#DESC
#DESC     "FIPS","PLANTID","POINTID","STACKID","SEGMENT","PLANT","SCC","ERPTYPE","SRCTYPE","STKHGT","STKDIAM","STKTEMP","STKFLOW","STKVEL","SIC_CODE","MACT","NAICS","CTYPE","LONG","LAT","UTMZ","POLL","ANN_EMISS","AVD_EMISS","CEFF","REFF","CPRI","CSEC","NEI_UNIQUE_ID","ORIS_FACILITY_CODE","ORIS_BOILER_ID","IPM_YN","DATA_SOURCE","STACK_DEFAULT_FLAG","LOCATION_DEFAULTS_FLAG","YEAR"
#EXPORT_DATE=Mon Apr 26 15:46:58 EDT 2010
#EXPORT_VERSION_NAME=change county codes
#EXPORT_VERSION_NUMBER=2
#REV_HISTORY 03/05/2009 Allan Beidler.    What:  CO for 5900,5188,4122    Why:  emissions value too high
#REV_HISTORY 03/09/2009 Allan Beidler.    What:  Replaced '35000' with '35001' for column fips Replaced '10000' with '10001' for column fips Replaced '11000' with '11001' for column fips Replaced '12000' with '12001' for column fips Replaced '13000' with '13001' for column fips Replaced '24000' with '24001' for column fips Replaced '46000' with '46001' for column fips Replaced '47000' with '47001' for column fips Replaced '48000' with '48001' for column fips Replaced '59000' with '59001' for column fips Replaced '61000' with '61001' for column fips Replaced '62000' with '62001' for column fips    Why:  add county codes of 001 for point source temporal matching.
"35001","7188","ON07188","","","TSTechCanada","39999999","01","01",,,,-9,,"0000","","336360","L",-79.430000000000007,44.07,,"PM2_5",0.33000000000000002,,0,100,-9,-9,"","","","","","","","2006",,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
"35001","10177","ON10177","","",,"39999999","01","01",,,,-9,,"0000","","911910","L",-75.730000000000004,45.399999999999999,,"PM2_5",0.40999999999999998,,0,100,-9,-9,"","","","","","","","2006",,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "canada_point2_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL      POINT
#TYPE     Point Sources
#COUNTRY  CANADA
#YEAR     2006
#DESC     ANNUAL
#DESC     patricia.nguyen@ec.gc.ca, david.niemi@ec.gc.ca, mourad.sassi@ec.gc.ca
#DESC     Emissions, in short tons, for facilities reporting to the NPRI
#DESC     CB05 speciation
#DESC     Emissions in short tons for facilities reporting to the NPRI
#DESC     This version correct from previous version with invalid speciated emissions values (values that were too high).
#DESC     CB5 speciation Created: 4 March 2009
#DESC
#DESC     Dropped unused columns: CITY, PROVSTATE, CLASS_CAT, CLASS_CODE, SUB_CLASS_CODE, NOTE
#DESC     Moved YEAR from the CC position to the JJ position
#DESC     Changed all SCCs to 399999999
#DESC     Removed ", Groupe des produits forestiers" from all "Tembec Inc." PLANT names.
#DESC     3/5/2009 James Beidler <beidler.james@epa.gov>
#DESC
#DESC     "FIPS","PLANTID","POINTID","STACKID","SEGMENT","PLANT","SCC","ERPTYPE","SRCTYPE","STKHGT","STKDIAM","STKTEMP","STKFLOW","STKVEL","SIC_CODE","MACT","NAICS","CTYPE","LONG","LAT","UTMZ","POLL","ANN_EMISS","AVD_EMISS","CEFF","REFF","CPRI","CSEC","NEI_UNIQUE_ID","ORIS_FACILITY_CODE","ORIS_BOILER_ID","IPM_YN","DATA_SOURCE","STACK_DEFAULT_FLAG","LOCATION_DEFAULTS_FLAG","YEAR"
#DESC 03/09/2009 Allan Beidler.    What:  Replaced '10000' with '10001'
#EXPORT_DATE=Mon Apr 26 15:47:13 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"10001","4316","100067","","","North Atlantic Refining","39999999","","01",,,,-9,,"0000","","","L",-53.990000000000002,47.789999999999999,,"ALD2",0.02,,0,100,-9,-9,"","","","","","","","2006",,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
"10001","4316","100067","","","North Atlantic Refining","39999999","","01",,,,-9,,"0000","","","L",-53.990000000000002,47.789999999999999,,"ALDX",0.0041044172,,0,100,-9,-9,"","","","","","","","2006",,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "canada_point3_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL      POINT
#TYPE     Point Sources
#COUNTRY  CANADA
#YEAR     2006
#DESC     ANNUAL
#DESC     patricia.nguyen@ec.gc.ca, david.niemi@ec.gc.ca, mourad.sassi@ec.gc.ca
#DESC     Emissions, in short tons, for facilities reporting to the NPRI
#DESC     Upstream Oil and Gas inventory
#DESC     Created: 10 Oct 2008
#DESC
#DESC     Dropped unused columns: CITY, PROVSTATE, CLASS_CAT, CLASS_CODE, SUB_CLASS_CODE, NOTE
#DESC     Moved YEAR from the CC position to the JJ position
#DESC     Mapped Canadian "facility name" to PLANT.
#DESC     2/3/2009 James Beidler <beidler.james@epa.gov>
#DESC
#DESC     "FIPS","PLANTID","POINTID","STACKID","SEGMENT","PLANT","SCC","ERPTYPE","SRCTYPE","STKHGT","STKDIAM","STKTEMP","STKFLOW","STKVEL","SIC_CODE","MACT","NAICS","CTYPE","LONG","LAT","UTMZ","POLL","ANN_EMISS","AVD_EMISS","CEFF","REFF","CPRI","CSEC","NEI_UNIQUE_ID","ORIS_FACILITY_CODE","ORIS_BOILER_ID","IPM_YN","DATA_SOURCE","STACK_DEFAULT_FLAG","LOCATION_DEFAULTS_FLAG","YEAR"
#EXPORT_DATE=Mon Apr 26 15:47:00 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"48012","040591         ","COMB           ","FLARE          ","               ","PCP LINDBERGH 12-36                   ","31000160  ","01","01",-9,-9,-9,-9,-9,"0000","   ","-9    ","L",-110.47,53.789999999999999,,"SO2     ",0.20999999999999999,,0,100,-9,-9,"","","","","","","","2006",,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
"48007","028002         ","COMB           ","PROP           ","               ","FIRST ALBERTA LEELA 104/4-26          ","10201002  ","01","01",-9,-9,-9,-9,-9,"0000","   ","-9    ","L",-110.61,52.289999999999999,,"SO2     ",1.0066625275049e-06,,0,100,-9,-9,"","","","","","","","2006",,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "mexico_point_border_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#IDA
#TYPE           POINT SOURCE INVENTORY
#COUNTRY        MEXICO
#YEAR           1999
#DESC           MEXICO BORDER STATES ONLY
#DESC           CLIENT: WESTERN REGIONAL AIR PARTNERSHIP (WRAP)
#DESC           MEXICO NATIONAL EMISSIONS INVENTORY, VER. 2.2, FOR THE SIX NORTHERN BORDER STATES.
#DESC           IDA CONVERSION BY MS. STACIE ENOCH, EASTERN RESEARCH GROUP, INC
#DESC           ON OCTOBER 27, 2006
#DESC           MINOR REVISIONS BY MR. REGI OOMMEN, EASTERN RESEARCH GROUP, INC ON JANUARY 18, 2007
#DATA           CO NH3 NOX PM10 PM2_5 SO2 VOC
#EXPORT_DATE=Mon Apr 26 15:47:06 EDT 2010
#EXPORT_VERSION_NAME=update stack diameter at 1 stack
#EXPORT_VERSION_NUMBER=1
#REV_HISTORY 03/03/2008 Rich Mason.    What:  old stack diameter was 777.55, new one, calc from flowrate & velocity indicated old value had truncation issue    Why:  stack diameter error
 2  102001001       1              1                         INDUSTRIA NAVAL DE CALIFORNIA S.A. DE C.31499900          29.81.8044141.  37.08449 14.50131                    0                                                     342931.863611-116.6108          4.94   0.01353425                    0.0                                                                0.07  1.917808E-4                    0.0            9.78   0.02679452                    0.0            9.42   0.02580822                    0.0            0.07  1.917808E-4                    0.0          7750.4     21.23397                    0.0   
 2  102001002       1              1                         CEMEX MEXICO S.A. DE C.V. PLANTA ENSENAD30500600          74.03.2808175.  418.8144 49.54068                    0                                                     324131.863611-116.6108                                                                                                                 22.61    0.0619452                    0.0           10.66   0.02920548                    0.0            7.18   0.01967123                    0.0          268.62    0.7359452                    0.0                                                       
`,
		},
		testData{
			name:   "mexico_point_interior_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#IDA
#TYPE    POINT SOURCE INVENTORY
#COUNTRY MEXICO
#YEAR    1999
#DESC    WESTERN REGIONAL AIR PARTNERSHIP (WRAP)
#DESC    INVENTORY OF LOWER TWENTY-SIX STATES FOR MEXICO
#DESC    VERSION 1 OF 1999 MEXICO NEI - 26 INTERIOR STATES
#DESC    IDA CONVERSION BY MS. STACIE ENOCH, EASTERN RESEARCH GROUP, INC
#DESC    ON August 18, 2006
#DESC    FILE REVISED JANUARY 8, 2007.  NULL GAS FLOW RATES WERE SET TO 0
#DATA    CO NH3 NOX PM10 PM2_5 SO2 VOC
#EXPORT_DATE=Mon Apr 26 15:46:59 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
 1  1311211         1              1                         PASTEURIZADORA AGUASCALIENTES S.A DE C.V302030            58.43.2964205.  392.4562  45.9856                    0                                                     202321.814615 -102.286       1.64068  0.004495013                    0.0                                                            8.861478   0.02427802                    0.0        1.053148  0.002885337                    0.0       0.6859682  0.001879365                    0.0        32.89054   0.09011105                    0.0       0.3592432  9.842279E-4                    0.0   
 1  1311212         1              1                         Evaporadora Mexicana, S.A. de C.V.      302030            58.43.2964205.  392.4562  45.9856                    0                                                     202321.814615 -102.286      5.199711   0.01424578                    0.0                                                            48.74904     0.133559                    0.0        28.47313   0.07800858                    0.0        20.56207   0.05633444                    0.0        616.7692     1.689778                    0.0        1.080706  0.002960838                    0.0   
`,
		},
		testData{
			name:   "point_offshore_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL      POINT
#TYPE     Point Inventory for CAPS
#COUNTRY  US
#YEAR     2005
#DESC     ANNUAL ptnonipm offshore oil (FIPS=85000 in original PQA file) -changed to FIPS=99000
#DESC     US (including AK and HI), PR, VI, and Tribal
#DESC     October 21, 2008 version of 2005 V2 NEI, original PQA file name:  ptinv_point_cap2005_11062008_orl.txt
#DESC   ptnonIPM AFTER TF: split and TF applied using split_point_ORL_IPM_2005v2_applyTF_20nov2008.sas
#EMF_START_DATE="1/1/2005 0:0"
#EMF_END_DATE="12/31/2005 23:59"
#EMF_TEMPORAL_RESOLUTION=Annual
#EMF_SECTOR=ptnonipm
#EMF_COUNTRY="US"
#EMF_REGION="US"
#EMF_PROJECT="2005 Platform, v2"
#DESC     FIPS,PLANTID,POINTID,STACKID,SEGMENT,PLANT,SCC,ERPTYPE,SRCTYPE,STKHGT,STKDIAM,STKTEMP,STKFLOW,STKVEL,SIC,MACT,NAICS,CTYPE,XLOC,YLOC,UTMZ,POLCODE,ANN_EMIS,AVD_EMIS,CEFF,REFF,CPRI,CSEC,NEI_UNIQUE_ID,ORIS_FACILITY_CODE,ORIS_BOILER_ID,IPM_YN,DATA_SOURCE,STACK_DEFAULT_FLAG,LOCATION_DEFAULT_FLAG,YEAR,TRIBAL_CODE,HORIZONTAL_AREA_FUGITIVE,RELEASE_HEIGHT_FUGITIVE,ZIPCODE,NAICS_FLAG,SIC_FLAG,MACT_FLAG,PROCESS_MACT_COMPLIANCE_STATUS,IPM_FACILITY,IPM_UNIT,BART_SOURCE,BART_UNIT,CONTROL_STATUS,START_DATE,END_DATE,WINTER_THROUGHPUT_PCT,SPRING_THROUGHPUT_PCT,SUMMER_THROUGHPUT_PCT,FALL_THROUGHPUT_PCT,ANNUAL_AVG_DAYS_PER_WEEK,ANNUAL_AVG_WEEKS_PER_YEAR,ANNUAL_AVG_HOURS_PER_DAY,ANNUAL_AVG_HOURS_PER_YEAR,PERIOD_DAYS_PER_WEEK,PERIOD_WEEKS_PER_PERIOD,PERIOD_HOURS_PER_DAY,PERIOD_HOURS_PER_PERIOD
#EXPORT_DATE=Mon Apr 26 15:47:08 EDT 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"99000","10063-1","BOI-01","BOI-01","BO>100","SPNResources,LLC(MMSID=02636)Platfo","10200601","02"," ",40,1,400,3.92699,5,"1311","none","211111","L",-93.557173000000006,28.242024000000001,-9,"CO",8.7443999999999998e-05,-9,,,,,"NEIMMS-10063-1"," "," "," ","G","00000"," ","2005","000",,,"77073"," "," "," "," "," "," "," "," ","NA","20050101","20051231",,,,,,,,,,,24,1464,,,,,,,
"99000","10121-1","BOI-01","BOI-01","BO>100","TheHoustonExplorationCompany(MMSID=","10200601","02"," ",67,1,400,3.92699,5,"1311","none","211111","L",-93.899291000000005,28.395147000000001,-9,"CO",0.14607599939999999,-9,,,,,"NEIMMS-10121-1"," "," "," ","G","00000"," ","2005","000",,,"77002-5215"," "," "," "," "," "," "," "," ","NA","20050101","20051231",,,,,,,,,,,24,8759,,,,,,,
`,
		},
		testData{
			name:   "alm_caps_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL      NONROAD ALM
#TYPE     NonRoad ALM Inventory for CAPS
#COUNTRY  US
#YEAR     2002
#DESC     ANNUAL
#DESC     US (including AK and HI), PR, VI
#DESC     March 27, 2007 version of NEI
#DESC     Pollutants in file are CO, NH3, NOX, PM10, PM2_5, SO2, VOC
#DESC
#DESC     Mar 28 2007, C. Allen: removed SCCs 2285002015, 2285004015, and 2285006015
#DESC     Nov 19 2007: C. Allen removed C3 SCCs (2280003100, 2280003200)
#DESC     Feb 20 2009: R. Mason removed all aircraft SCCs via UNIX grep '"2275' -for use with 2005v2 point inventory
#DESC                  Based on V1 (tribal data removed) of EMF dataset export arinv_alm_no_c3_cap2002v3_30jul2008_v1_orl.txt
#DESC     "FIPS","SCC","POLCODE","ANN_EMS","AVD_EMS","CEFF","REFF","RPEN","SRCTYPE","DATA_SOURCE","YEAR","TRIBAL_CODE","START_DATE","END_DATE","WINTER_THROUGHPUT_PCT","SPRING_THROUGHPUT_PCT","SUMMER_THROUGHPUT_PCT","FALL_THROUGHPUT_PCT","ANNUAL_AVG_DAYS_PER_WEEK","ANNUAL_AVG_WEEKS_PER_YEAR","ANNUAL_AVG_HOURS_PER_DAY","ANNUAL_AVG_HOURS_PER_YEAR","PERIOD_DAYS_PER_WEEK","PERIOD_WEEKS_PER_PERIOD","PERIOD_HOURS_PER_DAY","PERIOD_HOURS_PER_PERIOD","PRIMARY_DEVICE_TYPE_CODE"
#ORL      NONROAD ALM
#TYPE     NonRoad ALM Inventory for CAPS
#COUNTRY  US
#YEAR     2002
#DESC     ANNUAL
#DESC     US (including AK and HI), PR, VI
#DESC     March 27, 2007 version of NEI
#DESC     Pollutants in file are CO, NH3, NOX, PM10, PM2_5, SO2, VOC
#DESC
#DESC     Mar 28 2007, C. Allen: removed SCCs 2285002015, 2285004015, and 2285006015
#DESC     Nov 19 2007: C. Allen removed C3 SCCs (2280003100, 2280003200)
#DESC     "FIPS","SCC","POLCODE","ANN_EMS","AVD_EMS","CEFF","REFF","RPEN","SRCTYPE","DATA_SOURCE","YEAR","TRIBAL_CODE","START_DATE","END_DATE","WINTER_THROUGHPUT_PCT","SPRING_THROUGHPUT_PCT","SUMMER_THROUGHPUT_PCT","FALL_THROUGHPUT_PCT","ANNUAL_AVG_DAYS_PER_WEEK","ANNUAL_AVG_WEEKS_PER_YEAR","ANNUAL_AVG_HOURS_PER_DAY","ANNUAL_AVG_HOURS_PER_YEAR","PERIOD_DAYS_PER_WEEK","PERIOD_WEEKS_PER_PERIOD","PERIOD_HOURS_PER_DAY","PERIOD_HOURS_PER_PERIOD","PRIMARY_DEVICE_TYPE_CODE"
#REV_HISTORY 07/30/2008 Madeleine Strum.    What:  removed records with FIPS like '88%' (tribal data)    Why:  County-level tribal data cannot be spatially allocated andgets dropped in SMOKE and we don't want it in our summaries
#EXPORT_DATE=Wed Dec 22 12:11:52 EST 2010
#EXPORT_VERSION_NAME=changes for 2005cs
#EXPORT_VERSION_NUMBER=1
#REV_HISTORY v1(12/22/2010)  Chris Allen.   Updated diesel CMV port and underway emissions (SCC=2280002100 and 2280002200), NOX/SO2/PM, Delaware with those in EPA-HQ-OAR-2009-0491-3838.3_ram.xlsx  For TR1 Final (2005cs)
"26117","2280002100","CO",0.00076158035949999996,,,,,"03","S","2002","000","20020101","20021231",,,,,,,,,,,,,,,,
"26117","2280002100","NOX",0.0043735907157000002,,,,,"03","S","2002","000","20020101","20021231",,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "nonpt_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL  NONPOINT
#TYPE     NonPoint Inventory for CAPS
#COUNTRY  US
#YEAR     2002
#DESC     ANNUAL
#DESC     US (including AK and HI), PR, VI
#DESC     October 27, 2006 version of NEI
#DESC    oarea sector: nonpt_split.sas split out from arinv_non_point_cap2002nei_27oct2006_orl.txt
#DESC     02nov2006: non-point split into oarea minus ag and afdust and removed ALL catastrophic releases (SCC=28300XX000)
#DESC 14nov2006:  removed invalid SCC (2501060300) that was in RI did this in EMF
#DESC 09jan2007:  removed two fire SCCs (2810001000 and 2810015000) using "work/remove_fires_2.csh" (C. Allen)
#DESC  Additional Records 08mar2007- NY Refueling for CAPS, February 26, 2007 version from Roy Huntley
#DESC 04jan2008, C. Allen: Modified residential wood combustion (RWC) emissions. Original file was "arinv_nonpt_cap2002_nopfc_08mar2007_v0_orl.txt"
#DESC 08Sep2008, Re-calculated VOC for non-California SCC=2104008000 and 2104008001
#DESC  "FIPS","SCC","SIC","MACT","SRCTYPE","NAICS","POLCODE","ANN_EMS","AVD_EMS","CEFF","REFF","RPEN","CPRI","CSEC","DATA_SOURCE","YEAR","TRIBAL_CODE","MACT_FLAG","PROCESS_MACT_COMPLIANCE_STATUS","START_DATE","END_DATE","WINTER_THROUGHPUT_PCT","SPRING_THROUGHPUT_PCT","SUMMER_THROUGHPUT_PCT","FALL_THROUGHPUT_PCT","ANNUAL_AVG_DAYS_PER_WEEK","ANNUAL_AVG_WEEKS_PER_YEAR","ANNUAL_AVG_HOURS_PER_DAY","ANNUAL_AVG_HOURS_PER_YEAR","PERIOD_DAYS_PER_WEEK","PERIOD_WEEKS_PER_PERIOD","PERIOD_HOURS_PER_DAY","PERIOD_HOURS_PER_PERIOD"
#EMF_START_DATE=1/1/2002
#EMF_END_DATE=12/31/2002
#EMF_TEMPORAL_RESOLUTION=Annual
#EMF_SECTOR=nonpt
#EMF_REGION=US
#EMF_PROJECT="2002 Platform, v3.1"
#EXPORT_DATE=Tue Jan 04 13:43:29 EST 2011
#EXPORT_VERSION_NAME=Nebraska corrections for 2005cs
#EXPORT_VERSION_NUMBER=5
#REV_HISTORY v1(02/04/2009)  Rich Mason.   removed WRAP states other than California Oil and GAS emissions, SCCs=23100XXXXX  WRAP emissions provided separately
#REV_HISTORY v2(08/26/2009)  Rich Mason.   copied PM10 CEFF to PM2.5 CEFF for SCC=2610000400 in FIPS=18019,18043,18127  accuracy when considering direct PM controls
#REV_HISTORY v3(05/28/2010)  Rich Mason.   Oklahoma NEI oil and gas (2310000000) removed  OK DEQ replaced data May 2010
#REV_HISTORY v4(12/16/2010)  Chris Allen.   For Delaware, replaced NOX/SO2/PM2.5 emissions for Fuel Combustion and Open Burning with those in Delaware_EPA?HQ?OAR200904913823_NODA_comments_fnl.xls  For TR1 Final (2005cs)
#REV_HISTORY v4(12/16/2010)  Chris Allen.   For South Carolina, removed all SCC=2102005000 emissions for all pollutants  For TR1 Final (2005cs)
#REV_HISTORY v4(12/16/2010)  Chris Allen.   For Delaware, replaced PM10 emissions for Fuel Combustion and Open Burning: new_PM10 = (old_PM10/old_PM2_5)*new_PM2_5. PM10 was not provided in the spreadsheet used to make NOX/SO2/PM2.5 changes.  For TR1 Final (2005cs)
#REV_HISTORY v4(12/16/2010)  Chris Allen.   Removed Delaware residential wood emissions (SCC=2104008*) for NOX, SO2, PM10, PM2_5  These will be replaced in the next revision
#REV_HISTORY v4(12/16/2010)  Chris Allen.   Added Delaware residential wood emissions, NOX/SO2/PM10/PM2_5 (where PM10 = PM2_5), from Delaware_EPA?HQ?OAR200904913823_NODA_comments_fnl.xls. SCCs in that spreadsheet were mapped to those already part of the 2005 platform.  For TR1 Final (2005cs)
#REV_HISTORY v4(12/17/2010)  Chris Allen.   Completing earlier Delaware changes, removed NOX/SO2/PM emissions for 2102001000, 2102011000, 2103011000, 2610000500  Replacement emissions for these SCCs not provided in Delaware_EPA?HQ?OAR200904913823_NODA_comments_fnl.xls, which contains the entire set of Fuel Combustion and Open Burning emissions
#REV_HISTORY v4(12/20/2010)  Chris Allen.   Removed all Nebraska emissions for SCC=2102002000  Will be replaced with CENRAP emissions in the next revision
#REV_HISTORY v4(12/20/2010)  Chris Allen.   Added 2002 CENRAP version G inventory emissions for Nebraska, SCC=2102002000, all pollutants  For TR1 Final (2005cs)
#REV_HISTORY v5(01/04/2011)  Chris Allen.   For Nebraska, removed all emissions for eight fuel combustion SCCs  Will be replaced in the next step
#REV_HISTORY v5(01/04/2011)  Chris Allen.   Added 2002 CENRAP version G inventory emissions for Nebraska, eight fuel combustion SCCs, all pollutants  For TR1 Final (2005cs); original emissions were removed in the last revision
"01001","2102002000",,"0107-1","02",,"CO",0.01,,,0,0,,,"S-02-X","2002","000","SCC-D","03","20020101","20021231",25,25,25,25,6,0,0,0,0,0,0,0,,,,
"01001","2102002000",,"0107-1","02",,"NH3",0.00079090910000000005,,,,,,,"S-02-X-NR","2002","000","SCC-D","03","20020101","20021231",25,25,25,25,6,0,0,0,0,0,0,0,,,,
`,
		},
		testData{
			name:   "nonpt_mexico_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#IDA
#TYPE   NONPOINT SOURCE INVENTORY
#COUNTRY MEXICO
#YEAR    1999
#DESC    MEXICO NATIONAL EMISSIONS INVENTORY, VER. 2.2, FOR THE SIX NORTHERN BORDER STATES.
#DESC    PREPARED FOR THE WGA, EPA, COMMISSION FOR ENVIRONMENTAL COOPERATION, MEXICO'S
#DESC    SECRETARIAT OF NATURAL RESOURCES AND THE ENVIRONMENT (SEMARNAT), AND WRAP.
#DESC    PREPARED BY EASTERN RESEARCH GROUP, INC. (ERG), SACRAMENTO, CA.
#DESC    DOMAIN: BAJA CALIFORNIA, SONORA, CHIHUAHUA, COAHUILA, NUEVO LEON, AND TAMAULIPAS.
#DESC    CATEGORIES INCLUDE TRADITIONAL "AREA" SOURCES, PLUS LOCOMOTIVES, AIRCRAFT, AND
#DESC    COMMERICAL MARINE VESSELS; WILDFIRE AND AGRICULTURAL BURNING.
#DESC    ALSO INCLUDED ARE UNIQUE MEXICAN SOURCES/SCCS: BORDER CROSSINGS, BRICK KILNS,
#DESC    LPG DISTRIBUTION (NOT VIA PIPELINES), AND DOMESTIC AMMONIA.
#DESC    AREA SOURCES DO NOT INCLUDE PAVED AND UNPAVED ROAD DUST, AND WINDBLOWN DUST.
#DESC    FOR DETAILS ON MEXICO NEI DEVELOPMENT, INCLUDING EMISSIONS SUMMARIES, REFER TO:
#DESC    "MEXICO NATIONAL EMISSIONS INVENTORY, 1999. DRAFT FINAL." ERG, NOVEMBER 2005.
#DESC    IDA CONVERSION BY ERG, MORRISVILLE, NC, ON OCTOBER 27, 2006.
#DESC    Removed SCC 5555555555 Domestic Ammonia, A. Beidler, Dec. 21, 2006
#DATA    CO NH3 NOX PM10 PM2_5 SO2 VOC
#EXPORT_DATE=Fri Jan 07 15:07:30 EST 2011
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
 2  121020040004.157395360.01139012                                  0.0       0.0                           19.95549770.05467259                           0.831479070.00227802                           0.19955498 5.4672E-4                           4.961953160.01359439                           0.16629581  4.556E-4                           
 2  12102005000 12.2804450.03364505                                  0.0       0.0                            115.436180.31626349                           78.59968560.21534159                           51.18119040.14022243                           1426.263423.90757107                           0.687704920.00188412                           
`,
		},
		testData{
			name:   "ptinv_cem_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL     POINT
#TYPE    Point Source Inventory for CAPS
#COUNTRY US
#YEAR    2005
#DESC    Annual PTIPM
#DESC    US excluding AK, PR, VI and HI. Includes Tribal
#DESC    NEI  2005 Version 2.0
#DESC    Original inventory is EMF dataset ptipm_cap2005v2_revised12mar2009, version 5 (used in 2005cr_05b).
#DESC    Starting from that inventory, the following changes were made for 2005cs_05b:
#DESC    - Added ORIS IDs to_NEI UNIQUE_ID NEI2VA00040 (C. Allen, CSC, 11/30/2010)
#DESC      ORIS_FACILITY_CODE is 7839, ORIS_BOILER_IDs are 1 and 2 (for POINTIDs 1 and 2, respectively)
#DESC    - Applied PM10 and PM2_5 reduction factors for Natural Gas (boilers and turbines),Process Gas,and IGCC Units
#DESC         per Madeleine Strum's spreadsheet 2005cr_adjustmnts3.xlsx (C. Allen, CSC 12/1/2010)
#DESC    - Applied Pechan lat/lon coordinate fixes by ORIS FACILITY ID according to
#DESC      NEEDS410SupplementalFileLatLon111910fromPechanRev120310comparisonwithIPM.xls (C. Allen, CSC 12/7/2010)
#DESC      TD_creating_2005cs_ptipm_and_intermediate_ptnonipm_20DEC10_v4.xlsx changes (J. Beidler, CSC 12/21/2010)
#EMF_START_DATE="1/1/2005 0:0"
#EMF_END_DATE="12/31/2005 0:0"
#EMF_TEMPORAL_RESOLUTION=Annual
#EMF_SECTOR=ptipm
#EMF_COUNTRY="US"
#EMF_REGION="US"
#EMF_PROJECT="Transport Rule 1 Final (/tr1_f)"
#DESC FIPS,PLANTID,POINTID,STACKID,SEGMENT,PLANT,SCC,ERPTYPE,SRCTYPE,STKHGT,STKDIAM,STKTEMP,STKFLOW,STKVEL,SIC,MACT,NAICS,CTYPE,XLOC,YLOC,UTMZ,POLCODE,ANN_EMIS,AVD_EMIS,CEFF,REFF,CPRI,CSEC,NEI_UNIQUE_ID,ORIS_FACILITY_CODE,ORIS_BOILER_ID,IPM_YN,DATA_SOURCE,STACK_DEFAULT_FLAG,LOCATION_DEFAULT_FLAG,YEAR,TRIBAL_CODE,HORIZONTAL_AREA_FUGITIVE,RELEASE_HEIGHT_FUGITIVE,ZIPCODE,NAICS_FLAG,MACT_FLAG,PROCESS_MACT_COMPLIANCE_STATUS,IPM_FACILITY,IPM_UNIT,BART_SOURCE,BART_UNIT,CONTROL_STATUS,START_DATE,END_DATE,WINTER_THROUGHPUT_PCT,SPRING_THROUGHPUT_PCT,SUMMER_THROUGHPUT_PCT,FALL_THROUGHPUT_PCT,ANNUAL_AVG_DAYS_PER_WEEK,ANNUAL_AVG_WEEKS_PER_YEAR,ANNUAL_AVG_HOURS_PER_DAY,ANNUAL_AVG_HOURS_PER_YEAR,PERIOD_DAYS_PER_WEEK,PERIOD_WEEKS_PER_PERIOD,PERIOD_HOURS_PER_DAY,PERIOD_HOURS_PER_PERIOD,DESIGN_CAPACITY
#EXPORT_DATE=Wed Dec 29 13:07:20 EST 2010
#EXPORT_VERSION_NAME=Add FAKECEM72 and 82
#EXPORT_VERSION_NUMBER=1
#REV_HISTORY v1(12/29/2010)  James Beidler.   Added FAKECEM72 and FAKECEM82 NOX and SO2  For missing plants
"18043","00004","002","1","1","PSIENERGY-GALLAGHER","10100202","02","01",441.05000000000001,17.201000000000001,303,15176.16,65.307599999999994,"4911","1808-1","221112","L",-85.838099999999997,38.263599999999997,-9,"CO",92.247893099999999,,,,,,"NEI31676","1008","2","Y","E-E","11111"," ","2005","000",,,"47150",,,,"01",,"Y",,,"NA","20050101","20051231",,,,,,,,,,,,,,,,,,,
"18043","00004","002","1","1","PSIENERGY-GALLAGHER","10100202","02","01",441.05000000000001,17.201000000000001,303,15176.16,65.307599999999994,"4911","1808-1","221112","L",-85.838099999999997,38.263599999999997,-9,"NH3",0.1042403001,,,,,,"NEI31676","1008","2","Y","E-E","11111"," ","2005","000",,,"47150",,,,"01",,"Y",,,"NA","20050101","20051231",,,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "ptnonipm_cap_2005",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL      POINT
#TYPE     Point Inventory for CAPS
#COUNTRY  US
#YEAR     2005
#DESC     ANNUAL ptnonipm NOT including offshore oil (FIPS=85000 in original PQA file)
#DESC     US (including AK and HI), PR, VI, and Tribal
#DESC     October 21, 2008 version of 2005 V2 NEI, original PQA file name:  ptinv_point_cap2005_11062008_orl.txt
#DESC   ptnonIPM AFTER TF: split and TF applied using split_point_ORL_IPM_2005v2_applyTF_20nov2008.sas
#DESC      TD_creating_2005cs_ptipm_and_intermediate_ptnonipm_20DEC10_v4.xlsx changes (J. Beidler, CSC 12/21/2010)
#EMF_START_DATE="1/1/2005 0:0"
#EMF_END_DATE="12/31/2005 23:59"
#EMF_TEMPORAL_RESOLUTION=Annual
#EMF_SECTOR=ptnonipm
#EMF_COUNTRY="US"
#EMF_REGION="US"
#EMF_PROJECT="2005 Platform, v2"
#DESC     FIPS,PLANTID,POINTID,STACKID,SEGMENT,PLANT,SCC,ERPTYPE,SRCTYPE,STKHGT,STKDIAM,STKTEMP,STKFLOW,STKVEL,SIC,MACT,NAICS,CTYPE,XLOC,YLOC,UTMZ,POLCODE,ANN_EMIS,AVD_EMIS,CEFF,REFF,CPRI,CSEC,NEI_UNIQUE_ID,ORIS_FACILITY_CODE,ORIS_BOILER_ID,IPM_YN,DATA_SOURCE,STACK_DEFAULT_FLAG,LOCATION_DEFAULT_FLAG,YEAR,TRIBAL_CODE,HORIZONTAL_AREA_FUGITIVE,RELEASE_HEIGHT_FUGITIVE,ZIPCODE,NAICS_FLAG,SIC_FLAG,MACT_FLAG,PROCESS_MACT_COMPLIANCE_STATUS,IPM_FACILITY,IPM_UNIT,BART_SOURCE,BART_UNIT,CONTROL_STATUS,START_DATE,END_DATE,WINTER_THROUGHPUT_PCT,SPRING_THROUGHPUT_PCT,SUMMER_THROUGHPUT_PCT,FALL_THROUGHPUT_PCT,ANNUAL_AVG_DAYS_PER_WEEK,ANNUAL_AVG_WEEKS_PER_YEAR,ANNUAL_AVG_HOURS_PER_DAY,ANNUAL_AVG_HOURS_PER_YEAR,PERIOD_DAYS_PER_WEEK,PERIOD_WEEKS_PER_PERIOD,PERIOD_HOURS_PER_DAY,PERIOD_HOURS_PER_PERIOD
#REV_HISTORY 01/09/2009 Madeleine Strum.    What:  begain removing records    Why:  duplicates
#REV_HISTORY 01/09/2009 Madeleine Strum.    What:  deleted more records    Why:  duplicates
#REV_HISTORY 01/09/2009 Madeleine Strum.    What:  removed records with fips=30777    Why:  invalid fips, point sources located in  northern Wyoming near Montana and  emissions not that large.
#REV_HISTORY 01/16/2009 Allan Beidler.      What:  fixed Lat/Lon coordinates for larger sources  Why: Didn't match the FIPS code
#REV_HISTORY 01/16/2009 Allan Beidler.      What:  Dropped sources located well in the ocean or lakes  Why: Outside of US
#REV_HISTORY v1(06/14/2010)  Madeleine Strum.   deleted all records associated with  both kilns - 1 and 2- from Atlanta LaFarge Cement Plant  Per Dan McCain, George DEP,unit 2 shut down in 2002 and the other unit in 2004 Dan: (404)362-2778.  Confrimed by SPPD project lead Elineth.  Grinding operations for this facility  remain in the NEI
#REV_HISTORY v1(06/14/2010)  Chris Allen.   Removed 175 duplicate records. See /orchid/share/em_v4/inputs/2005cr_05b/ptnonipm/work/2005ck_ptnonipm_cap_true_duplicates.csv for list; all records with year=2002 in that file were removed  These records were duplicates in that records from both 2002 and 2005 were in the inventory, but weren'tcaught before due to extra leading zeroes in the plant IDs, point IDs, etc for one year or the other
#REV_HISTORY v3(07/01/2010)  Madeleine Strum.   REmoved BlueRidge Paper - CantonMill records that were for year 2006 (none were state-submitted).  In this case:  Plantid= '3708700159' and (DATA_SOURCE = 'R' or year='2006')  these records double count state-reported emissions
#REV_HISTORY v3(07/08/2010)  Madeleine Strum.   Removed Unit 017 from Domtar Paper FIPS=47163, plantid='0022', and pointid = '017'  Email from James.R Smith, Division ofAir Pollution Control , state of Tennessee, 8July2010 confirmed:   Yes I did hear back from Domtar.  We had a conference call on yesterday evening.  They are re-submitting their form to us.  They stated that unit 017 is not operating.  It is being removed from the site but not all of it is gone completely.  They stated that they submitted a notice to EPA back in 2003 about the changes.  The unit was taken out of commission from operation sometime in 2002.
#REV_HISTORY v4(07/08/2010)  Madeleine Strum.   changed unit 039 of domtar paper "end date" back to 20021231  mistakenly changed unit 039 enddate to 2005 when I was changing unit 041 so I needed to change it back
#REV_HISTORY v3(07/01/2010)  Madeleine Strum.   removed unit B001from P. H. Glatfelter Company - Chillicothe  Facility (0671010028).  No other units were removed as theywere  confirmed to emit in 2008 by the Ohio EPA  (tom Velais)  EIS indicates that unit B001 shut down in 2002 and  OHIO  State contact (Tom Velais, "Tom Velalis" <tom.velalis@epa.state.oh.us>) confirmed it shut down mid 2002. email from Velais sent to Strum:  07/01/2010 03:06 PM
#REV_HISTORY v3(07/08/2010)  Madeleine Strum.   pointid 041, plantid = '420470005' which is DOMTAR PAPER and poll in ('SO2', 'NOX') so2 and nox to reflect 2005 data  provided by PA DEP in a fax details in reference  SO2 was from 2002 and did not reflect the fact that the boilers added scrubber so2002  emissions were overestimated
#REV_HISTORY v5(07/22/2010)  Madeleine Strum.   Removed perceived duplicate records- removed state submitted 2005 data and used the Industry-based 2006 data gathered by SPPD which was very similar in magnitude of emissions.  Chose to keep the SPPD data because it had all the pollutants present whereas state data had only a subset.  For the same unitid, stackid and SCC there were two records for the same polllutant with different data source codes but similar emissions values.  they appeared to be duplicates so they were removed.
#REV_HISTORY v2(06/30/2010)  Rich Mason.   lat/lons for 3 NEI plantsReplaced '-95.8027' with '-95.7832' for column xloc using filter 'fips = '27083' and plantid = '2708300038'' Replaced '44.4710' with '44.4753' for column yloc using filter 'fips = '27083' and plantid = '2708300038'' Replaced '-88.6450' with '-88.6518' for column xloc using filter 'fips = '55139' and plantid = '471006470'' Replaced '44.0700' with '43.981' for column yloc using filter 'fips = '55139' and plantid = '471006470'' Also changedFIPS='20173' and plantid='0053'.  RFS2 OTAQ x/y's were better
#REV_HISTORY v6(08/16/2010)  Madeleine Strum.   Replaced '45063' with '45017' for column fips using filter 'NEI_UNIQUE_ID =   'NEI41393''  chaged to facilitate future year tagging and projections From:  45063 (Lexington)  to 45017 (Calhoun) because this facility straddles both counties, but Boiler MACT ICR database puts all the boilers in 45017 (Calhoun)
#EXPORT_DATE=Thu Jan 06 09:56:16 EST 2011
#EXPORT_VERSION_NAME=DE PM reduction at IKON
#EXPORT_VERSION_NUMBER=4
#REV_HISTORY v1(12/28/2010)  Allan Beidler.   plantid = '11500021', F4 removed  not in spreadsheet
#REV_HISTORY v2(01/03/2011)  Rich Mason.   removed 2 WV facilities erroneously not deleted. docket #2525  fips=54039, plantid=0002, fips=54079, plantid=0001
#REV_HISTORY v3(01/04/2011)  Allan Beidler.   added 13115,11500021, F4 & F5  should not have ben deleted
#REV_HISTORY v4(01/06/2011)  Rich Mason.   PM2.5 and PM10 reduced at plantid=1000300087, pointid=007.  PM10=oldPM10/PM25 * newPM25. segment1=120.2960/103.2186*2=2.3309,segment2=367.0724/314.9622*2=2.3309  comment in docket 3838.3.xls
"06001","01130314665","4","63","1","HEWLETTPACKARDCORPORATION","20200102","02","02",26.489999999999998,1.4179999999999999,490,95.114170000000001,60.211799999999997,"3751","0105-2","336991","L",-121.926106,37.469099,-9,"CO",0.014,,,,,,"NEI2CA314665"," "," ","","S","11111","Exact","2005","000",,,"94538"," "," ","SCC-Default","03"," "," ",""," ","NA","20050101","20051231",25,25,25,25,7,52,24,,7,52,24,,,,,,,,
"06007","041604670","1","1","1","WORLDCOM","20100105","02"," ",30.149999999999999,1.667,588,135.85599999999999,62.247900000000001,"4813","NONE","517","L",-121.87587000000001,39.837409999999998,-9,"NOX",0.13,,,,,,"NEI2CA604670"," "," ","","S","22222","SITEAVG","2005","000",,,"95926"," "," "," "," "," "," "," "," ","NA","20050101","20051231",,,,,,,,,7,52,24,,,,,,,,
`,
		},
		testData{
			name:   "secac3_2005_caps",
			freq:   Annually,
			period: Annual,
			fileData: `#ORL      POINT
#TYPE     Point Inventory for CAPs only
#COUNTRY  US
#YEAR     2005
#DESC     ANNUAL
#DESC      Ocean Going Class 3 Commercial Marine: Port and Underway
#DESC     See http://coast.cms.udel.edu/NorthAmericanSTEEM/  This inventory NOT on that website, rather, is a RERUN provided by Penny Carey 6/12/2007
#DESC      R Mason converted ANNUAL raster data to SMOKE PTINV using mk_SECA_2002pf31_rerun_JUNE2008.sas
#DESC    ** THIS INVENTORY contains emissions for all US states and non-US throughout 36km CMAQ domain...
#DESC    MODIFIED June2010 to project to 2005 and future years (by pollutant and region) using OTAQ factors C3 ctl inv adj_4-19-10.xls
#DESC    June2010 PC SAS code mk_ECA_IMO_GISbased_llxref.sas (initial grid cell to FIPS/growth region assignments) then ...
#DESC    ..OCTOBER2010 PC SAS code C:\CAIR\TR1\NODA Fall 2010\mk_ECA_IMO_GISbased_llxref_FALL2010.sas then garnet SAS mk_ecaIMO_2002_2005_20XX_FALL2010.sas to create this file
#DESC  JUNE2010: GIS polygons used to assign Canadian FIPS for ECA-IMO areas in EEZ (British Columbia and deafulted to Nova Scotia for Atlantic Canada (East Coast region)...
#DESC            Gulf Coast region (affects projection factors) redefined to include all waterways that impact the Gulf.
#DESC            CONTAINS FIPS and LAT/LON from CSC xref, modified June2010 for ECA-IMO control regions outside contiguous US **
#DESC  MODIFIED OCTOBER 2010:  1) Used NEI2008 CMV shapefiles to restrict US FIPS to state waters (~3-10 miles offshore),...
#DESC     2)  this caused new FIPS assignments of 8500X for offshore but within EEZ waters for region GFs, 3)  port shapefile
#DESC     4) Suffolk county NY offshore was erroneously assigned as 36013 (from Laurel's older EEZ shipping lane polygon file)..
#DESC     ... this was replaced with 85004 and more importantly, with EC not GL growth factors.  OTHERWISE, EMISSIONS ARE UNCHANGED
#DESC     NEI2008 website with polygons:  ftp://ftp.epa.gov/EmisInventory/2008_nei/mobile/rail_cmv_shapefiles/shipping_lanes_111309.zip and port_032310.zip
#DESC     Port shapefile used to assign grid cells as port (SCC=2280003200) if within 2km, otherwise assigned as underway (2280003100)
#DESC  Canada (FIPS=120000,135000, and 159000) are in separate ORL file
#DESC  MODIFIED 12/9/10 on garnet .../2005cs/mk_ecaIMO_2005_20XX_with_DE_TR1updates.sas to incorporate DE TR1 NODA C3 cty-level tots for PM2.5, SO2, and NOX -port and underway.
#DESC  NEW DE emissions for 2005, 2012 and 2014 from docket EPA-HQ-OAR-2009-0491-3838.3.xls (NODA 11/22/2010) **
#DESC         Also scaled PM10 same ratio as PM2.5 was adjusted (by county/SCC) and fixed SCCs -they were backwards in OCT2010 ORL files!
#DESC FIPS,PLANTID (row),POINTID (column),STACKID (colrow),SEGMENT (region),PLANT,SCC,ERPTYPE,SRCTYPE,STKHGT,STKDIAM,STKTEMP,STKFLOW,STKVEL,SIC,MACT,NAICS,CTYPE,XLOC,YLOC,UTMZ,POLCODE,ANN_EMS
#EMF_START_DATE="1/1/2005 0:0"
#EMF_END_DATE="12/31/2005 0:0"
#EMF_TEMPORAL_RESOLUTION=Annual
#EMF_SECTOR=seca_c3
#EMF_COUNTRY="US"
#EMF_REGION="US"
#EMF_PROJECT="Transport Rule 1 Final(/tr1_f)"
#EXPORT_DATE=Fri Dec 10 08:52:40 EST 2010
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
"10001","1096","3159","31591096","EC","SECA_C3","2280003200","02","03",65.620000000000005,2.625,539.60000000000002,0,82.019999999999996,"",""," ","L",-75.492564130000005,39.364973204000002,,"NOX",12.878310506,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
"10001","1096","3160","31601096","EC","SECA_C3","2280003200","02","03",65.620000000000005,2.625,539.60000000000002,0,82.019999999999996,"",""," ","L",-75.456632400000004,39.364973204000002,,"NOX",80.512382521000006,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "ptnonipm_2011",
			freq:   Annually,
			period: Annual,
			fileData: `#FORMAT=FF10_POINT
#COUNTRY=US
#YEAR=2011
#SELECTION_NAME=2011 NEI V2 with Biogenics
#INVENTORY_VERSION=General Purpose Release
#INVENTORY_LABEL=2011 NEI V2 with Biogenics
#CREATION_DATE=20140913 09:09
#CREATOR_NAME=Jonathan Miller
#VALUE_UNITS=TON
#INCLUDES_CAPS=true
#INCLUDES_HAPS=all
# Newly identified ptegu units removed from /garnet/oaqps/em_v6.2/2011platform/2011eg_nata_v6_11g/inputs/ptnonipm/ptnonipm_2011NEIv2_POINT_20140913_revised_20141007_09dec2014_v9.csv
#  using /garnet/oaqps/em_v6.2/2011platform/work/point_neiv2_final/ptipm_final/remove_ptegu_from_ptnonipm.sas
#DATA_SET_ID="543;2011EPA_Airports;Laurel Driver;2011"
#DATA_SET_ID="780;2011EPA_BOEM;Ron Ryan;2011"
#DATA_SET_ID="643;2011EPA_CarryForward-PreviousYearData;Madeleine Strum;2011"
#DATA_SET_ID="642;2011EPA_EGU;Ron Ryan;2011"
#DATA_SET_ID="621;2011EPA_HAP-Augmentation;Madeleine Strum;2011"
#DATA_SET_ID="601;2011EPA_LF;Ron Ryan;2011"
#DATA_SET_ID="600;2011EPA_Other;Madeleine Strum;2011"
#DATA_SET_ID="620;2011EPA_PM-Augmentation;Madeleine Strum;2011"
#DATA_SET_ID="542;2011EPA_Rail;Laurel Driver;2011"
#DATA_SET_ID="644;2011EPA_TRI;Ron Ryan;2011"
#DATA_SET_ID="560;2011EPA_chrom_split;Madeleine Strum;2011"
#DATA_SET_ID="-1412;Alabama Department of Environmental Management;Alabama Department of Environmental Management;2011"
#DATA_SET_ID="-1413;Alaska Department of Environmental Conservation;Alaska Department of Environmental Conservation;2011"
#DATA_SET_ID="-1380;Allegheny County Health Department;Allegheny County Health Department;2011"
#DATA_SET_ID="-1414;Arizona Department of Environmental Quality;Arizona Department of Environmental Quality;2011"
#DATA_SET_ID="-1415;Arkansas Department of Environmental Quality;Arkansas Department of Environmental Quality;2011"
#DATA_SET_ID="-1416;California Air Resources Board;California Air Resources Board;2011"
#DATA_SET_ID="-1382;Chattanooga Air Pollution Control Bureau (CHCAPCB);Chattanooga Air Pollution Control Bureau (CHCAPCB);2011"
#DATA_SET_ID="-1385;City of Albuquerque;City of Albuquerque;2011"
#DATA_SET_ID="-1391;City of Huntsville Division of Natural Resources and Environmental Mgmt;City of Huntsville Division of Natural Resources and Environmental Mgmt;2011"
#DATA_SET_ID="-1378;Clark County Department of Air Quality and Environmental Management;Clark County Department of Air Quality and Environmental Management;2011"
#DATA_SET_ID="-1474;Coeur d'Alene Tribe;Coeur d'Alene Tribe;2011"
#DATA_SET_ID="-1417;Colorado Department of Public Health and Environment;Colorado Department of Public Health and Environment;2011"
#DATA_SET_ID="-1479;"Confederated Tribes of the Colville Reservation, Washington";Confederated Tribes of the Colville Reservation, Washington;2011"
#DATA_SET_ID="-1418;Connecticut Department Of Environmental Protection;Connecticut Department Of Environmental Protection;2011"
#DATA_SET_ID="-1420;DC Department of Health Air Quality Division;DC Department of Health Air Quality Division;2011"
#DATA_SET_ID="-1419;Delaware Deparment of Natural Resources and Environmental Control;Delaware Deparment of Natural Resources and Environmental Control;2011"
#DATA_SET_ID="581;EPA NV Gold Mines;Sally Dombrowski;2011"
#DATA_SET_ID="-1421;Florida Department of Environmental Protection;Florida Department of Environmental Protection;2011"
#DATA_SET_ID="-1368;Forsyth County Environmental Affairs Department;Forsyth County Environmental Affairs Department;2011"
#DATA_SET_ID="-1422;Georgia Department of Natural Resources;Georgia Department of Natural Resources;2011"
#DATA_SET_ID="-1423;Hawaii Department of Health Clean Air Branch;Hawaii Department of Health Clean Air Branch;2011"
#DATA_SET_ID="-1424;Idaho Department of Environmental Quality;Idaho Department of Environmental Quality;2011"
#DATA_SET_ID="-1425;Illinois Environmental Protection Agency;Illinois Environmental Protection Agency;2011"
#DATA_SET_ID="-1426;Indiana Department of Environmental Management;Indiana Department of Environmental Management;2011"
#DATA_SET_ID="-1427;Iowa Department of Natural Resources;Iowa Department of Natural Resources;2011"
#DATA_SET_ID="-1362;Jefferson County (AL) Department of Health;Jefferson County (AL) Department of Health;2011"
#DATA_SET_ID="-1428;Kansas Department of Health and Environment;Kansas Department of Health and Environment;2011"
#DATA_SET_ID="-1429;Kentucky Division for Air Quality;Kentucky Division for Air Quality;2011"
#DATA_SET_ID="-1504;Kickapoo Tribe of Indians of the Kickapoo Reservation in Kansas;Kickapoo Tribe of Indians of the Kickapoo Reservation in Kansas;2011"
#DATA_SET_ID="-1383;Knox County Department of Air Quality Management;Knox County Department of Air Quality Management;2011"
#DATA_SET_ID="-1408;Lane Regional Air Pollution Authority;Lane Regional Air Pollution Authority;2011"
#DATA_SET_ID="-1377;Lincoln/Lancaster County Health Department;Lincoln/Lancaster County Health Department;2011"
#DATA_SET_ID="-1430;Louisiana Department of Environmental Quality;Louisiana Department of Environmental Quality;2011"
#DATA_SET_ID="-1366;Louisville Metro Air Pollution Control District;Louisville Metro Air Pollution Control District;2011"
#DATA_SET_ID="-1431;Maine Department of Environmental Protection;Maine Department of Environmental Protection;2011"
#DATA_SET_ID="-1363;Maricopa County Air Quality Department;Maricopa County Air Quality Department;2011"
#DATA_SET_ID="-1432;Maryland Department of the Environment;Maryland Department of the Environment;2011"
#DATA_SET_ID="-1433;Massachusetts Department of Environmental Protection;Massachusetts Department of Environmental Protection;2011"
#DATA_SET_ID="-1369;Mecklenburg County Air Quality;Mecklenburg County Air Quality;2011"
#DATA_SET_ID="-1387;Memphis and Shelby County Health Department - Pollution Control;Memphis and Shelby County Health Department - Pollution Control;2011"
#DATA_SET_ID="-1388;Metro Public Health of Nashville/Davidson County;Metro Public Health of Nashville/Davidson County;2011"
#DATA_SET_ID="-1434;Michigan Department of Environmental Quality;Michigan Department of Environmental Quality;2011"
#DATA_SET_ID="-1435;Minnesota Pollution Control Agency;Minnesota Pollution Control Agency;2011"
#DATA_SET_ID="-1436;Mississippi Dept of Environmental Quality ;Mississippi Dept of Environmental Quality ;2011"
#DATA_SET_ID="-1437;Missouri Department of Natural Resources;Missouri Department of Natural Resources;2011"
#DATA_SET_ID="-1438;Montana Department of Environmental Quality;Montana Department of Environmental Quality;2011"
#DATA_SET_ID="-1477;Navajo Nation;Navajo Nation;2011"
#DATA_SET_ID="-1439;Nebraska Environmental Quality;Nebraska Environmental Quality;2011"
#DATA_SET_ID="-1440;Nevada Division of Environmental Protection;Nevada Division of Environmental Protection;2011"
#DATA_SET_ID="-1441;New Hampshire Department of Environmental Services;New Hampshire Department of Environmental Services;2011"
#DATA_SET_ID="-1442;New Jersey Department of Environment Protection;New Jersey Department of Environment Protection;2011"
#DATA_SET_ID="-1443;New Mexico Environment Department Air Quality Bureau;New Mexico Environment Department Air Quality Bureau;2011"
#DATA_SET_ID="-1444;New York State Department of Environmental Conservation;New York State Department of Environmental Conservation;2011"
#DATA_SET_ID="-1475;Nez Perce Tribe;Nez Perce Tribe;2011"
#DATA_SET_ID="-1445;North Carolina Department of Environment and Natural Resources;North Carolina Department of Environment and Natural Resources;2011"
#DATA_SET_ID="-1446;North Dakota Department of Health;North Dakota Department of Health;2011"
#DATA_SET_ID="-1447;Ohio Environmental Protection Agency;Ohio Environmental Protection Agency;2011"
#DATA_SET_ID="-1448;Oklahoma Department of Environmental Quality;Oklahoma Department of Environmental Quality;2011"
#DATA_SET_ID="-1389;Olympic Region Clean Air Agency;Olympic Region Clean Air Agency;2011"
#DATA_SET_ID="-1384;Omaha Air Quality Control Division;Omaha Air Quality Control Division;2011"
#DATA_SET_ID="-1449;Oregon Department of Environmental Quality;Oregon Department of Environmental Quality;2011"
#DATA_SET_ID="-1450;Pennsylvania Department of Environmental Protection;Pennsylvania Department of Environmental Protection;2011"
#DATA_SET_ID="-1381;Philadelphia Air Management Services;Philadelphia Air Management Services;2011"
#DATA_SET_ID="-1396;Pinal County;Pinal County;2011"
#DATA_SET_ID="-1469;Puerto Rico;Puerto Rico;2011"
#DATA_SET_ID="-1390;Puget Sound Clean Air Agency;Puget Sound Clean Air Agency;2011"
#DATA_SET_ID="-1451;Rhode Island Department of Environmental Management;Rhode Island Department of Environmental Management;2011"
#DATA_SET_ID="-1481;Shoshone-Bannock Tribes of the Fort Hall Reservation of Idaho;Shoshone-Bannock Tribes of the Fort Hall Reservation of Idaho;2011"
#DATA_SET_ID="-1452;South Carolina Department of Health and Environmental Control;South Carolina Department of Health and Environmental Control;2011"
#DATA_SET_ID="-1453;South Dakota Department of Environment and Natural Resources;South Dakota Department of Environment and Natural Resources;2011"
#DATA_SET_ID="-1502;Southern Ute Indian Tribe;Southern Ute Indian Tribe;2011"
#DATA_SET_ID="563;Southwest Clean Air Agency;Southwest Clean Air Agency;2011"
#DATA_SET_ID="-1454;Tennessee Department of Environmental Conservation;Tennessee Department of Environmental Conservation;2011"
#DATA_SET_ID="-1455;Texas Commission on Environmental Quality;Texas Commission on Environmental Quality;2011"
#DATA_SET_ID="-1456;Utah Division of Air Quality;Utah Division of Air Quality;2011"
#DATA_SET_ID="-1457;Vermont Department of Environmental Conservation;Vermont Department of Environmental Conservation;2011"
#DATA_SET_ID="-1458;Virginia Department of Environmental Quality;Virginia Department of Environmental Quality;2011"
#DATA_SET_ID="-1459;Washington State Department of Ecology;Washington State Department of Ecology;2011"
#DATA_SET_ID="-1379;Washoe County Health District;Washoe County Health District;2011"
#DATA_SET_ID="-1460;West Virginia Division of Air Quality;West Virginia Division of Air Quality;2011"
#DATA_SET_ID="-1367;Western North Carolina Regional Air Quality Agency (Buncombe Co.);Western North Carolina Regional Air Quality Agency (Buncombe Co.);2011"
#REV_HISTORY v1(01/14/2015)  Allan Beidler.   removed oil and gas facilities  moved to pt_oilgas
#REV_HISTORY v2(01/14/2015)  Allan Beidler.   facility 6981111  moved from pt oil_gas as per Madeleine
#Removed additional ptegu sources from ptnonipm_2011NEIv2_POINT_20140913_revised_20141007_04dec2014_egu_removed_20150106_14jan2015_v2.csv
# Dataset formerly named ptnonipm_2011NEIv2_POINT_20140913_revised_20141007_egu_removed_20150115
#EXPORT_DATE=Wed Jun 03 08:53:11 EDT 2015
#EXPORT_VERSION_NAME=add facility 14563211
#EXPORT_VERSION_NUMBER=2
#REV_HISTORY v1(02/03/2015)  Chris Allen.   removed facility 14444611 (Caliente, NV)  this is not an active railyard, according to Madeleine Strum
#REV_HISTORY v2(02/09/2015)  Allan Beidler.   added 14563211 Landfill  dropped beforecountry_cd,region_cd,tribal_code,facility_id,unit_id,rel_point_id,process_id,agy_facility_id,agy_unit_id,agy_rel_point_id,agy_process_id,scc,poll,ann_value,ann_pct_red,facility_name,erptype,stkhgt,stkdiam,stktemp,stkflow,stkvel,naics,longitude,latitude,ll_datum,horiz_coll_mthd,design_capacity,design_capacity_units,reg_codes,fac_source_type,unit_type_code,control_ids,control_measures,current_cost,cumulative_cost,projection_factor,submitter_id,calc_method,data_set_id,facil_category_code,oris_facility_code,oris_boiler_id,ipm_yn,calc_year,date_updated,fug_height,fug_width_ydim,fug_length_xdim,fug_angle,zipcode,annual_avg_hours_per_year,jan_value,feb_value,mar_value,apr_value,may_value,jun_value,jul_value,aug_value,sep_value,oct_value,nov_value,dec_value,jan_pctred,feb_pctred,mar_pctred,apr_pctred,may_pctred,jun_pctred,jul_pctred,aug_pctred,sep_pctred,oct_pctred,nov_pctred,dec_pctred,comment
"US","01001",,"10583011","62385813","50346112","83296814",,,,,"2275050011","100414",0.0042677399999999999,,"Autauga County","1",,,,,,"48811",-86.5104500000000058,32.4387800000000013,,,,,,"100","300",,,,,,"USEPA",8,"2011EPA_Air","UNK",,,,"2011",20130210,,,,,"00000",,,,,,,,,,,,,,,,,,,,,,,,,,
"US","01001",,"10583011","62385813","50346112","83296814",,,,,"2275050011","100425",0.000987096999999999931,,"Autauga County","1",,,,,,"48811",-86.5104500000000058,32.4387800000000013,,,,,,"100","300",,,,,,"USEPA",8,"2011EPA_Air","UNK",,,,"2011",20130210,,,,,"00000",,,,,,,,,,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "nonpt_2011",
			freq:   Annually,
			period: Annual,
			fileData: `#FORMAT=FF10_NONPOINT
#COUNTRY=US
#YEAR=2011
#SELECTION_NAME=2011 NEI V2 with Biogenics
#INVENTORY_VERSION=General Purpose Release
#INVENTORY_LABEL=2011 NEI V2 with Biogenics
#CREATION_DATE=20141108 03:11
#CREATOR_NAME=Jonathan Miller
#VALUE_UNITS=TON
#INCLUDES_CAPS=true
#INCLUDES_HAPS=all
#DATA_SET_ID="720;2011EPA_AgBurningSF2;Sally Dombrowski;2011"
#DATA_SET_ID="543;2011EPA_Airports;Laurel Driver;2011"
#DATA_SET_ID="541;2011EPA_CMV;Laurel Driver;2011"
#DATA_SET_ID="740;2011EPA_CMVLADCO;Laurel Driver;2011"
#DATA_SET_ID="621;2011EPA_HAP-Augmentation;Madeleine Strum;2011"
#DATA_SET_ID="760;2011EPA_NP_Mercury;Jennifer Snyder;2011"
#DATA_SET_ID="500;2011EPA_NP_NoOverlap_w_Pt;Roy Huntley;2011"
#DATA_SET_ID="540;2011EPA_NP_Overlap_w_Pt;Roy Huntley;2011"
#DATA_SET_ID="620;2011EPA_PM-Augmentation;Madeleine Strum;2011"
#DATA_SET_ID="542;2011EPA_Rail;Laurel Driver;2011"
#DATA_SET_ID="640;2011EPA_biogenics;Madeleine Strum;2011"
#DATA_SET_ID="560;2011EPA_chrom_split;Madeleine Strum;2011"
#DATA_SET_ID="-1412;Alabama Department of Environmental Management;Alabama Department of Environmental Management;2011"
#DATA_SET_ID="-1413;Alaska Department of Environmental Conservation;Alaska Department of Environmental Conservation;2011"
#DATA_SET_ID="-1403;Bishop Paiute Tribe;Bishop Paiute Tribe;2011"
#DATA_SET_ID="-1416;California Air Resources Board;California Air Resources Board;2011"
#DATA_SET_ID="-1382;Chattanooga Air Pollution Control Bureau (CHCAPCB);Chattanooga Air Pollution Control Bureau (CHCAPCB);2011"
#DATA_SET_ID="-1378;Clark County Department of Air Quality and Environmental Management;Clark County Department of Air Quality and Environmental Management;2011"
#DATA_SET_ID="-1474;Coeur d'Alene Tribe;Coeur d'Alene Tribe;2011"
#DATA_SET_ID="-1417;Colorado Department of Public Health and Environment;Colorado Department of Public Health and Environment;2011"
#DATA_SET_ID="-1418;Connecticut Department Of Environmental Protection;Connecticut Department Of Environmental Protection;2011"
#DATA_SET_ID="-1420;DC Department of Health Air Quality Division;DC Department of Health Air Quality Division;2011"
#DATA_SET_ID="-1419;Delaware Deparment of Natural Resources and Environmental Control;Delaware Deparment of Natural Resources and Environmental Control;2011"
#DATA_SET_ID="-1492;Eastern Band of Cherokee Indians;Eastern Band of Cherokee Indians;2011"
#DATA_SET_ID="-1421;Florida Department of Environmental Protection;Florida Department of Environmental Protection;2011"
#DATA_SET_ID="-1422;Georgia Department of Natural Resources;Georgia Department of Natural Resources;2011"
#DATA_SET_ID="-1423;Hawaii Department of Health Clean Air Branch;Hawaii Department of Health Clean Air Branch;2011"
#DATA_SET_ID="-1424;Idaho Department of Environmental Quality;Idaho Department of Environmental Quality;2011"
#DATA_SET_ID="-1425;Illinois Environmental Protection Agency;Illinois Environmental Protection Agency;2011"
#DATA_SET_ID="-1426;Indiana Department of Environmental Management;Indiana Department of Environmental Management;2011"
#DATA_SET_ID="-1427;Iowa Department of Natural Resources;Iowa Department of Natural Resources;2011"
#DATA_SET_ID="-1428;Kansas Department of Health and Environment;Kansas Department of Health and Environment;2011"
#DATA_SET_ID="-1504;Kickapoo Tribe of Indians of the Kickapoo Reservation in Kansas;Kickapoo Tribe of Indians of the Kickapoo Reservation in Kansas;2011"
#DATA_SET_ID="-1383;Knox County Department of Air Quality Management;Knox County Department of Air Quality Management;2011"
#DATA_SET_ID="-1491;Kootenai Tribe of Idaho;Kootenai Tribe of Idaho;2011"
#DATA_SET_ID="-1430;Louisiana Department of Environmental Quality;Louisiana Department of Environmental Quality;2011"
#DATA_SET_ID="-1366;Louisville Metro Air Pollution Control District;Louisville Metro Air Pollution Control District;2011"
#DATA_SET_ID="-1431;Maine Department of Environmental Protection;Maine Department of Environmental Protection;2011"
#DATA_SET_ID="-1363;Maricopa County Air Quality Department;Maricopa County Air Quality Department;2011"
#DATA_SET_ID="-1432;Maryland Department of the Environment;Maryland Department of the Environment;2011"
#DATA_SET_ID="-1433;Massachusetts Department of Environmental Protection;Massachusetts Department of Environmental Protection;2011"
#DATA_SET_ID="-1387;Memphis and Shelby County Health Department - Pollution Control;Memphis and Shelby County Health Department - Pollution Control;2011"
#DATA_SET_ID="-1388;Metro Public Health of Nashville/Davidson County;Metro Public Health of Nashville/Davidson County;2011"
#DATA_SET_ID="-1434;Michigan Department of Environmental Quality;Michigan Department of Environmental Quality;2011"
#DATA_SET_ID="-1435;Minnesota Pollution Control Agency;Minnesota Pollution Control Agency;2011"
#DATA_SET_ID="-1437;Missouri Department of Natural Resources;Missouri Department of Natural Resources;2011"
#DATA_SET_ID="-1441;New Hampshire Department of Environmental Services;New Hampshire Department of Environmental Services;2011"
#DATA_SET_ID="-1442;New Jersey Department of Environment Protection;New Jersey Department of Environment Protection;2011"
#DATA_SET_ID="-1444;New York State Department of Environmental Conservation;New York State Department of Environmental Conservation;2011"
#DATA_SET_ID="-1475;Nez Perce Tribe;Nez Perce Tribe;2011"
#DATA_SET_ID="-1445;North Carolina Department of Environment and Natural Resources;North Carolina Department of Environment and Natural Resources;2011"
#DATA_SET_ID="-1371;Northern Cheyenne Tribe;Northern Cheyenne Tribe;2011"
#DATA_SET_ID="-1447;Ohio Environmental Protection Agency;Ohio Environmental Protection Agency;2011"
#DATA_SET_ID="-1448;Oklahoma Department of Environmental Quality;Oklahoma Department of Environmental Quality;2011"
#DATA_SET_ID="-1449;Oregon Department of Environmental Quality;Oregon Department of Environmental Quality;2011"
#DATA_SET_ID="-1450;Pennsylvania Department of Environmental Protection;Pennsylvania Department of Environmental Protection;2011"
#DATA_SET_ID="-1410;Prairie Band of Potawatomi Indians;Prairie Band of Potawatomi Indians;2011"
#DATA_SET_ID="-1478;Sac and Fox Nation of Missouri in Kansas and Nebraska Reservation;Sac and Fox Nation of Missouri in Kansas and Nebraska Reservation;2011"
#DATA_SET_ID="-1503;"Santee Sioux Nation, Nebraska";Santee Sioux Nation, Nebraska;2011"
#DATA_SET_ID="-1481;Shoshone-Bannock Tribes of the Fort Hall Reservation of Idaho;Shoshone-Bannock Tribes of the Fort Hall Reservation of Idaho;2011"
#DATA_SET_ID="-1452;South Carolina Department of Health and Environmental Control;South Carolina Department of Health and Environmental Control;2011"
#DATA_SET_ID="-1454;Tennessee Department of Environmental Conservation;Tennessee Department of Environmental Conservation;2011"
#DATA_SET_ID="-1455;Texas Commission on Environmental Quality;Texas Commission on Environmental Quality;2011"
#DATA_SET_ID="-1486;Tohono O'Odham Nation Reservation;Tohono O'Odham Nation Reservation;2011"
#DATA_SET_ID="-1456;Utah Division of Air Quality;Utah Division of Air Quality;2011"
#DATA_SET_ID="-1457;Vermont Department of Environmental Conservation;Vermont Department of Environmental Conservation;2011"
#DATA_SET_ID="-1458;Virginia Department of Environmental Quality;Virginia Department of Environmental Quality;2011"
#DATA_SET_ID="-1459;Washington State Department of Ecology;Washington State Department of Ecology;2011"
#DATA_SET_ID="-1379;Washoe County Health District;Washoe County Health District;2011"
#DATA_SET_ID="-1400;Washoe Tribe of California and Nevada;Washoe Tribe of California and Nevada;2011"
#DATA_SET_ID="-1460;West Virginia Division of Air Quality;West Virginia Division of Air Quality;2011"
#DATA_SET_ID="-1461;Wisconsin Department of Natural Resources;Wisconsin Department of Natural Resources;2011"
#DATA_SET_ID="-1462;Wyoming Department of Environmenal Quality;Wyoming Department of Environmenal Quality;2011"
#REV_HISTORY v1(11/11/2014)  James Beidler.   removed  non nonpoint sccs  for sectorization
#EXPORT_DATE=Wed Jan 21 10:06:42 EST 2015
#EXPORT_VERSION_NAME=remove 2102005000 from NC
#EXPORT_VERSION_NUMBER=5
#REV_HISTORY v1(11/11/2014)  James Beidler.   remove residential pfc  in PFC file
#REV_HISTORY v2(12/04/2014)  Chris Allen.   removed FIPS ending in 777  not modeled. only emissions with these FIPS were LEAD (CAS 7439921), which is why this wasn't caught until the NATA run
#REV_HISTORY v3(12/08/2014)  James Beidler.   removed PR and VI SCCs for replacement  updated emissions
#REV_HISTORY v3(12/08/2014)  James Beidler.   PR and VI updates  new PR and VI emissions
#REV_HISTORY v5(01/21/2015)  Chris Allen.   removed SCC 2102005000 in North Carolina  will be tagged in the next NEI version (if there is one)country_cd,region_cd,tribal_code,census_tract_cd,shape_id,scc,emis_type,poll,ann_value,ann_pct_red,control_ids,control_measures,current_cost,cumulative_cost,projection_factor,reg_codes,calc_method,calc_year,date_updated,data_set_id,jan_value,feb_value,mar_value,apr_value,may_value,jun_value,jul_value,aug_value,sep_value,oct_value,nov_value,dec_value,jan_pctred,feb_pctred,mar_pctred,apr_pctred,may_pctred,jun_pctred,jul_pctred,aug_pctred,sep_pctred,oct_pctred,nov_pctred,dec_pctred,comment
"US","54015",,,,"2102002000",,"92524",6.98020000000000053e-08,,,,,,,"63DDDDD&63DDDDD&63DDDDD&63DDDDD",5,2011,20140828,"2011EPA_HAP-Aug",,,,,,,,,,,,,,,,,,,,,,,,,
"US","54015",,,,"2102002000",,"532274",2.87420000000000015e-07,,,,,,,"63DDDDD&63DDDDD&63DDDDD&63DDDDD",5,2011,20140828,"2011EPA_HAP-Aug",,,,,,,,,,,,,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "nonroad_2011",
			freq:   Annually,
			period: Annual,
			fileData: `#FORMAT   FF10_NONROAD
#COUNTRY  US
#YEAR     2011
#DESC     Nonroad Source Inventory
#DESC     FF10 Nonroad format
#DESC     2011NEI Draft California nonroad inventory
#DESC     Based on eis_report_carb_nonroad_2011.zip dated April 30, 2013 from Madeleine Strum
#DESC     Ratioed missing VOC for SCCs with emissions from 2011 CARB modeler's data
# Dataset formerly named nonroad_ff10_california_2011neidraft_augmented_VOC
#EXPORT_DATE=Tue Dec 30 14:54:36 EST 2014
#EXPORT_VERSION_NAME=revise VOC augmentation
#EXPORT_VERSION_NUMBER=7
#REV_HISTORY v1(07/10/2013)  Chris Allen.   removed 63 VOC records  These SCCs are invalid, or duplicated in point (airport ground support)
#REV_HISTORY v2(07/11/2013)  James Beidler.   Replaced 'EXH__PM10' with 'EXH__PM10-PRI' for column 'poll'  Replaced 'EXH__PM2_5' with 'EXH__PM25-PRI' for column 'poll'For -PRI testing
#REV_HISTORY v3(08/12/2013)  Allan Beidler.
#REV_HISTORY v4(08/13/2013)  Allan Beidler.   more VOCs  missing
#REV_HISTORY v6(08/13/2013)  Allan Beidler.   fixed duplicate  error in scc
#REV_HISTORY v7(09/10/2013)  Allan Beidler.   removed version 3 & 4 records  will replaced with other augmented VOC
#REV_HISTORY v7(09/10/2013)  Allan Beidler.   new VOC augmentation  other was too high. this is B+A+F
#REV_HISTORY v7(09/10/2013)  Allan Beidler.   removed duplicate  duplicatecountry_cd,region_cd,tribal_code,census_tract_cd,shape_id,scc,emis_type,poll,ann_value,ann_pct_red,control_ids,control_measures,current_cost,cumulative_cost,projection_factor,reg_codes,calc_method,calc_year,date_updated,data_set_id,jan_value,feb_value,mar_value,apr_value,may_value,jun_value,jul_value,aug_value,sep_value,oct_value,nov_value,dec_value,jan_pctred,feb_pctred,mar_pctred,apr_pctred,may_pctred,jun_pctred,jul_pctred,aug_pctred,sep_pctred,oct_pctred,nov_pctred,dec_pctred,comment
"US","06001",,,,"2260001010","X","EXH__100414",0.103299000000000002,,,,,,,,,2011,,,0.0061979438999999999,0.0061979438999999999,0.00723093250000000042,0.00826391700000000083,0.0092969178999999999,0.0103298863000000005,0.0103298863000000005,0.0103298863000000005,0.0092969178999999999,0.0092969178999999999,0.0092969178999999999,0.00723093250000000042,,,,,,,,,,,,,
"US","06001",,,,"2260001010","X","EXH__100425",0.00872005000000000002,,,,,,,,,2011,,,0.000523202800000000011,0.000523202800000000011,0.000610403600000000018,0.000697603700000000004,0.000784804500000000011,0.000872005300000000018,0.000872005300000000018,0.000872005300000000018,0.000784804500000000011,0.000784804500000000011,0.000784804500000000011,0.000610403600000000018,,,,,,,,,,,,,
`,
		},
		testData{
			name:   "ptegu_daily_2011",
			freq:   Annually,
			period: Annual,
			fileData: `#FORMAT     FF10_DAILY_POINT
#COUNTRY    US
#YEAR       2011
#DESC       Non-CEM sources
#DESC       Created with /garnet/oaqps/em_v6.2/2011platform/work/point_neiv2_final/ptday/ptday_noncem.sas
#DESC       Ptday based on 2011NEIv2
#EXPORT_DATE=Thu Jan 29 10:21:34 EST 2015
#EXPORT_VERSION_NAME=additional EGUs
#EXPORT_VERSION_NUMBER=2
#REV_HISTORY v1(12/03/2014)  Allan Beidler.   removed Anadarko plant 1046611  tpo be replaced
#REV_HISTORY v1(12/03/2014)  Allan Beidler.   replaced facility 1046611  new temporalization
#REV_HISTORY v2(01/29/2015)  Chris Allen.   appended additional EGUs from ptipm_ff10_noncem_2011eg_additional_csv_06jan2015_nf_v1  so we have a complete single dataset for ptegu
#REV_HISTORY v2(01/29/2015)  Chris Allen.   appended additional EGUs from ptipm_ff10_noncem_2011eg_additional_20150115_15jan2015_v0  so we have a single complete datasetfor ptegucountry_cd,region_cd,tribal_code,facility_id,unit_id,rel_point_id,process_id,scc,poll,op_type_cd,calc_method,date_updated,monthnum,monthtot,dayval1,dayval2,dayval3,dayval4,dayval5,dayval6,dayval7,dayval8,dayval9,dayval10,dayval11,dayval12,dayval13,dayval14,dayval15,dayval16,dayval17,dayval18,dayval19,dayval20,dayval21,dayval22,dayval23,dayval24,dayval25,dayval26,dayval27,dayval28,dayval29,dayval30,dayval31,comment
"US","01033",,"7212811","10817213","10769412","61093014","20100101","NOX",,,,1,0.0307227831999999992,0,0,0.00153071099999999992,0,0.000895592999999999975,0.000396853000000000017,0.00230423400000000006,0.000138094999999999998,0,0.0025971509999999998,0,0.00278636899999999991,0.00413633499999999966,0.0159374439999999985,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,
"US","01033",,"7212811","10817213","10769412","61093014","20100101","NOX",,,,2,0.0370616142000000023,0,0.00103074599999999992,0,0.000140504000000000009,0,0.00269412700000000015,0.00543288899999999977,0.000692959000000000045,0.0102448769999999995,0.00389012099999999986,0.0125307749999999992,0,0,0,0.000118152000000000003,0,0,0.000117011999999999997,0,0,0,1.5237e-05,0.000154217000000000004,0,0,0,0,0,,,,
`,
		},
		testData{
			name:   "ptegu_annual_2011",
			freq:   Annually,
			period: Annual,
			fileData: `#FORMAT=FF10_POINT
#COUNTRY=US
#YEAR=2011
#SELECTION_NAME=2011 NEI V2 with Biogenics
#INVENTORY_VERSION=General Purpose Release
#INVENTORY_LABEL=2011 NEI V2 with Biogenics
#CREATION_DATE=20140913 09:09
#CREATOR_NAME=Jonathan Miller
#VALUE_UNITS=TON
#INCLUDES_CAPS=true
#INCLUDES_HAPS=all
#REV_HISTORY v1(10/28/2014)  Allan Beidler.   changed HG emissions for facility 7072311  as per email from Madeleine Oct. 28, 2014
#REV_HISTORY v1(11/10/2014)  Allan Beidler.   added additional info  for MN HG emissions
#EXPORT_DATE=Wed Jun 03 08:53:05 EDT 2015
#EXPORT_VERSION_NAME=append additional EGUs
#EXPORT_VERSION_NUMBER=1
#REV_HISTORY v1(01/29/2015)  Chris Allen.   appended additional EGUs from 2011NEIv2_move_ptnonipm_ptipm_20150105_06jan2015_v0.csv  so that we have a single inventory forptegu
#REV_HISTORY v1(01/29/2015)  Chris Allen.   appended additional EGUs from 2011NEIv2ipm_xwalkmove_ptnonipm_ptipm_20150114_15jan2015_v0.csv  so that we have a single complete dataset for ptegucountry_cd,region_cd,tribal_code,facility_id,unit_id,rel_point_id,process_id,agy_facility_id,agy_unit_id,agy_rel_point_id,agy_process_id,scc,poll,ann_value,ann_pct_red,facility_name,erptype,stkhgt,stkdiam,stktemp,stkflow,stkvel,naics,longitude,latitude,ll_datum,horiz_coll_mthd,design_capacity,design_capacity_units,reg_codes,fac_source_type,unit_type_code,control_ids,control_measures,current_cost,cumulative_cost,projection_factor,submitter_id,calc_method,data_set_id,facil_category_code,oris_facility_code,oris_boiler_id,ipm_yn,calc_year,date_updated,fug_height,fug_width_ydim,fug_length_xdim,fug_angle,zipcode,annual_avg_hours_per_year,jan_value,feb_value,mar_value,apr_value,may_value,jun_value,jul_value,aug_value,sep_value,oct_value,nov_value,dec_value,jan_pctred,feb_pctred,mar_pctred,apr_pctred,may_pctred,jun_pctred,jul_pctred,aug_pctred,sep_pctred,oct_pctred,nov_pctred,dec_pctred,comment
"US","01001",,"10583111","52263713","50910612","71808614","0010","X001A","001A","01","20100201","100414",0.154481499999999994,,"Southern Power Company-E B Harris Generating Plant","2",160,19,167,14504.7999999999993,51.2000000000000028,"221112",-86.5738309999999984,32.3816589999999991,,,2260,"E6BTU/HR","R63-0083","125","140",,,,,,"USEPA",8,"2011 EPA EG","HAPCA","7897","1A","7897_G_CT1A","2011",20130317,,,,,"36067",,,,,,,,,,,,,,,,,,,,,,,,,,"calc from CAMD2011 heat input and 2008 Pechan EF"
"US","01001",,"10583111","52263813","50940312","71809514","0010","X001B","001B","01","20100201","100414",0.156291000000000013,,"Southern Power Company-E B Harris Generating Plant","2",160,19,167,14504.7999999999993,51.2000000000000028,"221112",-86.5738249999999994,32.3819889999999972,,,2260,"E6BTU/HR","R63-0083","125","140",,,,,,"USEPA",8,"2011 EPA EG","HAPCA","7897","1B","7897_G_ST1","2011",20130317,,,,,"36067",,,,,,,,,,,,,,,,,,,,,,,,,,"calc from CAMD2011 heat input and 2008 Pechan EF"
`,
		},
		testData{
			name:   "canada_point_2010",
			freq:   Annually,
			period: Annual,
			fileData: `#FORMAT=FF10_POINT
#COUNTRY=CANADA
#YEAR=2010
#YEAR     2010
#DESC     Point emissions for non-VOC species
#DESC     ANNUAL
#DESC     david.niemi@ec.gc.ca, mourad.sassi@ec.gc.ca
#DESC     VOC emissions for CB05 mechanism, in short tons, for facilities reporting to the NPRI
#DESC     Created: July 7, 2014
#DESC     2010v1
#EXPORT_DATE=Mon Jan 05 12:34:12 EST 2015
#EXPORT_VERSION_NAME=Initial Version
#EXPORT_VERSION_NUMBER=0
country_cd,region_cd,tribal_code,facility_id,unit_id,rel_point_id,process_id,agy_facility_id,agy_unit_id,agy_rel_point_id,agy_process_id,scc,poll,ann_value,ann_pct_red,facility_name,erptype,stkhgt,stkdiam,stktemp,stkflow,stkvel,naics,longitude,latitude,ll_datum,horiz_coll_mthd,design_capacity,design_capacity_units,reg_codes,fac_source_type,unit_type_code,control_ids,control_measures,current_cost,cumulative_cost,projection_factor,submitter_id,calc_method,data_set_id,facil_category_code,oris_facility_code,oris_boiler_id,ipm_yn,calc_year,date_updated,fug_height,fug_width_ydim,fug_length_xdim,fug_angle,zipcode,annual_avg_hours_per_year,jan_value,feb_value,mar_value,apr_value,may_value,jun_value,jul_value,aug_value,sep_value,oct_value,nov_value,dec_value,jan_pctred,feb_pctred,mar_pctred,apr_pctred,may_pctred,jun_pctred,jul_pctred,aug_pctred,sep_pctred,oct_pctred,nov_pctred,dec_pctred,comment
"CA","24001",,"3060","53998084","-9","-9",,,,,"30300199","PM2_5",50.1876681999999974,,"RioTintoAlcan",,-9,-9,-9,,-9,"331313",-71.1289000000000016,48.3006999999999991,,,,,,,,,,,,,,,,,,,,"2010",20130606,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
"CA","24001",,"3060","53998084","10579","-9",,,,,"30300199","SO2",1073.64442200000008,,"RioTintoAlcan",,164.041994799999998,11.1220472400000006,185,,49.2125984299999999,"331313",-71.1289000000000016,48.3006999999999991,,,,,,,,,,,,,,,,,,,,"2010",20130606,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
`,
		},
	}
	overallReport := new(InventoryReport)
	for _, test := range table {
		e, err := NewEmissionsReader(nil, test.freq, Ton)
		if err != nil {
			t.Errorf("%s: %v", test.name, err)
			continue
		}
		r := bytes.NewReader([]byte(test.fileData))
		invF, err := NewInventoryFile(test.name, r, test.period, e.inputConv)
		if err != nil {
			t.Errorf("%s: %v", test.name, err)
			continue
		}

		_, report, err := e.ReadFiles([]*InventoryFile{invF}, nil)
		if err != nil {
			t.Errorf("%s: %v", test.name, err)
			continue
		}
		overallReport.Files = append(overallReport.Files, report.Files...)
	}
	reportTable := overallReport.TotalsTable()
	wantReportTable := Table{
		[]string{"Group", "File", "100414 (kg)", "100425 (kg)", "106990 (kg)", "2381217 (kg)", "532274 (kg)", "92524 (kg)", "ALD2 (kg)", "ALDX (kg)", "CO (kg)", "NAPHTH_72 (kg)", "NH3 (kg)", "NOX (kg)", "PM10 (kg)", "PM2_5 (kg)", "SO2 (kg)", "VOC (kg)"},
		[]string{"", "ag2005", "", "", "", "", "", "", "", "", "", "", "17018.94482145", "", "", "", "", ""},
		[]string{"", "ptipm_annual_2005", "", "", "", "", "", "", "", "", "83685.9049019235", "", "94.5652366462185", "", "", "", "", ""},
		[]string{"", "avefire_ida_2005", "", "", "", "", "", "", "", "", "3.6149325535814995e+06", "", "16261.291125", "77549.07679199999", "351477.94203000003", "301449.6829905", "21262.874185499997", "827819.5546794449"},
		[]string{"", "avefire_orl_2005", "", "", "1323.77760597285", "29.717456460614994", "", "", "", "", "", "", "", "", "", "", "", ""},
		[]string{"", "afdust_2005", "", "", "", "", "", "", "", "", "", "", "", "", "46880.69449942499", "2772.1146880124998", "", ""},
		[]string{"", "nonroad_CA_jan_2005", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "63.50886680420762"},
		[]string{"", "nonroad_caps_jan_2005", "", "", "", "", "", "", "", "", "11.303781555106996", "", "0.0004493072914266338", "", "", "", "", "0.08535955941430691"},
		[]string{"", "onroad_runpm_jan_2005", "", "", "", "", "", "", "", "", "", "2.3001041250289997", "", "", "", "", "", ""},
		[]string{"", "onroad_startpm_jan_2005", "", "", "", "", "", "", "", "", "", "0.19069346970737497", "", "", "", "", "", ""},
		[]string{"", "onroad_CA_2005", "", "", "", "", "", "", "", "", "", "", "", "", "5.02826018108275", "", "", ""},
		[]string{"", "onroad_not2moves_jan_2005", "", "", "", "", "", "", "", "", "", "", "", "", "0.00763765345851925", "0.49510684293424995", "", ""},
		[]string{"", "onroad_moves_jan_2005", "", "", "", "", "", "", "", "", "", "", "", "", "2.0544119599066244", "", "", "18.29001020712975"},
		[]string{"", "onroad_canada_2005", "", "", "", "", "", "", "", "", "1.26119580255e+06", "", "", "", "", "", "", "617738.5539"},
		[]string{"", "onroad_mexico_border_2005", "", "", "", "", "", "", "", "", "1.157760746094e+07", "", "28171.541553", "479258.94089399994", "34258.535178599996", "31264.289439449996", "59954.350722899995", "1.4742562737465e+06"},
		[]string{"", "onroad_mexico_interior_2005", "", "", "", "", "", "", "", "", "3.3427420362405e+07", "", "48840.014881598996", "1.1605103195027248e+06", "87470.0250086055", "79820.0223884355", "153250.049365464", "3.8808012071004e+06"},
		[]string{"", "canada_point1_2005", "", "", "", "", "", "", "", "", "", "", "", "", "", "671.3168999999999", "", ""},
		[]string{"", "canada_point2_2005", "", "", "", "", "", "", "18.1437", "3.7234657175819996", "", "", "", "", "", "", "", ""},
		[]string{"", "canada_point3_2005", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "190.50976322914502", ""},
		[]string{"", "mexico_point_border_2005", "", "", "", "", "", "", "", "", "4481.4939", "", "0", "20574.955799999996", "18542.861399999998", "15059.270999999999", "243751.53764999998", "7.031046623999999e+06"},
		[]string{"", "mexico_point_interior_2005", "", "", "", "", "", "", "", "", "6205.500109335", "", "0", "52263.397771829994", "26785.79650743", "19275.901534466997", "589361.5712318999", "1306.300315002"},
		[]string{"", "point_offshore_2005", "", "", "", "", "", "", "", "", "132.59728340082899", "", "", "", "", "", "", ""},
		[]string{"", "alm_caps_2005", "", "", "", "", "", "", "", "", "0.6908942784330074", "", "", "3.967655893422304", "", "", "", ""},
		[]string{"", "nonpt_2005", "", "", "", "", "", "", "", "", "9.07185", "", "0.7175008718835", "", "", "", "", ""},
		[]string{"", "nonpt_mexico_2005", "", "", "", "", "", "", "", "", "14912.162206986599", "", "0", "122825.29913427448", "72058.76112115395", "46611.841497555295", "1.2983861901501545e+06", "774.73665224505"},
		[]string{"", "ptinv_cem_2005", "", "", "", "", "", "", "", "", "83685.9049019235", "", "94.5652366462185", "", "", "", "", ""},
		[]string{"", "ptnonipm_cap_2005", "", "", "", "", "", "", "", "", "12.70059", "", "", "117.93405", "", "", "", ""},
		[]string{"", "secac3_2005_caps", "", "", "", "", "", "", "", "", "", "", "", "84722.635853699", "", "", "", ""},
		[]string{"", "ptnonipm_2011", "3.8716297118999994", "0.8954795919449999", "", "", "", "", "", "", "", "", "", "", "", "", "", ""},
		[]string{"", "nonpt_2011", "", "", "", "", "0.0002607431127", "6.332332737e-05", "", "", "", "", "", "", "", "", "", ""},
		[]string{"", "nonroad_2011", "93.71130358715548", "7.910698922124", "", "", "", "", "", "", "", "", "", "", "", "", "", ""},
		[]string{"", "ptegu_daily_2011", "", "", "", "", "", "", "", "", "", "", "", "61.492991821185", "", "", "", ""},
		[]string{"", "ptegu_annual_2011", "281.92815041250003", "", "", "", "", "", "", "", "", "", "", "", "", "", "", ""},
		[]string{"", "canada_point_2010", "", "", "", "", "", "", "", "", "", "", "", "", "", "45529.49977601699", "973994.11497207", ""},
	}

	if len(wantReportTable) != len(reportTable) {
		t.Errorf("reportTable: want length does not equal have length")
	}

	for i, w := range wantReportTable {
		h := reportTable[i]
		if !reflect.DeepEqual(w, h) {
			t.Errorf("reportTable: want: %v, have %v", w, h)
			break
		}
	}
}

func TestPeriodToTime(t *testing.T) {
	start, end, err := Feb.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}
	startWant := "2005-02-01 00:00:00 +0000 UTC"
	endWant := "2005-03-01 00:00:00 +0000 UTC"
	if fmt.Sprint(start) != startWant {
		t.Errorf("want %s; got %s", startWant, start)
	}
	if fmt.Sprint(end) != endWant {
		t.Errorf("want %s; got %s", endWant, end)
	}

	start, end, err = Dec.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}
	startWant = "2005-12-01 00:00:00 +0000 UTC"
	endWant = "2006-01-01 00:00:00 +0000 UTC"
	if fmt.Sprint(start) != startWant {
		t.Errorf("want %s; got %s", startWant, start)
	}
	if fmt.Sprint(end) != endWant {
		t.Errorf("want %s; got %s", endWant, end)
	}

	start, end, err = Annual.TimeInterval("2005")
	if err != nil {
		t.Error(err)
	}
	startWant = "2005-01-01 00:00:00 +0000 UTC"
	endWant = "2006-01-01 00:00:00 +0000 UTC"
	if fmt.Sprint(start) != startWant {
		t.Errorf("want %s; got %s", startWant, start)
	}
	if fmt.Sprint(end) != endWant {
		t.Errorf("want %s; got %s", endWant, end)
	}

	start, end, err = Annual.TimeInterval("0001")
	if err != nil {
		t.Error(err)
	}
	startWant = "0001-01-01 00:00:00 +0000 UTC"
	endWant = "0002-01-01 00:00:00 +0000 UTC"
	if fmt.Sprint(start) != startWant {
		t.Errorf("want %s; got %s", startWant, start)
	}
	if fmt.Sprint(end) != endWant {
		t.Errorf("want %s; got %s", endWant, end)
	}
}
