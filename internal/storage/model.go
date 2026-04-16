package storage

type AppConfig struct {
	AppID       string `json:"app_id"`
	Name        string `json:"name"`
	InstallPath string `json:"install_path"`
	Repo        string `json:"repo"`
	Asset       string `json:"asset"`
	Token       string `json:"token"`
	APIToken    string `json:"api_token"`
	Version     string `json:"version"`
	Channel     string `json:"channel"`
	Preset      string `json:"preset"`
	ServerMode  string `json:"server_mode"`
	AutoStart   bool   `json:"auto_start"`
}

type Storage interface {
	Save(appID string, data AppConfig) error
	Load(appID string) (AppConfig, error)
	Exists(appID string) (bool, error)
}
