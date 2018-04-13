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

package slca

import (
	"context"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/gonum/floats"
	"github.com/spatialmodel/epi"
)

func TestCSTConfig_EmissionsSurrogate(t *testing.T) {
	f, err := os.Open("testdata/test_config.toml")
	if err != nil {
		t.Fatal(err)
	}
	c := new(CSTConfig)
	if _, err = toml.DecodeReader(f, c); err != nil {
		t.Fatal(err)
	}
	if err = c.Setup(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		ref  *SpatialRef
		sum  float64
	}{
		{
			name: "normalized",
			ref: &SpatialRef{
				SCCs: []SCC{"2280003010"},
				Type: Stationary,
			},
			sum: 1,
		},
		{
			name: "non-normalized",
			ref: &SpatialRef{
				SCCs:            []SCC{"2280003010"},
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 2.6536626364130992e+06,
		},
		{
			name: "non-normalized fraction",
			ref: &SpatialRef{
				SCCs:            []SCC{"2280003010"},
				SCCFractions:    []float64{0.5},
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 2.6536626364130992e+06 / 2,
		},
		{
			name: "Different year",
			ref: &SpatialRef{
				SCCs:            []SCC{"2280003010"},
				EmisYear:        2011,
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 2.6536626364130992e+06,
		},
	}

	var failed bool
	for _, test := range tests {
		if failed {
			continue
		}
		t.Run(test.name, func(t *testing.T) {
			result, err := c.EmissionsSurrogate(context.Background(), PM25, test.ref)
			if err != nil {
				failed = true
				t.Fatal(err)
			}
			sum := result.Sum()
			if sum != test.sum {
				t.Errorf("want %g, have %g", test.sum, sum)
			}
		})
	}
}

func TestCSTConfig_EmissionsSurrogate_adjusted(t *testing.T) {
	f, err := os.Open("testdata/test_config.toml")
	if err != nil {
		t.Fatal(err)
	}
	c := new(CSTConfig)
	if _, err = toml.DecodeReader(f, c); err != nil {
		t.Fatal(err)
	}

	// Change the name of the sector to the one that matches the FugitiveDustSector
	// field.
	c.InventoryConfig.NEIFiles["test_adj"] = c.InventoryConfig.NEIFiles["test"]
	delete(c.InventoryConfig.NEIFiles, "test")

	if err = c.Setup(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		ref  *SpatialRef
		sum  float64
	}{
		{
			name: "normalized",
			ref: &SpatialRef{
				SCCs: []SCC{"2280003010"},
				Type: Stationary,
			},
			sum: 1,
		},
		{
			name: "non-normalized",
			ref: &SpatialRef{
				SCCs:            []SCC{"2280003010"},
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 2.6536626364130992e+06 / 2,
		},
		{
			name: "non-normalized fraction",
			ref: &SpatialRef{
				SCCs:            []SCC{"2280003010"},
				SCCFractions:    []float64{0.5},
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 2.6536626364130992e+06 / 2 / 2,
		},
		{
			name: "Different year",
			ref: &SpatialRef{
				SCCs:            []SCC{"2280003010"},
				EmisYear:        2011,
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 1.3268313182065496e+06,
		},
	}

	var failed bool
	for _, test := range tests {
		if failed {
			continue
		}
		t.Run(test.name, func(t *testing.T) {
			result, err := c.EmissionsSurrogate(context.Background(), PM25, test.ref)
			if err != nil {
				failed = true
				t.Fatal(err)
			}
			sum := result.Sum()
			if sum != test.sum {
				t.Errorf("want %g, have %g", test.sum, sum)
			}
		})
	}
}

func TestCSTConfig_ConcentrationSurrogate(t *testing.T) {
	f, err := os.Open("testdata/test_config.toml")
	if err != nil {
		t.Fatal(err)
	}
	c := new(CSTConfig)
	if _, err = toml.DecodeReader(f, c); err != nil {
		t.Fatal(err)
	}
	if err = c.Setup(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		ref  *SpatialRef
		sum  float64
	}{
		{
			name: "normalized",
			ref: &SpatialRef{
				SCCs: []SCC{"2280003010"},
				Type: Stationary,
			},
			sum: 1.5468779703503674e-09,
		},
		{
			name: "non-normalized",
			ref: &SpatialRef{
				SCCs:            []SCC{"2280003010"},
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 0.003240866726950201,
		},
		{
			name: "non-normalized fraction",
			ref: &SpatialRef{
				SCCs:            []SCC{"2280003010"},
				SCCFractions:    []float64{0.5},
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 0.003240866726950201 / 2,
		},
		{
			name: "Different year",
			ref: &SpatialRef{
				SCCs:            []SCC{"2280003010"},
				EmisYear:        2011,
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 0.003240866726950201,
		},
	}

	var failed bool
	for _, test := range tests {
		if failed {
			continue
		}
		t.Run(test.name, func(t *testing.T) {
			result, err := c.ConcentrationSurrogate(context.Background(), test.ref)
			if err != nil {
				failed = true
				t.Fatal(err)
			}
			sum := floats.Sum(result.TotalPM25())
			if sum != test.sum {
				t.Errorf("want %g, have %g", test.sum, sum)
			}
		})
	}
}

func TestCSTConfig_EvaluationConcentrations(t *testing.T) {
	f, err := os.Open("testdata/test_config.toml")
	if err != nil {
		t.Fatal(err)
	}
	c := new(CSTConfig)
	if _, err = toml.DecodeReader(f, c); err != nil {
		t.Fatal(err)
	}
	if err = c.Setup(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		year int
		sum  float64
	}{
		{
			name: "2014",
			year: 2014,
			sum:  0.003240866726950201,
		},
		{
			name: "2011",
			year: 2011,
			sum:  0.003240866726950201,
		},
	}

	var failed bool
	for _, test := range tests {
		if failed {
			continue
		}
		t.Run(test.name, func(t *testing.T) {
			result, err := c.EvaluationConcentrations(context.Background(), test.year)
			if err != nil {
				failed = true
				t.Fatal(err)
			}
			sum := floats.Sum(result.TotalPM25())
			if sum != test.sum {
				t.Errorf("want %g, have %g", test.sum, sum)
			}
		})
	}
}

func TestCSTConfig_EvaluationHealth(t *testing.T) {
	f, err := os.Open("testdata/test_config.toml")
	if err != nil {
		t.Fatal(err)
	}
	c := new(CSTConfig)
	if _, err = toml.DecodeReader(f, c); err != nil {
		t.Fatal(err)
	}
	if err = c.Setup(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		year int
		sum  float64
	}{
		{
			name: "2014",
			year: 2014,
			sum:  0.0007516069350078184,
		},
		{
			name: "2011",
			year: 2011,
			sum:  0.0007516069350078184,
		},
	}

	var failed bool
	for _, test := range tests {
		if failed {
			continue
		}
		t.Run(test.name, func(t *testing.T) {
			result, err := c.EvaluationHealth(context.Background(), test.year, epi.NasariACS)
			if err != nil {
				failed = true
				t.Fatal(err)
			}
			sum := result["TotalPop"][totalPM25].Sum()
			if sum != test.sum {
				t.Errorf("want %g, have %g", test.sum, sum)
			}
		})
	}
}

func TestCSTConfig_HealthSurrogate(t *testing.T) {
	f, err := os.Open("testdata/test_config.toml")
	if err != nil {
		t.Fatal(err)
	}
	c := new(CSTConfig)
	if _, err = toml.DecodeReader(f, c); err != nil {
		t.Fatal(err)
	}
	if err = c.Setup(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		ref  *SpatialRef
		sum  float64
	}{
		{
			name: "normalized",
			ref: &SpatialRef{
				EmisYear: 2014,
				SCCs:     []SCC{"2280003010"},
				Type:     Stationary,
			},
			sum: 3.617337147934318e-10,
		},
		{
			name: "non-normalized",
			ref: &SpatialRef{
				EmisYear:        2014,
				SCCs:            []SCC{"2280003010"},
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 0.0007516069350078184,
		},
		{
			name: "non-normalized fraction",
			ref: &SpatialRef{
				EmisYear:        2014,
				SCCs:            []SCC{"2280003010"},
				SCCFractions:    []float64{0.5},
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 0.0003758034675039092,
		},
		{
			name: "Different year",
			ref: &SpatialRef{
				SCCs:            []SCC{"2280003010"},
				EmisYear:        2011,
				Type:            Stationary,
				NoNormalization: true,
			},
			sum: 0.0007516069350078184,
		},
	}

	var failed bool
	for _, test := range tests {
		if failed {
			continue
		}
		t.Run(test.name, func(t *testing.T) {
			result, err := c.HealthSurrogate(context.Background(), test.ref, epi.NasariACS)
			if err != nil {
				failed = true
				t.Fatal(err)
			}
			sum := result["TotalPop"][totalPM25].Sum()
			if sum != test.sum {
				t.Errorf("want %g, have %g", test.sum, sum)
			}
		})
	}
}
