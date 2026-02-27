package p2p

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// RPC holds any arbitary data that is being sent over the each transport between
// two nodes in the network
type RPC struct {
	Payload []byte

	// From holds the listen address of the sender peer
	From    string

	// Stream is a boolean value that indicated if the message incoming is a stream
	Stream bool
}
