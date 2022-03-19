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
		allChannels[i] = make(chan Message, NUM_OF_NODES*10)
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
	requestTimeStamp := TimeStamp{n.Id, time.Now()}
	m := Message{
		Sender:    n.Id,
		Type:      Acquire,
		TimeStamp: requestTimeStamp,
	}
	n.PriorityQueue = append(n.PriorityQueue, requestTimeStamp)

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
		n.PriorityQueue = append(n.PriorityQueue, m.TimeStamp)
		//check whether there are any earlier requests in the priority queue
		//AND that current node is not in critical section (or does not have lock)
		if n.State != HasLock && m.TimeStamp.IsEarliest(n.PriorityQueue) {
			//if earliest request in the queue, can reply
			fmt.Printf("%v : request received, sending reply\n", n.Id)
			n.AllChannels[m.Sender] <- Message{
				Sender:    n.Id,
				Type:      AcknowledgeAcquire,
				TimeStamp: TimeStamp{n.Id, time.Now()},
			}
			n.PriorityQueue = RemoveTimeStamp(n.PriorityQueue, m.Sender)
		}
		sort.Slice(n.PriorityQueue, func(i, j int) bool {
			return n.PriorityQueue[j].Time.After(n.PriorityQueue[i].Time)
		})

	case AcknowledgeAcquire:
		n.WaitingArray = append(n.WaitingArray, m.Sender)
		sort.Ints(n.WaitingArray)
		if len(n.WaitingArray) == cap(n.AllChannels)-1 {
			n.State = HasLock

			fmt.Printf(`-----------------------------------------------------------------------------------
----------%v : Has lock, executing critical section <Number to Add: %v>------------
-----------------------------------------------------------------------------------
`, n.Id, *n.Num)
			n.ExecuteCriticalSection(n.Num)
			n.WaitingArray = nil
			fmt.Printf(`-----------------------------------------------------------------------------------
--------%v : Executed critical section <Number to Add: %v> Releasing lock----------
-----------------------------------------------------------------------------------
`, n.Id, *n.Num)
			n.ReleaseAndReply()
			n.State = Idle
		} else {
			fmt.Printf("%v : waiting for %v more replies\n", n.Id, cap(n.AllChannels)-1-len(n.WaitingArray))
			fmt.Printf("%v : waiting array %v \n", n.Id, n.WaitingArray)
		}
	}
}

func (n *Node) ReleaseAndReply() {
	//pop self's timestamp off the priority queue
	n.PriorityQueue = RemoveTimeStamp(n.PriorityQueue, n.Id)

	//reply to earliest all time stamps in the queue.
	temp := len(n.PriorityQueue)
	for i := 0; i < temp; i++ {
		stamp := n.PriorityQueue[0]
		if stamp.Id == n.Id {
			continue
		}
		n.AllChannels[stamp.Id] <- Message{
			Sender:    n.Id,
			Type:      AcknowledgeAcquire,
			TimeStamp: TimeStamp{n.Id, time.Now()},
		}
		n.PriorityQueue = RemoveTimeStamp(n.PriorityQueue, stamp.Id)
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

func RemoveTimeStamp(tArray []TimeStamp, id int) []TimeStamp {
	for i := 0; i < len(tArray); i++ {
		if tArray[i].Id == id {
			return append(tArray[:i], tArray[i+1:]...)
		}
	}
	return tArray
}