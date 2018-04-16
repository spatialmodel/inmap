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
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
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
	os.Remove(outfile)
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
}
