package caskraft

import (
	"context"
	"math/rand"
	"runtime"
	"time"
)

// Network implementations enable an election to send messages to other peers.
//
// The network implementation does not need to block on a response.
type Network interface {
	Send(Message)
}

// Logger tracks log messages for an election.
type Logger interface {
	// Transition indicates a state transition.
	Transition(member string, next, prior State)
	// Send indicates that the member sent a message.
	Send(member string, state State, message Message)
	// Receive indicates that the member received a message.
	Receive(member string, state State, message Message)
	// Drop indicates that the member dropped a message due to a full message
	// buffer.
	Drop(member string, state State, message Message)
	// Timeout indicates that the member has set its next timeout.
	Timeout(member string, state State, timeout time.Duration)
	// Error indicates that the election passed through an invalid state or
	// received an invalid message.
	Error(Error)
}

// NewElection creates a new election.
func NewElection(member string, members []string, capacity int, network Network, logger Logger) *Election {
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	return &Election{
		member:    member,
		members:   members,
		network:   network,
		logger:    logger,
		stopping:  make(chan struct{}, 0),
		stopped:   make(chan struct{}, 0),
		messages:  make(chan Message, capacity),
		timer:     timer,
		timerRead: true,
	}
}

// Election is the logical core of a RAFT election.
type Election struct {
	// Member is the address of our own peer.
	member string
	// Members are the addresses of other peers.
	members []string
	// Network is a pluggable implementation of a network protocol for sending
	// messages to other peers.
	network Network
	// Logger is a pluggable implementation for logging messages.
	logger Logger

	state     State
	quorum    int
	messages  chan Message
	timer     *time.Timer
	timerRead bool
	start     time.Time
	stopping  chan struct{}
	stopped   chan struct{}
}

// Handle ingests a message from another member of the electorate.
func (e *Election) Handle(message Message) {
	select {
	case e.messages <- message:
		runtime.Gosched()
	default:
		e.logger.Drop(e.member, e.state, message)
	}
}

// Start begins running elections.
//
// Other methods must not be called until Start returns.
// Once Start returns, all other methods are safe to call concurrently.
func (e *Election) Start(ctx context.Context) error {
	e.start = time.Now()
	e.quorum = (len(e.members) + 3) / 2
	e.resetElectionTimer()
	go e.run()
	return nil
}

// Stop halts this election.
func (e *Election) Stop(ctx context.Context) error {
	close(e.stopping)
	<-e.stopped
	return nil
}

func (e *Election) run() {
Election:
	for {
		select {
		case message := <-e.messages:
			e.logger.Receive(e.member, e.state, message)

			if message.Term > e.state.Term {
				if message.Subject == Heartbeat {
					e.transition(State{Type: Follower, Term: message.Term, Leader: message.From})
					e.resetHeartbeatTimer()
				} else {
					e.transition(State{Type: Follower, Term: message.Term})
					e.resetElectionTimer()
				}
			}

			switch message.Subject {
			case RequestVote:
				if e.state.Leader == "" {
					e.transition(State{Type: Follower, Term: e.state.Term, Vote: message.From, NumVotes: e.state.NumVotes, Leader: e.state.Leader})
					e.send(Message{Subject: Vote, To: message.From, From: e.member, Term: e.state.Term})
				}

			case Vote:
				switch e.state.Type {
				case Follower:
					e.logger.Error(Error{Class: InvalidVote, Member: e.member, State: e.state, Message: message})
				case Candidate:
					// Receive vote
					voteCount := e.state.NumVotes + 1
					if voteCount == e.quorum {
						e.transition(State{Type: Leader, Term: e.state.Term, Leader: e.member})
						e.heartbeat()
						e.resetHeartbeatTimer()
					} else {
						e.transition(State{Type: Follower, Term: e.state.Term, Vote: e.state.Vote, NumVotes: voteCount})
					}
				}

			case Heartbeat:
				e.transition(State{Type: Follower, Term: message.Term, Leader: message.From})
				e.resetElectionTimer()

			default:
				e.logger.Error(Error{Class: InvalidSubject, Member: e.member, State: e.state, Message: message})

			}

		case <-e.timer.C:
			e.timerRead = true
			switch e.state.Type {
			case Candidate, Follower:
				// New election
				e.transition(State{Type: Candidate, Term: e.state.Term + 1, Vote: e.member, NumVotes: 1})
				for _, member := range e.members {
					e.send(Message{Subject: RequestVote, To: member, From: e.member, Term: e.state.Term})
				}
				e.resetElectionTimer()

			case Leader:
				e.heartbeat()
				e.resetHeartbeatTimer()

			default:
				e.logger.Error(Error{Class: InvalidState, Member: e.member, State: e.state})

			}

		case <-e.stopping:
			break Election
		}

		runtime.Gosched()
	}
	close(e.stopped)
}

func (e *Election) heartbeat() {
	for _, member := range e.members {
		e.send(Message{
			Subject: Heartbeat,
			From:    e.member,
			To:      member,
			Term:    e.state.Term,
		})
	}
}

func (e *Election) resetElectionTimer() {
	timeout := minElectionTimeout + time.Duration(rand.Int63n(int64(maxElectionTimeout-minElectionTimeout)))
	e.resetTimer(timeout)
}

func (e *Election) resetHeartbeatTimer() {
	e.resetTimer(heartbeatTimeout)
}

func (e *Election) resetTimer(timeout time.Duration) {
	if !e.timer.Stop() && !e.timerRead {
		<-e.timer.C
	}
	e.timerRead = false
	e.timer.Reset(timeout)
	e.logger.Timeout(e.member, e.state, timeout)
}

func (e *Election) transition(state State) {
	e.logger.Transition(e.member, state, e.state)
	e.state = state
}

func (e *Election) send(message Message) {
	e.logger.Send(e.member, e.state, message)
	e.network.Send(message)
}
