package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

type Node struct {
	Id               int
	Queue            []int
	State            StateType
	ReceivingChannel chan Message
	AllChannels      []chan Message
	Num              *int
	PriorityQueue    []TimeStamp
	WaitingArray     []int
}

type Message struct {
	Sender    int
	Type      MessageType
	TimeStamp TimeStamp
}

type TimeStamp struct {
	Id   int
	Time time.Time
}

type MessageType int
type StateType int

const (
	Acquire MessageType = iota
	AcknowledgeAcquire
	Release
)

const (
	HasLock StateType = iota
	WaitingForReplies
	Idle
)

const (
	NUM_OF_NODES = 11
)

func main() {
	allChannels := make([]chan Message, NUM_OF_NODES)

	for i := 0; i < NUM_OF_NODES; i++ {
		allChannels[i] = make(chan Message, NUM_OF_NODES)
	}

	valueToAdd := 0

	for i := 0; i < NUM_OF_NODES; i++ {
		node := Node{
			Id:               i,
			Queue:            make([]int, 0),
			State:            Idle,
			ReceivingChannel: allChannels[i],
			AllChannels:      allChannels,
			Num:              &valueToAdd,
			PriorityQueue:    make([]TimeStamp, 0),
			WaitingArray:     make([]int, 0),
		}

		go node.start()
	}

	for {
	}
}

func (n *Node) start() {
	for {
		select {
		case m := <-n.ReceivingChannel:
			n.HandleRequest(m)
		default:
			if n.State == Idle {
				n.RandomLockRequest()
			}
		}
	}
}

func (n *Node) ExecuteCriticalSection(num *int) {
	*num += 1
}

func (n *Node) RandomLockRequest() {
	time.Sleep(time.Second * time.Duration(rand.Intn(3)))
	n.State = WaitingForReplies
	fmt.Printf("%v : requesting lock, waiting for replies\n", n.Id)
	m := Message{
		Sender:    n.Id,
		Type:      Acquire,
		TimeStamp: TimeStamp{n.Id, time.Now()},
	}

	for i := 0; i < cap(n.AllChannels); i++ {
		if i == n.Id {
			continue
		}
		n.AllChannels[i] <- m
	}
}

func (n *Node) HandleRequest(m Message) {
	switch m.Type {
	case Acquire:
		//check whether there are any earlier requests in the priority queue
		n.PriorityQueue = append(n.PriorityQueue, m.TimeStamp)
		if m.TimeStamp.IsEarliest(n.PriorityQueue) {
			//if earliest request in the queue, can reply
			fmt.Printf("%v : earliest request received, sending reply\n", n.Id)
			n.AllChannels[m.Sender] <- Message{
				Sender:    n.Id,
				Type:      AcknowledgeAcquire,
				TimeStamp: TimeStamp{n.Id, time.Now()},
			}
		}
		sort.Slice(n.PriorityQueue, func(i, j int) bool {
			return n.PriorityQueue[j].Time.After(n.PriorityQueue[i].Time)
		})

	case AcknowledgeAcquire:
		n.WaitingArray = append(n.WaitingArray, m.Sender)
		if len(n.WaitingArray) == cap(n.AllChannels)-1 {
			n.State = HasLock

			fmt.Printf(`-----------------------------------------------------------------------------------
----------%v : Has lock, executing critical section <Number to Add: %v>------------
-----------------------------------------------------------------------------------
`, n.Id, *n.Num)
			n.ExecuteCriticalSection(n.Num)
			n.WaitingArray = nil
			n.Release()
			fmt.Printf(`-----------------------------------------------------------------------------------
--------%v : Executed critical section <Number to Add: %v> Releasing lock----------
-----------------------------------------------------------------------------------
`, n.Id, *n.Num)
			n.State = Idle
		}
	case Release:
		n.HandleRelease(m)
	}
}

func (n *Node) HandleRelease(m Message) {
	//pop sender's timestamp off the priority queue
	for i := 0; i < len(n.PriorityQueue); i++ {
		if n.PriorityQueue[i].Id == m.Sender {
			n.PriorityQueue = append(n.PriorityQueue[:i], n.PriorityQueue[i+1:]...)
		}
	}

	//check next earliest request, and reply to that.
	stamp := n.PriorityQueue[0]
	if stamp.IsEarliest(n.PriorityQueue) && stamp.Id != n.Id {
		n.AllChannels[stamp.Id] <- Message{
			Sender:    n.Id,
			Type:      AcknowledgeAcquire,
			TimeStamp: TimeStamp{n.Id, time.Now()},
		}
	}
}

func (n *Node) Release() {
	for i := 0; i < cap(n.AllChannels); i++ {
		if i == n.Id {
			continue
		}
		n.AllChannels[i] <- Message{
			Sender:    n.Id,
			Type:      Release,
			TimeStamp: TimeStamp{n.Id, time.Now()},
		}
	}
}

//Checks whether current timestamp is earlier than all other timestamps in the array
func (t *TimeStamp) IsEarliest(tArray []TimeStamp) bool {
	for i := 0; i < len(tArray); i++ {
		if t.Time.After(tArray[i].Time) {
			return false
		}

		//tie breaker using Id
		if t.Time.Equal(tArray[i].Time) && t.Id > tArray[i].Id {
			return false
		}
	}
	return true
}
