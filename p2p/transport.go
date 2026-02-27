package p2p

import "net"

// Peer is an interface that represents remote node
type Peer interface {

	// Send is used to write a []byte to a connection
	Send([]byte) error

	net.Conn
	CloseStream() 
}

// Tranport is anything that handles the communication between
// the node in the network. This can be tcp, udp, websockets... etc
type Transport interface {
	
	// Dial is used to connect peer to other peer
	Dial(string) error 

	// ListenAndAccept will start the transport server and will wait for a 
	// new connection. In case of a new connection it will handle the 
	// connection in a goroutine, and waits for another connection
	ListenAndAccept() error

	// Consume will return a read-only channel that will be used to access the 
	// data recieved from the connection on a Transport server
	Consume() <-chan RPC

	// Close will stop the Transport server
	Close() error

	// Addr will return the listening address of the transport server 
	Addr() string
}
