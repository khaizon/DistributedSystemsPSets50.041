package lib

import (
	"encoding/json"
	"time"
)

type CentralManager struct {
	Id                          int
	IsPrimary                   bool
	PChannels                   []chan Message
	CentralManagersIncoming     []chan Message
	CentralManagersConfirmation []chan Message
	Incoming                    chan Message
	ConfirmationChan            chan Message
	CurrentState                State //to track all changes for the secondary replica to take over
	Debug                       bool
	IsAlive                     bool
	CountDownToDeath            int
	FinalCountDownToDeath       int
	Die                         chan int
	ReallyDie                   chan int
}

// State would be sent to the secondary replica everytime it updates its state
type State struct {
	Entries             map[int]CMEntry  // {[pageId]: {CopyArray, Data}}
	InvalidationCounter map[int]int      // {[pageId]: {number of confirmations}}
	RequestMap          map[int]struct { // {[pageId]: {status, queue}}
		Status RequestStatus
		Queue  []Message
	}
}

type CMEntry struct {
	CopyArray []int
	Owner     int
}

func (cm *CentralManager) Start() {
	cm.log(true, "starting %v", "test")
	cm.Die = make(chan int, 20)
	cm.ReallyDie = make(chan int, 20)
	startTime := time.Now().UnixNano()

	// ticker to make regular requests
	for {
		select {
		case <-cm.ReallyDie:
			cm.log(false, "DIED -- Time Elapsed: %v ms", float32((time.Now().UnixNano()-startTime)/int64(time.Millisecond)))
			return
		case <-cm.Die:
			cm.IsAlive = false
			cm.log(false, "dead, ressurecting in 6 seconds")
			go cm.Ressurect()

		case m := <-cm.Incoming:
			if !cm.IsAlive {
				break
			}
			cm.EnqueueRequest(m)
			cm.ForwardState()

		case m := <-cm.ConfirmationChan:
			if !cm.IsAlive {
				break
			}
			cm.log(true, "%v message (%v) from %v for pageId %v", m.Type.toString(), m.Type, m.Sender, m.PageId)
			cm.HandleMessage(m)
			cm.ForwardState()

		default:
			if !cm.IsPrimary || cm.IsAlive {
				break
			}

			pageId := cm.LongestQueue()
			if pageId == -1 {
				// cm.log(false, "breaking cus no available req")
				break
			}
			cm.log(true, "pageId: %v, requestMap: %v", pageId, cm.CurrentState.RequestMap[pageId].Queue)
			queuedMessage := cm.CurrentState.RequestMap[pageId].Queue[0]
			cm.HandleMessage(queuedMessage)
			cm.ForwardState()
		}
	}
}

func (cm *CentralManager) EnqueueRequest(m Message) {
	// Enqueues any message into the request map
	pageStatus, ok := cm.CurrentState.RequestMap[m.PageId]
	if !ok {
		cm.CurrentState.RequestMap[m.PageId] = struct {
			Status RequestStatus
			Queue  []Message
		}{Status: RequestStatus{State: IDLE}, Queue: []Message{m}}
		return
	}

	if isDuplicate := CheckDuplicate(pageStatus.Queue, m); isDuplicate {
		cm.log(false, "found duplicate request, dropping")
		cm.log(false, "isPrimary: %v", cm.IsPrimary)
		return
	}

	pageStatus.Queue = append(pageStatus.Queue, m)
	cm.CurrentState.RequestMap[m.PageId] = pageStatus
	cm.log(true, "queued %v's request for pageId: %v. requestMap is %v", m.Sender, m.PageId, cm.CurrentState.RequestMap)
}

func (cm *CentralManager) HandleMessage(m Message) {
	cm.log(true, "Request map is: %v", cm.CurrentState.RequestMap)
	cm.log(true, "entry map when %v received is %v", m.Type.toString(), cm.CurrentState.Entries)
	cm.log(true, "%v message (%v) from %v for pageId %v", m.Type.toString(), m.Type, m.Sender, m.PageId)

	if cm.IsPrimary {
		if cm.CountDownToDeath--; cm.CountDownToDeath <= 0 {
			cm.Die <- 1
		}
		if cm.FinalCountDownToDeath--; cm.FinalCountDownToDeath <= 0 {
			cm.ReallyDie <- 1
		}
	}

	switch m.Type {
	case READ_REQUEST:
		cm.HandleReadReqeuest(m)
	case WRITE_REQUEST:
		cm.HandleWriteRequest(m)
	case READ_CONFIRMATION:
		cm.HandleReadConfirmation(m)
	case WRITE_CONFIRMATION:
		cm.HandleWriteConfirmation(m)
	case INVALIDATE_CONFIRMATION:
		cm.HandleInvalidateConfirmation(m)
	case FORWARD_STATE:
		cm.HandleForwardState(m)
	case ELECT:
		cm.HandleElect(m)
	case ANNOUNCE_PRIMARY:
		cm.HandleAnnouncePrimary(m)
	case CHECK_ALIVE:
		cm.PChannels[m.Sender] <- Message{Sender: cm.Id, Type: ACKNOWLEDGE}
	}
}

