package blackboxadapter

import "sync"

type Registry struct {
	mu      sync.RWMutex
	runners map[string]ProbeRunner
}

func NewRegistry() *Registry {
	return &Registry{
		runners: make(map[string]ProbeRunner),
	}
}

func (r *Registry) Register(name string, runner ProbeRunner) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.runners[name] = runner
}

func (r *Registry) Get(name string) (ProbeRunner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	runner, ok := r.runners[name]
	return runner, ok
}
