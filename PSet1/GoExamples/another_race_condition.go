package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

var (
	CLOSE_A = false

	DATA = make(map[int]bool)
)

func main() {
	rand.Seed(time.Now().Unix())

	if len(os.Args) < 3 {
		return
	}

	min, _ := strconv.Atoi(os.Args[1])
	max, _ := strconv.Atoi(os.Args[2])
	if min > max {
		return
	}

	A := make(chan int)
	B := make(chan int)

	go first(min, max, A)
	go second(B, A)
	third(B)
}

func first(min, max int, out chan<- int) {
	for {
		if CLOSE_A {
			close(out)
			return
		}
		out <- random(min, max)
	}
}

func second(out chan<- int, in <-chan int) {
	for x := range in {
		fmt.Print(x, " ")
		_, ok := DATA[x]
		if ok {
			CLOSE_A = true
		} else {
			DATA[x] = true
			out <- x
		}
	}
	fmt.Println()
	close(out)
}

func third(in <-chan int) {
	var sum int
	sum = 0

	for x2 := range in {
		sum = sum + x2
	}
	fmt.Println("Sum is ", sum)
}

func random(min, max int) int {
	return rand.Intn(max-min) + min
}
