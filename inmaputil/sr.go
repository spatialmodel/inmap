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

package inmaputil

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spatialmodel/inmap/sr"
)

// StartSR starts the SR matrix creator, getting configuration information from the
// global Cfg variable.
//
// jobName is a user-specified name for the SR creation job.
//
// cmds is a list of InMAP subcommands for the individual simulations.
//
// memoryGB is the RAM required for each simulation, in GB.
//
// VariableGridData is the path to the location of the variable-resolution gridded
// InMAP data.
//
// VarGrid provides information for specifying the variable resolution grid.
//
// begin and end specify the beginning and end grid indices to process.
//
// layers specifies which vertical layers to process.
//
// client is a client of the cluster that will run the simulations.
func StartSR(ctx context.Context, jobName string, cmds []string, memoryGB int32, VariableGridData string, VarGrid *inmap.VarGridConfig, begin, end int, layers []int, client cloudrpc.CloudRPCClient, cfg *Cfg) error {
	outChan := outChan()
	varGridReader, err := os.Open(maybeDownload(ctx, VariableGridData, outChan))
	if err != nil {
		return fmt.Errorf("starting SR matrix---can't open variable grid data file: %v", err)
	}
	sr, err := sr.NewSR(varGridReader, VarGrid, client)
	if err != nil {
		return err
	}
	version := cfg.GetString("version")
	if err = sr.Start(ctx, jobName, version, layers, begin, end, cfg.Root, cfg.Viper, cmds, cfg.InputFiles(), memoryGB); err != nil {
		return err
	}
	return nil
}

// SaveSR saves the SR matrix results to an output file.
//
// jobName is a user-specified name for the SR creation job.
//
// VariableGridData is the path to the location of the variable-resolution gridded
// InMAP data.
//
// OutputFile is the path where the output file is or should be created
// when creating a source-receptor matrix.
//
// VarGrid provides information for specifying the variable resolution grid.
//
// begin and end specify the beginning and end grid indices to save.
//
// layers specifies which vertical layers to save.
//
// client is a client of the cluster that will run the simulations.
func SaveSR(ctx context.Context, jobName, OutputFile string, VariableGridData string, VarGrid *inmap.VarGridConfig, begin, end int, layers []int, client cloudrpc.CloudRPCClient) error {
	varGridReader, err := os.Open(VariableGridData)
	if err != nil {
		return fmt.Errorf("saving SR matrix---can't open variable grid data file: %v", err)
	}
	sr, err := sr.NewSR(varGridReader, VarGrid, client)
	if err != nil {
		return err
	}
	return sr.Save(ctx, OutputFile, jobName, layers, begin, end)
}

// CleanSR cleans up remote data created during the SR matrix creation simulations.
func CleanSR(ctx context.Context, jobName, VariableGridData string, VarGrid *inmap.VarGridConfig, begin, end int, layers []int, client cloudrpc.CloudRPCClient) error {
	varGridReader, err := os.Open(VariableGridData)
	if err != nil {
		return fmt.Errorf("saving SR matrix---can't open variable grid data file: %v", err)
	}
	sr, err := sr.NewSR(varGridReader, VarGrid, client)
	if err != nil {
		return err
	}
	return sr.Clean(ctx, jobName, layers, begin, end)
}

// SRPredict uses the SR matrix specified in SROutputFile
// to predict concentrations resulting
// from the emissions in EmissionsShapefiles (optionally
// masked by emissionMask), outputting the
// results specified by outputVaraibles in OutputFile.
// EmissionUnits specifies the units
// of the emissions. VarGrid specifies the variable resolution grid.
func SRPredict(EmissionUnits, SROutputFile, OutputFile string, outputVariables map[string]string, EmissionsShapefiles []string, emissionMask geom.Polygon, VarGrid *inmap.VarGridConfig) error {
	msgLog := make(chan string)
	go func() {
		for {
			log.Println(<-msgLog)
		}
	}()

	vgsr, err := spatialRef(VarGrid)
	if err != nil {
		return err
	}

	emis, err := inmap.ReadEmissionShapefiles(vgsr, EmissionUnits, msgLog, emissionMask, EmissionsShapefiles...)
	if err != nil {
		return err
	}
	f, err := os.Open(SROutputFile)
	if err != nil {
		return err
	}
	r, err := sr.NewReader(f)
	if err != nil {
		return err
	}
	conc, err := r.Concentrations(emis.EmisRecords()...)
	if err != nil {
		if _, ok := err.(sr.AboveTopErr); ok {
			log.Printf("%v; calculating concentrations for emissions in SR matrix top layer.", err)
		} else {
			return err
		}
	}
	if err = r.SetConcentrations(conc); err != nil {
		return err
	}

	var upload uploader
	o := upload.maybeUpload(OutputFile)
	if upload.err != nil {
		return upload.err
	}

	if err = r.Output(o, outputVariables, nil, vgsr); err != nil {
		return err
	}

	if err := upload.uploadOutput(nil); err != nil {
		return err
	}

	return nil
}
