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

package ces_test

import (
	"context"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/gonum/floats"
	"github.com/spatialmodel/epi"
	"github.com/spatialmodel/inmap/emissions/slca/eieio"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/ces"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"
)

func TestCES(t *testing.T) {
	f, err := os.Open("../data/test_config.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var cfg eieio.ServerConfig
	_, err = toml.DecodeReader(f, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Config.Years = []eieio.Year{2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015}

	s, err := eieio.NewServer(&cfg, epi.NasariACS)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	/*	cfg := eieio.ServerConfig{
		SpatialConfig: &eieio.SpatialConfig{

			Years:                       []eieio.Year{2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015},
			DetailYear:                  2007,
			UseSummary:                  "../data/IOUse_Before_Redefinitions_PRO_1997-2015_Summary.xlsx",
			UseDetail:                   "../data/IOUse_Before_Redefinitions_PRO_2007_Detail.xlsx",
			ImportsSummary:              "../data/ImportMatrices_Before_Redefinitions_SUM_1997-2016.xlsx",
			ImportsDetail:               "../data/ImportMatrices_Before_Redefinitions_DET_2007.xlsx",
			TotalRequirementsSummary:    "../data/IxC_TR_1997-2015_Summary.xlsx",
			TotalRequirementsDetail:     "../data/IxC_TR_2007_Detail.xlsx",
			DomesticRequirementsSummary: "../data/IxC_Domestic_1997-2015_Summary.xlsx",
			DomesticRequirementsDetail:  "../data/IxC_Domestic_2007_Detail.xlsx",
		},
	}*/

	/*e, err := eieio.NewServer(&cfg)
	if err != nil {
		t.Fatal(err)
	}*/

	c, err := ces.NewCES(s)
	if err != nil {
		t.Fatal(err)
	}
	/*	sector := "Animal (except poultry) slaughtering, rendering, and processing"
		v, err := c.whiteOtherFrac(2007, sector)
		if err != nil {
			t.Error(err)
		}
		want := 0.7631713337219025
		if v != want {
			t.Errorf("want %g, have %g", want, v)
		}
		if _, err := c.whiteOtherFrac(3030, sector); err == nil {
			t.Error("bad year: this should cause an error")
		}

		if _, err := c.whiteOtherFrac(2007, "Animal (except poultry)"); err == nil {
			t.Error("missing sector: this should cause an error")
		}*/

	t.Run("demand", func(t *testing.T) {
		d, err := c.DemographicConsumption(context.Background(), &eieiorpc.DemographicConsumptionInput{
			Demograph: eieiorpc.Demograph_WhiteOther,
			Year:      2014,
		})
		if err != nil {
			t.Fatal(err)
		}
		have := floats.Sum(d.Data)
		want := 1.082388720998684e+13
		if have != want {
			t.Errorf("white/other = %g; want %g", have, want)
		}

		d, err = c.DemographicConsumption(context.Background(), &eieiorpc.DemographicConsumptionInput{
			Demograph: eieiorpc.Demograph_Black,
			Year:      2014,
		})
		if err != nil {
			t.Fatal(err)
		}
		have = floats.Sum(d.Data)
		want = 1.1414981404936084e+12
		if have != want {
			t.Errorf("black = %g; want %g", have, want)
		}

		d, err = c.DemographicConsumption(context.Background(), &eieiorpc.DemographicConsumptionInput{
			Demograph: eieiorpc.Demograph_Hispanic,
			Year:      2014,
		})
		if err != nil {
			t.Fatal(err)
		}
		have = floats.Sum(d.Data)
		want = 1.4404668535628293e+12
		if have != want {
			t.Errorf("latino = %g; want %g", have, want)
		}

		for _, year := range cfg.Config.Years {
			_, err = c.DemographicConsumption(context.Background(), &eieiorpc.DemographicConsumptionInput{
				Demograph: eieiorpc.Demograph_Hispanic,
				Year:      int32(year),
			})
			if err != nil {
				t.Error(err)
			}
		}

	})

	// This should not change the existing file.
	//if err := c.WriteXLSX("processed.xlsx"); err != nil {
	//	t.Fatal(err)
	//}
}
