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
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/kardianos/osext"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/sr"
)

// RunSR runs the SR matrix creator.
//
// VariableGridData is the path to the location of the variable-resolution gridded
// InMAP data, or the location where it should be created if it doesn't already
// exist.
//
// InMAPData is the path to location of baseline meteorology and pollutant data.
//
// LogDir is the directory that log files should be stored in when creating
// a source-receptor matrix.
//
// OutputFile is the path where the output file is or should be created
// when creating a source-receptor matrix.
//
// VarGrid provides information for specifying the variable resolution grid.
//
// configFile give the path to the configuration file.
//
// begin and end specify the beginning and end grid indices to process.
//
// layers specifies which vertical layers to process.
func RunSR(VariableGridData, InMAPData, LogDir, OutputFile string, VarGrid *inmap.VarGridConfig, configFile string, begin, end int, layers []int) error {
	nodes, err := sr.PBSNodes()
	if err != nil {
		log.Printf("Problem reading $PBS_NODEFILE: %v. Continuing on local machine.", err)
	}

	command, err := osext.Executable()
	if err != nil {
		return err
	}
	command = fmt.Sprintf("%s  worker --config=%s --rpcport=%s", command, configFile, sr.RPCPort)

	sr, err := sr.NewSR(VariableGridData, InMAPData, command,
		LogDir, VarGrid, nodes)
	if err != nil {
		return err
	}

	if err = sr.Run(OutputFile, layers, begin, end); err != nil {
		return err
	}

	return nil
}

// NewWorker starts a new worker.
//
// VariableGridData is the path to the location of the variable-resolution gridded
// InMAP data, or the location where it should be created if it doesn't already
// exist.
//
// InMAPData is the path to location of baseline meteorology and pollutant data.
//
// VarGrid provides information for specifying the variable resolution grid.
func NewWorker(VariableGridData, InMAPData string, VarGrid *inmap.VarGridConfig) (*sr.Worker, error) {
	r, err := os.Open(VariableGridData)
	if err != nil {
		return nil, fmt.Errorf("problem opening file to load VariableGridData: %v", err)
	}
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			inmap.Load(r, VarGrid, nil),
		},
	}
	if err = d.Init(); err != nil {
		return nil, err
	}

	worker := sr.NewWorker(VarGrid, InMAPData, d.GetGeometry(0, false))
	return worker, nil
}

// SRPredict uses the SR matrix specified in SROutputFile
// to predict concentrations resulting
// from the emissions in EmissionsShapefiles, outputting the
// results in OutputFile. EmissionUnits specifies the units
// of the emissions. VarGrid specifies the variable resolution grid.
func SRPredict(EmissionUnits, SROutputFile, OutputFile string, EmissionsShapefiles []string, VarGrid *inmap.VarGridConfig) error {
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

	emis, err := inmap.ReadEmissionShapefiles(vgsr, EmissionUnits, msgLog, EmissionsShapefiles...)
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
	type rec struct {
		geom.Polygon
		PNH4, PNO3, PSO4, SOA, PrimaryPM25, TotalPM25 float64
	}

	o, err := shp.NewEncoder(OutputFile, rec{})
	if err != nil {
		return err
	}

	g := r.Geometry()

	for i, tpm := range conc.TotalPM25() {
		r := rec{
			Polygon:     g[i].(geom.Polygon),
			PNH4:        conc.PNH4[i],
			PNO3:        conc.PNO3[i],
			PSO4:        conc.PSO4[i],
			PrimaryPM25: conc.PrimaryPM25[i],
			SOA:         conc.SOA[i],
			TotalPM25:   tpm,
		}
		err = o.Encode(r)
		if err != nil {
			return err
		}
	}
	o.Close()
	// Projection definition. This may need to be changed for a different
	// spatial domain.
	// TODO: Make this settable by the user, or at least check to make sure it
	// matches the InMAPProj configuration variable.
	const proj4 = `PROJCS["Lambert_Conformal_Conic",GEOGCS["GCS_unnamed ellipse",DATUM["D_unknown",SPHEROID["Unknown",6370997,0]],PRIMEM["Greenwich",0],UNIT["Degree",0.017453292519943295]],PROJECTION["Lambert_Conformal_Conic"],PARAMETER["standard_parallel_1",33],PARAMETER["standard_parallel_2",45],PARAMETER["latitude_of_origin",40],PARAMETER["central_meridian",-97],PARAMETER["false_easting",0],PARAMETER["false_northing",0],UNIT["Meter",1]]`
	// Create .prj file
	f, err = os.Create(OutputFile[0:len(OutputFile)-len(filepath.Ext(OutputFile))] + ".prj")
	if err != nil {
		return fmt.Errorf("error creating output prj file: %v", err)
	}
	fmt.Fprint(f, proj4)
	f.Close()

	return nil
}
