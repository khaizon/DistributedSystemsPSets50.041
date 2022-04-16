package lib

import (
	"math/rand"
	"time"
)

type Processor struct {
	Id              int
	CentralManagers []*CentralManager
	PrimaryCM       int
	Channels        []chan Message
	NumOfVariables  int
	RequestMap      map[int]RequestStatus
	Cache           map[int]PageCache
	Debug           bool
}

type PageCache struct {
	// {pageId: {isOwner, isValid, Data}}
	IsOwner bool
	IsValid bool
	Data    int
}

func (p *Processor) Start() {
	// ticker to make regular requests
	reqTicker := time.NewTicker(1 * time.Second)
	requestTimeout := make(chan int, 5) //fault tolerant mod: add timeout to requests
	for {
		select {
		case <-reqTicker.C:
			p.SendRandomRequest()
		case <-requestTimeout:
			// do timeout thing
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
		p.RequestMap[m.PageId] = IDLE
		p.CentralManagers[p.PrimaryCM].ConfirmationChan <- Message{Sender: p.Id, Type: READ_CONFIRMATION, PageId: m.PageId}

	case INVALIDATE_COPY:
		// update cache map
		p.Cache[m.PageId] = PageCache{IsOwner: false, IsValid: false}
		p.CentralManagers[p.PrimaryCM].ConfirmationChan <- Message{Sender: p.Id, Type: INVALIDATE_CONFIRMATION, PageId: m.PageId}
		p.log(false, "cache for pageId %v invalidated", m.PageId)

	case PAGE_TO_WRITE:
		// write to variable and send confirmation to CM
		p.RequestMap[m.PageId] = IDLE
		p.CentralManagers[p.PrimaryCM].ConfirmationChan <- Message{Sender: p.Id, Type: WRITE_CONFIRMATION, PageId: m.PageId}

	case PAGE_NOT_FOUND:
		p.log(true, "%v received, resetting request status to IDLE", MESSAGE_TYPES[PAGE_NOT_FOUND])
		p.RequestMap[m.PageId] = IDLE
	}
}

func (p *Processor) SendRandomRequest() {
	pageId := rand.Intn(p.NumOfVariables) // choose random page to read or write to
	readOrWrite := rand.Intn(2)           // randomly read or write

	if p.RequestMap[pageId] == PENDING_REQUEST_COMPLETION || //don't make any request if pending reply
		p.Cache[pageId].IsOwner || //don't make any READ or WRITE request if owner
		(readOrWrite == 0 && p.Cache[pageId].IsValid) { //don't make READ request if cache is valid
		return
	}

	if readOrWrite == 1 {
		p.Cache[pageId] = struct {
			IsOwner bool
			IsValid bool
			Data    int
		}{true, true, p.Id}
	}
	p.CentralManagers[p.PrimaryCM].Incoming <- Message{ // make request
		Sender:  p.Id,
		Type:    MessageType(readOrWrite), // 0 = READ_REQUEST, 1 = WRITE_REQUEST
		PageId:  pageId,
		Content: p.Id,
	}

	p.RequestMap[pageId] = PENDING_REQUEST_COMPLETION
	p.log(false, "%v request (%v) sent for page Id %v", MESSAGE_TYPES[readOrWrite], MessageType(readOrWrite), pageId)
}
