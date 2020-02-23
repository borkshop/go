package caskraft

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeNetwork map[string]*Election

func newFakeNetwork(t *testing.T, count int) fakeNetwork {
	fakeNetwork := make(fakeNetwork, count)
	testLogger := &testLogger{
		t:     t,
		start: time.Now(),
	}

	for i := 0; i < count; i++ {
		member := name(i)
		members := make([]string, 0, count-1)
		for j := 0; j < count; j++ {
			if j != i {
				members = append(members, name(j))
			}
		}
		fakeNetwork[member] = NewElection(member, members, 10, fakeNetwork, testLogger)
	}

	return fakeNetwork
}

func name(i int) string {
	if i <= 26 {
		return string('A' + i)
	}
	return strconv.Itoa(i)
}

func (n fakeNetwork) Start(ctx context.Context) error {
	for _, election := range n {
		_ = election.Start(ctx) // TODO error merging and abort
	}
	return nil
}

func (n fakeNetwork) Stop(ctx context.Context) error {
	for _, election := range n {
		_ = election.Stop(ctx) // TODO error merging and abort
	}
	return nil
}

func (n fakeNetwork) Send(message Message) {
	n[message.To].Handle(message)
}

type testLogger struct {
	t     *testing.T
	start time.Time
}

func (l *testLogger) Receive(member string, state State, message Message) {
	l.log(member, state, " <- %s", message)
}

func (l *testLogger) Drop(member string, state State, message Message) {
	l.log(member, state, " <- %s message dropped", message)
}

func (l *testLogger) Send(member string, state State, message Message) {
	l.log(member, state, " -> %s", message)
}

func (l *testLogger) Transition(member string, next, prior State) {
	l.log(member, next, "")
}

func (l *testLogger) Timeout(member string, state State, timeout time.Duration) {
	l.log(member, state, " sleep %s", timeout)
}

func (l *testLogger) Error(err Error) {
	l.log(err.Member, err.State, " %v", err)
	panic("protocol error")
}

func (l *testLogger) log(member string, state State, format string, rest ...interface{}) {
	l.t.Logf("%5d %s %s"+format+"\n", append([]interface{}{
		time.Since(l.start) / 1000000,
		state,
		member,
	}, rest...)...)
}

func TestRaft(t *testing.T) {
	network := newFakeNetwork(t, 3)
	network.Start(context.Background())
	time.Sleep(time.Second)
	network.Stop(context.Background())

	leaders := 0
	for _, member := range network {
		assert.Equal(t, 1, member.state.Term)
		if member.state.Type == Leader {
			leaders++
		}
	}
	assert.Equal(t, 1, leaders)
}
