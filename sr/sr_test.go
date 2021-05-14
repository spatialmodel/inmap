/*
Copyright © 2013 the InMAP authors.
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

package sr_test

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/ctessum/cdf"
	"github.com/gonum/floats"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud"
	"github.com/spatialmodel/inmap/inmaputil"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
	"github.com/spatialmodel/inmap/sr"
)

// Set up directory location for configuration files.
func init() {
	os.Setenv("INMAP_ROOT_DIR", "../")
}

// TestSaveSRGrid checks the ability to save a grid file
// for SR matrix generation tests.
func saveSRGrid(t *testing.T, filename string) {
	cfg, ctmdata, pop, popIndices, mr, mortIndices := inmap.VarGridTestData()
	cfg.HiResLayers = 6
	f, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}

	emis := inmap.NewEmissions()

	var m simplechem.Mechanism
	mutator, err := inmap.PopulationMutator(cfg, popIndices)
	if err != nil {
		t.Error(err)
	}
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			cfg.RegularGrid(ctmdata, pop, popIndices, mr, mortIndices, emis, m),
			cfg.MutateGrid(mutator, ctmdata, pop, mr, emis, m, nil),
			inmap.Save(f),
		},
	}
	if err := d.Init(); err != nil {
		t.Error(err)
	}
}

type config struct {
	InMAPData        string
	VariableGridData string
	VarGrid          inmap.VarGridConfig
}

func loadConfig(file string) (*config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("problem reading configuration file: %v", err)
	}

	cfg := new(config)
	_, err = toml.Decode(string(bytes), cfg)
	if err != nil {
		return nil, fmt.Errorf(
			"there has been an error parsing the configuration file: %v\n", err)
	}

	cfg.InMAPData = os.ExpandEnv(cfg.InMAPData)
	cfg.VariableGridData = os.ExpandEnv(cfg.VariableGridData)
	cfg.VarGrid.CensusFile = os.ExpandEnv(cfg.VarGrid.CensusFile)
	cfg.VarGrid.MortalityRateFile = os.ExpandEnv(cfg.VarGrid.MortalityRateFile)

	return cfg, err
}

func TestSR(t *testing.T) {
	checkConfig := func(cmd []string) {
		var foundOutputVars, foundEmisUnits, foundEmissionsShapefiles bool
		for _, a := range cmd {
			if strings.Contains(a, "EmissionsShapefiles") {
				foundEmissionsShapefiles = true
				if !strings.Contains(a, "file://") {
					t.Errorf("EmissionUnits should be have 'file://' in it but doesn't: %s", a)
				}
			}
			if strings.Contains(a, "EmissionUnits") {
				foundEmisUnits = true
				if !strings.Contains(a, "ug/s") {
					t.Errorf("EmissionUnits should be ug/s but is: %s", a)
				}
			}
			if strings.Contains(a, "OutputVariables") {
				foundOutputVars = true
				if !strings.Contains(a, "{\"SOA\":\"SOA\",\"PrimPM25\":\"PrimaryPM25\",\"pNH4\":\"pNH4\","+
					"\"pSO4\":\"pSO4\",\"pNO3\":\"pNO3\"}") {
					t.Errorf("wrong OutputVariables: %s", a)
				}
			}
		}
		if !foundEmissionsShapefiles {
			t.Error("didn't find emissions shapefiles")
		}
		if !foundEmisUnits {
			t.Error("didn't find emissions units")
		}
		if !foundOutputVars {
			t.Error("didn't find output variables")
		}
	}

	checkRun := func(o []byte, err error) {
		if err != nil {
			t.Error(err)
		}
		for _, l := range strings.Split(string(o), "\n") {
			if strings.Contains(strings.ToLower(l), "error") {
				t.Log(l)
			}
		}
	}

	cfg := inmaputil.InitializeConfig()

	ctx := context.WithValue(context.Background(), "user", "test_user")

	config, err := loadConfig("../cmd/inmap/configExample.toml")
	if err != nil {
		t.Fatal(err)
	}
	varGridFile := strings.TrimSuffix(config.VariableGridData, ".gob") + "_SR.gob"
	saveSRGrid(t, varGridFile)
	defer os.Remove(varGridFile)
	varGridReader, err := os.Open(varGridFile)
	if err != nil {
		t.Fatal(err)
	}
	os.Mkdir("test", os.ModePerm)
	client, err := cloud.NewFakeClient(checkConfig, checkRun, "file://test", cfg.Root, cfg.Viper, cfg.InputFiles(), cfg.OutputFiles())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("test")
	s, err := sr.NewSR(varGridReader, &config.VarGrid, cloud.FakeRPCClient{Client: client})
	if err != nil {
		t.Fatal(err)
	}
	outfile := "../cmd/inmap/testdata/testSR.ncf"
	defer os.Remove(outfile)
	layers := []int{0, 2, 4}
	begin := 0 // layer 0
	end := -1
	if err = s.Start(ctx, "sr_test2", "latest", layers, begin, end, cfg.Root, cfg.Viper, []string{"run", "steady"}, cfg.InputFiles(), 2); err != nil {
		t.Fatal(err)
	}
	if err = s.Save(ctx, outfile, "sr_test2", layers, begin, end); err != nil {
		t.Fatal(err)
	}

	// Run it again for different indices.
	varGridReader, err = os.Open(strings.TrimSuffix(config.VariableGridData, ".gob") + "_SR.gob")
	if err != nil {
		t.Fatal(err)
	}
	s, err = sr.NewSR(varGridReader, &config.VarGrid, cloud.FakeRPCClient{Client: client})
	if err != nil {
		t.Fatal(err)
	}
	begin = 20 // layer 2
	end = 22
	if err = s.Start(ctx, "sr_test", "latest", layers, begin, end, cfg.Root, cfg.Viper, []string{"run", "steady"}, cfg.InputFiles(), 2); err != nil {
		t.Fatal(err)
	}
	if err = s.Save(ctx, outfile, "sr_test", layers, begin, end); err != nil {
		t.Fatal(err)
	}
	t.Run("compare ncf", func(t *testing.T) {
		ncfWithinTol(t, "../cmd/inmap/testdata/testSR.ncf", "../cmd/inmap/testdata/testSR_golden.ncf", 1.e-9)
	})
}

// ncfWithinTol creates errors if the new and old files are more different
// than the given floating-point tolerance.
func ncfWithinTol(t *testing.T, newFile, oldFile string, tol float64) {
	newR, err := os.Open(newFile)
	if err != nil {
		t.Fatal(err)
	}
	oldR, err := os.Open(oldFile)
	if err != nil {
		t.Fatal(err)
	}
	new, err := cdf.Open(newR)
	if err != nil {
		t.Fatal(err)
	}
	old, err := cdf.Open(oldR)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("global attr", func(t *testing.T) {
		newGlobalAttr := new.Header.Attributes("")
		oldGlobalAttr := old.Header.Attributes("")
		if len(newGlobalAttr) != len(oldGlobalAttr) {
			t.Fatalf("global attr length: %d != %d", len(newGlobalAttr), len(oldGlobalAttr))
		}
		for i, attr := range newGlobalAttr {
			if attr != oldGlobalAttr[i] {
				t.Errorf("global attr %d, %s != %s", i, attr, oldGlobalAttr[i])
			}
		}
	})
	t.Run("variables", func(t *testing.T) {
		newVars := new.Header.Variables()
		oldVars := new.Header.Variables()
		if len(newVars) != len(oldVars) {
			t.Fatalf("number of variables: %d != %d", len(newVars), len(oldVars))
		}
		for i, v := range newVars {
			t.Run("variable "+v, func(t *testing.T) {
				oldV := oldVars[i]
				if v != oldV {
					t.Fatalf("%d: %s != %s", i, v, oldV)
				}
				t.Run("attr", func(t *testing.T) {
					newAttr := new.Header.Attributes(v)
					oldAttr := old.Header.Attributes(v)
					if len(newAttr) != len(oldAttr) {
						t.Fatalf("attr length: %d != %d", len(newAttr), len(oldAttr))
					}
					for i, attr := range newAttr {
						if attr != oldAttr[i] {
							t.Errorf("%d, %s != %s", i, attr, oldAttr[i])
						}
						newAttrVal := new.Header.GetAttribute(v, attr)
						oldAttrVal := old.Header.GetAttribute(v, attr)
						interfaceEqualWithinTol(t, newAttrVal, oldAttrVal, tol)
					}
				})
				t.Run("data", func(t *testing.T) {
					newDim := new.Header.Lengths(v)
					oldDim := old.Header.Lengths(v)
					if !reflect.DeepEqual(newDim, oldDim) {
						t.Fatalf("dimensions: %v != %v", newDim, oldDim)
					}
					newData := new.Header.ZeroValue(v, arrayLen(newDim))
					nr := new.Reader(v, nil, nil)
					if _, err := nr.Read(newData); err != nil {
						t.Fatal(err)
					}
					oldData := old.Header.ZeroValue(v, arrayLen(oldDim))
					or := old.Reader(v, nil, nil)
					if _, err := or.Read(oldData); err != nil {
						t.Fatal(err)
					}
					interfaceEqualWithinTol(t, newData, oldData, tol)
				})
			})
		}
	})
}

func arrayLen(dim []int) int {
	i := 1
	for _, d := range dim {
		i *= d
	}
	return i
}

func interfaceEqualWithinTol(t *testing.T, new, old interface{}, tol float64) {
	switch tp := new.(type) {
	case int, string, []int, []int32, []string:
		if !reflect.DeepEqual(new, old) {
			t.Errorf("%v != %v", new, old)
		}
	case float64:
		if !floats.EqualWithinAbsOrRel(new.(float64), old.(float64), tol, tol) {
			t.Errorf("%g != %g", new, old)
		}
	case []float64:
		newV := new.([]float64)
		oldV := old.([]float64)
		if len(newV) != len(oldV) {
			t.Errorf("length %d != %d", len(newV), len(oldV))
			return
		}
		for i, n := range newV {
			o := oldV[i]
			if !floats.EqualWithinAbsOrRel(n, o, tol, tol) {
				t.Errorf("%g != %g", n, o)
			}
		}
	case []float32:
		newV := new.([]float32)
		oldV := old.([]float32)
		if len(newV) != len(oldV) {
			t.Errorf("length %d != %d", len(newV), len(oldV))
			return
		}
		for i, n := range newV {
			o := oldV[i]
			if !floats.EqualWithinAbsOrRel(float64(n), float64(o), tol, tol) {
				t.Errorf("%d: %g != %g", i, n, o)
			}
		}
	default:
		t.Fatalf("invalid type %T", tp)
	}
}
