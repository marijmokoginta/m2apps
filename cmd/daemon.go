package cmd

import (
	"context"
	"fmt"
	"m2apps/internal/daemon"
	"m2apps/internal/service"
	"m2apps/internal/ui"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage M2Apps background daemon",
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install daemon OS service",
	Run: func(cmd *cobra.Command, args []string) {
		runServiceAction("Installing service...", "Service installed", "install service", func(m service.ServiceManager) error {
			return m.Install()
		})
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall daemon OS service",
	Run: func(cmd *cobra.Command, args []string) {
		runServiceAction("Uninstalling service...", "Service uninstalled", "uninstall service", func(m service.ServiceManager) error {
			return m.Uninstall()
		})
	},
}

var daemonEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable daemon service auto-start",
	Run: func(cmd *cobra.Command, args []string) {
		runServiceAction("Enabling service...", "Service enabled", "enable service", func(m service.ServiceManager) error {
			return m.Enable()
		})
	},
}

var daemonDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable daemon service auto-start",
	Run: func(cmd *cobra.Command, args []string) {
		runServiceAction("Disabling service...", "Service disabled", "disable service", func(m service.ServiceManager) error {
			return m.Disable()
		})
	},
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon service",
	Run: func(cmd *cobra.Command, args []string) {
		runServiceAction("Starting service...", "Service started", "start service", func(m service.ServiceManager) error {
			return m.Start()
		})
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop daemon service",
	Run: func(cmd *cobra.Command, args []string) {
		runServiceAction("Stopping service...", "Service stopped", "stop service", func(m service.ServiceManager) error {
			return m.Stop()
		})
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon service status",
	Run: func(cmd *cobra.Command, args []string) {
		manager := service.NewServiceManager()

		fmt.Println(ui.Info("[INFO] Checking service status..."))
		status, err := manager.Status()
		if err != nil {
			printServiceError("check service status", err)
			os.Exit(1)
		}

		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Service status: %s", status)))
	},
}

var daemonRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run daemon process (internal)",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		manager, err := daemon.NewManager()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		if err := manager.RunForeground(ctx); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
	},
}

func init() {
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)
	daemonCmd.AddCommand(daemonEnableCmd)
	daemonCmd.AddCommand(daemonDisableCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonRunCmd)
	rootCmd.AddCommand(daemonCmd)
}

func runServiceAction(infoMessage, successMessage, action string, fn func(service.ServiceManager) error) {
	manager := service.NewServiceManager()

	fmt.Println(ui.Info("[INFO] " + infoMessage))
	if err := fn(manager); err != nil {
		printServiceError(action, err)
		os.Exit(1)
	}
	fmt.Println(ui.Success("[OK] " + successMessage))
}

func printServiceError(action string, err error) {
	message := strings.TrimSpace(err.Error())
	if strings.HasPrefix(message, "[ERROR]") {
		fmt.Println(ui.Error(message))
		return
	}
	fmt.Println(ui.Error(fmt.Sprintf("[ERROR] Failed to %s: %s", action, message)))
}
