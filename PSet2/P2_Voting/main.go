package main

import (
	"fmt"
	"math"
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
	RequestTimeStamp TimeStamp
	HasVote          bool
	LastCompleted    TimeStamp
}

type Message struct {
	Sender    int
	Type      MessageType
	TimeStamp TimeStamp
}

type TimeStamp struct {
	Id   int
	Time int64
}

type MessageType int
type StateType int

const (
	Acquire MessageType = iota
	Vote
	Release
	Rescind
)

const (
	Idle StateType = iota
	WaitingForvotes
	HasLock
)

const (
	NUM_OF_NODES = 31
)

var start = make(chan struct{})

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
			HasVote:          true,
		}

		go node.start()
	}
	close(start)

	var input string
	fmt.Scanln(&input)
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
	fmt.Printf(`-----------------------------------------------------------------------------------
----------%v : Has lock, executing critical section <Number to Add: %v>-------------
-----------------------------------------------------------------------------------
`, n.Id, *num)
	*num += 1
	fmt.Printf(`-----------------------------------------------------------------------------------
--------%v : Executed critical section <Number to Add: %v> Releasing lock-----------
-----------------------------------------------------------------------------------
`, n.Id, *num)
}

func (n *Node) RandomLockRequest() {
	// time.Sleep(time.Second * time.Duration(rand.Intn(3)))
	<-start
	n.State = WaitingForvotes
	fmt.Printf("%v : requesting lock, waiting for votes\n", n.Id)
	requestTimeStamp := TimeStamp{n.Id, time.Now().UnixNano()}
	n.RequestTimeStamp = requestTimeStamp
	m := Message{
		Sender:    n.Id,
		Type:      Acquire,
		TimeStamp: requestTimeStamp,
	}

	for i := 0; i < cap(n.AllChannels); i++ {
		n.AllChannels[i] <- m
	}
}

func (n *Node) HandleRequest(m Message) {
	switch m.Type {
	case Acquire:
		if len(n.PriorityQueue) == 0 && n.HasVote {
			//vote
			fmt.Printf("%v : voting on request with no one waiting for %v \n", n.Id, m.TimeStamp)
			n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: Vote, TimeStamp: m.TimeStamp}
			n.HasVote = false
		} else
		// if I have no vote and message is not the earliest request, add to priority queue
		if m.TimeStamp.IsSmaller(n.PriorityQueue[0]) {
			if n.HasVote {
				//vote
				fmt.Printf("%v : voting on request for %v \n", n.Id, m.TimeStamp)
				n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: Vote, TimeStamp: m.TimeStamp}
				n.HasVote = false
			} else {
				//rescind
				fmt.Printf("%v : rescinding from %v for %v\n", n.Id, n.PriorityQueue[0], m.TimeStamp)
				n.AllChannels[n.PriorityQueue[0].Id] <- Message{Sender: n.Id, Type: Rescind}
			}
		}
		n.PriorityQueue = append(n.PriorityQueue, m.TimeStamp)
		n.PriorityQueue = SortQueue(n.PriorityQueue)
	case Vote:
		if n.State == Idle || n.LastCompleted == m.TimeStamp {
			n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: Release, TimeStamp: m.TimeStamp}
			break
		}
		//If I receive a vote I will check whether I have a vote from every one
		if ArrayContains(n.WaitingArray, m.Sender) {
			fmt.Printf("%v : duplicate of %v found\n", n.Id, m.Sender)
		}
		n.WaitingArray = append(n.WaitingArray, m.Sender)
		votesRequired := int(math.Floor(float64(cap(n.AllChannels))/2) + 1)
		fmt.Printf("%v : vote received from %v. %v votes remaining. %v \n", n.Id, m.Sender, votesRequired-len(n.WaitingArray), n.WaitingArray)

		if len(n.WaitingArray) == votesRequired {
			n.State = HasLock
			n.ExecuteCriticalSection(n.Num)
			n.LastCompleted = m.TimeStamp
			//vote to everyone else
			for i := 0; i < len(n.WaitingArray); i++ {
				n.AllChannels[n.WaitingArray[i]] <- Message{Sender: n.Id, Type: Release, TimeStamp: n.PriorityQueue[0]}
			}
			n.WaitingArray = nil
			n.State = Idle
		}
	case Release:
		n.HasVote = true
		if m.TimeStamp == n.PriorityQueue[0] {
			n.PriorityQueue = n.PriorityQueue[1:]
		}
		if len(n.PriorityQueue) > 0 {
			//Vote
			fmt.Printf("%v : voting on release from %v for %v \n", n.Id, m.Sender, n.PriorityQueue[0])
			n.AllChannels[n.PriorityQueue[0].Id] <- Message{Sender: n.Id, Type: Vote, TimeStamp: n.PriorityQueue[0]}
			n.HasVote = false
		}
	case Rescind:
		//release if not in critical section
		if n.State == HasLock {
			break
		}
		n.WaitingArray = ArrayRemove(n.WaitingArray, m.Sender)
		n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: Release}
	}
}

func (t *TimeStamp) IsEarliest(t_array []TimeStamp) bool {
	for i := 0; i < len(t_array); i++ {
		if t_array[i].Time < t.Time {
			return false
		}
		if t_array[i].Time == t.Time && t_array[i].Id < t.Id {
			return false
		}
	}
	return true
}

func SortQueue(q []TimeStamp) []TimeStamp {
	sort.Slice(q, func(i, j int) bool {
		return q[j].Time > q[i].Time
	})
	return q
}

func (t *TimeStamp) IsSmaller(t2 TimeStamp) bool {
	if t.Time == t2.Time {
		fmt.Printf("Here, %v\n", t.Id < t2.Id)
		return t.Id < t2.Id
	}
	return t.Time < t2.Time
}

func QueueContains(arr []TimeStamp, t TimeStamp) bool {
	for i := 0; i < len(arr); i++ {
		if arr[i] == t {
			return true
		}
	}
	return false
}

func ArrayContains(arr []int, t int) bool {
	for i := 0; i < len(arr); i++ {
		if arr[i] == t {
			return true
		}
	}
	return false
}

func ArrayRemove(arr []int, t int) []int {
	for i := 0; i < len(arr); i++ {
		if arr[i] == t {
			return append(arr[:i], arr[i+1:]...)
		}
	}
	return arr
}
