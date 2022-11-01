package cmd

import (
	"github.com/kinvin/kd/install"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initial your kubernetes HA Cluster",
	Long:  "initial your kubernetes HA Cluster",
	Run: func(cmd *cobra.Command, args []string) {
		install.Kubeinit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&install.ConfigFile, "config", "", "init config file")
}
