package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
)

//================================== STRUCTS AND HELPERS ======================================//
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
	VotesReceived    []int
}

type Message struct {
	Sender    int
	Type      MessageType
	TimeStamp TimeStamp
}

type TimeStamp struct {
	Id   int
	Time int
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
		if t.Time > tArray[i].Time {
			return false
		}

		//tie breaker using Id
		if t.Time == tArray[i].Time && t.Id > tArray[i].Id {
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
		return q[i].Time < q[j].Time
	})
	return q
}

//============================ END OF STRUCTS AND HELPERS =====================================//
//=============================================================================================//

//==================================== NODE LOGIC =============================================//
//=============================================================================================//

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
			VotesReceived:    []int{},
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
			temp := []int{}
			for i := 0; i < len(n.PriorityQueue); i++ {
				temp = append(temp, n.PriorityQueue[i].Id)
			}
			fmt.Printf("%v : vote with : %v\n time stamp: %v\n state: %v\n wait queue: %v\n", n.Id, n.VotedTo.Id, n.VotedTo.Time, n.State, n.VotesReceived)
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
		//check whether request is the earlist known request
		//if earliest, vote or rescind current vote
		if m.TimeStamp.IsEarliest(n.PriorityQueue) {
			if n.HasVote {
				n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: Vote}
				n.HasVote = false
				n.VotedTo = m.TimeStamp
				break
			}
			n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: RescindVote}
			n.PriorityQueue = append(n.PriorityQueue, n.VotedTo)
			n.PriorityQueue = SortQueue(n.PriorityQueue)
			break
		}
		//if not earliest, add to queue
		n.PriorityQueue = append(n.PriorityQueue, n.VotedTo)
		n.PriorityQueue = SortQueue(n.PriorityQueue)
		break
	case RescindVote:
		//simply release your vote and remove from your array of votes received
		for i := 0; i < len(n.VotesReceived); i++ {
			if i == m.Sender {
				n.VotesReceived = append(n.VotesReceived[:i], n.VotesReceived[i+1:]...)
				break
			}
		}
		n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: ReleaseVote}
		break
	case ReleaseVote:
		//check whether there are others to vote for
		//if not then do nothing
		n.HasVote = true
		n.PriorityQueue = SortQueue(n.PriorityQueue)
		if len(n.PriorityQueue) > 0 {
			stamp := n.PriorityQueue[0]
			n.AllChannels[stamp.Id] <- Message{Sender: n.Id, Type: Vote}
			n.HasVote = false
			n.VotedTo = stamp
			n.PriorityQueue = n.PriorityQueue[1:]
		}
		break
	case Vote:
		//if node is in idle mode, release vote
		//add to array of votes recieved
		if n.State == Idle || n.State == HasLock {
			n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: ReleaseVote}
			break
		}

		//compute value of majority
		majorityValue := int(math.Floor(float64(cap(n.AllChannels))/2) + 1)
		n.VotesReceived = append(n.VotesReceived, m.Sender)
		sort.Ints(n.VotesReceived)

		//check whether has majority
		if len(n.VotesReceived) == majorityValue {
			n.State = HasLock
			n.ExecuteCriticalSection(n.Num)
			//release votes
			for i := 0; i < len(n.VotesReceived); i++ {
				n.AllChannels[n.VotesReceived[i]] <- Message{Sender: n.Id, Type: ReleaseVote}
			}
			n.VotesReceived = nil
			n.State = Idle
		}
	}
}

func (n *Node) RandomLockRequest() {
	time.Sleep(time.Second * time.Duration(rand.Intn(3)))
	n.State = WaitingForReplies
	fmt.Printf("%v : requesting lock, waiting for replies\n", n.Id)
	requestTimeStamp := TimeStamp{n.Id, int(time.Now().UnixMicro())}

	m := Message{Sender: n.Id, Type: Acquire, TimeStamp: requestTimeStamp}

	for i := 0; i < cap(n.AllChannels); i++ {
		n.AllChannels[i] <- m
	}
}
