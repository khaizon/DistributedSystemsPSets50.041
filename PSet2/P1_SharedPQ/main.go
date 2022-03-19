package main

import (
	"fmt"
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
	Request          TimeStamp
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
	Reply
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
----------%v : Has lock, executing critical section <Number to Add: %v>------------
-----------------------------------------------------------------------------------
`, n.Id, *num)
	*num += 1
	fmt.Printf(`-----------------------------------------------------------------------------------
--------%v : Executed critical section <Number to Add: %v> Releasing lock----------
-----------------------------------------------------------------------------------
`, n.Id, *num)
}

func (n *Node) RandomLockRequest() {
	// time.Sleep(time.Second * time.Duration(rand.Intn(3)))
	n.State = WaitingForReplies
	fmt.Printf("%v : requesting lock, waiting for replies\n", n.Id)
	requestTimeStamp := TimeStamp{n.Id, time.Now().UnixNano()}
	n.Request = requestTimeStamp
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
		//if waiting for reply, compare to own request time
		if n.State == HasLock || n.State == WaitingForReplies && n.Request.IsSmaller(m.TimeStamp) {
			n.PriorityQueue = append(n.PriorityQueue, m.TimeStamp)
			n.PriorityQueue = SortQueue(n.PriorityQueue)
			break
		}
		if n.State == HasLock {
			n.PriorityQueue = append(n.PriorityQueue, m.TimeStamp)
			n.PriorityQueue = SortQueue(n.PriorityQueue)
			break
		}

		//else reply
		n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: Reply}

	case Reply:
		//If I receive a reply I will check whether I have a reply from every one
		if ArrayContains(n.WaitingArray, m.Sender) {
			fmt.Printf("%v : duplicate of %v found\n", n.Id, m.Sender)
		}
		n.WaitingArray = append(n.WaitingArray, m.Sender)
		repliesRequired := cap(n.AllChannels)
		fmt.Printf("%v : Reply received from %v. %v replies remaining. %v \n", n.Id, m.Sender, repliesRequired-len(n.WaitingArray), n.WaitingArray)
		if len(n.WaitingArray) == repliesRequired {
			n.State = HasLock
			n.ExecuteCriticalSection(n.Num)
			//reply to everyone else
			for i := 0; i < len(n.PriorityQueue); i++ {
				n.AllChannels[n.PriorityQueue[i].Id] <- Message{Sender: n.Id, Type: Reply}
			}
			n.PriorityQueue = nil
			n.WaitingArray = nil
			n.State = Idle
		}

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
