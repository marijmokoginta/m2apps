package process

import (
	"encoding/json"
	"fmt"
	"m2apps/internal/system"
	"os"
	"path/filepath"
	"strings"
)

type Registry struct {
	filePath string
}

func NewRegistry() *Registry {
	return &Registry{
		filePath: filepath.Join(system.GetBaseDir(), "processes.json"),
	}
}

func (r *Registry) LoadAll() ([]AppProcesses, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AppProcesses{}, nil
		}
		return nil, fmt.Errorf("failed to read process registry: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return []AppProcesses{}, nil
	}

	var apps []AppProcesses
	if err := json.Unmarshal(data, &apps); err != nil {
		return nil, fmt.Errorf("failed to parse process registry: %w", err)
	}
	return apps, nil
}

func (r *Registry) SaveAll(apps []AppProcesses) error {
	if err := os.MkdirAll(system.GetBaseDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	data, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize process registry: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write process registry: %w", err)
	}
	return nil
}

func (r *Registry) Get(appID string) (AppProcesses, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return AppProcesses{}, fmt.Errorf("app_id is required")
	}

	apps, err := r.LoadAll()
	if err != nil {
		return AppProcesses{}, err
	}

	for _, entry := range apps {
		if strings.TrimSpace(entry.AppID) == id {
			return normalizeEntry(entry, id), nil
		}
	}

	return AppProcesses{
		AppID:     id,
		Processes: []Process{},
	}, nil
}

func (r *Registry) Set(appID string, processes []Process) error {
	id := strings.TrimSpace(appID)
	if id == "" {
		return fmt.Errorf("app_id is required")
	}

	apps, err := r.LoadAll()
	if err != nil {
		return err
	}

	entry := AppProcesses{
		AppID:     id,
		Processes: normalizeProcesses(processes),
	}

	for i, existing := range apps {
		if strings.TrimSpace(existing.AppID) == id {
			apps[i] = entry
			return r.SaveAll(apps)
		}
	}

	apps = append(apps, entry)
	return r.SaveAll(apps)
}

func normalizeEntry(entry AppProcesses, appID string) AppProcesses {
	return AppProcesses{
		AppID:     appID,
		Processes: normalizeProcesses(entry.Processes),
	}
}

func normalizeProcesses(processes []Process) []Process {
	if len(processes) == 0 {
		return []Process{}
	}

	out := make([]Process, 0, len(processes))
	for _, process := range processes {
		name := strings.TrimSpace(process.Name)
		if name == "" {
			name = "process"
		}

		command := make([]string, 0, len(process.Command))
		for _, part := range process.Command {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			command = append(command, part)
		}

		status := strings.TrimSpace(strings.ToLower(process.Status))
		if status == "" {
			status = "stopped"
		}

		out = append(out, Process{
			Name:    name,
			PID:     process.PID,
			Port:    normalizePort(process.Port),
			Command: command,
			Status:  status,
		})
	}

	return out
}

func normalizePort(port int) int {
	if port < 0 {
		return 0
	}
	return port
}
