package main

import (
	"fmt"
	"math/rand"
	"time"
)

type CentralManager struct {
	Debug     bool
	Receiving chan Message
	Entries   map[int]struct { //int = pageId
		CopySet []int
		Owner   int
	}
}

type Processor struct {
	Id             int
	CMChannel      chan Message
	NumOfVariables int
	RequestMap     map[int]RequestStatus
	Cache          map[int]struct { // {pageId: {isValid, Data}}
		IsValid bool
		Data    int
	}
}

type RequestStatus int

const (
	Idle RequestStatus = iota
	PendingPageForward
	PendingVariableForward
)

type Message struct {
	Sender  int
	Type    MessageType
	PageId  int
	Content int
}

type MessageType int

const (
	ReadRequest MessageType = iota
	WriteRequest
	ReadForward
	PageForward
	ReadConfirmation
	InvalidateCopy
	InvalidateConfirmation
	WriteForward
	VariableForward
	WriteConfirmation
)

const NUM_OF_VARIABLES = 10

func main() {
	cm := CentralManager{
		Debug:     true,
		Receiving: make(chan Message),
		Entries: map[int]struct {
			CopySet []int
			Owner   int
		}{},
	}
	cm.Start()
}

func (p *Processor) Start() {
	// ticker to make regular requests
	reqTicker := time.NewTicker(2 * time.Second)
	select {
	case <-reqTicker.C:
		pageId := rand.Intn(p.NumOfVariables) // choose random page to read or write to
		readOrWrite := rand.Intn(1)           // randomly read or write

		if p.RequestMap[pageId] == PendingPageForward || p.RequestMap[pageId] == PendingVariableForward {
			//don't make any request if pending reply
			break
		}
		p.CMChannel <- Message{ // make request
			Sender: p.Id,
			Type:   MessageType(readOrWrite), // 0 = readRequest, 1 = writeRequest
			PageId: pageId,
		}
	case m := <-p.CMChannel:
		p.HandleMessage(m)
	}
}

func (p *Processor) HandleMessage(m Message) {
	switch m.Type {
	case WriteForward:
		// invalid my cahace
		// send pageVariable to forwadee
	case ReadForward:
		// send pageForward message to forwardee
	case PageForward:
		// send read confirmation to CM
	case InvalidateCopy:
		// update cache map
	case VariableForward:
		// write to variable and send confirmation to CM
	}
}

func (c *CentralManager) Start() {
	c.log(true, "starting %v", "test")
	// ticker to make regular requests
	select {
	case m := <-c.Receiving:
		c.log(true, "handling message from %v", m.Sender)
		c.HandleMessage(m)
	}
}

func (c *CentralManager) HandleMessage(m Message) {
	switch m.Type {
	case WriteRequest:
		//invalidate copies? else send write forward
	case InvalidateConfirmation:
		//send write forward
	case ReadRequest:
		//send read forward or error
	case WriteConfirmation:
		// add to page map
		//do nothing or process next request
	case ReadConfirmation:
		//add to page map
		// process next request
	}
}

func (c *CentralManager) log(args ...interface{}) {
	debug := args[0].(bool)
	mainString := args[1].(string)
	if c.Debug && debug {
		fmt.Printf("CM: %v", fmt.Sprintf(mainString, args[2:]...))
	}
}
