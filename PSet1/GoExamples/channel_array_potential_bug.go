package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	arr := []uint{3, 4, 5, 6, 1, 2}
	c := make(chan []uint)

	var wg sync.WaitGroup
	go func(c chan []uint, wg *sync.WaitGroup) {
		wg.Add(1)
		defer wg.Done()

		nArr := <-c
		fmt.Printf("%p %v\n", nArr, nArr)

		nArr[1] = 0
	}(c, &wg)

	// Need to copy first and send copy
	//nArr := make([]uint, len(arr))
	//copy(nArr, arr)
	//c <- nArr
	c <- arr
	time.Sleep(time.Second)
	fmt.Printf("%p %v\n", arr, arr)

	wg.Wait()
}
