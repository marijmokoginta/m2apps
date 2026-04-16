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
	"runtime"
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
	Short:  "Run daemon process",
	Hidden: false,
	Run: func(cmd *cobra.Command, args []string) {
		printBanner()
		fmt.Printf("%s %s\n\n", ui.Info("[INFO] Version:"), appVersion)

		manager, err := daemon.NewManager()
		if err != nil {
			_ = daemon.AppendServiceLog("ERROR", fmt.Sprintf("failed to initialize daemon manager: %v", err))
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		if daemonRunDetach {
			if err := manager.Start(); err != nil {
				_ = daemon.AppendServiceLog("ERROR", fmt.Sprintf("daemon detach start failed: %v", err))
				fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
				printManualDaemonRunFallback()
				os.Exit(1)
			}
			fmt.Println(ui.Success("[OK] Daemon started in background mode"))
			fmt.Printf("%s %s\n", ui.Info("[INFO] Service log:"), daemonServiceLogPath())
			fmt.Printf("%s %s\n", ui.Info("[INFO] Access log:"), daemonAccessLogPath())
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if !daemonRunNoLog {
			fmt.Printf("%s %s\n", ui.Info("[INFO] Streaming daemon service logs from:"), daemonServiceLogPath())
			fmt.Printf("%s %s\n", ui.Info("[INFO] Streaming local API access logs from:"), daemonAccessLogPath())
			fmt.Println(ui.Info("[INFO] Showing only current runtime logs (history excluded)."))
			fmt.Println(ui.Info("[INFO] Press Ctrl+C to stop foreground daemon run."))
			go streamDaemonLogs(ctx, daemonServiceLogPath(), "SERVICE", true)
			go streamDaemonLogs(ctx, daemonAccessLogPath(), "API", true)
		}

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		if err := manager.RunForeground(ctx); err != nil {
			_ = daemon.AppendServiceLog("ERROR", fmt.Sprintf("daemon runtime error: %v", err))
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
	},
}

var (
	daemonRunDetach bool
	daemonRunNoLog  bool
)

func init() {
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)
	daemonCmd.AddCommand(daemonEnableCmd)
	daemonCmd.AddCommand(daemonDisableCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonRunCmd)
	daemonRunCmd.Flags().BoolVarP(&daemonRunDetach, "detach", "d", false, "Run daemon in background and return immediately")
	daemonRunCmd.Flags().BoolVar(&daemonRunNoLog, "no-log", false, "Run in foreground without streaming daemon log output")
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
			if strings.EqualFold(strings.TrimSpace(action), "start") {
				fmt.Println(ui.Warning("[WARN] Supervisor start failed, trying process fallback..."))
				if fallbackErr := startDaemonProcessFallback(); fallbackErr == nil {
					fmt.Println(ui.Success("[OK] Daemon started using process fallback mode"))
					printManualDaemonRunFallback()
					return nil
				}
				printManualDaemonRunFallback()
			}
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
		return runDaemonStart()
	case "stop":
		return runDaemonStop()
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

func runDaemonStart() error {
	manager := service.NewServiceManager()

	fmt.Println(ui.Info("[INFO] Starting service..."))
	if err := manager.Start(); err != nil {
		if runtime.GOOS == "windows" {
			fmt.Println(ui.Warning("[WARN] Failed to start Windows service, switching to process fallback..."))
			if fallbackErr := startDaemonProcessFallback(); fallbackErr == nil {
				fmt.Println(ui.Success("[OK] Daemon started using process fallback mode"))
				printManualDaemonRunFallback()
				return nil
			}
		}

		printServiceError("start service", err)
		printManualDaemonRunFallback()
		return err
	}

	fmt.Println(ui.Success("[OK] Service started"))
	return nil
}

func runDaemonStop() error {
	serviceManager := service.NewServiceManager()
	daemonManager, dmErr := daemon.NewManager()
	if dmErr != nil {
		return dmErr
	}

	fmt.Println(ui.Info("[INFO] Stopping daemon service/process..."))

	stopServiceErr := serviceManager.Stop()
	if stopServiceErr != nil && !isIgnorableServiceStopErr(stopServiceErr) {
		printServiceError("stop service", stopServiceErr)
	}

	if err := daemonManager.Stop(); err != nil {
		printServiceError("stop daemon process", err)
		return err
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		state := collectLocalAPIState()
		if state.Port <= 0 || (!state.TCPReady && !state.HTTPReady) {
			fmt.Println(ui.Success("[OK] Daemon stopped"))
			return nil
		}
		time.Sleep(180 * time.Millisecond)
	}

	state := collectLocalAPIState()
	if state.Port > 0 && (state.TCPReady || state.HTTPReady) {
		return fmt.Errorf("daemon stop verification failed: local API is still reachable at %s", state.URL)
	}

	fmt.Println(ui.Success("[OK] Daemon stopped"))
	return nil
}

func isIgnorableServiceStopErr(err error) bool {
	if err == nil {
		return true
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if message == "" {
		return false
	}

	return strings.Contains(message, "service not found") ||
		strings.Contains(message, "has not been started") ||
		strings.Contains(message, "already stopped") ||
		strings.Contains(message, "could not find the specified service")
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

func startDaemonProcessFallback() error {
	manager, err := daemon.NewManager()
	if err != nil {
		return err
	}
	return manager.Start()
}

func printManualDaemonRunFallback() {
	fmt.Println(ui.Info("[INFO] Manual fallback: run `m2apps daemon run` (foreground with logs)."))
	fmt.Println(ui.Info("[INFO] Background mode: run `m2apps daemon run --detach`."))
}

func daemonServiceLogPath() string {
	return daemon.ServiceLogPath()
}

func daemonAccessLogPath() string {
	return daemon.AccessLogPath()
}

func streamDaemonLogs(ctx context.Context, path string, source string, startAtEOF bool) {
	var offset int64
	if startAtEOF {
		if info, err := os.Stat(path); err == nil {
			offset = info.Size()
		}
	}
	ticker := time.NewTicker(350 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			size := info.Size()
			if size < offset {
				offset = 0
			}
			if size == offset {
				continue
			}

			file, err := os.Open(path)
			if err != nil {
				continue
			}

			_, _ = file.Seek(offset, io.SeekStart)
			data, _ := io.ReadAll(file)
			_ = file.Close()

			offset += int64(len(data))
			for _, line := range strings.Split(string(data), "\n") {
				text := strings.TrimSpace(line)
				if text == "" {
					continue
				}
				printDaemonLogLine(source, text)
			}
		}
	}
}

func printDaemonLogLine(source, line string) {
	label := strings.ToUpper(strings.TrimSpace(source))
	if label == "" {
		label = "LOG"
	}
	prefix := fmt.Sprintf("[%s]", label)

	lower := strings.ToLower(line)
	switch {
	case strings.Contains(line, "[ERROR]") || strings.Contains(line, "[FAIL]") || strings.Contains(lower, " failed"):
		fmt.Printf("%s %s\n", ui.Error(prefix), ui.Error(line))
	case strings.Contains(line, "[WARN]") || strings.Contains(lower, " warning"):
		fmt.Printf("%s %s\n", ui.Warning(prefix), ui.Warning(line))
	default:
		fmt.Printf("%s %s\n", ui.Info(prefix), line)
	}
}
