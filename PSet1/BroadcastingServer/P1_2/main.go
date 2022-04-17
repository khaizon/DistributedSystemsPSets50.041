package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"
)

type ClientData struct {
	Id               int
	SendingChannel   chan Message
	ReceivingChannel chan Message
	NumberOfClients  int
	Terminate        chan *sync.WaitGroup
	ReportMutex      *sync.Mutex
}

type ServerData struct {
	ReceivingChannel     chan Message
	ClientsData          []ClientData
	BroadcastingChannels []chan Message
}

type Message struct {
	Content []int
	Sender  int
	Clock   float64
}

type ServerBroadcastInput struct {
	Message      Message
	Clients      []ClientData
	Delay        time.Duration
	EventChannel chan Message
}

func client(data ClientData) {
	message := make([]int, data.NumberOfClients)
	var clock float64 = 0
	var messagesToBeRead []Message
	for {
		sendMessageDelay := time.After(time.Millisecond * (time.Duration(rand.Intn(15000) + 2000)))
		printOrderDelay := time.After(time.Millisecond * time.Duration((data.Id+1)*1000+5500))

		select {
		// message received from server
		case msg := <-data.ReceivingChannel:
			fmt.Printf("\n%v received from server by client %d", msg.Content, data.Id)
			messagesToBeRead = append(messagesToBeRead, msg)
			clock = math.Max(clock, msg.Clock) + 1

		// random timeout to signal a send message
		case <-sendMessageDelay:
			message[data.Id] = message[data.Id] + 1
			sendCopy := make([]int, len(message))
			copy(sendCopy, message)

			clock += 1
			data.SendingChannel <- Message{sendCopy, data.Id, clock}
			fmt.Printf("\nClient %d has sent %v", data.Id, sendCopy)

		//print the order of messages to be read every 15 + Id seconds
		case <-printOrderDelay:
			fmt.Printf("\nMessages to be read are%v", messagesToBeRead)
			sort.Slice(messagesToBeRead[:], func(i, j int) bool {
				return messagesToBeRead[i].Clock < messagesToBeRead[j].Clock
			})
			fmt.Printf("\nTotal Order for Client %v:", data.Id)
			for _, msg := range messagesToBeRead {
				fmt.Printf("\nClock: %v, Message: %v", msg.Clock, msg.Content)
			}
		case wg := <-data.Terminate:
			(*data.ReportMutex).Lock()
			fmt.Printf("\n\n--------- CLIENT %v REPORT ------------", data.Id)
			sort.Slice(messagesToBeRead[:], func(i, j int) bool {
				return messagesToBeRead[i].Clock < messagesToBeRead[j].Clock
			})
			fmt.Printf("\nTotal Order for Client %v:", data.Id)
			for _, msg := range messagesToBeRead {
				fmt.Printf("\nClock: %v, Message: %v", msg.Clock, msg.Content)
			}
			fmt.Printf("\n-------- END OF CLIENT %v REPORT ------", data.Id)
			(*data.ReportMutex).Unlock()
			(*wg).Done()
			fmt.Printf("\n%v : Terminating", data.Id)
			return
		}
	}
}

func server(data ServerData) {
	var clock float64 = 0
	eventChannel := make(chan Message, 10)
	for {

		select {
		case messageReceived := <-data.ReceivingChannel:
			fmt.Printf("\n%v received from Client %v", messageReceived.Content, messageReceived.Sender)
			clock = math.Max(clock, messageReceived.Clock) + 1

			//add delay for broadcast
			broadcastDelay := time.Millisecond * (time.Duration(rand.Intn(9000) + 1000))
			broadcastInput := ServerBroadcastInput{messageReceived, data.ClientsData, broadcastDelay, eventChannel}
			go broadcast(broadcastInput)

		case eventMessage := <-eventChannel:
			fmt.Printf("\nEvent Log: Server sent %v to Clients", eventMessage.Content)
			clock += 1
		}

	}
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
	var numberOfClients int
	var err error
	var serverRecevingChannel chan Message
	var serverBroadcastingChannels []chan Message
	var clientTerminatingChannels []chan *sync.WaitGroup
	var clientArray []ClientData
	var reportMutex sync.Mutex

	for {
		if !processStarted {
			fmt.Printf("Hi Prof! Please input number of clients> ")
		}
		var input string
		var wg sync.WaitGroup
		fmt.Scanln(&input)
		if processStarted {
			for i := 0; i < numberOfClients; i++ {
				wg.Add(1)
				clientTerminatingChannels[i] <- &wg
			}
			wg.Wait()
			break
		}

		if numberOfClients, err = strconv.Atoi(input); err == nil {
			fmt.Printf("\n%q looks like a number. Creating %q clients. Press ENTER again to stop processes", input, input)
			processStarted = true

			serverRecevingChannel = make(chan Message, int(numberOfClients))
			serverBroadcastingChannels = make([]chan Message, int(numberOfClients))
			clientTerminatingChannels = make([]chan *sync.WaitGroup, int(numberOfClients))
			clientArray = make([]ClientData, int(numberOfClients))
			for i := 0; i < int(numberOfClients); i++ {
				serverBroadcastingChannels[i] = make(chan Message, 10)
				clientTerminatingChannels[i] = make(chan *sync.WaitGroup, 10)
			}
			for i := 0; i < int(numberOfClients); i++ {
				clientData := ClientData{
					Id:               i,
					SendingChannel:   serverRecevingChannel,
					ReceivingChannel: serverBroadcastingChannels[i],
					NumberOfClients:  int(numberOfClients),
					Terminate:        clientTerminatingChannels[i],
					ReportMutex:      &reportMutex,
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
