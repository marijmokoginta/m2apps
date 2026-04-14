package progress

import (
	"sync"
)

type Manager struct {
	mu    sync.Mutex
	items map[string]*Progress
}

func NewManager() *Manager {
	return &Manager{
		items: make(map[string]*Progress),
	}
}

func (m *Manager) Start(appID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[appID] = &Progress{
		AppID:   appID,
		Phase:   "init",
		Step:    "starting",
		Percent: 0,
		Logs:    []string{},
		Status:  "running",
	}
}

func (m *Manager) Update(appID, phase, step string, percent int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := m.ensure(appID)
	p.Phase = phase
	p.Step = step
	p.Percent = clampPercent(percent)
}

func (m *Manager) Log(appID, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := m.ensure(appID)
	p.Logs = append(p.Logs, message)
}

func (m *Manager) Complete(appID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := m.ensure(appID)
	p.Percent = 100
	p.Status = "completed"
	p.Step = "done"
}

func (m *Manager) Fail(appID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := m.ensure(appID)
	p.Status = "failed"
}

func (m *Manager) Get(appID string) (Progress, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.items[appID]
	if !ok {
		return Progress{}, false
	}
	cp := *p
	cp.Logs = append([]string{}, p.Logs...)
	return cp, true
}

func (m *Manager) ensure(appID string) *Progress {
	p, ok := m.items[appID]
	if ok {
		return p
	}
	p = &Progress{
		AppID:   appID,
		Phase:   "init",
		Step:    "starting",
		Percent: 0,
		Logs:    []string{},
		Status:  "running",
	}
	m.items[appID] = p
	return p
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

var defaultManager = NewManager()

func DefaultManager() *Manager {
	return defaultManager
}
