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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmaputil

import (
	"fmt"
	"log"
	"os"

	"github.com/spatialmodel/inmap"
)

// Preproc preprocesses chemical transport model
// output as specified by information in cfg
// and saves the result for use in future InMAP simulations.
//
// StartDate is the date of the beginning of the simulation.
// Format = "YYYYMMDD".
//
// EndDate is the date of the end of the simulation.
// Format = "YYYYMMDD".
//
// CTMType specifies what type of chemical transport
// model we are going to be reading data from. Valid
// options are "GEOS-Chem" and "WRF-Chem".
//
// WRFOut is the location of WRF-Chem output files.
// [DATE] should be used as a wild card for the simulation date.
//
// GEOSA1 is the location of the GEOS 1-hour time average files.
// [DATE] should be used as a wild card for the simulation date.
//
// GEOSA3Cld is the location of the GEOS 3-hour average cloud
// parameter files. [DATE] should be used as a wild card for
// the simulation date.
//
// GEOSA3Dyn is the location of the GEOS 3-hour average dynamical
// parameter files. [DATE] should be used as a wild card for
// the simulation date.
//
// GEOSI3 is the location of the GEOS 3-hour instantaneous parameter
// files. [DATE] should be used as a wild card for
// the simulation date.
//
// GEOSA3MstE is the location of the GEOS 3-hour average moist parameters
// on level edges files. [DATE] should be used as a wild card for
// the simulation date.
//
// GEOSApBp is the location of the pressure level variable file.
// It is optional; if it is not specified the Ap and Bp information
// will be extracted from the GEOSChem files.
//
// GEOSChem is the location of GEOS-Chem output files.
// [DATE] should be used as a wild card for the simulation date.
//
// OlsonLandMap is the location of the GEOS-Chem Olson land use map file,
// which is described here:
// http://wiki.seas.harvard.edu/geos-chem/index.php/Olson_land_map
//
// InMAPData is the path where the preprocessed baseline meteorology and pollutant
// data should be written.
//
// CtmGridXo is the lower left of Chemical Transport Model (CTM) grid [x].
//
// CtmGridYo is the lower left of grid [y]
//
// CtmGridDx is the grid cell size in the x direction [m].
//
// CtmGridDy is the grid cell size in the y direction [m].
//
// dash indicates whether GEOS-Chem variable names are in the form 'IJ-AVG-S__xxx'
// as opposed to 'IJ_AVG_S_xxx'.
func Preproc(StartDate, EndDate, CTMType, WRFOut, GEOSA1, GEOSA3Cld, GEOSA3Dyn, GEOSI3, GEOSA3MstE, GEOSApBp,
	GEOSChem, OlsonLandMap, InMAPData string, CtmGridXo, CtmGridYo, CtmGridDx, CtmGridDy float64, dash bool, recordDeltaStr, fileDeltaStr string, noChemHour bool) error {
	msgChan := make(chan string)
	go func() {
		for {
			log.Println(<-msgChan)
		}
	}()
	var ctm inmap.Preprocessor
	switch CTMType {
	case "GEOS-Chem":
		vars := []string{StartDate, EndDate, CTMType, GEOSA1, GEOSA3Cld, GEOSA3Dyn, GEOSI3, GEOSA3MstE, GEOSChem, OlsonLandMap, recordDeltaStr, fileDeltaStr}
		varNames := []string{"StartDate", "EndDate", "CTMType", "GEOSA1", "GEOSA3Cld", "GEOSA3Dyn", "GEOSI3", "GEOSA3MstE", "GEOSChem", "OlsonLandMap", "recordDeltaStr", "fileDeltaStr"}
		for i, v := range vars {
			if v == "" {
				return fmt.Errorf("inmap preprocessor: configuration variable %s is not specified", varNames[i])
			}
		}
		var err error
		ctm, err = inmap.NewGEOSChem(
			GEOSA1,
			GEOSA3Cld,
			GEOSA3Dyn,
			GEOSI3,
			GEOSA3MstE,
			GEOSApBp,
			GEOSChem,
			OlsonLandMap,
			StartDate,
			EndDate,
			dash,
			recordDeltaStr,
			fileDeltaStr,
			noChemHour,
			msgChan,
		)
		if err != nil {
			return err
		}
	case "WRF-Chem":
		vars := []string{StartDate, EndDate, CTMType, WRFOut}
		varNames := []string{"StartDate", "EndDate", "CTMType", "WRFOut"}
		for i, v := range vars {
			if v == "" {
				return fmt.Errorf("inmap preprocessor: configuration variable %s is not specified", varNames[i])
			}
		}
		var err error
		ctm, err = inmap.NewWRFChem(WRFOut, StartDate, EndDate, msgChan)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("inmap preprocessor: the CTMType you specified, '%s', is invalid. Valid options are WRF-Chem and GEOS-Chem", CTMType)
	}
	ctmData, err := inmap.Preprocess(ctm, CtmGridXo, CtmGridYo, CtmGridDx, CtmGridDy)
	if err != nil {
		return err
	}

	// Write out the result.
	ff, err := os.Create(InMAPData)
	if err != nil {
		return fmt.Errorf("inmap: preprocessor writing output file: %v", err)
	}
	if err := ctmData.Write(ff); err != nil {
		return fmt.Errorf("inmap: preprocessor writing output file: %v", err)
	}
	if err := ff.Close(); err != nil {
		return fmt.Errorf("inmap: preprocessor closing output file: %v", err)
	}

	return nil
}
