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

package cmd

import (
	"fmt"
	"os"
	"reflect"

	"github.com/spatialmodel/inmap"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(preprocCmd)
}

var preprocCmd = &cobra.Command{
	Use:   "preproc",
	Short: "Preprocess CTM output",
	Long: `preproc preprocesses chemical transport model
  output as specified by information in the configuration
  file and saves the result for use in future InMAP simulations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return labelErr(Preproc(Config))
	},
}

// Preproc preprocesses chemical transport model
// output as specified by information in cfg
// and saves the result for use in future InMAP simulations.
func Preproc(cfg *ConfigData) error {
	if err := preprocCheckConfig(cfg); err != nil {
		return err
	}
	var ctm inmap.Preprocessor
	switch cfg.Preproc.CTMType {
	case "GEOS-Chem":
		var err error
		ctm, err = inmap.NewGEOSChem(
			cfg.Preproc.GEOSChem.GEOSA1,
			cfg.Preproc.GEOSChem.GEOSA3Cld,
			cfg.Preproc.GEOSChem.GEOSA3Dyn,
			cfg.Preproc.GEOSChem.GEOSI3,
			cfg.Preproc.GEOSChem.GEOSA3MstE,
			cfg.Preproc.GEOSChem.GEOSChem,
			cfg.Preproc.GEOSChem.VegTypeGlobal,
			cfg.Preproc.StartDate,
			cfg.Preproc.EndDate,
		)
		if err != nil {
			return err
		}
	case "WRF-Chem":
		var err error
		ctm, err = inmap.NewWRFChem(cfg.Preproc.WRFChem.WRFOut, cfg.Preproc.StartDate, cfg.Preproc.EndDate)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("inmap preprocessor: the CTMType you specified, '%s', is invalid. Valid options are WRF-Chem and GEOS-Chem", cfg.Preproc.CTMType)
	}
	ctmData, err := inmap.Preprocess(ctm)
	if err != nil {
		return err
	}

	// Write out the result.
	ff, err := os.Create(cfg.InMAPData)
	if err != nil {
		return fmt.Errorf("inmap: preprocessor writing output file: %v", err)
	}
	ctmData.Write(ff, cfg.Preproc.CtmGridXo, cfg.Preproc.CtmGridYo, cfg.Preproc.CtmGridDx, cfg.Preproc.CtmGridDy)
	ff.Close()

	return nil
}

// preprocCheckConfig checks preprocessor-specific configuration
// information.
func preprocCheckConfig(cfg *ConfigData) error {

	cfg.Preproc.StartDate = os.ExpandEnv(cfg.Preproc.StartDate)
	cfg.Preproc.EndDate = os.ExpandEnv(cfg.Preproc.EndDate)

	switch cfg.Preproc.CTMType {
	case "WRF-Chem":
		if err := preprocCheckPaths(&cfg.Preproc.WRFChem); err != nil {
			return err
		}
	case "GEOS-Chem":
		if err := preprocCheckPaths(&cfg.Preproc.GEOSChem); err != nil {
			return err
		}
	default:
		return fmt.Errorf("inmap preprocessor: the CTMType you specified, '%s', is invalid. Valid options are WRF-Chem and GEOS-Chem", cfg.Preproc.CTMType)
	}
	return nil
}

// preprocCheckPaths makes sure that none of the String
// fields in the given variable are empty and expands
// any environment variables that they contain.
// The given variable must be a pointer to a struct.
func preprocCheckPaths(paths interface{}) error {
	v := reflect.ValueOf(paths).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Type().Kind() == reflect.String {
			s := f.String()
			if s == "" {
				name := v.Type().Field(i).Name
				return fmt.Errorf("inmap preprocessor: configuration file field %s is empty", name)
			}
			s = os.ExpandEnv(s)
			f.SetString(s)
		}
	}
	return nil
}
