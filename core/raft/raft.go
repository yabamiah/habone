package raft

import (
	"sync"
	"time"

	"github.com/yabamiah/habone/rpc"
)

// Node is a node in the cluster
type Node struct {
	ID 	 int
	Port string
}

// Raft is a server that maintains a replicated log of commands.
type RaftEngine struct {
	node 			  Node // Node is the local node.
	peerNodes 		  []Node // PeerNodes is the remote nodes.

	mu 				  sync.Mutex // Lock to protect shared access to this peer's state
	server 			  *rpc.Server // Server is the server that handles the RPCs.

	//storage 		  *Storage // Storage is the storage engine.

	commitChan 		  chan<- CommitEntry  // CommitChan is the channel to send committed commands to the client.
	newCommitReadyChan chan struct{} // newCommitReadyChan is the channel to notify the client that a new commit is ready.

	triggerAEChan 	  chan struct{} // triggerAEChan is the channel to trigger an AppendEntries RPC.

	currentTerm 	  int // CurrentTerm is the current term.
	votedFor 		  int // VotedFor is the candidate that received the vote in the current term.
	log 			  []LogEntry // Log is the log sent by the leader.

	commitIndex 	  int // CommitIndex is the highest log entry that is known to be committed.
	lastApplied 	  int // LastApplied is the highest log entry that is applied to the state machine.
	state 			  NState // State is the current state of the node.
	electionResetTime time.Time // ElectionTimeout is the timeout for the election.

	nextIndex 		  map[int]int // NextIndex is the index of the next log entry to send to that server.
	matchIndex 		  map[int]int // MatchIndex is the index of the highest log entry known to be replicated on server.
}

func (rf *RaftEngine) runElectionTimer() {
	timeoutDuration := randRange(150, 300)
	rf.mu.Lock()
	termStart := rf.currentTerm
	rf.mu.Unlock()
	rf.dlog("Election timer started %v", timeoutDuration)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		<- ticker.C

		rf.mu.Lock()
		if rf.state != Candidate && rf.state != Follower {
			rf.dlog("Election timer stopped, state=%s", rf.state)
			rf.mu.Unlock()
			return
		}

		if termStart != rf.currentTerm {
			rf.dlog("Election timer stopped, term changed from %d to %d", termStart, rf.currentTerm)
			rf.mu.Unlock()
			return
		}

		if rfElectionTime := time.Since(rf.electionResetTime); rfElectionTime >= timeoutDuration {
			rf.startElection()
			rf.mu.Unlock()
			return
		}
		rf.mu.Unlock()
	}
}

func (rf *RaftEngine) startElection() {
	rf.state = Candidate
	rf.currentTerm++
	savedTerm := rf.currentTerm
	rf.electionResetTime = time.Now()
	rf.votedFor = rf.node.ID
	rf.dlog("Election started, term=%d", savedTerm)

	votosReceived := 1

	// Send RequestVote RPCs to all other servers
	for _, peerNode := range rf.peerNodes {
		nodeID := peerNode.ID
		go func(nodeID int) {
			rf.mu.Lock()
			savedLastLogIndex, savedLastLogTerm := rf.getLastLogIndexAndTerm()
			rf.mu.Unlock()

			args := RequestVoteArgs{
				Term:         savedTerm,
				CandidateID:  rf.node.ID,
				LastLogIndex: savedLastLogIndex,
				LastLogTerm:  savedLastLogTerm,
			}

			rf.dlog("Sending ResquestVote to %d: %+v", peerNode, args)
			var reply RequestVoteReply

			// Make RPC call
			if err := rf.server.Call(nodeID, "RaftEngine.RequestVote", args, &reply); err == nil {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				rf.dlog("Received ResquestVoteReply %+v", reply)

				if rf.state != Candidate {
					rf.dlog("While waiting for reply, state = %s", rf.state)
					return
				}

				if reply.Term > savedTerm {
					rf.dlog("term %d is out of date, updating to %d", savedTerm, reply.Term)
					rf.becomeFollower(reply.Term)
					return
				} else if reply.Term == savedTerm {
					if reply.VoteGranted {
						votosReceived ++
						if votosReceived*2 > len(rf.peerNodes)+1 {
							rf.dlog("Election won with %d votes", votosReceived)
							rf.becomeMinter()
							return
						}
					}
				}
			}
		}(nodeID)
	}
	go rf.runElectionTimer()
}

