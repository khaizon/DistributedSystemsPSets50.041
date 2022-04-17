package main

import (
	"fmt"
	"sync"
	"time"
)

type Node struct {
	Id            int
	AllChannels   []chan Message
	HasLock       bool
	Server        int
	PriorityQueue []Message
	Num           *int
	Requesting    bool
	Start         chan struct{}
	WaitGroup     *sync.WaitGroup
	ShouldRequest bool
}

type Message struct {
	Sender int
	Type   MessageType
}

type MessageType int

const (
	Acquire MessageType = iota
	Release
	LockGranted
	Kill
)

const (
	NUM_OF_NODES = 11
)

func main() {

	num := 0
	for j := 1; j < NUM_OF_NODES+1; j++ {
		var wg sync.WaitGroup
		var start = make(chan struct{}, 0)
		wg.Add(j)
		all_channels := make([]chan Message, NUM_OF_NODES)
		for i := 0; i < NUM_OF_NODES; i++ {
			all_channels[i] = make(chan Message, NUM_OF_NODES)
		}
		for i := 0; i < NUM_OF_NODES; i++ {

			node := Node{
				Id:            i,
				AllChannels:   all_channels,
				HasLock:       i == 0,
				Server:        0,
				Num:           &num,
				Requesting:    false,
				ShouldRequest: i < j,
				Start:         start,
				WaitGroup:     &wg,
			}
			go node.start()
		}
		close(start)
		t1 := time.Now().UnixMicro()
		wg.Wait()
		fmt.Printf("%v out of %v concurrent requests done\n", j, NUM_OF_NODES)
		t2 := time.Now().UnixMicro()
		for i := 0; i < NUM_OF_NODES; i++ {
			all_channels[i] <- Message{Type: Kill}
		}
		fmt.Printf("Number of Nodes: %v,    Time taken: %v\n", j, t2-t1)
		// time.Sleep(2 * time.Second)
	}
	var input string
	fmt.Scanln(&input)
}

func (n *Node) start() {
	for {
		select {
		case m := <-n.AllChannels[n.Id]:
			if m.Type == Kill {
				return
			}
			n.HandleMessage(m)
		default:
			if !n.Requesting && n.ShouldRequest {
				<-n.Start
				// fmt.Printf("%v : requesting\n", n.Id)
				n.AllChannels[n.Server] <- Message{Sender: n.Id, Type: Acquire}
				n.Requesting = true
			}
		}
	}
}

func (n *Node) HandleMessage(m Message) {
	switch m.Type {
	case Acquire:
		if n.HasLock {
			n.HasLock = false
			n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: LockGranted}
			break
		}
		n.PriorityQueue = append(n.PriorityQueue, m)
	case LockGranted:
		n.ExecuteCriticalSection(n.Num)
		n.WaitGroup.Done()

		// n.Requesting = false
		n.AllChannels[n.Server] <- Message{Sender: n.Id, Type: Release}
	case Release:
		n.HasLock = true
		if len(n.PriorityQueue) == 0 {
			break
		}
		queuedMessage := n.PriorityQueue[0]
		n.PriorityQueue = n.PriorityQueue[1:]
		n.HasLock = false
		n.AllChannels[queuedMessage.Sender] <- Message{Sender: n.Id, Type: LockGranted}
	}
}

func (n *Node) Acquire() {

}

func (n *Node) ExecuteCriticalSection(num *int) {
	// 	fmt.Printf(`-----------------------------------------------------------------------------------
	// ----------%v : Has lock, executing critical section <Number to Add: %v>-------------
	// -----------------------------------------------------------------------------------
	// `, n.Id, *num)
	*num += 1
	// 	fmt.Printf(`-----------------------------------------------------------------------------------
	// --------%v : Executed critical section <Number to Add: %v> Releasing lock-----------
	// -----------------------------------------------------------------------------------
	// `, n.Id, *num)
}
