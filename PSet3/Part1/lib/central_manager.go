package lib

type CentralManager struct {
	PChannels           []chan Message
	Incoming            chan Message
	ConfirmationChan    chan Message
	Entries             map[int]CMEntry  // {[pageId]: {CopyArray, Data}}
	InvalidationCounter map[int]int      // {[pageId]: {number of confirmations}}
	RequestMap          map[int]struct { // {[pageId] : {status, queue}}
		Status RequestStatus
		Queue  []Message
	}
	Debug bool
}

type CMEntry struct {
	CopyArray []int
	Owner     int
}

func (c *CentralManager) Start() {
	c.log(true, "starting %v", "test")
	// ticker to make regular requests
	for {
		select {
		case m := <-c.Incoming:
			c.EnqueueRequest(m)
		case m := <-c.ConfirmationChan:

			c.log(true, "%v message (%v) from %v for pageId %v", MESSAGE_TYPES[m.Type], m.Type, m.Sender, m.PageId)

			c.HandleMessage(m)
		default:
			for pageId := range c.RequestMap {
				if c.RequestMap[pageId].Status == Idle && len(c.RequestMap[pageId].Queue) > 0 {
					queuedMessage := c.RequestMap[pageId].Queue[0]
					c.HandleMessage(queuedMessage)
				}
			}
		}
	}
}

func (c *CentralManager) EnqueueRequest(m Message) {
	// Enqueues any message into the request map
	pageStatus, ok := c.RequestMap[m.PageId]
	if !ok {
		c.RequestMap[m.PageId] = struct {
			Status RequestStatus
			Queue  []Message
		}{Status: Idle, Queue: []Message{m}}
		return
	}
	pageStatus.Queue = append(pageStatus.Queue, m)
	c.RequestMap[m.PageId] = pageStatus
	c.log(true, "queued %v's request for pageId: %v. requestMap is %v", m.Sender, m.PageId, c.RequestMap)
}

func (c *CentralManager) HandleMessage(m Message) {
	c.log(true, "Request map is: %v", c.RequestMap)
	c.log(true, "entry map when %v received is %v", MESSAGE_TYPES[m.Type], c.Entries)

	c.log(true, "%v message (%v) from %v for pageId %v", MESSAGE_TYPES[m.Type], m.Type, m.Sender, m.PageId)

	switch m.Type {
	case WriteRequest:
		// check status pageId: might be pending write request
		pageStatus, ok := c.RequestMap[m.PageId]
		if ok {
			if pageStatus.Status != Idle {
				break
			}
		}
		pageStatus.Status = PendingWriteConfirmation
		c.RequestMap[m.PageId] = pageStatus
		//entry doesn't exist? send write
		if _, ok := c.Entries[m.PageId]; !ok {
			c.Entries[m.PageId] = CMEntry{CopyArray: []int{}, Owner: m.Sender}        //set new owner
			c.PChannels[m.Sender] <- Message{Type: VariableForward, PageId: m.PageId} //send the pageVariable to alow the write
			c.log(true, "page variable sent to %v", m.Sender)
			break
		}
		//invalidate copies? else send write forward
		cmEntry := c.Entries[m.PageId]

		if len(cmEntry.CopyArray) != 0 {
			//send invalidate copies
			c.InvalidationCounter[m.PageId] = len(cmEntry.CopyArray)
			for i := 0; i < len(cmEntry.CopyArray); i++ {
				if cmEntry.CopyArray[i] == m.Sender {
					//don't invalidate the requester
					c.ConfirmationChan <- Message{Type: InvalidateConfirmation, PageId: m.PageId}
					continue
				}
				go func(copyHolder int, channels []chan Message) {
					channels[cmEntry.CopyArray[copyHolder]] <- Message{Type: InvalidateCopy, PageId: m.PageId}
				}(i, c.PChannels)
			}
			break
		}
		c.PChannels[cmEntry.Owner] <- Message{Sender: m.Sender, Type: WriteForward, PageId: m.PageId} //send the writeForward request to owner

	case InvalidateConfirmation:
		//do nothing if waiting for more confirmation, else send write forward
		c.InvalidationCounter[m.PageId]--
		if c.InvalidationCounter[m.PageId] != 0 {
			break
		}
		cmEntry := c.Entries[m.PageId]
		c.PChannels[cmEntry.Owner] <- Message{
			Sender: c.RequestMap[m.PageId].Queue[0].Sender,
			Type:   WriteForward,
			PageId: m.PageId} //send the writeForward request to owner

		cmEntry.CopyArray = []int{}
		c.Entries[m.PageId] = cmEntry
		c.log(true, "Entry map: %v after sending writeforward to %v", c.Entries, cmEntry.Owner)
	case ReadRequest:
		pageStatus, ok := c.RequestMap[m.PageId]
		if ok {
			if pageStatus.Status != Idle {
				c.log(true, "status not idle")
				// exit if status is not idle - might be redundant
				break
			}
		}
		cmEntry, ok := c.Entries[m.PageId]
		if !ok {
			c.log(false, "error, pagedId %v not found", m.PageId)
			request, _ := c.RequestMap[m.PageId]
			request.Queue = request.Queue[1:]
			c.RequestMap[m.PageId] = request
			c.PChannels[m.Sender] <- Message{Type: PageNotFoundError, PageId: m.PageId}
			break
		}
		pageStatus.Status = PendingReadConfirmation
		c.RequestMap[m.PageId] = pageStatus
		//send read forward or error
		c.PChannels[cmEntry.Owner] <- Message{Sender: m.Sender, Type: ReadForward, PageId: m.PageId} //send the writeForward request to owner

	case WriteConfirmation:
		// add to page map
		cmEntry, ok := c.Entries[m.PageId]
		if !ok {
			c.log(false, "error, pagedId %v not found", m.PageId)
			break
		}
		// update owner
		cmEntry.Owner = m.Sender
		c.Entries[m.PageId] = cmEntry
		//update request status for that pageId
		request, ok := c.RequestMap[m.PageId]
		c.log(true, "request map before removing %v: %v", m.Sender, c.RequestMap)
		request.Queue = request.Queue[1:]
		request.Status = Idle
		c.RequestMap[m.PageId] = request
		c.log(true, "request map: %v", c.RequestMap)
		c.log(false, "updated entries map to %v", c.Entries)
	case ReadConfirmation:
		//add to page map
		cmEntry, ok := c.Entries[m.PageId]
		if !ok {
			c.log(false, "error, pagedId %v not found", m.PageId)
			break
		}
		// update owner
		cmEntry.CopyArray = append(cmEntry.CopyArray, m.Sender)
		c.Entries[m.PageId] = cmEntry
		c.log(false, "updated entry map %v ", c.Entries)
		// set pageId request to Idle
		request, ok := c.RequestMap[m.PageId]
		request.Queue = request.Queue[1:]
		request.Status = Idle
		c.RequestMap[m.PageId] = request
	}
}
