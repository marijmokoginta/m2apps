package config

type InstallConfig struct {
	AppID        string              `json:"app_id"`
	Name         string              `json:"name"`
	Source       SourceConfig        `json:"source"`
	Auth         AuthConfig          `json:"auth"`
	Channel      string              `json:"channel"`
	Preset       string              `json:"preset"`
	ServerMode   string              `json:"server_mode"`
	Requirements []RequirementConfig `json:"requirements"`
}

type SourceConfig struct {
	Type    string `json:"type"`
	Repo    string `json:"repo"`
	Version string `json:"version"`
	Asset   string `json:"asset"`
}

type AuthConfig struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type RequirementConfig struct {
	Type    string `json:"type"`
	Version string `json:"version"`
}
