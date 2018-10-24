package caskraft

// Error is a RAFT protocol error, indicating an unexpected message for a
// member's state.
type Error struct {
	Class   ErrorClass
	Member  string
	State   State
	Message Message
}

// ErrorClass indicates the type of a RAFT protocol error.
type ErrorClass int

const (
	// InvalidVote indicates that a vote was received for the current term but
	// the member was not in the candidate state, indicating they never sent
	// out vote requests.
	InvalidVote ErrorClass = iota
	// InvalidSubject indicates an unrecognized message subject.
	InvalidSubject
	// InvalidState indicates that the RAFT state machine is in an invalid
	// state.
	InvalidState
)

func (err Error) Error() string {
	switch err.Class {
	case InvalidVote:
		return "received a vote for this term but not in the candidate state"
	default:
		return "unrecognized raft error class"
	}
}