func (rf *RaftEngine) becomeMinter() {
	rf.state = Minter
	rf.dlog("Became minter, term=%d, log=%v", rf.currentTerm, rf.log)

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			rf.minterPulsar()
			<- ticker.C

			rf.mu.Lock()
			if rf.state != Minter {
				rf.mu.Unlock()
				return
			}
			rf.mu.Unlock()
		}
	}()
}

func (rf *RaftEngine) minterPulsar() {
	rf.mu.Lock()
	saveTerm := rf.currentTerm
	rf.mu.Unlock()

	for _, peerNode := range rf.peerNodes {
		nodeID := peerNode.ID
		args := AppendEntriesArgs {
			Term: saveTerm,
			MinterID: rf.node.ID,
		}
		go func(nodeID int) {
			rf.dlog("Sending Minter Pulsar to %v: ni=%d, args=%+v", nodeID, 0, args )
			var reply AppendEntriesReply
			if err := rf.server.Call(nodeID, "RaftEngine.AppendEntries", args, &reply); err == nil {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				
				if reply.Term > saveTerm {
					rf.dlog("term %d is out of date in pulsar reply")
					rf.becomeFollower(reply.Term)
					return
				}
			}
		}(nodeID)
	}
}

func (rf *RaftEngine) becomeFollower(term int) {
	rf.dlog("Becomes Follower with term=%d; log=%v", term, rf.log)
	rf.state = Follower
	rf.currentTerm = term
	rf.votedFor = -1
	rf.electionResetTime = time.Now()

	go rf.runElectionTimer()
}

func (rf *RaftEngine) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state == Dead {
		return nil
	}
	lastLogIndex, lastLogTerm := rf.getLastLogIndexAndTerm()
	rf.dlog("RequestVote: %+v [currentTerm=%d, votedFor=%d, log index/term=(%d, %d)]", args, rf.currentTerm, rf.votedFor, lastLogIndex, lastLogTerm)

	if args.Term > rf.currentTerm {
		rf.dlog("term %d is out of date in RequestVote, updating to %d", rf.currentTerm, args.Term)
		rf.becomeFollower(args.Term)
	}

	if rf.currentTerm == args.Term && 
		(rf.votedFor == -1 || rf.votedFor == args.CandidateID) &&
		(args.LastLogTerm > lastLogIndex ||
			(args.LastLogTerm == lastLogIndex && args.LastLogIndex >= lastLogIndex)) {
		reply.VoteGranted = true
		rf.votedFor = args.CandidateID
		rf.electionResetTime = time.Now()
	} else {
		reply.VoteGranted = false
	}

	reply.Term = rf.currentTerm
	rf.dlog("RequestVoteReply: %+v", reply)
	return nil
}

func (rf *RaftEngine) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state == Dead {
		return nil
	}
	rf.dlog("AppendEntries: %+v", args)

	if args.Term > rf.currentTerm {
		rf.dlog("term %d is out of date in AppendEntries, updating to %d", rf.currentTerm, args.Term)
		rf.becomeFollower(args.Term)
	}

	reply.Success = false
	if args.Term == rf.currentTerm {
		if rf.state != Follower {
			rf.becomeFollower(args.Term)
		}
		rf.electionResetTime = time.Now()
		
		if args.PrevLogIndex == -1 ||
	       (args.PrevLogIndex < len(rf.log) && args.PrevLogTerm == rf.log[args.PrevLogIndex].Term) {
			reply.Success = true

			logInsertIndex := args.PrevLogIndex + 1
			newEntriesIndex := 0

			for {
				if logInsertIndex >= len(rf.log) || newEntriesIndex >= len(args.Entries) {
					break
				}
				logInsertIndex++
				newEntriesIndex++
			}

			if args.LeaderCommit > rf.commitIndex {
				rf.commitIndex = Min(args.LeaderCommit, len(rf.log)-1)
				rf.dlog("Setting commitIndex to %d", rf.commitIndex)
				rf.newCommitReadyChan <- struct{}{}
			}
			}
		}

	reply.Term = rf.currentTerm
	rf.dlog("AppendEntriesReply: %+v", *reply)

	return nil
}

