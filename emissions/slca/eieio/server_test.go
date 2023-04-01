/*
Copyright © 2018 the InMAP authors.
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

package eieio

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"
	"github.com/spatialmodel/inmap/epi"
)

func TestServer_grpc(t *testing.T) {
	f, err := os.Open("data/test_config.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var c ServerConfig
	_, err = toml.DecodeReader(f, &c)
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewServer(&c, "", epi.NasariACS)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ts := httptest.NewTLSServer(s)
	defer ts.Close()

	t.Run("index", func(t *testing.T) {
		client := ts.Client()

		res, err := client.Get(ts.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("Response code was %v; want 200", res.StatusCode)
		}

		expected := []byte("<!DOCTYPE html>")
		body := make([]byte, len(expected))
		_, err = res.Body.Read(body)
		if err != nil {
			t.Fatal(err)
		}

		if bytes.Compare(expected, body) != 0 {
			t.Errorf("Response body was '%s'; want '%s'", expected, body)
		}
	})

	tests := []struct {
		name      string
		f         func(ctx context.Context, in *eieiorpc.Selection) (*eieiorpc.Selectors, error)
		selection *eieiorpc.Selection
		selectors *eieiorpc.Selectors
	}{
		{
			name: "EndUseGroups",
			f:    s.EndUseGroups,
			selection: &eieiorpc.Selection{
				EndUseGroup:     All,
				EndUseSector:    All,
				EmitterGroup:    All,
				EmitterSector:   All,
				ImpactType:      "health",
				Population:      "TotalPop",
				FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
				Year:            2011,
				Pol:             &eieiorpc.Selection_Pollutant{eieiorpc.Pollutant_TotalPM25},
				AQM:             "isrm",
			},
			selectors: &eieiorpc.Selectors{
				Names:  []string{"All", "Food", "Goods", "Transportation", "Services", "Shelter", "Information and Entertainment", "Electricity"},
				Values: []float32{0.14021207, 0.038151223, 0.036187083, 0.024247574, 0.019742511, 0.01743482, 0.003491058, 0.0009578001},
			},
		},
		{
			name: "EndUseSectors_1",
			f:    s.EndUseSectors,
			selection: &eieiorpc.Selection{
				EndUseGroup:     All,
				EndUseSector:    All,
				EmitterGroup:    All,
				EmitterSector:   All,
				ImpactType:      "conc",
				Population:      "TotalPop",
				FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
				Year:            2011,
				Pol:             &eieiorpc.Selection_Pollutant{eieiorpc.Pollutant_TotalPM25},
				AQM:             "isrm",
			},
			selectors: &eieiorpc.Selectors{
				Names:  []string{"All"},
				Values: []float32{0.14021207},
			},
		},
		{
			name: "EndUseSectors_2",
			f:    s.EndUseSectors,
			selection: &eieiorpc.Selection{
				EndUseGroup:     "Electricity",
				EndUseSector:    All,
				EmitterGroup:    All,
				EmitterSector:   All,
				ImpactType:      "conc",
				Population:      "TotalPop",
				FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
				Year:            2011,
				Pol:             &eieiorpc.Selection_Pollutant{eieiorpc.Pollutant_TotalPM25},
				AQM:             "isrm",
			},
			selectors: &eieiorpc.Selectors{
				Names:  []string{"All", "Electric power generation, transmission, and distribution"},
				Values: []float32{0.0009578001, 0.0009578001},
			},
		},
		{
			name: "EmitterGroups",
			f:    s.EmitterGroups,
			selection: &eieiorpc.Selection{
				EndUseGroup:     All,
				EndUseSector:    All,
				EmitterGroup:    All,
				EmitterSector:   All,
				ImpactType:      "emis",
				FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
				Year:            2011,
				Pol:             &eieiorpc.Selection_Emission{eieiorpc.Emission_PM25},
				AQM:             "isrm",
			},
			selectors: &eieiorpc.Selectors{
				Names: []string{"All", "Industrial Fuel Comb.", "Other Industrial Processes",
					"Highway Veh., Light Duty, Gas", "Metals Processing", "Industrial Solvents", "Petroleum Prod. Production",
					"Mining & Mineral Processing", "Ag. Crops", "Ag. Livestock", "Fertilizer Application",
					"Elec. Util., Coal", "Food Prod. & Comm. Cooking", "Construction", "Highway Veh., Heavy Duty, Diesel"},
				Values: []float32{4.9888858e+08, 1.8044906e+08, 1.8044906e+08, 1.3799046e+08, 4.2458604e+07, 4.2458604e+07,
					3.980494e+07, 3.1843952e+07, 1.0614651e+07, 1.0614651e+07, 5.3073255e+06, 5.3073255e+06, 2.6536628e+06, 0, 0},
			},
		},
		{
			name: "EmitterSectors_1",
			f:    s.EmitterSectors,
			selection: &eieiorpc.Selection{
				EndUseGroup:     All,
				EndUseSector:    All,
				EmitterGroup:    All,
				EmitterSector:   All,
				ImpactType:      "emis",
				FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
				Year:            2011,
				Pol:             &eieiorpc.Selection_Emission{eieiorpc.Emission_PM25},
				AQM:             "isrm",
			},
			selectors: &eieiorpc.Selectors{
				Codes:  []string{"All"},
				Names:  []string{"All"},
				Values: []float32{4.9888858e+08},
			},
		},
		{
			name: "EmitterSectors_2",
			f:    s.EmitterSectors,
			selection: &eieiorpc.Selection{
				EndUseGroup:     All,
				EndUseSector:    All,
				EmitterGroup:    "Mining & Mineral Processing",
				EmitterSector:   All,
				ImpactType:      "emis",
				FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
				Year:            2011,
				Pol:             &eieiorpc.Selection_Emission{eieiorpc.Emission_PM25},
				AQM:             "isrm",
			},
			selectors: &eieiorpc.Selectors{
				Codes: []string{"All", "0030500310", "0030500505", "0030500622", "0030501062", "0030501207", "0030501403", "0030502006",
					"0030502601", "0030503704", "0030504034", "0030504099", "0030504142"},
				Names: []string{"All", "Industrial Processes;Mineral Products;Brick Manufacture;Curing and Firing: Sawdust Fired Tunnel Kilns",
					"Industrial Processes;Mineral Products;Castable Refractory;Molding and Shakeout",
					"Industrial Processes;Mineral Products;Cement Manufacturing (Dry Process);Preheater Kiln",
					"Industrial Processes;Mineral Products;Coal Mining, Cleaning, and Material Handling;Surface Mining Operations: Screening",
					"Industrial Processes;Mineral Products;Fiberglass Manufacturing;Unit Melter Furnace (Wool-type Fiber)",
					"Industrial Processes;Mineral Products;Glass Manufacture;Flat Glass: Melting Furnace",
					"Industrial Processes;Mineral Products;Stone Quarrying - Processing (See also 305320);Miscellaneous Operations: Screen/Convey/Handling",
					"Industrial Processes;Mineral Products;Diatomaceous Earth;Handling", "Industrial Processes;Mineral Products;Coated Abrasives Manufacturing;Drying",
					"Industrial Processes;Mineral Products;Mining and Quarrying of Nonmetallic Minerals;Screening",
					"Industrial Processes;Mineral Products;Mining and Quarrying of Nonmetallic Minerals;Other Not Classified",
					"Industrial Processes;Mineral Products;Clay processing: Kaolin;Calcining, flash calciner"},
				Values: []float32{3.1843952e+07, 2.6536628e+06, 2.6536628e+06, 2.6536628e+06, 2.6536628e+06,
					2.6536628e+06, 2.6536628e+06, 2.6536628e+06, 2.6536628e+06, 2.6536628e+06, 2.6536628e+06, 2.6536628e+06,
					2.6536628e+06},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r, err := test.f(context.Background(), test.selection)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(r, test.selectors) {
				t.Errorf("%#v != %#v", r, test.selectors)
			}
		})
	}

	t.Run("Geometry", func(t *testing.T) {
		r, err := s.getGeometry(context.Background(), &eieiorpc.GeometryInput{
			SpatialReference: "+proj=longlat",
			AQM:              "isrm",
		})
		if err != nil {
			t.Fatal(err)
		}
		want := []*eieiorpc.Rectangle{
			&eieiorpc.Rectangle{
				LL: &eieiorpc.Point{X: -97.04719, Y: 39.963825},
				LR: &eieiorpc.Point{X: -97.03539, Y: 39.96383},
				UL: &eieiorpc.Point{X: -97.047195, Y: 39.972866},
				UR: &eieiorpc.Point{X: -97.03539, Y: 39.97287},
			},
			&eieiorpc.Rectangle{
				LL: &eieiorpc.Point{X: -97.047195, Y: 39.972866},
				LR: &eieiorpc.Point{X: -97.03539, Y: 39.97287},
				UL: &eieiorpc.Point{X: -97.0472, Y: 39.981907},
				UR: &eieiorpc.Point{X: -97.0354, Y: 39.98191},
			},
			&eieiorpc.Rectangle{
				LL: &eieiorpc.Point{X: -97.0472, Y: 39.981907},
				LR: &eieiorpc.Point{X: -97.0236, Y: 39.981915},
				UL: &eieiorpc.Point{X: -97.04721, Y: 39.999992},
				UR: &eieiorpc.Point{X: -97.023605, Y: 39.999996},
			},
			&eieiorpc.Rectangle{
				LL: &eieiorpc.Point{X: -97.03539, Y: 39.96383},
				LR: &eieiorpc.Point{X: -97.02359, Y: 39.96383},
				UL: &eieiorpc.Point{X: -97.03539, Y: 39.97287},
				UR: &eieiorpc.Point{X: -97.0236, Y: 39.972874},
			},
			&eieiorpc.Rectangle{
				LL: &eieiorpc.Point{X: -97.03539, Y: 39.97287},
				LR: &eieiorpc.Point{X: -97.0236, Y: 39.972874},
				UL: &eieiorpc.Point{X: -97.0354, Y: 39.98191},
				UR: &eieiorpc.Point{X: -97.0236, Y: 39.981915},
			},
			&eieiorpc.Rectangle{
				LL: &eieiorpc.Point{X: -97.04721, Y: 39.999992},
				LR: &eieiorpc.Point{X: -97, Y: 40},
				UL: &eieiorpc.Point{X: -97.04723, Y: 40.036156},
				UR: &eieiorpc.Point{X: -97, Y: 40.036167},
			},
			&eieiorpc.Rectangle{
				LL: &eieiorpc.Point{X: -97.02359, Y: 39.96383},
				LR: &eieiorpc.Point{X: -97, Y: 39.963833},
				UL: &eieiorpc.Point{X: -97.0236, Y: 39.981915},
				UR: &eieiorpc.Point{X: -97, Y: 39.98192},
			},
			&eieiorpc.Rectangle{
				LL: &eieiorpc.Point{X: -97.0236, Y: 39.981915},
				LR: &eieiorpc.Point{X: -97, Y: 39.98192},
				UL: &eieiorpc.Point{X: -97.023605, Y: 39.999996},
				UR: &eieiorpc.Point{X: -97, Y: 40},
			},
			&eieiorpc.Rectangle{
				LL: &eieiorpc.Point{X: -97, Y: 39.963833},
				LR: &eieiorpc.Point{X: -96.95281, Y: 39.963825},
				UL: &eieiorpc.Point{X: -97, Y: 40},
				UR: &eieiorpc.Point{X: -96.95279, Y: 39.999992},
			},
			&eieiorpc.Rectangle{
				LL: &eieiorpc.Point{X: -97, Y: 40},
				LR: &eieiorpc.Point{X: -96.95279, Y: 39.999992},
				UL: &eieiorpc.Point{X: -97, Y: 40.036167},
				UR: &eieiorpc.Point{X: -96.95277, Y: 40.036156},
			},
		}

		if !reflect.DeepEqual(r, want) {
			t.Fatal("rectangles not equal")
		}
	})

	t.Run("MapInfo", func(t *testing.T) {
		for _, test := range []struct {
			impactType string
			pollutant  eieiorpc.Pollutant
			colors     [][]uint8
		}{
			{
				impactType: "health",
				pollutant:  eieiorpc.Pollutant_TotalPM25,
				colors: [][]uint8{[]uint8{0xff, 0xff, 0xff}, []uint8{0x0, 0x0, 0x0}, []uint8{0x0, 0x0, 0x0}, []uint8{0x0, 0x0, 0x0},
					[]uint8{0x0, 0x0, 0x0}, []uint8{0x0, 0x0, 0x0}, []uint8{0x0, 0x0, 0x0}, []uint8{0x0, 0x0, 0x0}, []uint8{0x0, 0x0, 0x0},
					[]uint8{0x0, 0x0, 0x0}},
			},
			{
				impactType: "conc",
				pollutant:  eieiorpc.Pollutant_TotalPM25,
				colors: [][]uint8{[]uint8{0x15, 0xd, 0x2d}, []uint8{0x8f, 0x0, 0xc3}, []uint8{0xfd, 0x8f, 0x33},
					[]uint8{0x3e, 0x12, 0xc6}, []uint8{0xf5, 0x5e, 0x3a}, []uint8{0x4, 0x17, 0xa9}, []uint8{0xee, 0x4f, 0x3b},
					[]uint8{0xff, 0xff, 0xff}, []uint8{0x8, 0x17, 0x9f}, []uint8{0x0, 0x0, 0x0}},
			},
			{
				impactType: "emis",
				pollutant:  eieiorpc.Pollutant_TotalPM25,
				colors: [][]uint8{[]uint8{0x0, 0x0, 0x0}, []uint8{0x0, 0x0, 0x0}, []uint8{0xff, 0xff, 0xff}, []uint8{0x0, 0x0, 0x0},
					[]uint8{0x0, 0x0, 0x0}, []uint8{0x27, 0x16, 0xb7}, []uint8{0x0, 0x0, 0x0}, []uint8{0xff, 0xff, 0xff},
					[]uint8{0x27, 0x16, 0xb7}, []uint8{0x27, 0x16, 0xb7}},
			},
		} {
			t.Run(test.impactType, func(t *testing.T) {
				mapInfo, err := s.MapInfo(context.Background(), &eieiorpc.Selection{
					EndUseGroup:     All,
					EndUseSector:    All,
					EmitterGroup:    All,
					EmitterSector:   All,
					ImpactType:      test.impactType,
					FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
					Year:            2011,
					Pol:             &eieiorpc.Selection_Pollutant{eieiorpc.Pollutant_TotalPM25},
					Population:      "TotalPop",
					AQM:             "isrm",
				})
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(mapInfo.RGB, test.colors) {
					t.Errorf("%#v != %v", mapInfo.RGB, test.colors)
				}
			})
		}
	})

	t.Run("DefaultSelection", func(t *testing.T) {
		ds, err := s.DefaultSelection(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		want := &eieiorpc.Selection{
			EndUseGroup:     "All",
			EndUseSector:    "All",
			EmitterGroup:    "All",
			EmitterSector:   "All",
			ImpactType:      "conc",
			FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
			Year:            2011,
			Population:      "TotalPop",
			Pol:             &eieiorpc.Selection_Pollutant{eieiorpc.Pollutant_TotalPM25},
			AQM:             "isrm",
		}
		if !reflect.DeepEqual(ds, want) {
			t.Errorf("%+v != %+v", ds, want)
		}
	})

	t.Run("Populations", func(t *testing.T) {
		p, err := s.Populations(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		want := &eieiorpc.Selectors{
			Names: []string{"TotalPop", "WhiteNoLat", "Black", "Native", "Asian", "Latino"},
		}
		if !reflect.DeepEqual(p, want) {
			t.Errorf("%+v != %+v", p, want)
		}
	})

	t.Run("Years", func(t *testing.T) {
		y, err := s.Years(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		want := &eieiorpc.Year{
			Years: []int32{2007, 2011, 2015},
		}
		if !reflect.DeepEqual(y, want) {
			t.Errorf("%+v != %+v", y, want)
		}
	})
}
