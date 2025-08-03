package cmd

import "github.com/spf13/cobra"

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "install a riceThing compatible configuration by pointing towards a directory or a github link.",
}

func init() {
	rootCmd.AddCommand(installCmd)
}
