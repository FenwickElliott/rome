package model

// State of node
type State uint8

const (
	Follower State = iota
	Candidate
	Leader
)

func (s State) String() string {
	return []string{"Follower", "Candidate", "Leader"}[s]
}
