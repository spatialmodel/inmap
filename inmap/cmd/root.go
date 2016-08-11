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

// Package cmd contains commands and subcommands for the InMAP command-line interface.
package cmd

import (
	"fmt"

	"github.com/spatialmodel/inmap"
	"github.com/spf13/cobra"
)

const year = "2016"

var (
	configFile string

	// Config holds the global configuration data.
	Config *ConfigData
)

// RootCmd is the main command.
var RootCmd = &cobra.Command{
	Use:   "inmap",
	Short: "A reduced-form air quality model.",
	Long: `InMAP is a reduced-form air quality model for fine particulate matter (PM2.5).
			Use the subcommands specified below to access the model functionality.
      Additional information is available at http://inmap.spatialmodel.com.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return labelErr(Startup(configFile))
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		completedMessage()
	},
}

// Startup reads the configuration file and prints a welcome message.
func Startup(configFile string) error {
	var err error
	Config, err = ReadConfigFile(configFile)
	if err != nil {
		return err
	}

	fmt.Println("\n" +
		"------------------------------------------------\n" +
		"                    Welcome!\n" +
		"  (In)tervention (M)odel for (A)ir (P)ollution  \n" +
		"                Version " + inmap.Version + "   \n" +
		"               Copyright 2013-" + year + "      \n" +
		"                the InMAP Authors               \n" +
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

	RootCmd.PersistentFlags().StringVar(&configFile, "config", "./inmap.toml", "configuration file location")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  "version prints the version number of this version of InMAP.",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("InMAP v%s\n", inmap.Version)
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
	},
}
