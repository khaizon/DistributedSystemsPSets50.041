package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

type ClientData struct {
	Id               int
	SendingChannel   chan Message
	ReceivingChannel chan Message
	NumberOfClients  int
}

type ServerData struct {
	ReceivingChannel     chan Message
	ClientsData          []ClientData
	BroadcastingChannels []chan Message
}

type Message struct {
	Content []int
	Sender  int
	//Clock int
}

type Event struct {
	Content  []int
	Sender   int
	Receiver int
}

type ServerBroadcastInput struct {
	Message      Message
	Clients      []ClientData
	Delay        time.Duration
	EventChannel chan Event
}

func client(data ClientData) {
	message := make([]int, data.NumberOfClients)
	for {
		// periodic timeout to signal client to sent the server a message
		timeout := time.After(time.Millisecond * (time.Duration(rand.Intn(15000) + 2000)))
		select {
		// message received from server
		case msg := <-data.ReceivingChannel:
			fmt.Printf("%v received from server by client %d\n", msg.Content, data.Id)

		// random timeout to signal a send message
		case <-timeout:
			message[data.Id] = message[data.Id] + 1
			sendCopy := make([]int, len(message))
			copy(sendCopy, message)
			data.SendingChannel <- Message{sendCopy, data.Id}
			fmt.Printf("Client %d has sent %v\n", data.Id, sendCopy)
		}
	}
}

func server(data ServerData) {
	eventChannel := make(chan Event, 10)
	for {
		select {
		case messageReceived := <-data.ReceivingChannel:
			fmt.Printf("%v received from Client %v\n", messageReceived.Content, messageReceived.Sender)

			//add delay for broadcast
			broadcastDelay := time.Millisecond * (time.Duration(rand.Intn(9000) + 1000))
			broadcastInput := ServerBroadcastInput{messageReceived, data.ClientsData, broadcastDelay, eventChannel}
			go broadcast(broadcastInput)

		case eventReceived := <-eventChannel:
			fmt.Printf("Event Log: Server sent %v to Client %v\n", eventReceived.Content, eventReceived.Receiver)
		}

	}
}

func broadcast(input ServerBroadcastInput) {
	<-time.After(input.Delay)
	fmt.Print("Starting to broadcast message from Server\n")
	for i := 0; i < len(input.Clients); i++ {
		if input.Clients[i].Id == input.Message.Sender {
			continue
		}
		input.Clients[i].ReceivingChannel <- input.Message
		//report to server completion of event
		input.EventChannel <- Event{input.Message.Content, -1, input.Clients[i].Id}
	}
}

func main() {
	var processStarted = false
	for {
		if !processStarted {
			fmt.Printf("Hi Prof! Please input number of clients> ")
		}
		var input string
		fmt.Scanln(&input)
		if processStarted {
			break
		}
		if numberOfClients, err := strconv.Atoi(input); err == nil {
			fmt.Printf("%q looks like a number. Creating %q clients. Press ENTER again to stop processes\n", input, input)
			processStarted = true

			var serverRecevingChannel = make(chan Message, int(numberOfClients))
			var serverBroadcastingChannels = make([]chan Message, int(numberOfClients))
			var clientArray = make([]ClientData, int(numberOfClients))
			for i := 0; i < int(numberOfClients); i++ {
				serverBroadcastingChannels[i] = make(chan Message, 10)
			}
			for i := 0; i < int(numberOfClients); i++ {
				clientData := ClientData{
					Id:               i,
					SendingChannel:   serverRecevingChannel,
					ReceivingChannel: serverBroadcastingChannels[i],
					NumberOfClients:  int(numberOfClients),
				}
				go client(clientData)
				clientArray[i] = clientData
			}
			go server(ServerData{
				ReceivingChannel:     serverRecevingChannel,
				ClientsData:          clientArray,
				BroadcastingChannels: serverBroadcastingChannels,
			})
		}
	}
}
