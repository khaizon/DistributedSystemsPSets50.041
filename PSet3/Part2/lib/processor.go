package lib

import (
	"math/rand"
	"time"
)

type Processor struct {
	Id                   int
	CentralManagers      []*CentralManager
	PrimaryCM            int
	Channels             []chan Message
	NumOfVariables       int
	RequestMap           map[int]RequestStatus
	Cache                map[int]PageCache
	Debug                bool
	TimeoutChan          chan int
	TimeoutDur           int
	InElection           bool
	AcknowledgementArray []int
}

type PageCache struct {
	// {pageId: {isOwner, isValid, Data}}
	IsOwner bool
	IsValid bool
	Data    int
}

func (p *Processor) Start() {
	// ticker to make regular requests

	reqTicker := time.NewTicker(time.Duration(rand.Intn(p.TimeoutDur * int(time.Second))))
	p.TimeoutChan = make(chan int, p.NumOfVariables) //fault tolerant mod: add timeout to requests
	for {
		select {
		case <-reqTicker.C:
			if p.InElection {
				break
			}
			p.SendRandomRequest()
			go p.StartRequestTimer()

		case <-p.TimeoutChan:
			p.HandleTimeout()

		case m := <-p.Channels[p.Id]:
			p.log(true, "%v message received", m.Type.toString())
			p.HandleMessage(m)
		}
	}
}

func (p *Processor) HandleMessage(m Message) {
	p.log(true, "%v message (%v) from %v for pageId %v", m.Type.toString(), m.Type, m.Sender, m.PageId)

	switch m.Type {
	case WRITE_FORWARD:
		// invalidate my cache
		p.Cache[m.PageId] = PageCache{IsOwner: false, IsValid: false}
		p.log(false, "cache for pageId %v invalidated", m.PageId)
		p.Channels[m.Sender] <- Message{Type: PAGE_TO_WRITE, PageId: m.PageId}

	case READ_FORWARD:
		// send PAGE_COPY_FORWARD message to forwardee
		content := p.Cache[m.PageId].Data
		p.Channels[m.Sender] <- Message{Type: PAGE_COPY_FORWARD, PageId: m.PageId, Content: content}

	case PAGE_COPY_FORWARD:
		// send read confirmation to CM
		p.Cache[m.PageId] = PageCache{IsOwner: false, IsValid: true, Data: m.Content}
		p.RequestMap[m.PageId] = RequestStatus{Timestamp: 0, State: IDLE}
		p.CentralManagers[p.PrimaryCM].ConfirmationChan <- Message{Sender: p.Id, Type: READ_CONFIRMATION, PageId: m.PageId}

	case INVALIDATE_COPY:
		// update cache map
		p.Cache[m.PageId] = PageCache{IsOwner: false, IsValid: false}
		p.CentralManagers[p.PrimaryCM].ConfirmationChan <- Message{Sender: p.Id, Type: INVALIDATE_CONFIRMATION, PageId: m.PageId}
		p.log(false, "cache for pageId %v invalidated", m.PageId)

	case PAGE_TO_WRITE:
		// write to variable and send confirmation to CM
		p.RequestMap[m.PageId] = RequestStatus{Timestamp: 0, State: IDLE}
		p.CentralManagers[p.PrimaryCM].ConfirmationChan <- Message{Sender: p.Id, Type: WRITE_CONFIRMATION, PageId: m.PageId}

	case PAGE_NOT_FOUND:
		p.log(true, "%v received, resetting request status to IDLE", MESSAGE_TYPES[PAGE_NOT_FOUND])
		p.RequestMap[m.PageId] = RequestStatus{Timestamp: 0, State: IDLE}

	case START_ELECTION:
		p.HandleStartElection(m)

	case ACKNOWLEDGE:
		p.HandleAcknowledge(m)

	case ANNOUNCE_PRIMARY:
		p.HandleAnnouncePrimary(m)
	}
}

func (p *Processor) SendRandomRequest() {
	pageId := rand.Intn(p.NumOfVariables) // choose random page to read or write to
	readOrWrite := rand.Intn(2)           // randomly read or write

	if p.RequestMap[pageId].State != IDLE || //don't make any request if pending reply
		p.Cache[pageId].IsOwner || //don't make any READ or WRITE request if owner
		(readOrWrite == 0 && p.Cache[pageId].IsValid) { //don't make READ request if cache is valid
		return
	}

	requestState := PENDING_READ_COMPLETION
	// if readOrWrite == 1 => write operation => set self to owner
	if readOrWrite == 1 {
		p.Cache[pageId] = struct {
			IsOwner bool
			IsValid bool
			Data    int
		}{true, true, p.Id}
		requestState = PENDING_WRITE_COMPLETION
	}

	request := Message{ // make request
		Sender:  p.Id,
		Type:    MessageType(readOrWrite), // 0 = READ_REQUEST, 1 = WRITE_REQUEST
		PageId:  pageId,
		Content: p.Id,
	}
	p.CentralManagers[p.PrimaryCM].Incoming <- request //send request

	p.RequestMap[pageId] = RequestStatus{Timestamp: time.Now().UnixNano(), State: requestState, Message: request}
	p.log(false, "%v request (%v) sent for page Id %v", MESSAGE_TYPES[readOrWrite], MessageType(readOrWrite), pageId)
}

