package raft

type NState uint

type LogEntry struct {
	Command interface{}
	Term    int
}

type CommitEntry struct {
	Command interface{}

	Index int

	Term int
}

const (
	Follower NState = iota
	Candidate
	Minter
	Dead
)

type RequestVoteArgs struct {
	Term	     int
	CandidateID  int

	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term         int
	MinterID     int

	PrevLogIndex int
	PrevLogTerm  int
	Entries 	 []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}
