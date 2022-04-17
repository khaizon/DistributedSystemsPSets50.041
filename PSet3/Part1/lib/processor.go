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
	Debug bool
}

func (p *Processor) Start() {
	// ticker to make regular requests
	reqTicker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-reqTicker.C:
			pageId := rand.Intn(p.NumOfVariables) // choose random page to read or write to
			readOrWrite := rand.Intn(2)           // randomly read or write

			if p.RequestMap[pageId] == PendingRequestCompletion || //don't make any request if pending reply
				p.Cache[pageId].IsOwner || //don't make any READ or WRITE request if owner
				(readOrWrite == 0 && p.Cache[pageId].IsValid) { //don't make READ request if cache is valid
				break
			}

			if readOrWrite == 1 {
				p.Cache[pageId] = struct {
					IsOwner bool
					IsValid bool
					Data    int
				}{true, true, p.Id}
			}

			p.CMRequestChan <- Message{ // make request
				Sender:  p.Id,
				Type:    MessageType(readOrWrite), // 0 = readRequest, 1 = writeRequest
				PageId:  pageId,
				Content: p.Id,
			}

			p.RequestMap[pageId] = PendingRequestCompletion
			p.log(false, "%v request (%v) sent for page Id %v", MESSAGE_TYPES[readOrWrite], MessageType(readOrWrite), pageId)
		case m := <-p.Channels[p.Id]:
			p.log(true, "%v message received", MESSAGE_TYPES[m.Type])
			p.HandleMessage(m)
		}
	}
}

func (p *Processor) HandleMessage(m Message) {
	p.log(true, "%v message (%v) from %v for pageId %v", MESSAGE_TYPES[m.Type], m.Type, m.Sender, m.PageId)

	switch m.Type {
	case WriteForward:
		// invalidate my cache
		p.Cache[m.PageId] = struct {
			IsOwner bool
			IsValid bool
			Data    int
		}{IsOwner: false, IsValid: false}
		p.log(false, "cache for pageId %v invalidated", m.PageId)
		p.Channels[m.Sender] <- Message{Type: VariableForward, PageId: m.PageId}
	case ReadForward:
		// send pageForward message to forwardee
		content := p.Cache[m.PageId].Data
		p.Channels[m.Sender] <- Message{Type: PageForward, PageId: m.PageId, Content: content}
	case PageForward:
		// send read confirmation to CM
		p.Cache[m.PageId] = struct {
			IsOwner bool
			IsValid bool
			Data    int
		}{IsOwner: false, IsValid: true, Data: m.Content}
		p.RequestMap[m.PageId] = Idle
		p.CMConfirmationChan <- Message{Sender: p.Id, Type: ReadConfirmation, PageId: m.PageId}
	case InvalidateCopy:
		// update cache map
		p.Cache[m.PageId] = struct {
			IsOwner bool
			IsValid bool
			Data    int
		}{IsOwner: false, IsValid: false}
		p.CMConfirmationChan <- Message{Sender: p.Id, Type: InvalidateConfirmation, PageId: m.PageId}
		p.log(false, "cache for pageId %v invalidated", m.PageId)
	case VariableForward:
		// write to variable and send confirmation to CM
		p.RequestMap[m.PageId] = Idle
		p.CMConfirmationChan <- Message{Sender: p.Id, Type: WriteConfirmation, PageId: m.PageId}
	case PageNotFoundError:
		p.log(true, "%v received, resetting request status to idle", MESSAGE_TYPES[PageNotFoundError])
		p.RequestMap[m.PageId] = Idle
	}
}
