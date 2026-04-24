package preset

import (
	"fmt"
	"m2apps/internal/logger"
	"m2apps/internal/system"
	"m2apps/internal/ui"
	"os"
	"path/filepath"
	"strings"
)

type smartStep struct {
	Command   string
	ShouldRun func(workDir string) (bool, string, error)
	BeforeRun func(workDir string) error
}

type presetHandler struct {
	Install        []smartStep
	Update         []smartStep
	PostUpdate     []string
	OnAppURLChange []string
	RestartTargets []string
}

var presetHandlers = map[string]presetHandler{
	"laravel":         newLaravelPresetHandler(),
	"laravel-inertia": newLaravelPresetHandler(),
}

func RunInstallPreset(name, workDir string) error {
	handler, ok := presetHandlers[normalizePreset(name)]
	if !ok {
		steps, err := GetPreset(name)
		if err != nil {
			return err
		}
		return RunSteps(steps, workDir)
	}

	total := len(handler.Install)
	for i, step := range handler.Install {
		if step.BeforeRun != nil {
			if err := step.BeforeRun(workDir); err != nil {
				return err
			}
		}

		if step.ShouldRun != nil {
			shouldRun, reason, err := step.ShouldRun(workDir)
			if err != nil {
				return err
			}
			if !shouldRun {
				msg := strings.TrimSpace(reason)
				if msg == "" {
					msg = "already satisfied"
				}
				fmt.Println(ui.Info(fmt.Sprintf("[SKIP] %s (%s)", step.Command, msg)))
				continue
			}
		}

		if err := runCommandStep(workDir, step.Command, i+1, total); err != nil {
			return err
		}
	}

	return nil
}

func RunUpdatePreset(name, workDir string) error {
	handler, ok := presetHandlers[normalizePreset(name)]
	if !ok {
		steps, err := GetPreset(name)
		if err != nil {
			return err
		}
		return RunSteps(steps, workDir)
	}

	if len(handler.Update) == 0 {
		return nil
	}

	total := len(handler.Update)
	for i, step := range handler.Update {
		if step.BeforeRun != nil {
			if err := step.BeforeRun(workDir); err != nil {
				return err
			}
		}

		if step.ShouldRun != nil {
			shouldRun, reason, err := step.ShouldRun(workDir)
			if err != nil {
				return err
			}
			if !shouldRun {
				msg := strings.TrimSpace(reason)
				if msg == "" {
					msg = "already satisfied"
				}
				fmt.Println(ui.Info(fmt.Sprintf("[SKIP] %s (%s)", step.Command, msg)))
				continue
			}
		}

		if err := runCommandStep(workDir, step.Command, i+1, total); err != nil {
			return err
		}
	}

	return nil
}

func RunPostUpdate(name, workDir string) error {
	handler, ok := presetHandlers[normalizePreset(name)]
	if !ok || len(handler.PostUpdate) == 0 {
		return nil
	}

	for i, command := range handler.PostUpdate {
		if err := runCommandStep(workDir, command, i+1, len(handler.PostUpdate)); err != nil {
			return err
		}
	}
	return nil
}

func RunOnAppURLChange(name, workDir string) error {
	handler, ok := presetHandlers[normalizePreset(name)]
	if !ok || len(handler.OnAppURLChange) == 0 {
		return nil
	}

	for i, command := range handler.OnAppURLChange {
		if err := runCommandStep(workDir, command, i+1, len(handler.OnAppURLChange)); err != nil {
			return err
		}
	}
	return nil
}

