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
)

const NUM_OF_VARIABLES = 10
const NUM_OF_PROCESSORS = 1

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
}
