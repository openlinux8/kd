package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var Version string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print kdeploy version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kdeploy %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
