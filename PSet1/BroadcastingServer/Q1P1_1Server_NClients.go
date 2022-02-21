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
}

func client(data ClientData) {
	message := make([]int, data.NumberOfClients)
	for {
		// periodic timeout to signal client to sent the server a message
		timeout := time.After(time.Millisecond * (time.Duration(rand.Intn(15000) + 2000)))
		select {
		// message received from server
		case msg := <-data.ReceivingChannel:
			fmt.Printf("\n%v received from server by client %d\n", msg.Content, data.Id)

		// random timeout to signal a send message
		case <-timeout:
			message[data.Id] = message[data.Id] + 1
			sendCopy := make([]int, len(message))
			copy(sendCopy, message)
			data.SendingChannel <- Message{sendCopy, data.Id}
			fmt.Printf("\nClient %d has sent %v\n", data.Id, sendCopy)
		}
	}
}

func server(data ServerData) {
	for {
		messageReceived := <-data.ReceivingChannel
		fmt.Printf("\n%v received from Client %v\n", messageReceived.Content, messageReceived.Sender)
		//random delay block
		<-time.After(time.Millisecond * (time.Duration(rand.Intn(5000) + 1000)))
		fmt.Printf("\nStarting to broadcast message from Client %v\n", messageReceived.Sender)

		for i := 0; i < len(data.ClientsData); i++ {
			if data.ClientsData[i].Id == messageReceived.Sender {
				continue
			}
			data.ClientsData[i].ReceivingChannel <- messageReceived
		}
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
		if numberOfClients, err := strconv.Atoi(input); err == nil {
			fmt.Printf("\n%q looks like a number. Creating %q clients.\n", input, input)
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
