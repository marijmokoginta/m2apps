package cmd

import (
	"fmt"
	"m2apps/internal/daemon"
	"m2apps/internal/storage"
	"m2apps/internal/system"
	"m2apps/internal/ui"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <app_id>",
	Short: "Delete installed application",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appID := strings.TrimSpace(args[0])
		if !confirmAction(fmt.Sprintf("Delete application %s?", appID)) {
			fmt.Println(ui.Warning("[WARN] Delete cancelled."))
			return
		}

		fmt.Println(ui.Info(fmt.Sprintf("[INFO] Deleting application %s...", appID)))

		if err := runDelete(appID); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		fmt.Println(ui.Success(fmt.Sprintf("[OK] Application %s deleted", appID)))
	},
}

func runDelete(appID string) error {
	id := strings.TrimSpace(appID)
	if id == "" {
		return fmt.Errorf("app_id is required")
	}

	store, err := storage.New()
	if err != nil {
		return err
	}

	cfg, err := store.Load(id)
	if err != nil {
		return fmt.Errorf("failed to load app metadata: %w", err)
	}

	installPath := filepath.Clean(strings.TrimSpace(cfg.InstallPath))
	if installPath == "" {
		return fmt.Errorf("invalid install path for app %s", id)
	}
	if installPath == string(os.PathSeparator) {
		return fmt.Errorf("refusing to delete root path")
	}

	if _, err := os.Stat(installPath); err == nil {
		if err := os.RemoveAll(installPath); err != nil {
			return fmt.Errorf("failed to remove installed app directory: %w", err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to access installed app directory: %w", err)
	}

	if err := os.RemoveAll(system.GetAppDir(id)); err != nil {
		return fmt.Errorf("failed to remove app metadata directory: %w", err)
	}

	daemonManager, err := daemon.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize daemon manager: %w", err)
	}
	if err := daemonManager.UnregisterApp(id); err != nil {
		return fmt.Errorf("failed to update daemon app registry: %w", err)
	}

	return nil
}
