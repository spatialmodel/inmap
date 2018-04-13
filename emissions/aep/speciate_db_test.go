/*
Copyright © 2017 the InMAP authors.
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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.*/

package aep

import (
	"bytes"
	"math"
	"testing"
)

var (
	speciesPropertiesExample = `ID,CAS,CAS no hyphen,EPAID,SAROAD,PAMS,HAPS,NAME,SYMBOL,SPEC_MW,NonVOCTOG,NOTE,SRS ID,Molecular Formula,Smiles Notation
592,"106-97-8","106978",,"43212",1,0,"N-butane","N_BUTA",5.8122199999999992e+01,0,,"24026","C4H10","C(CC)C"
605,"109-66-0","109660",,"43220",1,0,"N-pentane","N_PENT",7.2148780000000002e+01,0,,"26021","C5H12","C(CCC)C"
671,"74-98-6","74986",,"43204",1,0,"Propane","N_PROP",4.4095619999999997e+01,0,,"5207","C3H8","C(C)C"
2284,"N/A","N/A",,"99999",0,0,"Unidentified","UNID",1.3719212445472201e+02,0,,,,
118,"540-84-1","540841",,"43276",1,1,"2,2,4-trimethylpentane","PA224M",1.1422852000000000e+02,0,,"51961","C8H18","C(CC(C)C)(C)(C)C"
508,"78-78-4","78784",,"98132",1,0,"Isopentane (or 2-Methylbutane)","IPENTA",7.2148780000000002e+01,0,,"7310","C5H12","C(CC)(C)C"
601,"110-54-3","110543",,"43231",1,1,"N-hexane","N_HEX",8.6175359999999998e+01,0,,"26740","C6H14","C(CCCC)C"
717,"108-88-3","108883",,"45202",1,1,"Toluene","TOLUE",9.2138419999999996e+01,0,,"25452","C7H8","c1(ccccc1)C"
199,"107-83-5","107835",,"43229",1,0,"2-methylpentane (isohexane)","PENA2M",8.6175359999999998e+01,0,,"24646","C6H14","C(CCC)(C)C"
248,"96-14-0","96140",,"43230",1,0,"3-methylpentane","PENA3M",8.6175359999999998e+01,0,,"16634","C6H14","C(CC)(CC)C"
2605,"10102-43-9","10102439",,,0,0,"Nitrogen Monoxide (Nitric Oxide)","NO",30.00,0,,"167916","NO","N=O"
613,"14797-55-8","14797558",,"12306",0,0,"Nitrate","NO3-",62.00,0,,"197186","NO3","[O-][N+](=O)[O-]"
700,"7704-34-9","7704349",,"12169",0,0,"Sulfur","S",32.07,0,,"152744","S","S"
2606,"10102-44-0","10102440",,,0,0,"Nitrogen Dioxide","NO2",46.00,0,,"197194","NO2","[O-]N=O"
797,"7440-44-0","7440440",,"12116",0,0,"Elemental Carbon","EC",12.01,0,,"150037","C","C"
699,"14808-79-8","14808798",,"12403",0,0,"Sulfate","SO4=",96.05,0,,"197301","O4S","[O-]S(=O)(=O)[O-]"
626,,,"E701250","11102",0,0,"Organic carbon","OC",12.01,0,,"701250",,
2669,,,,,0,0,"Particulate Non-Carbon Organic Matter","PNCOM",,0,"Particulate Non-Carbon Organic Matter is calculated for each source category by multiplying OC emissions by a source-category specific OM/OC (OM = organic matter) ratio to calculate an OM emission, and subtracting OC from OM.  PNCOM = POC*(OM/OC Ratio - 1)  where POC is from each profile and OM/OC ratio is based on the ""Supporting information for: Emissions Inventory of PM2:5 Trace Elements across the United States"", By Adam Reff, Prakash V. Bhave, Heather Simon, Thompson G. Pace, George A.Pouliot, J. David Mobley, and Marc Houyoux (http://www.epa.gov/AMD/peer/products/Reff_ES&T2008_supportInfo.pdf), e.g., 1.25 for on/off-road motor vehicle exhaus, 1.7 for wood combustion, 1.4 for other sources including marine vessel engines.",,,
2671,,,,,0,0,"Other Unspeciated PM2.5","PMO",,0,"Calculated by subtracting the sum of speciated compounds in a profile from 100% of PM2.5 mass",,,
`

	gasProfileExample = `P_NUMBER,NAME,QUALITY,P_DATE,NOTES,TOTAL,MASTER_POL,T_METHOD,NORM_BASIS,ORIG_COMPO,STANDARD,J_RATING,V_RATING,D_RATING,REGION,SIBLING,VERSION,VOCtoTOG,CONTROLS,TEST_YEAR
"2487","Composite of 7 Emission Profiles from Crude Oil Storage Tanks - 1993","N/A","07/01/99 00:00:00","Profile normalized to equal 100% for the sum of the 55 PAMS (Photochemical Assessment Monitoring Stations) pollutants + MTBE, UNIDENTIFIED and OTHER.",1.00000000e+02,,,,,1,,,,,,"3.2",1.04351461e+00,"Not Available",
"8869","Gasoline Headspace Vapor - 0% Ethanol (E0) Combined - EPAct/V2/E-89 Program ","B","12/09/12 00:00:00","Repeat analysis for instrumental precision is more commonly performed with repeatability of 5% or better.  The concentration of ethanol was corrected for the reduced FID response due to the presence of an oxygen atom. For ethanol, the carbon atom attached to the hydroxyl (OH) group has about 50% reduced response with the FID compared to the other carbon in the compound. Consequently the correction for the 2 carbons in ethanol is given as 2/1.5 or 1.33.",1.00000000e+02,"VOC","To prepare for testing, 25 ml aliquots were put into 50 ml Erlenmeyer flasks then placed into a constant water-temperature bath maintained at 25 C.  After a 20 min equilibration time, 100 μL aliquots of the headspace vapors were taken from the Erlenmeyer flask and injected into the SUMMA canisters.  The prepared canisters were allowed to equilibrate for a 12 to 15 h period before performing both GC/FID and GC/MS analyses.","VOC","C",1,5.0000000000000000e+00,5.0000000000000000e+00,3.0000000000000000e+00,"United States","8863; 8866","4.4",,"None","2010"
"8870","Gasoline Headspace Vapor - 10% Ethanol (E10) Combined - EPAct/V2/E-89 Program ","B","12/09/12 00:00:00","Repeat analysis for instrumental precision is more commonly performed with repeatability of 5% or better.  The concentration of ethanol was corrected for the reduced FID response due to the presence of an oxygen atom. For ethanol, the carbon atom attached to the hydroxyl (OH) group has about 50% reduced response with the FID compared to the other carbon in the compound. Consequently the correction for the 2 carbons in ethanol is given as 2/1.5 or 1.33.",1.00000000e+02,"VOC","To prepare for testing, 25 ml aliquots were put into 50 ml Erlenmeyer flasks then placed into a constant water-temperature bath maintained at 25 C.  After a 20 min equilibration time, 100 μL aliquots of the headspace vapors were taken from the Erlenmeyer flask and injected into the SUMMA canisters.  The prepared canisters were allowed to equilibrate for a 12 to 15 h period before performing both GC/FID and GC/MS analyses.","VOC","C",1,5.0000000000000000e+00,5.0000000000000000e+00,3.0000000000000000e+00,"United States","8864; 8867","4.4",,"None","2010"
`

	gasSpeciesExample = `ID,SPECIES_ID,P_NUMBER,WEIGHT_PER,UNCERTAINT,ANLYMETHOD,UNC_METHOD
100479,592,"2487",2.4500000000000000e+01,-9.9000000000000000e+01,"Not Available","N/A"
100488,605,"2487",1.2770000000000000e+01,-9.9000000000000000e+01,"Not Available","N/A"
184718,118,"8869",5.7602109436999998e+00,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
184779,508,"8869",2.4589274217000000e+01,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
184785,592,"8869",5.4417833466000003e+00,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
184788,601,"8869",6.0573857346000004e+00,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
184791,605,"8869",6.5569568292999998e+00,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
184798,717,"8869",6.8650783927000001e+00,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
184838,118,"8870",5.3981473934000004e+00,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
184864,199,"8870",1.1033755298000001e+01,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
184877,248,"8870",6.5965412536999999e+00,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
184900,508,"8870",1.0291311908000001e+01,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
184919,717,"8870",5.2128155925000002e+00,-9.9000000000000000e+01,"GC-MS and GC-FID","N/A"
`

	otherGasesSpeciesExample = `ID,SPECIES_ID,P_NUMBER,ANLYMETHOD,PHASE,WEIGHT_PER,SPECIES EMISSION RATE,UNCERTAINT,UNC_METHOD,PM EMISSION RATE,VOC EMISSION RATE,OTHER EMISSION RATE,EMISSION RATE UNIT
298,2605,"HONO","Tunable Infrared Laser Differential Absorption Spectrometer (TILDAS)","Gas",9e+01,,,,,,,
299,2606,"HONO","Tunable Infrared Laser Differential Absorption Spectrometer (TILDAS)","Gas",15e+00,,,,,,,
300,2605,"6181","Tunable Infrared Laser Differential Absorption Spectrometer (TILDAS)","Gas",1.6888888888888889e+01,,,,,,,
301,2606,"6181","Tunable Infrared Laser Differential Absorption Spectrometer (TILDAS)","Gas",8.3111111111111114e+01,,,,,,,
`

	pmSpeciesExample = `ID,SPECIES_ID,P_NUMBER,WEIGHT_PER,UNCERTAINT,UNC_METHOD,ANLYMETHOD
182464,626,"91112",2.5e+01,-9.9000000000000000e+01,"N/A","Thermal/Optical Transmission"
182465,797,"91112",3.8e+01,-9.9000000000000000e+01,"N/A","Thermal/Optical Transmission"
182466,613,"91112",2.1e+00,-9.9000000000000000e+01,"N/A","Ion Chromatography (IC)"
182467,699,"91112",8.5e+00,-9.9000000000000000e+01,"N/A","Ion Chromatography (IC)"
182468,2669,"91112",9.8e+00,-9.9000000000000000e+01,"N/A","Inferred"
182469,700,"91112",2.8e+00,-9.9000000000000000e+01,"N/A","X-Ray Fluorescence (XRF)"
182470,2671,"91112",1.6e+01,-9.9000000000000000e+01,"N/A","Inferred"
`
)