func (cm *CentralManager) HandleReadConfirmation(m Message) {
	/**
	1. Error handling - return if pageId not foung
	2. Adds sender of READ_CONFIRMATION to CopyArray to track who has a copy of the info
	3. Remove request from the queue, and update request state for that given pageId to IDLE.
	*/

	cmEntry, ok := cm.CurrentState.Entries[m.PageId]
	if !ok {
		cm.log(false, "error, pagedId %v not found", m.PageId)
		return
	}
	// update copy array
	cmEntry.CopyArray = append(cmEntry.CopyArray, m.Sender)
	cm.CurrentState.Entries[m.PageId] = cmEntry
	cm.log(false, "updated entry map %v ", cm.CurrentState.Entries)

	// set pageId request to IDLE
	request, ok := cm.CurrentState.RequestMap[m.PageId]
	request.Queue = request.Queue[1:]
	request.Status = RequestStatus{State: IDLE}
	cm.CurrentState.RequestMap[m.PageId] = request
}

func (cm *CentralManager) HandleWriteConfirmation(m Message) {
	// add to page map
	cmEntry, ok := cm.CurrentState.Entries[m.PageId]
	if !ok {
		cm.log(false, "error, pagedId %v not found", m.PageId)
		return
	}
	// update owner
	cmEntry.Owner = m.Sender
	cm.CurrentState.Entries[m.PageId] = cmEntry
	//update request status for that pageId
	request, ok := cm.CurrentState.RequestMap[m.PageId]
	cm.log(false, "request map before removing %v: %v", m.Sender, cm.CurrentState.RequestMap)
	request.Queue = request.Queue[1:]
	request.Status = RequestStatus{State: IDLE}
	cm.CurrentState.RequestMap[m.PageId] = request
	cm.log(false, "request map: %v", cm.CurrentState.RequestMap)
	cm.log(false, "updated entries map to %v", cm.CurrentState.Entries)
}

func (cm *CentralManager) HandleReadReqeuest(m Message) {
	pageStatus, ok := cm.CurrentState.RequestMap[m.PageId]
	if ok {
		if pageStatus.Status.State != IDLE {
			cm.log(true, "status not IDLE")
			// exit if status is not IDLE - might be redundant
			return
		}
	}
	cmEntry, ok := cm.CurrentState.Entries[m.PageId]
	if !ok {
		cm.log(false, "error, pagedId %v not found", m.PageId)
		request, _ := cm.CurrentState.RequestMap[m.PageId]
		request.Queue = request.Queue[1:]
		cm.CurrentState.RequestMap[m.PageId] = request
		cm.PChannels[m.Sender] <- Message{Type: PAGE_NOT_FOUND, PageId: m.PageId}
		return
	}
	pageStatus.Status = RequestStatus{State: PENDING_READ_COMPLETION}
	cm.CurrentState.RequestMap[m.PageId] = pageStatus
	//send read forward or error
	cm.PChannels[cmEntry.Owner] <- Message{Sender: m.Sender, Type: READ_FORWARD, PageId: m.PageId} //send the WRITE_FORWARD request to owner
}

func (cm *CentralManager) HandleWriteRequest(m Message) {
	// check status pageId: might be pending write request
	pageStatus, _ := cm.CurrentState.RequestMap[m.PageId]

	pageStatus.Status = RequestStatus{State: PENDING_WRITE_COMPLETION}
	cm.CurrentState.RequestMap[m.PageId] = pageStatus
	//entry doesn't exist? send write
	if _, ok := cm.CurrentState.Entries[m.PageId]; !ok {
		cm.CurrentState.Entries[m.PageId] = CMEntry{CopyArray: []int{}, Owner: m.Sender} //set new owner
		cm.PChannels[m.Sender] <- Message{Type: PAGE_TO_WRITE, PageId: m.PageId}         //send the pageVariable to alow the write
		cm.log(true, "page variable sent to %v", m.Sender)
		return
	}
	//invalidate copies? else send write forward
	cmEntry := cm.CurrentState.Entries[m.PageId]

	if len(cmEntry.CopyArray) != 0 {
		//send invalidate copies
		cm.CurrentState.InvalidationCounter[m.PageId] = len(cmEntry.CopyArray)
		for i := 0; i < len(cmEntry.CopyArray); i++ {
			if cmEntry.CopyArray[i] == m.Sender {
				//don't invalidate the requester
				cm.ConfirmationChan <- Message{Type: INVALIDATE_CONFIRMATION, PageId: m.PageId}
				continue
			}
			go func(copyHolder int, channels []chan Message) {
				channels[cmEntry.CopyArray[copyHolder]] <- Message{Type: INVALIDATE_COPY, PageId: m.PageId}
			}(i, cm.PChannels)
		}
		return
	}

	// If no copies to invalidate, send the write forward message
	cm.PChannels[cmEntry.Owner] <- Message{Sender: m.Sender, Type: WRITE_FORWARD, PageId: m.PageId} //send the WRITE_FORWARD request to owner
}

