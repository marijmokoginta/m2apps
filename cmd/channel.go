package cmd

import (
	"fmt"
	"m2apps/internal/storage"
	"m2apps/internal/ui"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage application update channel",
}

var channelSetCmd = &cobra.Command{
	Use:   "set <app_id> <channel>",
	Short: "Set update channel for an installed application",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		appID := strings.TrimSpace(args[0])
		channel := normalizeChannelInput(args[1])

		message, err := runSetChannel(appID, channel)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}
		fmt.Println(ui.Success(message))
	},
}

func init() {
	channelCmd.AddCommand(channelSetCmd)
	rootCmd.AddCommand(channelCmd)
}

func normalizeChannelInput(channel string) string {
	return strings.ToLower(strings.TrimSpace(channel))
}

func isValidChannel(channel string) bool {
	switch channel {
	case "stable", "beta", "alpha":
		return true
	default:
		return false
	}
}

func runSetChannel(appID string, channel string) (string, error) {
	if !isValidChannel(channel) {
		return "", fmt.Errorf("invalid channel. use one of: stable, beta, alpha")
	}

	store, err := storage.New()
	if err != nil {
		return "", err
	}

	config, err := store.Load(appID)
	if err != nil {
		return "", fmt.Errorf("failed to load app metadata: %w", err)
	}

	if normalizeChannelInput(config.Channel) == channel {
		return fmt.Sprintf("[OK] Channel for %s is already %s", config.AppID, channel), nil
	}

	config.Channel = channel
	if err := store.Save(config.AppID, config); err != nil {
		return "", fmt.Errorf("failed to update channel: %w", err)
	}

	return fmt.Sprintf("[OK] Channel for %s set to %s", config.AppID, channel), nil
}
