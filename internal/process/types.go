package process

type Process struct {
	Name    string   `json:"name"`
	PID     int      `json:"pid"`
	Port    int      `json:"port"`
	Command []string `json:"command"`
	Status  string   `json:"status"`
}

type AppProcesses struct {
	AppID     string    `json:"app_id"`
	Processes []Process `json:"processes"`
}
