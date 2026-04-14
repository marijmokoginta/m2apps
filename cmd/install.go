package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install application from install.json",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting installation...")
	},
}
