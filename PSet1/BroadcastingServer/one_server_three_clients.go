package main

import (
	"fmt"
	"math/rand"
	"time"
)

type clientData struct {
	id               int
	sendingChannel   chan []int
	receivingChannel chan []int
	clock            int
}

type serverInput struct {
	receivingChannel chan []int
	clients          []clientData
}

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

		localClock[0] = localClock[0] + 1

		randomDelay := time.Millisecond * (time.Duration(rand.Intn(5000) + 1000))
		randomDelayChannel := time.After(randomDelay)
		<-randomDelayChannel

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
		fmt.Printf("%v", input)

	}

}
