package main

import (
  "fmt"
  "time"
	"math/rand"
)


func sender(id int, clock chan []int) {
	send := []int{1,1,1,1,1,1,1,1,1,1,1};
  for i := 0; ; i++ {
		// send an integer array
    fmt.Println("Sending ", send)
    clock <- send
    amt := time.Duration(rand.Intn(1000))
    time.Sleep(time.Millisecond * amt)
		send[id] = send[id]+1
  }
}

func receiver(clock chan []int) {
  for {
		// receive integer from channel and print it
    msg := <- clock
    fmt.Println("Received ", msg)
		//sleep sporadically
  }
}

func main() {
	// create a channel of type integer
  var clock chan []int = make(chan []int)

	// launch two go routines "sender" and "receiver"
  go sender(0, clock)
  go sender(1, clock)
  go receiver(clock)

  var input string
  fmt.Scanln(&input)
}
