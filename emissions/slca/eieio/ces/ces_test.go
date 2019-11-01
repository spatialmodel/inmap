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
	"github.com/spatialmodel/inmap/emissions/slca/eieio"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/ces"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"
	"github.com/spatialmodel/inmap/epi"
)

// Set up directory location
func init() {
	os.Setenv("INMAP_ROOT_DIR", "../../../../")
}

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

	s, err := eieio.NewServer(&cfg, "", epi.NasariACS)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	c, err := ces.NewCES(s, ".")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("demand", func(t *testing.T) {
		d, err := c.DemographicConsumption(context.Background(), &eieiorpc.DemographicConsumptionInput{
			Demograph: eieiorpc.Demograph_WhiteOther,
			Year:      2014,
		})
		if err != nil {
			t.Fatal(err)
		}
		haveWhiteOther := floats.Sum(d.Data)
		want := 1.2266264876604041e+13
		if haveWhiteOther != want {
			t.Errorf("white/other = %g; want %g", haveWhiteOther, want)
		}

		d, err = c.DemographicConsumption(context.Background(), &eieiorpc.DemographicConsumptionInput{
			Demograph: eieiorpc.Demograph_Black,
			Year:      2014,
		})
		if err != nil {
			t.Fatal(err)
		}
		haveBlack := floats.Sum(d.Data)
		want = 1.2810049534676526e+12
		if haveBlack != want {
			t.Errorf("black = %g; want %g", haveBlack, want)
		}

		d, err = c.DemographicConsumption(context.Background(), &eieiorpc.DemographicConsumptionInput{
			Demograph: eieiorpc.Demograph_Hispanic,
			Year:      2014,
		})
		if err != nil {
			t.Fatal(err)
		}
		haveHispanic := floats.Sum(d.Data)
		want = 1.6193962594997866e+12
		if haveHispanic != want {
			t.Errorf("hispanic = %g; want %g", haveHispanic, want)
		}

		d, err = c.DemographicConsumption(context.Background(), &eieiorpc.DemographicConsumptionInput{
			Demograph: eieiorpc.Demograph_All,
			Year:      2014,
		})
		if err != nil {
			t.Fatal(err)
		}
		haveAll := floats.Sum(d.Data)
		allSum := haveBlack + haveHispanic + haveWhiteOther

		if !floats.EqualWithinAbsOrRel(haveAll, allSum, 1e-10, 1e-10) {
			t.Errorf("total demographic: %g != %g", haveAll, allSum)
		}

		t.Run("dt", func(t *testing.T) {
			var overallTotal float64
			for _, dt := range []eieiorpc.FinalDemandType{
				eieiorpc.FinalDemandType_PersonalConsumption,
				eieiorpc.FinalDemandType_PrivateResidential,
				eieiorpc.FinalDemandType_PrivateStructures,
				eieiorpc.FinalDemandType_PrivateEquipment,
				eieiorpc.FinalDemandType_PrivateIP,
				eieiorpc.FinalDemandType_InventoryChange} {

				d, err := s.FinalDemand(context.TODO(), &eieiorpc.FinalDemandInput{
					FinalDemandType: dt,
					Year:            2014,
					Location:        eieiorpc.Location_Domestic,
				})
				if err != nil {
					t.Fatal(err)
				}
				overallTotal += floats.Sum(d.Data)
			}

			if !floats.EqualWithinAbsOrRel(haveAll, overallTotal, 1e-8, 1e-8) {
				t.Errorf("overall total: %g != %g", haveAll, overallTotal)
			}
		})

		t.Run("mask", func(t *testing.T) {
			ctx := context.Background()

			totalCons, err := c.DemographicConsumption(ctx, &eieiorpc.DemographicConsumptionInput{
				Year:      2014,
				Demograph: eieiorpc.Demograph_Black,
			})
			if err != nil {
				t.Fatal(err)
			}
			totalDeaths, err := s.Health(ctx, &eieiorpc.HealthInput{
				Demand:     totalCons,
				Pollutant:  eieiorpc.Pollutant_TotalPM25,
				Population: "TotalPop",
				Year:       2014,
				Location:   eieiorpc.Location_Domestic,
				HR:         "NasariACS",
				AQM:        "isrm",
			})
			if err != nil {
				t.Fatal(err)
			}

			ioAbbrevs, err := s.EndUseGroupAbbrevs(ctx, &eieiorpc.StringInput{})
			if err != nil {
				t.Fatal(err)
			}
			var sum []float64
			var deathSum float64
			for j, useAbbrev := range ioAbbrevs.List {
				useMask, err := s.EndUseMask(ctx, &eieiorpc.StringInput{String_: useAbbrev})
				if err != nil {
					t.Fatal(err)
				}
				cons, err := c.DemographicConsumption(ctx, &eieiorpc.DemographicConsumptionInput{
					Year:       2014,
					EndUseMask: useMask,
					Demograph:  eieiorpc.Demograph_Black,
				})
				if err != nil {
					t.Fatal(err)
				}
				if j == 0 {
					sum = make([]float64, len(cons.Data))
				}
				floats.Add(sum, cons.Data)

				deaths, err := s.Health(ctx, &eieiorpc.HealthInput{
					Demand:     cons,
					Pollutant:  eieiorpc.Pollutant_TotalPM25,
					Population: "TotalPop",
					Year:       2014,
					Location:   eieiorpc.Location_Domestic,
					HR:         "NasariACS",
					AQM:        "isrm",
				})
				if err != nil {
					t.Fatal(err)
				}
				deathSum += floats.Sum(deaths.Data)
			}
			if len(totalCons.Data) != len(sum) {
				t.Fatalf("consumption length %d != %d", len(totalCons.Data), len(sum))
			}
			for i, v := range totalCons.Data {
				if !floats.EqualWithinAbsOrRel(sum[i], v, 1e-10, 1e-10) {
					t.Errorf("consumption %d: %g != %g", i, sum[i], v)
				}
			}
			if !floats.EqualWithinAbsOrRel(floats.Sum(totalDeaths.Data), deathSum, 1e-10, 1e-10) {
				t.Errorf("deaths: %g != %g", floats.Sum(totalDeaths.Data), deathSum)
			}
		})

		t.Run("years", func(t *testing.T) {
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
	})
}
