package main

import (
	"fmt"
	"time"
)

type MessageType int

const (
	Rejection = iota
	CoordinatorRequest
	NewCoordinator
	Hello
)

type MachineData struct {
	Id          int
	Timeout     int            //Message Propagation Time + Message Handling time - timeout until initiating an election
	Coordinator int            //Current coordinator among other machines
	IsDown      bool           //either Down or Up state
	Channels    []chan Message //Receiving channels for the machines
	IsSender    bool           //whether this machine will be be down for bully algorithm to start
}

type ElectionData struct {
	Initiator int            //The machine starting/calling for the election
	Channels  []chan Message //Receiving channels of all machines
	Timeout   int            //timeout in seconds before declaring self election
}

type Message struct {
	Sender int
	Type   MessageType // 4 Types: (1) Hello, (2) RejectCoordinator (3) RequestToBeCoordinator (4) reject coordinator
}

func main() {
	processStarted := false
	for {
		if !processStarted {
			fmt.Printf("Hi Prof! Please best(b) or worst (w) case> ")
		}
		var input string
		var sender int
		fmt.Scanln(&input)
		if processStarted {
			break
		}
		//if best case, then isSender is true for client n-1
		if input == "b" {
			fmt.Print("best case scenario selected\n")
			sender = 3
			processStarted = true
		} else {
			fmt.Print("worst case scenario selected\n")
			//if worst case, then isSender is true for client 1 (or 0)
			sender = 0
			processStarted = true
		}
		fmt.Print(sender)
		channels := make([]chan Message, 5)
		for i := 0; i < 5; i++ {
			channels[i] = make(chan Message, 5)
		}

		for i := 0; i < 5; i++ {
			go machine(MachineData{Id: i, Timeout: 3, Coordinator: 4, IsDown: i == 4, Channels: channels, IsSender: i == sender})
		}
	}

	fmt.Print("program has ended \n")
}

func machine(data MachineData) {
	//check state => if Down, don't respond to messages
	messageNotSent := true
	for {
		if data.IsDown {
			// don't respond to messages
			continue
		}
		//try to send message if selected as the sender
		for data.IsSender && messageNotSent {
			fmt.Printf("%v: sending message to machine %v\n", data.Id, data.Coordinator)
			data.Channels[data.Coordinator] <- Message{data.Id, Hello}
			select {
			case reply := <-data.Channels[data.Id]:
				fmt.Printf("reply received: %v\n", reply)

			case <-time.After(time.Duration(data.Timeout) * time.Second):
				if messageNotSent {
					fmt.Printf("%v: no replies received. Timeout. Starting election\n", data.Id)
					//start election?
					electionRejected := start_election(ElectionData{data.Id, data.Channels, data.Timeout})
					if !electionRejected {
						// If election request is not rejected by machines with higher Id, declare self as coordinator
						fmt.Printf("%v: no rejection received. declaring self as coordinator\n", data.Id)
						go broadcast_coordinator(data)
						messageNotSent = false
					}
				}
			}

		}
		select {
		case msg := <-data.Channels[data.Id]:
			fmt.Printf("%v : message recieved\n", data.Id)
			if msg.Type == CoordinatorRequest || msg.Type == NewCoordinator {
				// check to reject the coordinator request
				if msg.Sender < data.Id {
					data.Channels[msg.Sender] <- Message{Sender: data.Id, Type: Rejection}
					electionRejected := start_election(ElectionData{data.Id, data.Channels, data.Timeout})
					if !electionRejected {
						// If election request is not rejected by machines with higher Id, declare self as coordinator
						fmt.Printf("%v: no rejection received. declaring self as coordinator\n", data.Id)
						go broadcast_coordinator(data)
						messageNotSent = false
					}
				} else {
					// else update coordinator
					data.Coordinator = msg.Sender
					fmt.Printf("%v : Updated coordinator to :%v\n", data.Id, msg.Sender)
				}
			}
		}
	}
}

func start_election(data ElectionData) bool {
	// returns true if election request is denied
	//send to first machine of higher Id
	for i, c := range data.Channels {
		if i <= data.Initiator {
			continue //ignore machines of lower Id
		}
		c <- Message{data.Initiator, CoordinatorRequest}
	}
	//wait for an answer until timeout has been reached
	for {
		timeout := time.After(time.Duration(data.Timeout) * time.Second)
		select {
		case msg := <-data.Channels[data.Initiator]:
			fmt.Printf("%v : message %v received\n", data.Initiator, msg)
			if msg.Type != Rejection {
				continue
			}
			return true
		case <-timeout:
			return false
		}
	}
}

func broadcast_coordinator(self MachineData) {
	//send to all machines in the ring that current machine is the coordinator
	message := Message{Sender: self.Id, Type: NewCoordinator}

	for i := 0; i < cap(self.Channels); i++ {
		if i == self.Id {
			continue
		}
		self.Channels[i] <- message
	}
}
