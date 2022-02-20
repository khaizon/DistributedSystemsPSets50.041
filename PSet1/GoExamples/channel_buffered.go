package main

import (
  "fmt"
  "time"
	"math/rand"
)


func sender(id int, clock chan int) {
	send := []int{1,1,1,1,1,1,1,1,1,1};
  for i := 0; ; i++ {
		// send an integer 
		msg := (send[i%10]+i);
		fmt.Println(id, " Sending ", msg)
    clock <- msg;
		//sleep sporadically
    amt := time.Duration(rand.Intn(500))
    time.Sleep(time.Millisecond * amt)
		//careful about this update
		//send[id] = send[id]+1
  }
}

func receiver(clock chan int) {
  for {
		// receive integer from channel and print it
    msg := <- clock
    fmt.Println("Receiving ", msg)
		//sleep sporadically
    //amt := time.Duration(rand.Intn(2000))
    //time.Sleep(time.Millisecond * amt)
  }
}

func main() {
	// create a channel of type integer array, size = 1
  var clock chan int = make(chan int, 10)

	// launch two go routines "sender" and "receiver"
  go sender(0, clock)
  go sender(1, clock)
  go receiver(clock)

  var input string
  fmt.Scanln(&input)
	close(clock)
}
