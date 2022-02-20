package main

import (
  "fmt"
  "time"
	"math/rand"
)

//func another_sender(clock chan int) {
//  for i := -1; ; i--{
		// send a negative integer 
//    clock <- i
//  }
//}

func sender(clock chan int) {
  for i := 0; ; i++ {
		// send an integer 
    fmt.Println("Sending ", i)
    clock <- i
		//sleep sporadically
    amt := time.Duration(rand.Intn(1000))
    time.Sleep(time.Millisecond * amt)
  }
}

func receiver(clock chan int) {
  for {
		// receive integer from channel and print it
    msg := <- clock
    fmt.Println("Receiving ", msg)
		//sleep sporadically
  }
}

func main() {
	// create a channel of type integer
  var clock chan int = make(chan int)

	// launch two go routines "sender" and "receiver"
  go sender(clock)
  //go another_sender(clock)
  go receiver(clock)

  var input string
  fmt.Scanln(&input)
}
