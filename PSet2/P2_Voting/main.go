package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
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
			HasVote:          true,
			AllChannels:      allChannels,
			Num:              &valueToAdd,
			PriorityQueue:    make([]TimeStamp, 0),
			WaitingArray:     make([]int, 0),
		}

		go node.start()
	}
	var input string
	fmt.Print("Press Enter to Stop")
	fmt.Scanln(&input)
}

func (n *Node) start() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case m := <-n.ReceivingChannel:
			n.HandleRequest(m)

		case <-ticker.C:
			fmt.Printf("%v : vote with : %v\n time stamp: %v\n state: %v\n wait queue: %v\n", n.Id, n.VotedTo.Id, n.VotedTo.Time.Unix(), n.State, n.WaitingArray)
		default:
			if n.State == Idle {
				n.RandomLockRequest()
			}
		}
	}
}

func (n *Node) HandleRequest(m Message) {
	switch m.Type {
	case Acquire:
		n.PriorityQueue = append(n.PriorityQueue, m.TimeStamp)
		//check whether request is earliest in the priority queue:
		// AND whether current node has a vote
		if m.TimeStamp.Time.UnixNano() <= n.PriorityQueue[0].Time.UnixNano() {
			if n.HasVote {
				//vote for machine
				fmt.Printf("%v : request received, voting to %v\n", n.Id, m.Sender)
				n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: Vote}

				n.PriorityQueue = RemoveTimeStamp(n.PriorityQueue, m.Sender)
				n.PriorityQueue = SortQueue(n.PriorityQueue)

				n.HasVote = false
				n.VotedTo = m.TimeStamp
				break
			}
			// if no vote and timestamp is earliest, rescind vote
			fmt.Printf("%v : acquire time stamp: %v, but my vote's timestamp is %v\n", n.Id, m.TimeStamp, n.VotedTo)

			fmt.Printf("%v : request received, rescinding vote from%v\n", n.Id, n.VotedTo.Id)
			n.AllChannels[m.Sender] <- Message{Sender: n.VotedTo.Id, Type: RescindVote}

			//request should be added back into the priority queue
			n.PriorityQueue = append(n.PriorityQueue, n.VotedTo)
			n.PriorityQueue = SortQueue(n.PriorityQueue)
		}
	case Vote:
		fmt.Printf("%v : vote received from %v \n", n.Id, m.Sender)

		if n.State != WaitingForReplies {
			//release vote
			n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: ReleaseVote}
			break
		}

		//compute value of majority
		majorityValue := int(math.Floor(float64(cap(n.AllChannels))/2) + 1)
		n.WaitingArray = append(n.WaitingArray, m.Sender)
		sort.Ints(n.WaitingArray)

		//check whether has majority
		if len(n.WaitingArray) == majorityValue {
			n.State = HasLock
			n.ExecuteCriticalSection(n.Num)
			n.ReleaseAndReply()
			n.State = Idle
		} else {
			fmt.Printf("%v : waiting for %v more replies\n", n.Id, majorityValue-len(n.WaitingArray))
			fmt.Printf("%v : waiting array %v \n", n.Id, n.WaitingArray)
		}
	case ReleaseVote:
		n.HasVote = true
		//vote if there is a request waiting in the queue
		if len(n.PriorityQueue) > 0 {
			stamp := n.PriorityQueue[0]

			// cast vote
			fmt.Printf("%v : released, voting to %v \n", n.Id, stamp.Id)
			n.AllChannels[stamp.Id] <- Message{Sender: n.Id, Type: Vote}

			n.VotedTo = stamp
			n.HasVote = false
			n.PriorityQueue = RemoveTimeStamp(n.PriorityQueue, stamp.Id)
			n.PriorityQueue = SortQueue(n.PriorityQueue)
		}
	case RescindVote:
		//remove vote and release
		for i := 0; i < len(n.WaitingArray); i++ {
			if i == m.Sender {
				n.WaitingArray = append(n.WaitingArray[:i], n.WaitingArray[i+1:]...)
				break
			}
		}

		//release vote
		n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: ReleaseVote}
	}
}

func (n *Node) ReleaseAndReply() {
	//pop self's timestamp off the priority queue
	n.PriorityQueue = RemoveTimeStamp(n.PriorityQueue, n.Id)

	//release vote to those who voted for me
	for i := 0; i < len(n.WaitingArray); i++ {
		n.AllChannels[n.WaitingArray[i]] <- Message{Sender: n.Id, Type: ReleaseVote}
	}

	// clear waiting array and set state to idle
	n.WaitingArray = nil
}
func (n *Node) RandomLockRequest() {
	time.Sleep(time.Second * time.Duration(rand.Intn(3)))
	n.State = WaitingForReplies
	fmt.Printf("%v : requesting lock, waiting for replies\n", n.Id)
	requestTimeStamp := TimeStamp{n.Id, time.Now()}

	m := Message{Sender: n.Id, Type: Acquire, TimeStamp: requestTimeStamp}

	n.PriorityQueue = append(n.PriorityQueue, requestTimeStamp)

	for i := 0; i < cap(n.AllChannels); i++ {
		n.AllChannels[i] <- m
	}
}

//=============================== STRUCTS AND HELPERS =========================================//
//=============================================================================================//

type Node struct {
	Id               int
	Queue            []int
	State            StateType
	HasVote          bool
	VotedTo          TimeStamp
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
	Vote
	ReleaseVote
	RescindVote
)

const (
	HasLock StateType = iota
	WaitingForReplies
	Idle
)

const (
	NUM_OF_NODES = 11
)

//Checks whether current timestamp is earlier than all other timestamps in the array
func (t *TimeStamp) IsEarliest(tArray []TimeStamp) bool {
	for i := 0; i < len(tArray); i++ {
		if t.Time.UnixNano() > tArray[i].Time.UnixNano() {
			return false
		}

		//tie breaker using Id
		if t.Time.Equal(tArray[i].Time) && t.Id > tArray[i].Id {
			return false
		}
	}
	return true
}

//Removes machine <id>'s timestamp from a timestamp array
func RemoveTimeStamp(tArray []TimeStamp, id int) []TimeStamp {
	for i := 0; i < len(tArray); i++ {
		if tArray[i].Id == id {
			return append(tArray[:i], tArray[i+1:]...)
		}
	}
	return tArray
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

func SortQueue(q []TimeStamp) []TimeStamp {
	sort.Slice(q, func(i, j int) bool {
		return q[j].Time.UnixNano() > q[i].Time.UnixNano()
	})
	return q
}
