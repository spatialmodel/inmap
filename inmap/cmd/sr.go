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

package cmd

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
	"github.com/spf13/cobra"
)

var (
	layers []int
	begin  int
	end    int
)

func init() {
	RootCmd.AddCommand(srCmd)

	srCmd.Flags().IntSliceVar(&layers, "layers", []int{0, 2, 4, 6},
		"List of layer numbers to create matrices for.")
	srCmd.Flags().IntVar(&begin, "begin", 0, "Beginning row index.")
	srCmd.Flags().IntVar(&end, "end", -1, "End row index. Default is -1 (the last row).")

	srCmd.AddCommand(srPredictCmd)

	RootCmd.AddCommand(workerCmd)

	srCmd.Flags().StringVar(&sr.RPCPort, "rpcport", "6060",
		"Set the port to be used for RPC communication.")
	workerCmd.Flags().StringVar(&sr.RPCPort, "rpcport", "6060",
		"Set the port to be used for RPC communication.")
}

// srCmd is a command that creates an SR matrix.
var srCmd = &cobra.Command{
	Use:   "sr",
	Short: "Create an SR matrix.",
	Long: `Create a source-receptor matrix from InMAP simulations.
    Simulations will be run on the cluster defined by $PBS_NODEFILE.
    If $PBS_NODEFILE doesn't exist, the simulations will run on the
    local machine.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return labelErr(RunSR(begin, end, layers))
	},
	DisableAutoGenTag: true,
}

// RunSR runs the SR matrix creator.
func RunSR(begin, end int, layers []int) error {
	nodes, err := sr.PBSNodes()
	if err != nil {
		log.Printf("Problem reading $PBS_NODEFILE: %v. Continuing on local machine.", err)
	}

	command, err := osext.Executable()
	if err != nil {
		return err
	}
	command = fmt.Sprintf("%s  worker --config=%s --rpcport=%s", command, configFile, sr.RPCPort)

	sr, err := sr.NewSR(Config.VariableGridData, Config.InMAPData, command,
		Config.SR.LogDir, &Config.VarGrid, nodes)
	if err != nil {
		return err
	}

	if err = sr.Run(Config.SR.OutputFile, layers, begin, end); err != nil {
		return err
	}

	return nil
}

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start an InMAP worker.",
	Long: `Start an InMAP worker that listens over RPC for simulation requests,
		does the simulations, and returns results.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		worker, err := NewWorker()
		if err != nil {
			return labelErr(err)
		}
		return labelErr(sr.WorkerListen(worker, sr.RPCPort))
	},
	DisableAutoGenTag: true,
}

// NewWorker starts a new worker.
func NewWorker() (*sr.Worker, error) {
	r, err := os.Open(Config.VariableGridData)
	if err != nil {
		return nil, fmt.Errorf("problem opening file to load VariableGridData: %v", err)
	}
	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			inmap.Load(r, &Config.VarGrid, nil),
		},
	}
	if err = d.Init(); err != nil {
		return nil, err
	}

	worker := sr.NewWorker(&Config.VarGrid, Config.InMAPData, d.GetGeometry(0, false))
	return worker, nil
}

// srPredictCmd is a command that makes predictions using the SR matrix.
var srPredictCmd = &cobra.Command{
	Use:   "predict",
	Short: "Predict concentrations",
	Long: `Use the SR matrix specified in the configuration file
	 field SR.OutputFile to predict concentrations resulting
	 from the emissions specified in the EmissionsShapefiles field in the configuration
	 file, outputting the results in the shapefile specified in OutputFile field.
	 of the configuration file. The EmissionUnits field in the configuration
	 file specifies the units of the emissions. Output units are μg particulate
	 matter per m³ air.

	 Output variables:
	 PNH4: Particulate ammonium
	 PNO3: Particulate nitrate
	 PSO4: Particulate sulfate
	 SOA: Secondary organic aerosol
	 PrimaryPM25: Primarily emitted PM2.5
	 TotalPM25: The sum of the above components`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return labelErr(SRPredict(Config))
	},
	DisableAutoGenTag: true,
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

	emis, err := inmap.ReadEmissionShapefiles(Config.sr, Config.EmissionUnits,
		msgLog, Config.EmissionsShapefiles...)
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
		err := o.Encode(r)
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
