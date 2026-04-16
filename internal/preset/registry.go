package preset

import "fmt"

type Step struct {
	Type string
	Run  string
}

var Presets = map[string][]Step{
	"laravel": {
		{Type: "command", Run: "composer install"},
		{Type: "command", Run: "php artisan key:generate"},
		{Type: "command", Run: "php artisan migrate --force"},
		{Type: "command", Run: "npm install"},
		{Type: "command", Run: "npm run build"},
		{Type: "command", Run: "php artisan storage:link"},
		{Type: "command", Run: "php artisan optimize:clear"},
	},
	"laravel-inertia": {
		{Type: "command", Run: "composer install"},
		{Type: "command", Run: "php artisan key:generate"},
		{Type: "command", Run: "php artisan migrate --force"},
		{Type: "command", Run: "npm install"},
		{Type: "command", Run: "npm run build"},
		{Type: "command", Run: "php artisan storage:link"},
		{Type: "command", Run: "php artisan optimize:clear"},
	},
}

func GetPreset(name string) ([]Step, error) {
	steps, ok := Presets[name]
	if !ok {
		return nil, fmt.Errorf("preset %q not found", name)
	}
	return steps, nil
}
