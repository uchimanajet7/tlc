package cmd

import "github.com/spf13/cobra"

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:           "tlc",
	Short:         "This tool is OneTab URL list cleaner",
	Long:          "This tool is OneTab URL list cleaner",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	cobra.OnInitialize()
}
