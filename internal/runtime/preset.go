package runtime

import (
	"fmt"
	"strings"
)

type ProcessCommand struct {
	Name    string
	Command []string
}

var presetCommands = map[string][]ProcessCommand{
	"laravel": {
		{Name: "web", Command: []string{"php", "artisan", "serve", "--host=127.0.0.1", "--port={PORT}"}},
		{Name: "queue", Command: []string{"php", "artisan", "queue:work"}},
		{Name: "scheduler", Command: []string{"php", "artisan", "schedule:work"}},
	},
	"laravel-inertia": {
		{Name: "web", Command: []string{"php", "artisan", "serve", "--host=127.0.0.1", "--port={PORT}"}},
		{Name: "queue", Command: []string{"php", "artisan", "queue:work"}},
		{Name: "scheduler", Command: []string{"php", "artisan", "schedule:work"}},
	},
	"node": {
		{Name: "app", Command: []string{"npm", "run", "start"}},
	},
	"nodejs": {
		{Name: "app", Command: []string{"npm", "run", "start"}},
	},
}

var defaultPorts = map[string]int{
	"laravel":         8000,
	"laravel-inertia": 8000,
	"node":            3000,
	"nodejs":          3000,
	"nextjs":          3000,
	"flutter":         5000,
}

func DefaultPort(name string) int {
	key := strings.TrimSpace(strings.ToLower(name))
	if key == "" {
		return 0
	}

	port, ok := defaultPorts[key]
	if !ok {
		return 0
	}
	return port
}

func LoadPreset(name string) ([]ProcessCommand, error) {
	key := strings.TrimSpace(strings.ToLower(name))
	if key == "" {
		return nil, fmt.Errorf("runtime preset is required")
	}

	commands, ok := presetCommands[key]
	if !ok {
		return nil, fmt.Errorf("runtime preset %q is not supported", name)
	}

	out := make([]ProcessCommand, 0, len(commands))
	for _, process := range commands {
		if strings.TrimSpace(process.Name) == "" || len(process.Command) == 0 {
			continue
		}

		command := make([]string, 0, len(process.Command))
		for _, part := range process.Command {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			command = append(command, part)
		}
		if len(command) == 0 {
			continue
		}

		out = append(out, ProcessCommand{
			Name:    process.Name,
			Command: command,
		})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("runtime preset %q has no executable process commands", name)
	}

	return out, nil
}
