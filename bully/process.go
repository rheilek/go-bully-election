package bully

import "sync"

type electProcess struct {
	lock sync.Mutex
	wait chan struct{}
	done bool
}

func (p *electProcess) Restart() {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.done {
		p.wait = make(chan struct{})
		p.done = false
	}
}

func (p *electProcess) Done() {
	p.lock.Lock()
	defer p.lock.Unlock()

	if !p.done {
		close(p.wait)
		p.done = true
	}

}

func (p *electProcess) Wait() {
	<-p.wait
}
