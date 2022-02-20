package main

import (
	"fmt"
	"time"
)

const (
	maxGoroutines = 10
)

func main() {
	for i := 0; i < maxGoroutines; i++ {
		go func() {
			fmt.Println(i)
		}()
	}
	time.Sleep(time.Second)
}
