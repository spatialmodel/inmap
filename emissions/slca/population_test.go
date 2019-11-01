package slca

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"
	"github.com/spatialmodel/inmap/epi"
)

// Set up directory location for configuration files.
func init() {
	os.Setenv("INMAP_ROOT_DIR", "../../")
}

func TestPopulationIncidence(t *testing.T) {
	f, err := os.Open("testdata/test_config.toml")
	if err != nil {
		t.Fatal(err)
	}
	c := new(CSTConfig)
	if _, err = toml.DecodeReader(f, c); err != nil {
		t.Fatal(err)
	}
	if err = c.Setup(epi.NasariACS); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		year int32
		pop  []float64
		io   []float64
	}{
		{
			year: 2000,
			pop:  []float64{50000, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			io:   []float64{399.9996241961794, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			year: 2001,
			pop:  []float64{50000, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			io:   []float64{399.9996241961794, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			year: 2002,
			pop:  []float64{50000*0.9 + 100000*0.1, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			io:   []float64{439.9995866157974, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // 400*0.9 + 800*0.1
		},
		{
			year: 2012,
			pop:  []float64{100000, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			io:   []float64{799.9992483923589, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			year: 2014,
			pop:  []float64{100000, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			io:   []float64{799.9992483923588, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			year: 2015,
			pop:  []float64{100000, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			io:   []float64{799.9992483923588, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test.year), func(t *testing.T) {
			p, err := c.PopulationIncidence(context.Background(), &eieiorpc.PopulationIncidenceInput{
				Year: test.year, Population: "TotalPop", HR: "NasariACS", AQM: "inmap"})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(test.pop, p.Population) {
				t.Errorf("population: %v != %v", p.Population, test.pop)
			}
			if !reflect.DeepEqual(test.io, p.Incidence) {
				t.Errorf("incidence: %v != %v", p.Incidence, test.io)
			}
		})
	}
}
