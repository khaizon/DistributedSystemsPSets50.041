package main

import (
	"fmt"
	"math"
	"sync"
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
	AllowedToRequest bool
	WaitGroup        *sync.WaitGroup
	Done             chan int
	Start            chan struct{}
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
	Kill
)

const (
	Idle StateType = iota
	WaitingForvotes
	HasLock
)

const (
	NUM_OF_NODES = 20
)

func main() {
	allChannels := make([]chan Message, NUM_OF_NODES)

	for i := 0; i < NUM_OF_NODES; i++ {
		allChannels[i] = make(chan Message, NUM_OF_NODES*10)
	}

	valueToAdd := 0
	for j := 1; j < NUM_OF_NODES+1; j++ {
		var wg sync.WaitGroup
		var start = make(chan struct{}, 0)
		wg.Add(j)
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
				AllowedToRequest: i < j,
				WaitGroup:        &wg,
				Done:             make(chan int, 1),
				Start:            start,
			}

			go node.start()
		}
		t1 := time.Now().UnixMicro()
		close(start)
		wg.Wait()
		fmt.Printf("%v out of %v concurrent requests done\n", j, NUM_OF_NODES)
		t2 := time.Now().UnixMicro()
		for i := 0; i < cap(allChannels); i++ {
			allChannels[i] <- Message{Type: Kill}
		}
		fmt.Printf("Number of Nodes: %v,    Time taken: %v\n", j, t2-t1)
		time.Sleep(2 * time.Second)
	}

	var input string
	fmt.Scanln(&input)
}

func (n *Node) start() {
	requested := false
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
		case m := <-n.ReceivingChannel:
			if m.Type == Kill {
				return
			}
			n.HandleRequest(m)
		default:
			if n.State == Idle && !requested && n.AllowedToRequest {
				<-n.Start
				n.RandomLockRequest()
				requested = true
			}
		}
	}
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

func (n *Node) RandomLockRequest() {
	// time.Sleep(time.Second * time.Duration(rand.Intn(3)))
	n.State = WaitingForvotes
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
			n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: Vote, TimeStamp: m.TimeStamp}
			n.HasVote = false
		} else if m.TimeStamp.IsSmaller(n.PriorityQueue[0]) {
			if n.HasVote {
				//vote
				n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: Vote, TimeStamp: m.TimeStamp}
				n.HasVote = false
			} else {
				//rescind
				fmt.Printf("%v : rescinding from %v\n", n.Id, n.PriorityQueue[0].Id)
				n.AllChannels[n.PriorityQueue[0].Id] <- Message{Sender: n.Id, Type: Rescind}
			}
		}
		n.PriorityQueue = AddTimeStamp(n.PriorityQueue, m.TimeStamp)
		// n.PriorityQueue = SortQueue(n.PriorityQueue)
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

		if len(n.WaitingArray) == votesRequired {
			n.State = HasLock
			n.ExecuteCriticalSection(n.Num)
			n.WaitGroup.Done()
			n.LastCompleted = m.TimeStamp
			//release vote to everyone else
			for i := 0; i < len(n.WaitingArray); i++ {
				n.AllChannels[n.WaitingArray[i]] <- Message{Sender: n.Id, Type: Release, TimeStamp: n.PriorityQueue[0]}
			}
			n.WaitingArray = nil
			n.State = Idle
		}
	case Release:
		n.HasVote = true
		if len(n.PriorityQueue) == 0 {
			return
		}
		if m.TimeStamp == n.PriorityQueue[0] {
			n.PriorityQueue = n.PriorityQueue[1:]
		}
		if len(n.PriorityQueue) > 0 {
			//Vote
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

func (t *TimeStamp) IsSmaller(t2 TimeStamp) bool {
	if t.Time == t2.Time {
		// fmt.Printf("Here, %v\n", t.Id < t2.Id)
		return t.Id < t2.Id
	}
	return t.Time < t2.Time
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

func AddTimeStamp(arr []TimeStamp, t TimeStamp) []TimeStamp {
	result := []TimeStamp{}
	if len(arr) == 0 {
		return append(arr, t)
	}

	for i := len(arr) - 1; i >= 0; i-- {
		if t.Time > arr[i].Time {
			result = append([]TimeStamp{t}, arr[i+1:]...)
			result = append(arr[:i+1], result...)
			return result
		}
	}
	return append([]TimeStamp{t}, arr...)
}
