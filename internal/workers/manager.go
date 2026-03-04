package workers

import (
	"sync"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/logger"
)

type Manager struct {
	cfg     *config.Config
	workers map[string]chan struct{}
	mu      sync.Mutex
	stop    chan struct{}
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg:     cfg,
		workers: make(map[string]chan struct{}),
		stop:    make(chan struct{}),
	}
}

func (m *Manager) Start() {
	logger.Log.Info("[Worker] Starting Worker Manager...")

	// Start initial workers
	m.StartWorker("anidb", StartAniDBWorker)
	m.StartWorker("monitor", func(stopCh <-chan struct{}) {
		StartMonitorWorker(stopCh)
	})
}

func (m *Manager) Stop() {
	close(m.stop)
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, ch := range m.workers {
		logger.Log.Infof("[Worker] Stopping %s worker...", name)
		close(ch)
	}
}

// Reload stops then restarts a specific worker with new config/intervals
func (m *Manager) Reload(name string) {
	m.mu.Lock()
	if ch, ok := m.workers[name]; ok {
		close(ch)
		delete(m.workers, name)
	}
	m.mu.Unlock()

	// Logic for which worker to restart
	switch name {
	case "anidb":
		m.StartWorker("anidb", StartAniDBWorker)
	case "monitor":
		m.StartWorker("monitor", func(stopCh <-chan struct{}) {
			StartMonitorWorker(stopCh)
		})
	}
}

func (m *Manager) StartWorker(name string, startFunc func(<-chan struct{})) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stopCh := make(chan struct{})
	m.workers[name] = stopCh

	go startFunc(stopCh)
	logger.Log.Infof("[Worker] Started %s worker", name)
}
