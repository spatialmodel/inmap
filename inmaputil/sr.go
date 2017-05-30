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
func RunSR(cfg *ConfigData, configFile string, begin, end int, layers []int) error {
	nodes, err := sr.PBSNodes()
	if err != nil {
		log.Printf("Problem reading $PBS_NODEFILE: %v. Continuing on local machine.", err)
	}

	command, err := osext.Executable()
	if err != nil {
		return err
	}
	command = fmt.Sprintf("%s  worker --config=%s --rpcport=%s", command, configFile, sr.RPCPort)

	sr, err := sr.NewSR(cfg.VariableGridData, cfg.InMAPData, command,
		cfg.SR.LogDir, &cfg.VarGrid, nodes)
	if err != nil {
		return err
	}

	if err = sr.Run(cfg.SR.OutputFile, layers, begin, end); err != nil {
		return err
	}

	return nil
}

// NewWorker starts a new worker.
func NewWorker(cfg *ConfigData) (*sr.Worker, error) {
	r, err := os.Open(cfg.VariableGridData)
	if err != nil {
		return nil, fmt.Errorf("problem opening file to load VariableGridData: %v", err)
	}
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			inmap.Load(r, &cfg.VarGrid, nil),
		},
	}
	if err = d.Init(); err != nil {
		return nil, err
	}

	worker := sr.NewWorker(&cfg.VarGrid, cfg.InMAPData, d.GetGeometry(0, false))
	return worker, nil
}

// SRPredict uses the SR matrix specified in cfg.OutputFile
// to predict concentrations resulting
// from the emissions in cfg.EmissionsShapefiles, outputting the
// results in cfg.OutputFile. cfg.EmissionUnits specifies the units
// of the emissions.
func SRPredict(cfg *ConfigData) error {
	msgLog := make(chan string)
	go func() {
		for {
			log.Println(<-msgLog)
		}
	}()

	emis, err := inmap.ReadEmissionShapefiles(cfg.sr, cfg.EmissionUnits, msgLog, cfg.EmissionsShapefiles...)
	if err != nil {
		return err
	}
	f, err := os.Open(cfg.SR.OutputFile)
	if err != nil {
		return err
	}

	r, err := sr.NewReader(f)
	if err != nil {
		return err
	}

	conc, err := r.Concentrations(emis.EmisRecords()...)
	if err != nil {
		return err
	}

	type rec struct {
		geom.Polygon
		PNH4, PNO3, PSO4, SOA, PrimaryPM25, TotalPM25 float64
	}

	o, err := shp.NewEncoder(cfg.OutputFile, rec{})
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
	f, err = os.Create(cfg.OutputFile[0:len(cfg.OutputFile)-len(filepath.Ext(cfg.OutputFile))] + ".prj")
	if err != nil {
		return fmt.Errorf("error creating output prj file: %v", err)
	}
	fmt.Fprint(f, proj4)
	f.Close()

	return nil
}
