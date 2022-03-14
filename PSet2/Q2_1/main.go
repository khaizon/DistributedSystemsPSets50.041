package main

import (
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

func main() {

}

func (n *Node) start() {
	//1. node send to all other nodes to ask for lock
	//2. node performs check on whether they can reply
	//3. node waits for all replies until can execute critical section
	for {
		select {
		case m := <-n.ReceivingChannel:
			n.HandleRequest(m)
		default:
			if n.State == Idle || n.State == HasLock {
				n.RandomLockRequest()
			}
		}
	}
}

func (n *Node) ExecuteCriticalSection(num *int) {
	*num += 1
}

func (n *Node) RandomLockRequest() {
	time.Sleep(time.Second * time.Duration(rand.Intn(5)))
	n.State = WaitingForReplies
	m := Message{
		Sender: n.Id,
		Type:   Acquire,
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
			n.AllChannels[m.Sender] <- Message{
				Sender:    n.Id,
				Type:      AcknowledgeAcquire,
				TimeStamp: TimeStamp{n.Id, time.Now()},
			}
		}
		//sort the priority queue
		sort.Slice(n.PriorityQueue, func(i, j int) bool {
			return n.PriorityQueue[j].Time.After(n.PriorityQueue[i].Time)
		})

	case AcknowledgeAcquire:
		n.WaitingArray = append(n.WaitingArray, m.Sender)
		if len(n.WaitingArray) == cap(n.AllChannels)-1 {
			n.State = HasLock
			go n.ExecuteCriticalSection(n.Num)
			n.WaitingArray = nil
			n.Release()
		}
	case Release:
		//pop sender's timestamp off the priority queue
		for i := 0; i < len(n.PriorityQueue); i++ {
			if n.PriorityQueue[i].Id == m.Sender {
				n.PriorityQueue = append(n.PriorityQueue[:i], n.PriorityQueue[i+1:]...)
			}
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
	for i := 0; i < cap(tArray); i++ {
		if t.Time.After(tArray[i].Time) {
			return false
		}
	}
	return true
}
