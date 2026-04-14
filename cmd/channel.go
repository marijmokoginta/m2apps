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

		if !isValidChannel(channel) {
			fmt.Println(ui.Error("[ERROR] Invalid channel. Use one of: stable, beta, alpha"))
			os.Exit(1)
		}

		store, err := storage.New()
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] %v", err)))
			os.Exit(1)
		}

		config, err := store.Load(appID)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] failed to load app metadata: %v", err)))
			os.Exit(1)
		}

		if normalizeChannelInput(config.Channel) == channel {
			fmt.Println(ui.Success(fmt.Sprintf("[OK] Channel for %s is already %s", config.AppID, channel)))
			return
		}

		config.Channel = channel
		if err := store.Save(config.AppID, config); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("[ERROR] failed to update channel: %v", err)))
			os.Exit(1)
		}

		fmt.Println(ui.Success(fmt.Sprintf("[OK] Channel for %s set to %s", config.AppID, channel)))
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
