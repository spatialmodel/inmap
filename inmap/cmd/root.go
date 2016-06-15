package cmd

import (
	"fmt"

	"github.com/spatialmodel/inmap"
	"github.com/spf13/cobra"
)

const year = "2016"

var (
	configFile string
	config     *configData
)

// RootCmd is the main command.
var RootCmd = &cobra.Command{
	Use:   "inmap",
	Short: "A reduced-form air quality model.",
	Long: `A reduced-form air quality model for fine particulate matter (PM2.5).
          Additional information is available at http://inmap.spatialmodel.com.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return Startup(configFile)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		completedMessage()
	},
}

// Startup reads the configuration file and prints a welcome message.
func Startup(configFile string) error {
	if configFile == "" {
		return fmt.Errorf("need to specify configuration file as in " +
			"`inmap -config=configFile.toml`")
	}
	var err error
	config, err = readConfigFile(configFile)
	if err != nil {
		return err
	}

	fmt.Println("\n" +
		"------------------------------------------------\n" +
		"                    Welcome!\n" +
		"  (In)tervention (M)odel for (A)ir (P)ollution  \n" +
		"                Version " + inmap.Version + "   \n" +
		"               Copyright 2013-" + year + "      \n" +
		"     Regents of the University of Minnesota     \n" +
		"------------------------------------------------")
	return nil
}

func completedMessage() {
	fmt.Println("\n" +
		"------------------------------------\n" +
		"           InMAP Completed!\n" +
		"------------------------------------")
}

func init() {
	RootCmd.AddCommand(versionCmd)

	RootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is ./inmap.toml)")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of InMAP",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("InMAP v%s", inmap.Version)
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
	},
}
