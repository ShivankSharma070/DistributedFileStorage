package p2p

// Peer is an interface that represents remote node 
type Peer interface{
	Close() error
}

// Tranport is anything that handles the communication between 
// the node in the network. This can be tcp, udp, websockets... etc
type Transport interface{
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
