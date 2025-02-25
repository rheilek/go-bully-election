package bully

import (
	"errors"
	"time"
)

var ErrShutdown = errors.New("peer is down")

type testPeer struct {
	id       string
	rank     int
	election Election
	shutdown bool
	alive    time.Time
}

func (p *testPeer) Id() string {
	return p.id
}

func (p *testPeer) Rank() int {
	return p.rank
}

func (p *testPeer) Shutdown() {
	p.shutdown = true
}

func (p *testPeer) Start() {
	p.shutdown = false
}

func (p *testPeer) Elect(from Peer) error {
	if p.shutdown {
		return ErrShutdown
	}
	p.election.HandleElect(from)
	return nil
}

func (p *testPeer) Ping(id string, from Peer) error {
	if p.shutdown {
		return ErrShutdown
	}
	p.election.HandlePing(id, from)
	return nil
}

func (p *testPeer) Elected(id string, leader Peer) error {
	if p.shutdown {
		return ErrShutdown
	}
	p.election.HandleElected(id, leader)
	return nil
}
