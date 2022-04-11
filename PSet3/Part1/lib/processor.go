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
	Cache              map[int]struct { // {pageId: {isValid, Data}}
		IsValid bool
		Data    int
	}
	Debug bool
}

func (p *Processor) Start() {
	// ticker to make regular requests
	reqTicker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-reqTicker.C:
			pageId := rand.Intn(p.NumOfVariables) // choose random page to read or write to
			readOrWrite := rand.Intn(2)           // randomly read or write

			if p.RequestMap[pageId] == PendingRequestCompletion {
				//don't make any request if pending reply
				break
			}
			p.Cache[pageId] = struct {
				IsValid bool
				Data    int
			}{true, p.Id}
			p.CMRequestChan <- Message{ // make request
				Sender: p.Id,
				Type:   MessageType(readOrWrite), // 0 = readRequest, 1 = writeRequest
				PageId: pageId,
			}
			p.RequestMap[pageId] = PendingRequestCompletion
			p.log(true, "%v request sent for page Id %v", MESSAGE_TYPES[readOrWrite], pageId)
		case m := <-p.Channels[p.Id]:
			p.log(true, "%v message received", MESSAGE_TYPES[m.Type])
			p.HandleMessage(m)
		}
	}
}

func (p *Processor) HandleMessage(m Message) {
	switch m.Type {
	case WriteForward:
		// invalidate my cache
		p.Cache[m.PageId] = struct {
			IsValid bool
			Data    int
		}{IsValid: false}
		// send pageVariable to forwadee
		p.Channels[m.Sender] <- Message{Type: VariableForward, PageId: m.PageId}
	case ReadForward:
		// send pageForward message to forwardee
		content := p.Cache[m.PageId].Data
		p.Channels[m.Sender] <- Message{Type: VariableForward, PageId: m.PageId, Content: content}
	case PageForward:
		// send read confirmation to CM
		p.Cache[m.PageId] = struct {
			IsValid bool
			Data    int
		}{IsValid: true, Data: m.Content}
		p.RequestMap[m.PageId] = Idle
		p.CMConfirmationChan <- Message{Type: WriteConfirmation, PageId: m.PageId}
	case InvalidateCopy:
		// update cache map
		p.Cache[m.PageId] = struct {
			IsValid bool
			Data    int
		}{IsValid: false}
		p.CMConfirmationChan <- Message{Type: InvalidateConfirmation, PageId: m.PageId}
	case VariableForward:
		// write to variable and send confirmation to CM
		p.RequestMap[m.PageId] = Idle
		p.CMConfirmationChan <- Message{Type: WriteConfirmation, PageId: m.PageId}

	}
}
