package monitor

import (
	"sync"

	"gitee.com/kolonse_zhjsh/gpbx/datacenter"
)

func (n *Monitor) data_clear() {
	datacenter.GetTaskManager().RemoveCompletedCall()
	datacenter.GetGlobalSessionManager().RemoveCompletedCall()
	datacenter.GetSessionManager().RemoveCompletedCall()
	jobs.run()
}

type Jobs struct {
	jobs []func()
	mu   sync.Mutex
}

func (p *Jobs) register(f func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jobs = append(p.jobs, f)
}

func (p *Jobs) run() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, f := range p.jobs {
		f()
	}
}

var jobs = &Jobs{}

func Register(f func()) {
	jobs.register(f)
}