func (rf *RaftEngine) Submit(command interface{}) bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	rf.dlog("Submit received %v: %v", rf.state, command)
	if rf.state == Minter {
		rf.log = append(rf.log, LogEntry{Command: command, Term: rf.currentTerm} )
		rf.dlog("Submit: log=%v", rf.log)
		return true
	}
	return false
}

func (rf *RaftEngine) leaderSendHeartBeats() {
	rf.mu.Lock()
	saveTerm := rf.currentTerm
	rf.mu.Unlock()

	for _, peerNode := range rf.peerNodes {
		nodeID := peerNode.ID
		go func(nodeID int) {
			rf.mu.Lock()
			nextIndex := rf.nextIndex[nodeID]
			prevLogIndex := nextIndex - 1
			prevLogTerm := -1

			if prevLogIndex >= 0 {
				prevLogTerm = rf.log[prevLogIndex].Term
			}
			entries := rf.log[nextIndex:]

			args := AppendEntriesArgs {
				Term: saveTerm,
				MinterID: rf.node.ID,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm: prevLogTerm,
				Entries: entries,
				LeaderCommit: rf.commitIndex,
			}
			rf.mu.Unlock()
			rf.dlog("Sending AppendEntries to %v: ni=%d, args=%+v", nodeID, nextIndex, args )

			var reply AppendEntriesReply
			if err := rf.server.Call(nodeID, "RaftEngine.AppendEntries", args, &reply); err == nil {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				if reply.Term > saveTerm {
					rf.dlog("term %d is out of date in AppendEntries reply", saveTerm)
					rf.becomeFollower(reply.Term)
					return
				}

				if rf.state == Minter && saveTerm == reply.Term {
					if reply.Success {
						rf.nextIndex[nodeID] = nextIndex + len(entries)
						rf.matchIndex[nodeID] = rf.nextIndex[nodeID] - 1
						rf.dlog("AppendEntries success: ni=%d, mi=%d", rf.nextIndex[nodeID], rf.matchIndex[nodeID])

						savedCommitIndex := rf.commitIndex
						for i := rf.commitIndex + 1; i < len(rf.log); i++ {
							if rf.log[i].Term == rf.currentTerm {
								count := 1
								for _, peerNode := range rf.peerNodes {
									nodeID := peerNode.ID
									if rf.matchIndex[nodeID] >= i {
										count++
									}
								}
								if count*2 > len(rf.peerNodes) + 1 {
									rf.commitIndex = i
								}	
							}
						}
						if rf.commitIndex != savedCommitIndex {
							rf.dlog("Leader sets commitIndex to %d", rf.commitIndex)
							rf.newCommitReadyChan <- struct{}{}
						}
					} else {
						rf.nextIndex[nodeID] = nextIndex - 1
						rf.dlog("AppendEntries failed, decrementing nextIndex to %d", rf.nextIndex[nodeID])
					}
				}
			}	
		}(nodeID)
	}
}

func (rf *RaftEngine) commitChanSender() {
	for range rf.newCommitReadyChan {
		rf.mu.Lock()
		savedTerm := rf.currentTerm
		sevedLastApplied := rf.lastApplied
		var entries []LogEntry
		if rf.commitIndex > rf.lastApplied {
			entries = rf.log[rf.lastApplied+1 : rf.commitIndex+1]
			rf.lastApplied = rf.commitIndex
		}
		rf.mu.Unlock()
		rf.dlog("commitChanSender: lastApplied=%d entries=%v", sevedLastApplied, entries)

		for i, entry := range entries {
			rf.commitChan <- CommitEntry{
				Command: entry.Command,
				Index: sevedLastApplied + i + 1,
				Term: savedTerm,
			}
		}
	}
	rf.dlog("commitChanSender: exiting")
}

func (rf *RaftEngine) getLastLogIndexAndTerm() (int, int) {
	if len(rf.log) > 0 {
		lastIndex := len(rf.log) - 1
		return lastIndex, rf.log[lastIndex].Term
	} else {
		return -1, -1
	}
}