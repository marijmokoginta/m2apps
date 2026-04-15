// Copyright (c) M2CodeApps
// Author: Marij Mokoginta
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func confirmAction(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", strings.TrimSpace(prompt))
	answer, _ := reader.ReadString('\n')
	normalized := strings.ToLower(strings.TrimSpace(answer))
	return normalized == "y" || normalized == "yes"
}
