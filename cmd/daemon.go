package cmd

import (
	"context"
	"fmt"
	"io"
	"m2apps/internal/daemon"
	"m2apps/internal/privilege"
	"m2apps/internal/service"
	"m2apps/internal/system"
	"m2apps/internal/ui"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

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
		if err := runDaemonCommand("install"); err != nil {
			os.Exit(1)
		}
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall daemon OS service",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDaemonCommand("uninstall"); err != nil {
			os.Exit(1)
		}
	},
}

var daemonEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable daemon service auto-start",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDaemonCommand("enable"); err != nil {
			os.Exit(1)
		}
	},
}

var daemonDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable daemon service auto-start",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDaemonCommand("disable"); err != nil {
			os.Exit(1)
		}
	},
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon service",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDaemonCommand("start"); err != nil {
			os.Exit(1)
		}
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop daemon service",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDaemonCommand("stop"); err != nil {
			os.Exit(1)
		}
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon service status",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDaemonStatus(); err != nil {
			os.Exit(1)
		}
	},
}

var daemonRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run daemon process (internal)",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		manager, err := daemon.NewManager()
		if err != nil {
			_ = daemon.AppendRuntimeLog("ERROR", fmt.Sprintf("failed to initialize daemon manager: %v", err))
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
			_ = daemon.AppendRuntimeLog("ERROR", fmt.Sprintf("daemon runtime error: %v", err))
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

func runServiceAction(infoMessage, successMessage, action string, fn func(service.ServiceManager) error) error {
	manager := service.NewServiceManager()

	fmt.Println(ui.Info("[INFO] " + infoMessage))
	if err := fn(manager); err != nil {
		printServiceError(action, err)
		return err
	}
	fmt.Println(ui.Success("[OK] " + successMessage))
	return nil
}

func printServiceError(action string, err error) {
	message := strings.TrimSpace(err.Error())
	if strings.HasPrefix(message, "[ERROR]") {
		fmt.Println(ui.Error(message))
		return
	}
	fmt.Println(ui.Error(fmt.Sprintf("[ERROR] Failed to %s: %s", action, message)))
}

func runDaemonStatus() error {
	manager := service.NewServiceManager()

	fmt.Println(ui.Info("[INFO] Checking service status..."))
	status, err := manager.Status()
	if err != nil {
		printServiceError("check service status", err)
		return err
	}

	fmt.Println(ui.Info(fmt.Sprintf("[INFO] Service status: %s", status)))
	printLocalAPISummary()
	return nil
}

func runDaemonCommand(action string) error {
	if daemonActionNeedsSupervisor(action) {
		launched, err := triggerDaemonSupervisorPopup(action)
		if err != nil {
			printServiceError(action, err)
			return err
		}
		if launched {
			return nil
		}
	}

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "install":
		return runServiceAction("Installing service...", "Service installed", "install service", func(m service.ServiceManager) error {
			return m.Install()
		})
	case "uninstall":
		return runServiceAction("Uninstalling service...", "Service uninstalled", "uninstall service", func(m service.ServiceManager) error {
			return m.Uninstall()
		})
	case "enable":
		return runServiceAction("Enabling service...", "Service enabled", "enable service", func(m service.ServiceManager) error {
			return m.Enable()
		})
	case "disable":
		return runServiceAction("Disabling service...", "Service disabled", "disable service", func(m service.ServiceManager) error {
			return m.Disable()
		})
	case "start":
		return runServiceAction("Starting service...", "Service started", "start service", func(m service.ServiceManager) error {
			return m.Start()
		})
	case "stop":
		return runServiceAction("Stopping service...", "Service stopped", "stop service", func(m service.ServiceManager) error {
			return m.Stop()
		})
	case "status":
		return runDaemonStatus()
	case "api_info":
		return runLocalAPIInfo()
	case "api_ping":
		return runLocalAPIPing()
	default:
		return fmt.Errorf("unsupported daemon action %q", action)
	}
}

func daemonActionNeedsSupervisor(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "install", "uninstall", "enable", "disable", "start", "stop":
		return true
	default:
		return false
	}
}