func RestartProcessTargets(name string) []string {
	handler, ok := presetHandlers[normalizePreset(name)]
	if !ok || len(handler.RestartTargets) == 0 {
		return nil
	}
	out := make([]string, 0, len(handler.RestartTargets))
	for _, target := range handler.RestartTargets {
		t := strings.TrimSpace(target)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func newLaravelPresetHandler() presetHandler {
	return presetHandler{
		Install: []smartStep{
			{
				Command: "composer install",
				ShouldRun: func(workDir string) (bool, string, error) {
					ok, err := pathExists(filepath.Join(workDir, "vendor"))
					if err != nil {
						return false, "", err
					}
					if ok {
						return false, "vendor directory already exists", nil
					}
					return true, "", nil
				},
			},
			{
				Command:   "php artisan key:generate",
				BeforeRun: ensureLaravelEnvFile,
			},
			{Command: "php artisan migrate --force"},
			{
				Command: "npm install",
				ShouldRun: func(workDir string) (bool, string, error) {
					ok, err := pathExists(filepath.Join(workDir, "public", "build"))
					if err != nil {
						return false, "", err
					}
					if ok {
						return false, "public/build already exists", nil
					}
					return true, "", nil
				},
			},
			{
				Command: "npm run build",
				ShouldRun: func(workDir string) (bool, string, error) {
					ok, err := pathExists(filepath.Join(workDir, "public", "build"))
					if err != nil {
						return false, "", err
					}
					if ok {
						return false, "public/build already exists", nil
					}
					return true, "", nil
				},
			},
			{
				Command: "php artisan storage:link",
				ShouldRun: func(workDir string) (bool, string, error) {
					ok, err := pathExists(filepath.Join(workDir, "public", "storage"))
					if err != nil {
						return false, "", err
					}
					if ok {
						return false, "public/storage already exists", nil
					}
					return true, "", nil
				},
			},
			{Command: "php artisan optimize:clear"},
		},
		Update: []smartStep{
			{
				Command: "composer install",
				ShouldRun: func(workDir string) (bool, string, error) {
					ok, err := pathExists(filepath.Join(workDir, "vendor"))
					if err != nil {
						return false, "", err
					}
					if ok {
						return false, "vendor directory already exists", nil
					}
					return true, "", nil
				},
			},
			{
				Command:   "php artisan migrate --force",
				BeforeRun: ensureLaravelEnvFile,
			},
			{
				Command: "npm install",
				ShouldRun: func(workDir string) (bool, string, error) {
					ok, err := pathExists(filepath.Join(workDir, "public", "build"))
					if err != nil {
						return false, "", err
					}
					if ok {
						return false, "public/build already exists", nil
					}
					return true, "", nil
				},
			},
			{
				Command: "npm run build",
				ShouldRun: func(workDir string) (bool, string, error) {
					ok, err := pathExists(filepath.Join(workDir, "public", "build"))
					if err != nil {
						return false, "", err
					}
					if ok {
						return false, "public/build already exists", nil
					}
					return true, "", nil
				},
			},
		},
		PostUpdate:     []string{"php artisan storage:link", "php artisan optimize:clear"},
		OnAppURLChange: []string{"npm run build"},
		RestartTargets: []string{"queue", "scheduler"},
	}
}

func ensureLaravelEnvFile(workDir string) error {
	envPath := filepath.Join(workDir, ".env")
	ok, err := pathExists(envPath)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	examplePath := filepath.Join(workDir, ".env.example")
	exampleExists, err := pathExists(examplePath)
	if err != nil {
		return err
	}
	if !exampleExists {
		return fmt.Errorf("missing .env and .env.example in %s", workDir)
	}

	content, err := os.ReadFile(examplePath)
	if err != nil {
		return fmt.Errorf("failed to read .env.example: %w", err)
	}
	if err := os.WriteFile(envPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to create .env from .env.example: %w", err)
	}
	fmt.Println(ui.Info("[INFO] Created .env from .env.example"))
	return nil
}

func runCommandStep(workDir, command string, stepIndex, total int) error {
	commandLine := strings.TrimSpace(command)
	if commandLine == "" {
		return fmt.Errorf("empty command in step %d", stepIndex)
	}

	logWriter := logger.Writer()
	createdLogger := false
	if logWriter == nil {
		if err := logger.Init(); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		createdLogger = true
		logWriter = logger.Writer()
		if logWriter == nil {
			logger.Close()
			return fmt.Errorf("logger is not initialized")
		}
	}
	if createdLogger {
		defer logger.Close()
	}

	if _, err := logWriter.WriteString(fmt.Sprintf("\n=== Step [%d/%d]: %s ===\n", stepIndex, total, commandLine)); err != nil {
		return fmt.Errorf("failed to write step log: %w", err)
	}

	cmd := system.NewShellCommand(commandLine)
	cmd.Dir = workDir
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	cmd.Env = append(os.Environ(),
		"CI=true",
		"NPM_CONFIG_LOGLEVEL=silent",
		"NO_COLOR=1",
	)

	spinner := ui.NewSpinner()
	spinner.Start(fmt.Sprintf("[%d/%d] Running: %s", stepIndex, total, commandLine))
	if err := cmd.Run(); err != nil {
		spinner.Stop(ui.Error(fmt.Sprintf("[FAIL] %s", commandLine)))
		return fmt.Errorf("step failed: %s (see logs)", commandLine)
	}
	spinner.Stop(ui.Success(fmt.Sprintf("[OK] %s", commandLine)))
	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check path %s: %w", path, err)
}

func normalizePreset(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
