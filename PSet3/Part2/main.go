package main

import (
	"fmt"

	lib "main/lib"
)

const NUM_OF_VARIABLES = 4
const NUM_OF_PROCESSORS = 10
const NUM_OF_CENTRAL_MANAGERS = 4
const TIMEOUT_DURATION = 5

func main() {
	processorChannels := make([]chan lib.Message, NUM_OF_PROCESSORS)
	for i := 0; i < NUM_OF_PROCESSORS; i++ {
		processorChannels[i] = make(chan lib.Message, 10*NUM_OF_PROCESSORS)
	}

	cmArray := startCentralManagers(NUM_OF_CENTRAL_MANAGERS, processorChannels)

	for i := 0; i < NUM_OF_PROCESSORS; i++ {
		p := lib.Processor{
			Id:              i,
			CentralManagers: cmArray,
			PrimaryCM:       0,
			Channels:        processorChannels,
			NumOfVariables:  NUM_OF_VARIABLES,
			RequestMap:      map[int]lib.RequestStatus{},
			Cache:           map[int]lib.PageCache{},
			Debug:           false,
			TimeoutDur:      TIMEOUT_DURATION,
		}
		go p.Start()
	}
	fmt.Scanln()
}

func startCentralManagers(numOfCentralMangers int, processorChannels []chan lib.Message) []*(lib.CentralManager) {
	cmArray := make([]*(lib.CentralManager), numOfCentralMangers)

	// make channels
	cmIncomingChanArray := make([]chan lib.Message, numOfCentralMangers)
	cmConfirmationChanArray := make([]chan lib.Message, numOfCentralMangers)
	for i := 0; i < numOfCentralMangers; i++ {
		cmIncomingChanArray[i] = make(chan lib.Message, len(processorChannels)*10)
		cmConfirmationChanArray[i] = make(chan lib.Message, len(processorChannels)*10)
	}

	// make Central Mangers
	for i := 0; i < numOfCentralMangers; i++ {
		cm := lib.CentralManager{
			Id:                          i,
			Incoming:                    cmIncomingChanArray[i],
			ConfirmationChan:            cmConfirmationChanArray[i],
			CentralManagersIncoming:     cmIncomingChanArray,
			CentralManagersConfirmation: cmConfirmationChanArray,
			CurrentState: lib.State{
				InvalidationCounter: map[int]int{},
				Entries:             map[int]lib.CMEntry{},
				RequestMap: map[int]struct {
					Status lib.RequestStatus
					Queue  []lib.Message
				}{},
			},
			PChannels:        processorChannels,
			IsPrimary:        i == 0,
			Debug:            false,
			CountDownToDeath: 100 + 100*i,
		}
		cmArray[i] = &cm

		go cm.Start()
	}

	//return array containing addresses
	return cmArray
}
