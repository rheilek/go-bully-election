package bully

import (
	"fmt"
	"sync"
	"time"

	"log"
)

const (
	heartbeatInterval = 30 * time.Second
	heartbeatTimeout  = 1 * time.Minute
)

type Election interface {
	Id() string
	SetupElection(peers Peers, me Peer, shutdownCh chan struct{})
	GetLeader() Peer
	GetPeer(id string) Peer
	HandlePing(id string, from Peer)
	HandleElect(from Peer)
	HandleElected(id string, leader Peer)
}

func NewElection() Election {
	return &bullyElection{
		electProcess: &electProcess{wait: make(chan struct{})},
	}
}

type bullyElection struct {
	lock         sync.Mutex
	heartbeater  sync.Once
	electProcess *electProcess

	me     Peer
	peers  Peers
	leader Peer

	state struct {
		id    string
		lock  sync.Mutex
		peers map[Peer]peerState
	}
}

func (b *bullyElection) Id() string {
	return b.state.id
}

func (b *bullyElection) SetupElection(peers Peers, me Peer, shutdownCh chan struct{}) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if peers.equals(b.peers) && equals(me, b.me) {
		log.Println("same setup no election needed")
		return
	}

	b.peers = peers
	b.me = me

	b.state = struct {
		id    string
		lock  sync.Mutex
		peers map[Peer]peerState
	}{peers: make(map[Peer]peerState)}

	b.heartbeater.Do(func() {
		t := time.NewTicker(heartbeatInterval)
		go func() {
			for {
				select {
				case <-shutdownCh:
					t.Stop()
					return
				case <-t.C:
					b.heartbeat()
				}
			}
		}()
	})
	b.elect()
}

func (b *bullyElection) GetPeer(id string) Peer {
	for _, p := range b.peers {
		if p.Id() == id {
			return p
		}
	}
	return nil
}

func (b *bullyElection) GetLeader() Peer {
	log.Println("waiting for leader...")
	b.electProcess.Wait()
	log.Printf("leader is %q\n", b.leader.Id())
	return b.leader
}

func (b *bullyElection) setLeader(id string, leader Peer) {
	b.state.lock.Lock()
	defer b.state.lock.Unlock()

	log.Printf("[%v] set leader %q\n", id, leader.Id())
	b.state.id = id
	b.leader = leader
	b.electProcess.Done()
}

func (b *bullyElection) updateOnline(id string, p Peer) {
	b.state.lock.Lock()
	defer b.state.lock.Unlock()

	b.state.peers[p] = peerState{id: id, time: time.Now()}
}

func (b *bullyElection) isOnline(p Peer) bool {
	b.state.lock.Lock()
	defer b.state.lock.Unlock()

	state, ok := b.state.peers[p]
	if ok && b.state.id == state.id && state.time.Add(heartbeatTimeout).After(time.Now()) {
		return true
	}
	return false
}

func (b *bullyElection) elect() {
	log.Println("start")
	defer log.Println("end")

	b.electProcess.Restart()
	b.leader = nil

	for _, n := range b.peers {
		if n.Rank() > b.me.Rank() {
			err := n.Elect(b.me)
			if err != nil {
				log.Printf("peer %q not available: %v\n", n.Id(), err)
				continue
			}
			log.Printf("peer %q online, waiting for elected message...\n", n.Id())
			return
		}
	}

	b.state.lock.Lock()
	b.state.id = fmt.Sprintf("%d", time.Now().UnixNano()) // use UUID generator
	b.state.peers = make(map[Peer]peerState)
	b.state.lock.Unlock()

	log.Printf("[%v] become leader\n", b.state.id)
	b.setLeader(b.state.id, b.me)

	wg := sync.WaitGroup{}
	for _, n := range b.peers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := n.Elected(b.state.id, b.leader)
			if err != nil {
				log.Printf("[%v] peer %q not respond: %v\n", b.state.id, n.Id(), err)
				return
			}
			log.Printf("[%v] send elected to peer %q\n", b.state.id, n.Id())
			b.updateOnline(b.state.id, n)
		}()
	}
	wg.Wait()

}

func (b *bullyElection) heartbeat() {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.leader != nil {

		if b.me == b.leader {
			for _, p := range b.peers {
				if b.isOnline(p) {
					log.Printf("[%v] peer %q is online\n", b.state.id, p.Id())
					continue
				}
				err := p.Elected(b.state.id, b.leader)
				if err != nil {
					continue
				}
				log.Printf("[%v] send elected to peer %q\n", b.state.id, p.Id())
				b.updateOnline(b.state.id, p)
			}

			return
		}

		err := b.leader.Ping(b.state.id, b.me)
		if err != nil {
			log.Printf("[%v] leader %q not respond: %v\n", b.state.id, b.leader.Id(), err)
			b.elect()
			return
		}
	}
}

func (b *bullyElection) HandlePing(id string, from Peer) {
	b.updateOnline(id, from)
}

func (b *bullyElection) HandleElect(_ Peer) {
	// nop
}

func (b *bullyElection) HandleElected(id string, leader Peer) {
	b.setLeader(id, leader)
}
