package cmd

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const colorReset = "\033[0m"

func rgb(r, g, b int) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func gradientLine(s string, sr, sg, sb, er, eg, eb int) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return colorReset
	}
	denom := n - 1
	if denom < 1 {
		denom = 1
	}
	var buf strings.Builder
	buf.Grow(n * 20)
	for i, ch := range runes {
		t := float64(i) / float64(denom)
		r := clamp(int(math.Round(float64(sr)+t*float64(er-sr))), 0, 255)
		g := clamp(int(math.Round(float64(sg)+t*float64(eg-sg))), 0, 255)
		b := clamp(int(math.Round(float64(sb)+t*float64(eb-sb))), 0, 255)
		buf.WriteString(rgb(r, g, b))
		buf.WriteRune(ch)
	}
	buf.WriteString(colorReset)
	return buf.String()
}

func gradientLineShifted(s string, phase float64) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return colorReset
	}
	denom := n - 1
	if denom < 1 {
		denom = 1
	}
	var buf strings.Builder
	buf.Grow(n * 20)
	for i, ch := range runes {
		raw := float64(i)/float64(denom) + phase*0.35
		t := math.Mod(math.Abs(raw), 2.0)
		if t > 1.0 {
			t = 2.0 - t
		}

		var r, g, b int
		if t < 0.5 {
			u := t / 0.5
			r = clamp(int(math.Round(0+u*120)), 0, 255)
			g = clamp(int(math.Round(220+u*(80-220))), 0, 255)
			b = clamp(int(math.Round(255+u*(255-255))), 0, 255)
		} else {
			u := (t - 0.5) / 0.5
			r = clamp(int(math.Round(120+u*(255-120))), 0, 255)
			g = clamp(int(math.Round(80+u*(50-80))), 0, 255)
			b = clamp(int(math.Round(255+u*(200-255))), 0, 255)
		}
		buf.WriteString(rgb(r, g, b))
		buf.WriteRune(ch)
	}
	buf.WriteString(colorReset)
	return buf.String()
}

var bannerLines = []string{
	` __  __ ___   ____          _        _                      `,
	`|  \/  |__ \ / ___|___   __| | ___  / \   _ __  _ __  ___  `,
	`| |\/| | / /| |   / _ \ / _` + "`" + `/ _ \/ _ \ | '_ \| '_ \/ __| `,
	`| |  | |/ /_| |__| (_) | (_| |  __/ ___ \| |_) | |_) \__ \ `,
	`|_|  |_|____|\____\___/ \__,_|\___/_/   \_\ .__/| .__/|___/ `,
	`                                          |_|   |_|         `,
}

func printBanner() {
	totalLines := float64(len(bannerLines) - 1)
	for i, line := range bannerLines {
		phase := float64(i) / totalLines
		fmt.Println(gradientLineShifted(line, phase))
	}

	subtitle := "  Auto Updater Engine"
	fmt.Println(gradientLine(subtitle, 255, 220, 0, 255, 140, 0))

	author := "  by Marij Mokoginta"
	fmt.Println(gradientLine(author, 255, 200, 180, 255, 100, 80))

	fmt.Println()
}

var rootCmd = &cobra.Command{
	Use:   "m2apps",
	Short: "M2Apps CLI",
	Run: func(cmd *cobra.Command, args []string) {
		printBanner()
		_ = cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(listCmd)
}
