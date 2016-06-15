package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spatialmodel/inmap"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(gridCmd)
}

// gridCmd is a command that creates and saves a new variable resolution grid.
var gridCmd = &cobra.Command{
	Use:   "grid",
	Short: "Create a variable resolution grid",
	Long: "Create and save a variable resolution grid as specified by the " +
		"information in the configuration file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return Grid()
	},
}

// Grid creates and saves a new variable resolution grid.
func Grid() error {

	// Start a function to receive and print log messages.
	msgLog := make(chan string)
	go func() {
		for msg := range msgLog {
			log.Println(msg)
		}
	}()

	ctmData, err := getCTMData()
	if err != nil {
		return err
	}

	log.Println("Loading population and mortality rate data")

	pop, popIndices, mr, err := config.VarGrid.LoadPopMort()
	if err != nil {
		return err
	}

	w, err := os.Create(config.VariableGridData)
	if err != nil {
		return fmt.Errorf("problem creating file to store variable grid data in: %v", err)
	}

	d := &inmap.InMAP{
		InitFuncs: []inmap.DomainManipulator{
			config.VarGrid.RegularGrid(ctmData, pop, popIndices, mr, nil),
			config.VarGrid.MutateGrid(inmap.PopulationMutator(&config.VarGrid, popIndices),
				ctmData, pop, mr, nil),
			inmap.Save(w),
		},
	}
	if err := d.Init(); err != nil {
		return err
	}
	log.Printf("Grid successfully created at %s", config.VariableGridData)
	return nil
}
