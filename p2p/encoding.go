package p2p

import (
	"io"
)

// Decoder is an interface that defines the method for decoding incoming
// data from a remote peer. It takes an io.Reader as source of data and
// populates the provided RPC struct with the decoded information
type Decoder interface {
	Decode(io.Reader, *RPC) error
}

// DefaultDecoder is the default implementation of the Decoder interface.
// It reads the first byte to determine if the incoming message is a stream
// or a regular message
type DefaultDecoder struct{}

// Decode reads data from the provided io.Reader and populates the RPC struct.
// It peeks at the first byte to check if it's an incoming stream flag.
// If the first byte matches IncomingStream, it sets the Stream field to true
// and returns without reading further data. Otherwise, it reads up to 1024
// bytes into the Payload field of the RPC struct
func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return nil
	}

	// In case of stream we are not decoding  what is being send over the network.
	// We are just setting stream as true so that we can handle that in our logic
	stream := peekBuf[0] == IncomingStream
	if stream {
		msg.Stream = true
		return nil
	}

	buff := make([]byte, 1024)
	n, err := r.Read(buff)
	if err != nil {
		return err
	}

	msg.Payload = buff[:n]

	return nil
}
