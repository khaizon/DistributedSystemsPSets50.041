package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func client(id int, sendingChannel chan []int, receivingChannel chan []int, size int) {
	send := make([]int, size)
	for {
		// periodic timeout to signal client to sent the server a message
		timeout := time.After(time.Millisecond * (time.Duration(rand.Intn(15000) + 2000)))

		select {
		// message received from server
		case msg := <-receivingChannel:
			fmt.Printf("\n%v received from server by client %d\n", msg, id)

		// random timeout to signal a send message
		case <-timeout:
			send[id] = send[id] + 1
			sendCopy := make([]int, len(send))
			copy(sendCopy, send)
			sendingChannel <- sendCopy
			fmt.Printf("\nClient %d has sent %v\n", id, sendCopy)
		}
	}
}

func server(sendingChannels []chan []int, receivingChannels []chan []int) {

	localClock := make([]int, cap(sendingChannels)+1)

	for {
		set := []reflect.SelectCase{}
		for _, ch := range receivingChannels {
			set = append(set, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ch),
			})
		}
		from, valValue, _ := reflect.Select(set)
		var val = valValue.Interface().([]int)
		fmt.Printf("\nServer received %d from Channel%d\n", val, from)
		localClock[0] = localClock[0] + 1

		randomDelay := time.Millisecond * (time.Duration(rand.Intn(5000) + 1000))
		randomDelayChannel := time.After(randomDelay)
		fmt.Printf("\nServer random delay of %v before broadcasting %v\n", randomDelay, val)
		<-randomDelayChannel

		fmt.Printf("\nServer is broadcasting %d to all other channels\n", val)
		for i := 0; i < cap(sendingChannels); i++ {
			if i != from {
				sendingChannels[i] <- val
				localClock[0] = localClock[0] + 1
			}
		}

	}
}

func main() {
	// var input string
	// fmt.Scanln(&input)
	var processStarted = false
	for {
		if !processStarted {
			fmt.Printf("Hi Prof! Please input number of clients> ")
		}
		noOfClientsString, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Printf("\nInput detected: %s \n", noOfClientsString)

		tokens := strings.Fields(noOfClientsString)
		if len(tokens) == 0 {
			fmt.Println("No input detected. Please enter a number.")
			continue
		}

		noOfClients := tokens[0]
		if intVar, err := strconv.ParseInt(noOfClients, 10, 0); err == nil {
			fmt.Printf("%q looks like a number. Creating %q clients.\n", noOfClients, noOfClients)

			var clientToServer = make([]chan []int, int(intVar))
			var serverToClient = make([]chan []int, int(intVar))
			for i := 0; i < int(intVar); i++ {
				clientToServer[i] = make(chan []int, 10)
				serverToClient[i] = make(chan []int, 10)
			}

			for i := 0; i < int(intVar); i++ {
				go client(i, clientToServer[i], serverToClient[i], int(intVar))
			}
			go server(serverToClient, clientToServer)
			processStarted = true
		}

	}

}
