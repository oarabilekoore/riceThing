package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ricething",
	Short: "riceThing helps you to clone, quickly install and share your riced system.",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
