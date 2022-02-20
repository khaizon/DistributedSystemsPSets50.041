package main

import (
  "fmt"
)

func f(n int, search []int, key int) {
  for i := 0; i < len(search); i++ {
    fmt.Println(n, ":", i, "is searching")
    if search[i]==key {
			fmt.Println(n, ":", i, " search is successful");
			return;
    }
  }
	fmt.Println("search for ", n, " is unsuccessful");
}

func main() {
	search := []int{1,2,34,5,6,9,10,7,11,12};
	var key int = 11;
  for n := 0; n < 10; n++ {
		// create a go routine that executes asynchronously
    go f(n, search, key)
  }
  var input string
	// wait for the input, as otherwise, the program will not wait
  fmt.Scanln(&input)
}
