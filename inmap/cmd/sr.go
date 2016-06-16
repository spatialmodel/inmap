package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/ctessum/rpccluster"
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
		return RunSR(begin, end, layers)
	},
}

// RunSR runs the SR matrix creator.
func RunSR(begin, end int, layers []int) error {
	nodes, err := rpccluster.PBSNodes()
	if err != nil {
		log.Printf("Problem reading $PBS_NODEFILE: %v. Continuing on local machine.", err)
	}

	command, err := osext.Executable()
	if err != nil {
		return err
	}
	command += " worker --rpcport=" + sr.RPCPort

	sr, err := sr.NewSR(Config.VariableGridData, Config.InMAPData, command,
		Config.SRLogDir, &Config.VarGrid, nodes)
	if err != nil {
		return err
	}

	if err = sr.Run(Config.SROutputFile, layers, begin, end); err != nil {
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
		worker, err := InitWorker()
		if err != nil {
			return err
		}
		return worker.Listen(sr.RPCPort)
	},
}

// InitWorker starts a new worker.
func InitWorker() (*sr.Worker, error) {

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

	worker, err := sr.NewWorker(&Config.VarGrid, Config.InMAPData, d.GetGeometry(0, false))
	if err != nil {
		return nil, err
	}
	return worker, nil
}
