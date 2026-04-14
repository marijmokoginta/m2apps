package progress

type Progress struct {
	AppID   string   `json:"app_id"`
	Phase   string   `json:"phase"`
	Step    string   `json:"step"`
	Percent int      `json:"percent"`
	Logs    []string `json:"logs"`
	Status  string   `json:"status"`
}
