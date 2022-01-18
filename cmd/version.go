package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of this program",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("1.1.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