func (p *Processor) StartRequestTimer() {
	/**
	after a duration, do a request timeout
	*/
	time.Sleep(time.Duration(p.TimeoutDur) * time.Second)
	p.TimeoutChan <- 1
}

func (p *Processor) HandleTimeout() {
	/**
	after a timeout, check for requests that are NOT IDLE, and have exceeded specified timeout duration
	if request is not Idle and exceeded timeout duration => start election
	*/

	if p.InElection {
		// check for acknowledgement messages and elect
		p.ElectNewCM()
		return
	}
	// p.InElection = true

	for key := range p.RequestMap {
		requestState := p.RequestMap[key]
		if requestState.State == IDLE {
			// Don't check if request is in idle mode (means the request has completed)
			continue
		}

		//Check whether : currentTime >= requestTimestamp + timeoutDuration
		if time.Now().UnixNano() >= requestState.Timestamp+int64(p.TimeoutDur*int(time.Second)) {
			p.log(false, "timeout for pageId: %v, operation: %v", requestState.Message.PageId, requestState.Message.Type.toString())
			p.StartElection()
		}
	}
}

func (p *Processor) StartElection() {
	/**
	set state to be in election mode
	*/
	if p.InElection {
		// do nothing if already in election mode
		return
	}

	p.log(false, "broadcasting election")
	for i := 0; i < len(p.Channels); i++ {
		p.Channels[i] <- Message{Sender: p.Id, Type: START_ELECTION}
	}
}

func (p *Processor) HandleStartElection(m Message) {
	/**
	Loop through list of CMs
	Reply id of the first CM that is alive to the initating processor
	*/

	if p.InElection {
		// do nothing if already in election mode
		return
	}

	p.AcknowledgementArray = nil
	p.InElection = true
	p.log(false, "starting election")

	// for i := 0; i < len(p.CentralManagers); i++ {
	// 	go func(n int) {
	// 		p.CentralManagers[n].ConfirmationChan <- Message{Sender: p.Id, Type: CHECK_ALIVE}
	// 	}(i)
	// }

	if len(p.Channels) == p.Id+1 { //current channel has highest Id
		//do something
		for i := 0; i < len(p.CentralManagers); i++ {
			go func(n int) {
				p.log(false, "test sending to CM%v", n)
				p.CentralManagers[n].ConfirmationChan <- Message{Sender: p.Id, Type: CHECK_ALIVE}
				p.log(false, "test sent to CM%v", n)
			}(i)
			p.log(false, "before timeout")
			go p.StartRequestTimer()

		}
	}
}

func (p *Processor) HandleAcknowledge(m Message) {
	p.log(false, "CM %v alive, chosen", m.Sender)
	p.AcknowledgementArray = append(p.AcknowledgementArray, m.Sender)
}

func (p *Processor) HandleAnnouncePrimary(m Message) {
	/**
	1. Set primary CM to announcer
	2. Sent pending requests again
	*/
	p.PrimaryCM = m.Sender
	p.InElection = false
	for pageId := range p.RequestMap {
		requestStatus := p.RequestMap[pageId]
		if requestStatus.State == IDLE {
			continue
		}

		p.CentralManagers[p.PrimaryCM].Incoming <- requestStatus.Message
		p.RequestMap[pageId] = RequestStatus{Timestamp: time.Now().UnixNano(), State: requestStatus.State, Message: requestStatus.Message}
		go p.StartRequestTimer()
	}
}

func (p *Processor) ElectNewCM() {
	if len(p.AcknowledgementArray) == 0 {
		p.log(false, "no responses gg ")
		return
	}
	newCM := p.AcknowledgementArray[0]
	for i := 1; i < len(p.AcknowledgementArray); i++ {
		if p.AcknowledgementArray[i] < newCM {
			newCM = p.AcknowledgementArray[i]
		}
	}
	p.log(false, "new Cm elected is CM %v", newCM)
	p.CentralManagers[newCM].ConfirmationChan <- Message{Sender: p.Id, Type: ELECT}
}
