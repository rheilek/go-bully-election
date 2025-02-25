package bully

import (
	"time"
)

const ()

type Peer interface {
	Id() string
	Rank() int
	Ping(id string, from Peer) error
	Elect(from Peer) error
	Elected(id string, leader Peer) error
}

func equals(p1, p2 Peer) bool {
	if p1 == nil && p2 == nil {
		return true
	}
	if p1 != nil && p2 == nil {
		return false
	}
	if p2 != nil && p1 == nil {
		return false
	}
	return p1.Id() == p2.Id() && p1.Rank() == p2.Rank()
}

type Peers []Peer

func (peers Peers) equals(others Peers) bool {
	if len(peers) == len(others) {
		for i, p := range peers {
			if !equals(p, others[i]) {
				return false
			}
		}
		return true
	}
	return false
}

type peerState struct {
	id   string
	time time.Time
}
