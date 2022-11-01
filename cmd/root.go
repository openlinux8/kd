package cmd

import (
	"github.com/spf13/cobra"
	"log"
)

var rootCmd = &cobra.Command{
	Use:   "kdeploy",
	Short: "kdeploy is a kubernetes install tools",
	Long:  "kdeploy is a kubernetes install tools",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
