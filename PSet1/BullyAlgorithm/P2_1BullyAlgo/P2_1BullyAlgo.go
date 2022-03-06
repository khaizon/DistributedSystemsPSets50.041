package main

import (
	"fmt"
	"math/rand"
	"time"
)

type MachineState int

const (
	Down MachineState = iota
	Up
)

type MachineData struct {
	Id       int
	Timeout  int          //Message Propagation Time + Message Handling time - timeout until initiating an election
	Ring     []int        //Ring structure
	State    MachineState //either Down or Up state
	Channels []chan int   //Receiving channels for the machines
	WillDie  bool         //whether this machine will be be down for bully algorithm to start
}

type ElectionData struct {
	Initiator int        //The machine starting/calling for the election
	Ring      []int      //Rind structure
	Channels  []chan int //Receiving channels of all machines
	Timeout   int        //timeout before declaring self election
}

func main() {
	fmt.Print("hello world")
}

func machine(data MachineData) {
	//check state => if Down, don't respond to messages
	for {
		if data.State == Down {
			// don't respond to messages
			continue
		}
		//try to send message
		randomChannel := rand.Intn(len(data.Channels))
		for randomChannel == data.Id {
			randomChannel = rand.Intn(len(data.Channels))
		}
		data.Channels[randomChannel] <- data.Id
		select {
		case <-time.After(time.Second * 2):
			//start election?
		case msg := <-data.Channels[data.Id]:
			fmt.Printf("%v\n", msg)
		}

	}
}

func start_election(data ElectionData) {
	//a way to start the election
	//send to all machines of higher Id
	//wait for an answer until timeout has been reached
}
