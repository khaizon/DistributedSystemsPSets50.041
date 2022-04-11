package main

import (
	"fmt"

	lib "github.com/khaizon/DistributedSystemsPSets50.041/blob/main/PSet3/Part1/lib"
)

func main() {
	cmIncomingChan := make(chan lib.Message, lib.NUM_OF_PROCESSORS)
	cmConfirmationChan := make(chan lib.Message, lib.NUM_OF_PROCESSORS)
	processorChannels := make([]chan lib.Message, lib.NUM_OF_PROCESSORS)
	for i := 0; i < lib.NUM_OF_PROCESSORS; i++ {
		processorChannels[i] = make(chan lib.Message, lib.NUM_OF_PROCESSORS)
	}
	cm := lib.CentralManager{
		Debug:               true,
		Incoming:            cmIncomingChan,
		ConfirmationChan:    cmConfirmationChan,
		InvalidationCounter: map[int]int{},
		Entries:             map[int]lib.CMEntry{},
		RequestMap: map[int]struct {
			Status lib.RequestStatus
			Queue  []lib.Message
		}{},
		PChannels: processorChannels,
	}
	go cm.Start()
	for i := 0; i < lib.NUM_OF_PROCESSORS; i++ {
		p := lib.Processor{
			Id:                 i,
			CMRequestChan:      cmIncomingChan,
			CMConfirmationChan: cmConfirmationChan,
			Channels:           processorChannels,
			NumOfVariables:     lib.NUM_OF_VARIABLES,
			RequestMap:         map[int]lib.RequestStatus{},
			Cache: map[int]struct {
				IsValid bool
				Data    int
			}{},
			Debug: true,
		}
		go p.Start()
	}
	fmt.Scanln()
}
