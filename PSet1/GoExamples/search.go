package main

import (
  "fmt"
  "time"
  "math/rand"
)

func f(n int, search []int, key int) {
  for i := 0; i < len(search); i++ {
    fmt.Println(n, ":", i, "is searching")
    if search[i]==key {
			fmt.Println(n, ":", i, " search is successful");
			return;
    }
    amt := time.Duration(rand.Intn(1000))
    time.Sleep(time.Millisecond * amt)
  }
	fmt.Println("search for ", n, " is unsuccessful");
}

func main() {
	search := []int{1,2,34,5,6,9,10,7,11,12};
	var key int = 11;
  for n := 0; n < 10; n++ {
    go f(n, search, key)
  }
  var input string
  fmt.Scanln(&input)
}
