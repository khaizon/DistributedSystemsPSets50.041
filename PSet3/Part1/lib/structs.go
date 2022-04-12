package lib

type RequestStatus int

const (
	Idle RequestStatus = iota
	PendingRequestCompletion
	PendingWriteConfirmation
	PendingReadConfirmation
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
	PageNotFoundError
)

var MESSAGE_TYPES []string = []string{
	"ReadRequest",
	"WriteRequest",
	"ReadForward",
	"PageForward",
	"ReadConfirmation",
	"InvalidateCopy",
	"InvalidateConfirmation",
	"WriteForward",
	"VariableForward",
	"WriteConfirmation",
	"PageNotFoundError",
}