func (cm *CentralManager) HandleInvalidateConfirmation(m Message) {
	//do nothing if waiting for more confirmation, else send write forward
	cm.CurrentState.InvalidationCounter[m.PageId]--
	if cm.CurrentState.InvalidationCounter[m.PageId] != 0 {
		return
	}
	cmEntry := cm.CurrentState.Entries[m.PageId]
	cm.PChannels[cmEntry.Owner] <- Message{
		Sender: cm.CurrentState.RequestMap[m.PageId].Queue[0].Sender,
		Type:   WRITE_FORWARD,
		PageId: m.PageId} //send the WRITE_FORWARD request to owner

	cmEntry.CopyArray = []int{}
	cm.CurrentState.Entries[m.PageId] = cmEntry
	cm.log(true, "Entry map: %v after sending WRITE_FORWARD to %v", cm.CurrentState.Entries, cmEntry.Owner)
}

func (cm *CentralManager) HandleElect(m Message) {
	//update IsPrimary and acts as per normal
	cm.log(false, "elected as new primary")
	cm.IsPrimary = true
	for i := range cm.PChannels {
		cm.PChannels[i] <- Message{Sender: cm.Id, Type: ANNOUNCE_PRIMARY}
	}
	for pageId := range cm.CurrentState.RequestMap {
		pageStatus := cm.CurrentState.RequestMap[pageId]
		if len(pageStatus.Queue) == 0 {
			continue
		}
		lastRequest := pageStatus.Queue[0]
		if pageStatus.Status.State == PENDING_READ_COMPLETION && lastRequest.Type == READ_REQUEST {
			pageOwner := cm.CurrentState.Entries[pageId].Owner
			cm.PChannels[pageOwner] <- Message{Sender: lastRequest.Sender, Type: READ_FORWARD, PageId: pageId} //send the READ_FORWARD request to owner
		}
		if pageStatus.Status.State == PENDING_WRITE_COMPLETION && lastRequest.Type == WRITE_REQUEST {
			cm.HandleWriteRequest(lastRequest)
		}
	}
	for i := range cm.CentralManagersConfirmation {
		if i == cm.Id {
			continue
		}
		cm.CentralManagersConfirmation[i] <- Message{Type: ANNOUNCE_PRIMARY}
	}
}

func (cm *CentralManager) HandleForwardState(m Message) {
	if err := json.Unmarshal(m.State, &cm.CurrentState); err != nil {
		cm.log(false, "Unmarshal Error: %v", err)
	}
	cm.log(true, "currentstate set to:  %v", cm.CurrentState)
}

func (cm *CentralManager) ForwardState() {
	stateBytes, err := json.Marshal(cm.CurrentState)
	if err != nil {
		cm.log(false, "Serialize Error: %v", err)
		return
	}

	if !cm.IsPrimary {
		// don't forward state if not primary CM
		return
	}
	for i := 0; i < len(cm.CentralManagersIncoming); i++ {
		if i == cm.Id {
			// don't forward to self
			continue
		}
		go func(n int) {
			cm.CentralManagersConfirmation[n] <- Message{Type: FORWARD_STATE, State: stateBytes}
		}(i)
	}
}

func (cm *CentralManager) LongestQueue() int {
	maxSeenLength := 0
	result := -1
	for pageId := range cm.CurrentState.RequestMap {
		resStatus := cm.CurrentState.RequestMap[pageId]
		if resStatus.Status.State == IDLE {
			if len(resStatus.Queue) > maxSeenLength {
				maxSeenLength = len(cm.CurrentState.RequestMap[pageId].Queue)
				result = pageId
			}
		}
	}
	return result
}

func (cm *CentralManager) Ressurect() {
	time.Sleep(5 * time.Second)
	cm.IsAlive = true
	cm.CountDownToDeath = 100
	for i := 0; i < len(cm.PChannels); i++ {
		cm.PChannels[i] <- Message{Type: START_ELECTION}
	}
}

func (cm *CentralManager) HandleAnnouncePrimary(m Message) {
	cm.IsPrimary = false
}
