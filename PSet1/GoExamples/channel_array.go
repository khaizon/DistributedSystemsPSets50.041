package main

import (
  "fmt"
  "time"
	"math/rand"
)


func sender(clock chan []int) {
	send := []int{1,2,34,5,6,9,10,7,11,12};
  for i := 0; ; i++ {
		// send an integer array 
    fmt.Println("Sending ", send)
    clock <- send
    amt := time.Duration(rand.Intn(1000))
    time.Sleep(time.Millisecond * amt)
  }
}

func receiver(clock chan []int) {
  for {
		// receive integer from channel and print it
    msg := <- clock
    fmt.Println("Receiving ", msg)
		//sleep sporadically
  }
}

func main() {
	// create a channel of type integer
  var clock chan []int = make(chan []int)

	// launch two go routines "sender" and "receiver"
  go sender(clock)
  go receiver(clock)

  var input string
  fmt.Scanln(&input)
}
