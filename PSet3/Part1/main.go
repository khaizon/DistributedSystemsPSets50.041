package main

import (
	"fmt"

	lib "github.com/khaizon/DistributedSystemsPSets50.041/blob/main/PSet3/Part1/lib"
)

const NUM_OF_VARIABLES = 2
const NUM_OF_PROCESSORS = 2

func main() {
	cmIncomingChan := make(chan lib.Message, 10*NUM_OF_PROCESSORS)
	cmConfirmationChan := make(chan lib.Message, 10*NUM_OF_PROCESSORS)
	processorChannels := make([]chan lib.Message, 10*NUM_OF_PROCESSORS)
	for i := 0; i < NUM_OF_PROCESSORS; i++ {
		processorChannels[i] = make(chan lib.Message, 10*NUM_OF_PROCESSORS)
	}
	cm := lib.CentralManager{
		Debug:               false,
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
	for i := 0; i < NUM_OF_PROCESSORS; i++ {
		p := lib.Processor{
			Id:                 i,
			CMRequestChan:      cmIncomingChan,
			CMConfirmationChan: cmConfirmationChan,
			Channels:           processorChannels,
			NumOfVariables:     NUM_OF_VARIABLES,
			RequestMap:         map[int]lib.RequestStatus{},
			Cache: map[int]struct {
				IsOwner bool
				IsValid bool
				Data    int
			}{},
			Debug: false,
		}
		go p.Start()
	}
	fmt.Scanln()
}