func TestSpeciateDB(t *testing.T) {
	speciesProperties := bytes.NewBuffer([]byte(speciesPropertiesExample))
	gasProfile := bytes.NewBuffer([]byte(gasProfileExample))
	gasSpecies := bytes.NewBuffer([]byte(gasSpeciesExample))
	otherGasesSpecies := bytes.NewBuffer([]byte(otherGasesSpeciesExample))
	pmSpecies := bytes.NewBuffer([]byte(pmSpeciesExample))
	specDB, err := NewSpeciateDB(speciesProperties, gasProfile, gasSpecies, otherGasesSpecies, pmSpecies)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		code   string
		Type   SpeciationType
		result map[string]float64
	}{
		{
			code: "HONO",
			Type: "NOx",
			result: map[string]float64{
				"2605": 0.9 / (0.9 + 0.15),
				"2606": 0.15 / (0.9 + 0.15),
			},
		},
		{
			code: "2487",
			Type: "VOC",
			result: map[string]float64{
				"592": 2.45e+01 / (2.45e+01 + 1.277e+01),
				"605": 1.277e+01 / (2.45e+01 + 1.277e+01),
			},
		},
		{
			code: "91112",
			Type: "PM2.5",
			result: map[string]float64{
				"626":  2.5e+01 / (2.5e+01 + 3.8e+01 + 2.1e+00 + 8.5e+00 + 9.8e+00 + 2.8e+00 + 1.6e+01),
				"797":  3.8e+01 / (2.5e+01 + 3.8e+01 + 2.1e+00 + 8.5e+00 + 9.8e+00 + 2.8e+00 + 1.6e+01),
				"613":  2.1e+00 / (2.5e+01 + 3.8e+01 + 2.1e+00 + 8.5e+00 + 9.8e+00 + 2.8e+00 + 1.6e+01),
				"699":  8.5e+00 / (2.5e+01 + 3.8e+01 + 2.1e+00 + 8.5e+00 + 9.8e+00 + 2.8e+00 + 1.6e+01),
				"2669": 9.8e+00 / (2.5e+01 + 3.8e+01 + 2.1e+00 + 8.5e+00 + 9.8e+00 + 2.8e+00 + 1.6e+01),
				"700":  2.8e+00 / (2.5e+01 + 3.8e+01 + 2.1e+00 + 8.5e+00 + 9.8e+00 + 2.8e+00 + 1.6e+01),
				"2671": 1.6e+01 / (2.5e+01 + 3.8e+01 + 2.1e+00 + 8.5e+00 + 9.8e+00 + 2.8e+00 + 1.6e+01),
			},
		},
	}

	for _, test := range tests {
		var profile map[string]float64
		profile, err = specDB.Speciation(test.code, test.Type)
		if err != nil {
			t.Error(err)
		}
		if mapDifferent(test.result, profile) {
			t.Errorf("test %+v: want %v, got %v", test, test.result, profile)
		}
	}

	id, err := specDB.IDFromName("N-pentane")
	if err != nil {
		t.Error(err)
	}
	if id != "605" {
		t.Errorf("wrong ID for N-pentane: have %s, want 605", id)
	}
}

// mapDifferent returns true if a and b are significantly different.
func mapDifferent(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return true
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return true
		}
		diff := 2 * math.Abs(va-vb) / (va + vb)
		if diff > 1.e-8 || math.IsNaN(diff) {
			return true
		}
	}
	return false
}
