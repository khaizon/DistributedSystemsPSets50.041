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
	ReceivingChannel chan Message
	ClientsData      []ClientData
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
			fmt.Printf("\n%v received from server by client %d\n", msg, data.Id)

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
		var messageReceived Message
		var randomDelayChannel <-chan time.Time
		select {
		case messageReceived := <-data.ReceivingChannel:
			fmt.Printf("\n%v received from %v", messageReceived.Content, messageReceived.Sender)
			randomDelay := time.Millisecond * (time.Duration(rand.Intn(5000) + 1000))
			randomDelayChannel = time.After(randomDelay)

		case <-randomDelayChannel:
			fmt.Printf("\nStarting to broadcast message from%v\n", messageReceived.Sender)
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
			fmt.Printf("%q looks like a number. Creating %q clients.\n", input, input)

			var serverRecevingChannel = make(chan []Message, int(numberOfClients))
			var serverBroadcastingChannels = make([]chan []Message, int(numberOfClients))
			var clients = make([]ClientData, int(numberOfClients))
			for i := 0; i < int(numberOfClients); i++ {
				serverBroadcastingChannels[i] = make(chan []Message, 10)
			}
			for i := 0; i < int(numberOfClients); i++ {
				clientData := ClientData{
					Id:               i,
					SendingChannel:   serverRecevingChannel,
					ReceivingChannel: serverBroadcastingChannels[i],
					NumberOfClients:  int(numberOfClients),
				}
				go client(clientData)
				clients = append(clients, clientData)
			}
			go server(ServerData{
				ReceivingChannel: serverRecevingChannel,
				ClientsData:      clients,
			})
		}
	}
}
