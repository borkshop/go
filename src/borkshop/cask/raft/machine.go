package caskraft

import (
	"fmt"
	"time"
)

const (
	minElectionTimeout = 150 * time.Millisecond
	maxElectionTimeout = 300 * time.Millisecond
	heartbeatTimeout   = 100 * time.Millisecond
)

// StateType is one of "follower", "candidate", and "leader" in the RAFT
// algorithm.
type StateType int

const (
	// Follower indicates that the elector is waiting to receive vote requests
	// or heartbeats from the current leader until the election times out.
	Follower StateType = iota
	// Candidate indicates that the elector is waiting to receive votes
	// back from potential followers.
	Candidate
	// Leader indicates that the leader is sending heartbeats to assert its
	// leadership over other members.
	Leader
)

func (t StateType) String() string {
	switch t {
	case Follower:
		return "follow"
	case Candidate:
		return "candat"
	case Leader:
		return "LEADER"
	}
	return "unknow"
}

// State represents a state of the RAFT algorithm.
type State struct {
	// Type is one of follower, candidate, or leader.
	Type StateType
	// Term is the election term.
	Term int
	// Leader is the current leader.
	Leader string
	// Vote is the member this node on the network votes for.
	Vote string
	// NumVotes is the number of votes this node has received in this election
	// term.
	NumVotes int
}

// String returns a representation of the state.
func (s State) String() string {
	return fmt.Sprintf("[%s term:%d vote:%1s leader:%1s votes:%d]", s.Type, s.Term, s.Vote, s.Leader, s.NumVotes)
}

// Subject identifies the type of message sent between members of the
// electorate.
type Subject int

const (
	// RequestVote is the subject of a vote request message.
	RequestVote Subject = iota
	// Vote is the subject of a vote message.
	Vote
	// Heartbeat is the subject of a heartbeat message.
	Heartbeat
)

func (t Subject) String() string {
	switch t {
	case RequestVote:
		return "plea"
	case Vote:
		return "vote"
	case Heartbeat:
		return "poll"
	}
	return "unkn"
}

// Message represents a message between members of the electorate.
type Message struct {
	// Type is one of request vote, vote, or heartbeat.
	Subject Subject
	// To is the member that this message is sent to.
	To string
	// From is the member the message pertains to.
	//
	// For request vote, it is the candidate.
	// For vote, it is the voter.
	// For heartbeat, it is the leader.
	From string
	// Term is the election term that the sender recognized when sending the
	// message.
	Term int
}

// String returns a representation of the message.
func (m Message) String() string {
	return fmt.Sprintf("[to:%s subject:%s from:%s term:%d]", m.To, m.Subject, m.From, m.Term)
}
