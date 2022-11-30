package raft

import (
	"time"
	"math/rand"
	"fmt"
	"log"
)
const DebugRaft = 1 // Debug the state of the raft engine

func randRange(min, max int) time.Duration {
	rand.Seed(time.Now().UnixNano())

	return  time.Duration(rand.Intn(max - min) + min)
}

func millisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// dlog is a debug log function
func (rf *RaftEngine) dlog(format string, args ...interface{}) {
	if DebugRaft > 0 {
		format = fmt.Sprintf("[%d] ", rf.node.ID) + format
		log.Printf(format, args...)
	}
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}