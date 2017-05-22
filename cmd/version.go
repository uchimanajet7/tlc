package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of tlc",
	Long:  `All software has versions. This is tlc's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("OneTab URL list cleaner v0.1 -- HEAD")
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
