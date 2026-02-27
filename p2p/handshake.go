package p2p

// HandshakeFunc takes a Peer as argument and returns a error.
// It will be executed upon a new connection.
type HandshakeFunc func(Peer) error 

// Default HandshakeFunc -> doesn't do anything
func NOPHandshakeFunc(Peer) error { return nil}
