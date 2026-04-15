package cmd

import (
	"fmt"
	"m2apps/internal/selfupdate"
	"m2apps/internal/ui"
	"os"

	"github.com/spf13/cobra"
)

var (
	internalSelfUpdateTarget    string
	internalSelfUpdateNewBinary string
	internalSelfUpdateParentPID int
	internalSelfUpdateRestart   bool
	internalSelfUpdateDaemon    bool
)

var internalCmd = &cobra.Command{
	Use:    "internal",
	Short:  "Internal commands",
	Hidden: true,
}

var internalSelfUpdateCmd = &cobra.Command{
	Use:    "self-update",
	Short:  "Run internal self-update helper",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := selfupdate.RunInternalSelfUpdate(
			internalSelfUpdateTarget,
			internalSelfUpdateNewBinary,
			internalSelfUpdateParentPID,
			internalSelfUpdateRestart,
			internalSelfUpdateDaemon,
		); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
	},
}

func init() {
	internalSelfUpdateCmd.Flags().StringVar(&internalSelfUpdateTarget, "target", "", "target executable path")
	internalSelfUpdateCmd.Flags().StringVar(&internalSelfUpdateNewBinary, "new", "", "new executable path")
	internalSelfUpdateCmd.Flags().IntVar(&internalSelfUpdateParentPID, "parent-pid", 0, "parent process PID")
	internalSelfUpdateCmd.Flags().BoolVar(&internalSelfUpdateRestart, "restart", true, "restart application after replacing binary")
	internalSelfUpdateCmd.Flags().BoolVar(&internalSelfUpdateDaemon, "restart-daemon", false, "restart daemon service after replacing binary")
	internalSelfUpdateCmd.MarkFlagRequired("target")
	internalSelfUpdateCmd.MarkFlagRequired("new")

	internalCmd.AddCommand(internalSelfUpdateCmd)
	rootCmd.AddCommand(internalCmd)
}