func triggerDaemonSupervisorPopup(action string) (bool, error) {
	if privilege.IsElevated() {
		return false, nil
	}

	fmt.Println(ui.Info("[INFO] Supervisor permission required. Opening OS authentication popup..."))
	if err := privilege.RelaunchElevated([]string{"daemon", strings.TrimSpace(action)}); err != nil {
		return false, err
	}

	return true, nil
}

type localAPIState struct {
	Port       int
	PID        int
	URL        string
	TCPReady   bool
	HTTPReady  bool
	HTTPStatus int
	Error      string
}

func collectLocalAPIState() localAPIState {
	state := localAPIState{}
	state.PID = readDaemonPID()

	manager, err := daemon.NewManager()
	if err != nil {
		state.Error = err.Error()
		return state
	}

	port, err := manager.Port()
	if err != nil {
		state.Error = err.Error()
		return state
	}
	state.Port = port
	state.URL = fmt.Sprintf("http://127.0.0.1:%d", port)

	tcpConn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err == nil {
		state.TCPReady = true
		_ = tcpConn.Close()
	}

	httpClient := &http.Client{Timeout: 3 * time.Second}
	resp, err := httpClient.Get(state.URL + "/health")
	if err != nil {
		if state.Error == "" {
			state.Error = err.Error()
		}
		return state
	}
	defer resp.Body.Close()

	state.HTTPStatus = resp.StatusCode
	if resp.StatusCode > 0 {
		state.HTTPReady = true
	}
	return state
}

func printLocalAPISummary() {
	spinner := ui.NewSpinner()
	spinner.Start("[INFO] Checking Local API...")
	state := collectLocalAPIState()
	spinner.Stop(ui.Success("[OK] Local API check completed"))

	fmt.Println(ui.Info("[INFO] Local API Summary:"))
	if state.Port <= 0 {
		fmt.Printf("%s %s\n", ui.Warning("[WARN] Port:"), "not available")
		if strings.TrimSpace(state.Error) != "" {
			fmt.Printf("%s %s\n", ui.Warning("[WARN] Detail:"), state.Error)
		}
		return
	}

	fmt.Printf("%s %d\n", ui.Info("[INFO] PID:"), state.PID)
	fmt.Printf("%s %d\n", ui.Info("[INFO] Port:"), state.Port)
	fmt.Printf("%s %s\n", ui.Info("[INFO] URL:"), state.URL)
	fmt.Printf("%s %t\n", ui.Info("[INFO] TCP Health:"), state.TCPReady)
	fmt.Printf("%s %t\n", ui.Info("[INFO] HTTP Health:"), state.HTTPReady)
	if state.HTTPStatus > 0 {
		fmt.Printf("%s %d\n", ui.Info("[INFO] Last HTTP Status:"), state.HTTPStatus)
	}
	if strings.TrimSpace(state.Error) != "" {
		fmt.Printf("%s %s\n", ui.Warning("[WARN] Detail:"), state.Error)
	}
}

func runLocalAPIInfo() error {
	printLocalAPISummary()
	return nil
}

func runLocalAPIPing() error {
	spinner := ui.NewSpinner()
	spinner.Start("[INFO] Pinging Local API endpoint...")
	state := collectLocalAPIState()
	if state.Port <= 0 || strings.TrimSpace(state.URL) == "" {
		spinner.Stop(ui.Error("[FAIL] Local API ping failed"))
		return fmt.Errorf("local API port is not available: %s", state.Error)
	}

	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Get(state.URL + "/ping")
	if err != nil {
		spinner.Stop(ui.Error("[FAIL] Local API ping failed"))
		return fmt.Errorf("failed to ping local API: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	spinner.Stop(ui.Success("[OK] Local API ping completed"))
	fmt.Printf("%s %s\n", ui.Info("[INFO] URL:"), state.URL+"/ping")
	fmt.Printf("%s %d\n", ui.Info("[INFO] HTTP Status:"), resp.StatusCode)
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "(empty)"
	}
	fmt.Printf("%s %s\n", ui.Info("[INFO] Response:"), trimmed)
	return nil
}

func readDaemonPID() int {
	data, err := os.ReadFile(filepath.Join(system.GetDaemonDir(), "daemon.pid"))
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}
