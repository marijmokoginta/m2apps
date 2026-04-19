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
	internalSelfUpdateSource    string
	internalSelfUpdateParentPID int
	internalSelfUpdateRestart   bool
	internalSelfUpdateDaemon    bool
	internalSelfUpdateStatus    string
	internalDaemonAction        string
	internalDaemonActionStatus  string
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
			internalSelfUpdateSource,
			internalSelfUpdateParentPID,
			internalSelfUpdateRestart,
			internalSelfUpdateDaemon,
			internalSelfUpdateStatus,
		); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
	},
}

var internalDaemonActionCmd = &cobra.Command{
	Use:    "daemon-action",
	Short:  "Run daemon action in elevated/internal context",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		writeStatus := func(success bool, message string) {
			_ = writeDaemonActionStatus(internalDaemonActionStatus, daemonActionStatus{
				Success: success,
				Message: message,
			})
		}

		if err := runDaemonCommandWithoutSupervisor(internalDaemonAction); err != nil {
			writeStatus(false, err.Error())
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		writeStatus(true, "")
	},
}

func init() {
	internalSelfUpdateCmd.Flags().StringVar(&internalSelfUpdateTarget, "target", "", "target executable path")
	internalSelfUpdateCmd.Flags().StringVar(&internalSelfUpdateNewBinary, "new", "", "new executable path")
	internalSelfUpdateCmd.Flags().StringVar(&internalSelfUpdateSource, "source", "", "source executable path before migration")
	internalSelfUpdateCmd.Flags().IntVar(&internalSelfUpdateParentPID, "parent-pid", 0, "parent process PID")
	internalSelfUpdateCmd.Flags().BoolVar(&internalSelfUpdateRestart, "restart", true, "restart application after replacing binary")
	internalSelfUpdateCmd.Flags().BoolVar(&internalSelfUpdateDaemon, "restart-daemon", false, "restart daemon service after replacing binary")
	internalSelfUpdateCmd.Flags().StringVar(&internalSelfUpdateStatus, "status-file", "", "windows helper status file")
	internalSelfUpdateCmd.MarkFlagRequired("target")
	internalSelfUpdateCmd.MarkFlagRequired("new")

	internalDaemonActionCmd.Flags().StringVar(&internalDaemonAction, "action", "", "daemon action to execute")
	internalDaemonActionCmd.Flags().StringVar(&internalDaemonActionStatus, "status-file", "", "status output file")
	internalDaemonActionCmd.MarkFlagRequired("action")

	internalCmd.AddCommand(internalSelfUpdateCmd)
	internalCmd.AddCommand(internalDaemonActionCmd)
	rootCmd.AddCommand(internalCmd)
}
