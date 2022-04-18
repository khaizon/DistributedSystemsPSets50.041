package lib

import (
	"math/rand"
	"time"
)

type Processor struct {
	Id                 int
	CMRequestChan      chan Message
	CMConfirmationChan chan Message
	Channels           []chan Message
	NumOfVariables     int
	RequestMap         map[int]RequestStatus
	Cache              map[int]struct { // {pageId: {isOwner, isValid, Data}}
		IsOwner bool
		IsValid bool
		Data    int
	}
	Debug      bool
	TimeoutDur int
}

func (p *Processor) Start() {
	// ticker to make regular requests
	reqTicker := time.NewTicker(time.Duration(rand.Intn(p.TimeoutDur * int(time.Second))))
	for {
		select {
		case <-reqTicker.C:

			p.SendRandomRequest()

		case m := <-p.Channels[p.Id]:
			p.log(true, "%v message received", MESSAGE_TYPES[m.Type])
			p.HandleMessage(m)
		}
	}
}

func (p *Processor) HandleMessage(m Message) {
	p.log(true, "%v message (%v) from %v for pageId %v", MESSAGE_TYPES[m.Type], m.Type, m.Sender, m.PageId)

	switch m.Type {
	case WRITE_FORWARD:
		// invalidate my cache
		p.Cache[m.PageId] = struct {
			IsOwner bool
			IsValid bool
			Data    int
		}{IsOwner: false, IsValid: false}
		p.log(false, "cache for pageId %v invalidated", m.PageId)
		p.Channels[m.Sender] <- Message{Type: PAGE_TO_WRITE, PageId: m.PageId}
	case READ_FORWARD:
		// send pageForward message to forwardee
		content := p.Cache[m.PageId].Data
		p.Channels[m.Sender] <- Message{Type: PAGE_COPY_FORWARD, PageId: m.PageId, Content: content}
	case PAGE_COPY_FORWARD:
		// send read confirmation to CM
		p.Cache[m.PageId] = struct {
			IsOwner bool
			IsValid bool
			Data    int
		}{IsOwner: false, IsValid: true, Data: m.Content}
		p.RequestMap[m.PageId] = RequestStatus{State: IDLE}
		p.CMConfirmationChan <- Message{Sender: p.Id, Type: READ_CONFIRMATION, PageId: m.PageId}
	case INVALIDATE_COPY:
		// update cache map
		p.Cache[m.PageId] = struct {
			IsOwner bool
			IsValid bool
			Data    int
		}{IsOwner: false, IsValid: false}
		p.CMConfirmationChan <- Message{Sender: p.Id, Type: INVALIDATE_CONFIRMATION, PageId: m.PageId}
		p.log(false, "cache for pageId %v invalidated", m.PageId)
	case PAGE_TO_WRITE:
		// write to variable and send confirmation to CM
		p.RequestMap[m.PageId] = RequestStatus{State: IDLE}
		p.CMConfirmationChan <- Message{Sender: p.Id, Type: WRITE_CONFIRMATION, PageId: m.PageId}
	case PAGE_NOT_FOUND:
		p.log(true, "%v received, resetting request status to idle", MESSAGE_TYPES[PAGE_NOT_FOUND])
		p.RequestMap[m.PageId] = RequestStatus{State: IDLE}
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
	p.CMRequestChan <- request //send request

	p.RequestMap[pageId] = RequestStatus{Timestamp: time.Now().UnixNano(), State: requestState, Message: request}
	p.log(false, "%v request (%v) sent for page Id %v", MESSAGE_TYPES[readOrWrite], MessageType(readOrWrite), pageId)
}
