package rpc

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/yabamiah/habone/raft"
)

type Server struct {
	mu 	sync.Mutex

	serverID 	int
	peerIDs 	[]int

	rfEngine 	*raft.RaftEngine
	storage 	Storage
	rpcProxy 	*RPCProxy

	rpcServer 	*rpc.Server
	listener 	net.Listener

	commitChan 	chan<- raft.CommitEntry
	peerClients map[int]*rpc.Client

	ready 		<-chan interface{}
	quit 	 	chan interface{}
	wg 			sync.WaitGroup
}