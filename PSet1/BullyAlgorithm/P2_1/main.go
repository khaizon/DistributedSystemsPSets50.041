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
	Acknowledge
)

type MachineData struct {
	Id               int
	Timeout          int            //Message Propagation Time + Message Handling time - timeout until initiating an election
	Coordinator      int            //Current coordinator among other machines
	IsDown           bool           //either Down or Up state
	Channels         []chan Message //Receiving channels for the machines
	IsSender         bool           //whether this machine will be be down for bully algorithm to start
	IsInElection     bool           //Whether there is currently an election
	NumberOfMachines int            //Number of machines to communicate with
	Terminate        chan int
}

type Message struct {
	Sender int
	Type   MessageType // 4 Types: (1) Hello, (2) RejectCoordinator (3) RequestToBeCoordinator (4) reject coordinator
}

const numberOfMachines = 5

func main() {
	processStarted := false
	for {
		if !processStarted {
			fmt.Printf("Hi Prof! Please best(b) or worst (w) case> ")
		}
		channels := make([]chan Message, numberOfMachines)
		terminationChannels := make([]chan int, numberOfMachines)
		for i := 0; i < numberOfMachines; i++ {
			channels[i] = make(chan Message, 10*numberOfMachines)
			terminationChannels[i] = make(chan int, numberOfMachines)
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
			sender = numberOfMachines - 2
		} else {
			fmt.Print("worst case scenario selected\n")
			//if worst case, then isSender is true for client 1 (or 0)
			sender = 0
		}
		processStarted = true

		for i := 0; i < numberOfMachines; i++ {
			go machine(MachineData{
				Id:               i,
				Timeout:          4,
				Coordinator:      4,
				IsDown:           i == numberOfMachines-1,
				Channels:         channels,
				IsSender:         i == sender,
				Terminate:        terminationChannels[i],
				NumberOfMachines: numberOfMachines,
			})
		}
	}

	fmt.Print("program has ended \n")
}

func machine(self MachineData) {
	//check state => if Down, don't respond to messages
	ticker := time.NewTicker(time.Duration(self.Id+self.NumberOfMachines) * time.Second) //used for regular ping checks
	electionTimeoutChannel := make(chan int, 5)
	pingTimeoutChannel := make(chan int, 2)

	machinesStillAlive := make([]bool, self.NumberOfMachines)
	for i := 0; i < self.NumberOfMachines; i++ {
		machinesStillAlive[i] = true
	}
	for {
		if self.IsDown {
			// don't respond to messages
			continue
		}

		select {
		case <-self.Terminate:
			ticker.Stop()
			fmt.Printf("%v : Terminating now\n", self.Id)
			return
		case <-ticker.C:
			if !self.IsInElection && self.IsSender {
				fmt.Printf("%v : regular ping checks\n", self.Id)
				if self.Id != self.Coordinator {
					self.Channels[self.Coordinator] <- Message{Sender: self.Id, Type: Hello}
					machinesStillAlive[self.Coordinator] = false
				}

				go start_timeout(pingTimeoutChannel, self.Timeout)
			}
		case <-pingTimeoutChannel: //regular ping - check node failure
			if !self.IsInElection {
				//check for machine failure
				fmt.Printf("%v : Regular ping timeout. Checking for machine failure\n", self.Id)

				machineFailureDetected := false
				for i := 0; i < self.NumberOfMachines; i++ {
					if !machinesStillAlive[i] {
						machineFailureDetected = true
						fmt.Printf("%v : machine %v failure detected\n", self.Id, i)
						fmt.Printf("%v : machine %v is coordinator? %v \n", self.Id, i, self.Coordinator == i)
					}
				}
				if machineFailureDetected && !machinesStillAlive[self.Coordinator] {
					//start election
					fmt.Printf("%v : starting election\n", self.Id)
					self.IsInElection = true
					for i := 0; i < self.NumberOfMachines; i++ {
						if i <= self.Id {
							continue //only ask machines of high id
						}
						self.Channels[i] <- Message{Sender: self.Id, Type: CoordinatorRequest}
					}
					self.Coordinator = self.Id //self elect, a reply would override this before timeout
					go start_timeout(electionTimeoutChannel, self.Timeout)
				}
			}
		case <-electionTimeoutChannel: //timeout handler
			//election request time - check whether self.Coordinator is overriden
			if self.IsInElection {
				if self.Coordinator == self.Id {
					// election succeeded - start broadcasting
					fmt.Printf("%v : election succeeeded. Starting broadcast\n", self.Id)
					for i := 0; i < self.NumberOfMachines; i++ {
						if i == self.Id {
							continue //no need to broadcast to self
						}
						self.Channels[i] <- Message{Sender: self.Id, Type: NewCoordinator}
					}
				}
				if self.Coordinator != self.Id {
					//Election failed do nothing
					fmt.Printf("%v : Election failed. \n", self.Id)
				}
			}

		case msg := <-self.Channels[self.Id]: //message handler
			switch msg.Type {
			case Hello:
				fmt.Printf("%v : ping from %v received\n", self.Id, msg.Sender)
				self.Channels[msg.Sender] <- Message{Sender: self.Id, Type: Acknowledge} //reply the ping message

			case Acknowledge:
				fmt.Printf("%v : ping acknowledgement from %v received\n", self.Id, msg.Sender)
				machinesStillAlive[msg.Sender] = true //update that machine is still alive

			case CoordinatorRequest:
				fmt.Printf("%v : coordinator request from %v received\n", self.Id, msg.Sender)
				if msg.Sender < self.Id { //reply no if Id is higher
					fmt.Printf("%v : Rejecting coordinator request from %v received\n", self.Id, msg.Sender)
					self.Channels[msg.Sender] <- Message{Sender: self.Id, Type: Rejection}

					//start election
					fmt.Printf("%v : starting election\n", self.Id)
					self.IsInElection = true
					for i := 0; i < self.NumberOfMachines; i++ {
						if i <= self.Id {
							continue //only ask machines of high id
						}
						self.Channels[i] <- Message{Sender: self.Id, Type: CoordinatorRequest}
					}
					self.Coordinator = self.Id //self elect, a reply would override this before timeout
					go start_timeout(electionTimeoutChannel, self.Timeout)
				}

			case Rejection:
				self.Coordinator = msg.Sender
				fmt.Printf("%v : Rejection. new coordinator is temporarily set to %v\n", self.Id, self.Coordinator)

			case NewCoordinator: // set coordinator to sender
				self.Coordinator = msg.Sender
				fmt.Printf("%v : new coordinator is  %v\n", self.Id, self.Coordinator)
				self.IsInElection = false
			}

		}

	}
}

func start_timeout(timeoutChannel chan int, duration int) {
	time.Sleep(time.Second * time.Duration(duration))
	timeoutChannel <- 0
}
