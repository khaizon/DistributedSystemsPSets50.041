package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
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
	Clock   []float64
}

type ServerBroadcastInput struct {
	Message      Message
	Clients      []ClientData
	Delay        time.Duration
	EventChannel chan Message //test
}

func client(data ClientData) {
	message := make([]int, data.NumberOfClients)
	clock := make([]float64, data.NumberOfClients+1)
	var messagesToBeRead []Message
	for {
		select {
		// message received from server
		case msg := <-data.ReceivingChannel:
			fmt.Printf("\n%v received from server by client %d", msg.Content, data.Id)
			messagesToBeRead = append(messagesToBeRead, msg)
			var internalClockCopy = make([]float64, data.NumberOfClients+1)
			var messageClockCopy = make([]float64, data.NumberOfClients+1)
			copy(internalClockCopy, clock)
			copy(messageClockCopy, msg.Clock)
			clock = vectorMax(internalClockCopy, messageClockCopy)
			clock[data.Id] += 1

			// random timeout to signal a send message
		case <-time.After(time.Millisecond * (time.Duration(rand.Intn(15000) + 2000))):
			message[data.Id] = message[data.Id] + 1
			sendCopy := make([]int, len(message))
			copy(sendCopy, message)

			tempClockCopy := make([]float64, data.NumberOfClients+1)
			clock[data.Id] += 1
			copy(tempClockCopy, clock)

			data.SendingChannel <- Message{sendCopy, data.Id, tempClockCopy}
			fmt.Printf("\nClient %d has sent %v", data.Id, tempClockCopy)

		//print the order of messages to be read every 15 + Id seconds
		case <-time.After(time.Millisecond * time.Duration((data.Id+1)*1000+5500)):
			fmt.Printf("\nMessages to be read are%v", messagesToBeRead)
			sort.Slice(messagesToBeRead[:], func(i, j int) bool {
				return compareVectors(messagesToBeRead[i].Clock, messagesToBeRead[j].Clock)
			})
			fmt.Printf("\nTotal Order for Client %v:", data.Id)
			for _, msg := range messagesToBeRead {
				fmt.Printf("\nClock: %v, Message: %v", msg.Clock, msg.Content)
			}
		}
	}
}

func compareVectors(smaller, greater []float64) bool {
	//return true is smaller vector is
	//strictly smaller than greater vector
	for c, element := range smaller {
		if element > greater[c] {
			return false
		}
	}
	return true
}

func server(data ServerData) {
	//array size = number of clients + 1
	//=> thus server id =number of clients since indexing begins from 0
	Id := cap(data.ClientsData)
	clock := make([]float64, cap(data.ClientsData)+1)
	eventChannel := make(chan Message, 10)
	for {

		select {
		case messageReceived := <-data.ReceivingChannel:
			fmt.Printf("\n%v received from Client %v", messageReceived.Content, messageReceived.Sender)
			var internalClockCopy = make([]float64, cap(data.ClientsData)+1)
			var messageClockCopy = make([]float64, cap(data.ClientsData)+1)

			copy(internalClockCopy, clock)
			copy(messageClockCopy, messageReceived.Clock)
			clock = vectorMax(internalClockCopy, messageClockCopy)
			clock[Id] += 1

			//add delay for broadcast
			broadcastDelay := time.Millisecond * (time.Duration(rand.Intn(9000) + 1000))
			broadcastInput := ServerBroadcastInput{messageReceived, data.ClientsData, broadcastDelay, eventChannel}
			go broadcast(broadcastInput)

		case eventMessage := <-eventChannel:
			fmt.Printf("\nEvent Log: Server sent %v to Clients", eventMessage.Content)
			clock[Id] += 1
		}

	}
}

func vectorMax(v1, v2 []float64) []float64 {
	result := make([]float64, cap(v2))
	for i := 0; i < cap(v2); i++ {
		result[i] = math.Max(v1[i], v2[i])
	}
	return result
}

func broadcast(input ServerBroadcastInput) {
	<-time.After(input.Delay)
	fmt.Print("\nStarting to broadcast message from Server")
	for i := 0; i < len(input.Clients); i++ {
		if input.Clients[i].Id == input.Message.Sender {
			continue
		}
		input.Clients[i].ReceivingChannel <- input.Message
		//report to server completion of event
	}
	input.EventChannel <- input.Message
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
			fmt.Printf("\n%q looks like a number. Creating %q clients. Press ENTER again to stop processes", input, input)
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
