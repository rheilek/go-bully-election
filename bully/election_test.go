package bully

import (
	"testing"
)

func TestElection(t *testing.T) {

	shutdown := make(chan struct{})

	node1Election := NewElection()
	node1 := &testPeer{election: node1Election, id: "node1", rank: 1}

	node2Election := NewElection()
	node2 := &testPeer{election: node2Election, id: "node2", rank: 2}

	node3Election := NewElection()
	node3 := &testPeer{election: node3Election, id: "node3", rank: 3}

	node1.Shutdown()
	node3.Shutdown()

	node2Election.SetupElection(Peers{node1, node3}, node2, shutdown)
	assertEqual(t, node2Election.GetLeader().Id(), "node2")

	node3.Start()
	node3Election.SetupElection(Peers{node1, node2}, node3, shutdown)
	assertEqual(t, node2Election.GetLeader().Id(), "node3")
	assertEqual(t, node3Election.GetLeader().Id(), "node3")

	node1.Start()
	node1Election.SetupElection(Peers{node2, node3}, node1, shutdown)
	node3Election.(*bullyElection).heartbeat() // elect new peers
	assertEqual(t, node1Election.GetLeader().Id(), "node3")

	node3.Shutdown()
	node2.Shutdown()

	node1Election.(*bullyElection).heartbeat() // restart election
	assertEqual(t, node1Election.GetLeader().Id(), "node1")
}

func assertEqual[V comparable](t *testing.T, got, expected V) {
	t.Helper()
	if expected != got {
		t.Errorf("got: %v, expected: %v", got, expected)
		t.FailNow()
	}
}
