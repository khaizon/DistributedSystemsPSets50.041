package main

import (
	"fmt"
	"math"
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
	if len(tArray) == 0 {
		return true
	}
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

func Contains(arr []int, num int) bool {
	for i := 0; i < len(arr); i++ {
		if arr[i] == num {
			return true
		}
	}
	return false
}

func (t1 *TimeStamp) SmallerThan(t2 TimeStamp) bool {
	if t1.Time == t2.Time {
		return t1.Id < t2.Id
	}
	return t1.Time < t2.Time
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
			PriorityQueue:    []TimeStamp{},
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
			fmt.Printf("%v : state: %v\n wait queue: %v\n Priority Queue: %v\n", n.Id, n.State, n.VotesReceived, n.PriorityQueue)
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
		fmt.Printf("%v : acquire request from %v. Has VOte: %v. message is earliest: %v \n", n.Id, m.Sender, n.HasVote, m.TimeStamp.IsEarliest(n.PriorityQueue))
		if m.TimeStamp.IsEarliest(n.PriorityQueue) {
			if n.HasVote {
				fmt.Printf("%v : has vote. Voting for %v\n", n.Id, m.Sender)
				n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: Vote}
				n.HasVote = false
			} else {
				fmt.Printf("%v : rescinding from %v\n", n.Id, n.PriorityQueue[0].Id)
				n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: RescindVote}
			}
		}
		//if not earliest, add to queue
		n.PriorityQueue = append(n.PriorityQueue, m.TimeStamp)
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
		n.PriorityQueue = n.PriorityQueue[1:]
		n.HasVote = true
		n.PriorityQueue = SortQueue(n.PriorityQueue)
		if len(n.PriorityQueue) > 0 {
			stamp := n.PriorityQueue[0]
			fmt.Printf("%v : vote was released by %v. Voting for %v\n", n.Id, m.Sender, stamp.Id)
			n.AllChannels[stamp.Id] <- Message{Sender: n.Id, Type: Vote}
			n.HasVote = false
		}
		break
	case Vote:
		//if node is in idle mode, release vote
		//add to array of votes recieved
		if n.State == Idle {
			n.AllChannels[m.Sender] <- Message{Sender: n.Id, Type: ReleaseVote}
		}
		//compute value of majority
		majorityValue := int(math.Floor(float64(cap(n.AllChannels))/2) + 1)
		if Contains(n.VotesReceived, m.Sender) {
			fmt.Printf("%v : error flag, vote from %v already present---------------------\n", n.Id, m.Sender)
			break
		}
		fmt.Printf("%v : vote from %v received\n", n.Id, m.Sender)
		n.VotesReceived = append(n.VotesReceived, m.Sender)
		fmt.Printf("%v : current votes received%v\n", n.Id, n.VotesReceived)

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
	// time.Sleep(time.Second * time.Duration(rand.Intn(3)))
	n.State = WaitingForReplies
	fmt.Printf("%v : requesting lock, waiting for replies\n", n.Id)
	requestTimeStamp := TimeStamp{n.Id, int(time.Now().UnixMicro())}

	m := Message{Sender: n.Id, Type: Acquire, TimeStamp: requestTimeStamp}

	for i := 0; i < cap(n.AllChannels); i++ {
		n.AllChannels[i] <- m
	}
}
