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

package sr

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"bitbucket.org/ctessum/cdf"
	"github.com/BurntSushi/toml"
	"github.com/gonum/floats"
	"github.com/spatialmodel/inmap"
)

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
	cfg, err := loadConfig("../cmd/inmap/configExample.toml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.VariableGridData = strings.TrimSuffix(cfg.VariableGridData, ".gob") + "_SR.gob"
	command := "nocommand"
	logDir := "noLogs"
	var nodes []string // no nodes means it will run locally.
	sr, err := NewSR(cfg.VariableGridData, cfg.InMAPData, command,
		logDir, &cfg.VarGrid, nodes)
	if err != nil {
		t.Fatal(err)
	}
	outfile := "../cmd/inmap/testdata/testSR.ncf"
	defer os.Remove(outfile)
	layers := []int{0, 2, 4}
	begin := 0 // layer 0
	end := -1
	if err = sr.Run(outfile, layers, begin, end); err != nil {
		t.Fatal(err)
	}

	// Run it again for different indices.
	sr, err = NewSR(cfg.VariableGridData, cfg.InMAPData, command,
		logDir, &cfg.VarGrid, nodes)
	if err != nil {
		t.Fatal(err)
	}
	begin = 20 // layer 2
	end = 22
	if err = sr.Run(outfile, layers, begin, end); err != nil {
		t.Fatal(err)
	}
	t.Run("compare ncf", func(t *testing.T) {
		ncfWithinTol(t, "../cmd/inmap/testdata/testSR.ncf", "../cmd/inmap/testdata/testSR.ncf", 1.e-10)
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
				t.Errorf("%g != %g", n, o)
			}
		}
	default:
		t.Fatalf("invalid type %T", tp)
	}
}
