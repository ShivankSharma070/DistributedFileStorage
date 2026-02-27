package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCPPeer implements Peer. It represents the remote node over the established
// TCP connection
type TCPPeer struct {
	// Conn is the underlying connection of the peer
	net.Conn

	// Outbound is a boolean value used to represent the type of connection
	// If we dial and retreive a conn: outbound : true
	// If we accept and retreive a conn: outbound : false
	Outbound bool

	// wg is used to ensure that data is read from a
	// connection in a orderly manner, and to avoid race conditions
	wg *sync.WaitGroup
}

// CloseStream implement Peer. It calls the p.wg.Done() which will resume the
// handling of incomming data by handleConnection
func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

// Send implements Peer. It writes a []byte to the underlying
// connection object of the peer
func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Write(b)
	return err
}

// NewTCPPeer will create and return a *TCPPeer with the given connection
func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		Outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

// TCPTransportOpts will have all the required and configurable
// fields that will be used to create a new TCPTransport
type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder

	// OnPeer is a custom function that is executed upon recieving a new connection.
	OnPeer func(Peer) error
}

// TCPTransport implement Transport for tcp protocol
type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener

	// rpcChan is a buffered channel of RPC used by handleConnection
	// to pass down the recieved data for further handling
	rpcChan chan RPC
}

// NewTCPTransport will create and return a *TCPTransport with the provided opts
func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcChan:          make(chan RPC, 1024),
	}
}

// Dial implements Transport interface. It uses net.Dail() to connect our
// transport with other peers on tcp
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConnection(conn, true)
	return nil
}

// ListenAndAccept implements Transport. It starts listening for
// incoming connection on your TCPTransport using net.Listen() and calls
// startAcceptLoop() for a new connection in a goroutine and waits for
// another conneciton
func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	log.Printf("Server Started on port: %s\n", t.ListenAddr)

	go t.startAcceptLoop()
	return nil
}

// Close implement Transport interface. It calls net.listener.Close()
// Blocked accept operations will be unblocked and return a net.ErrClosed
func (t *TCPTransport) Close() error {
	log.Println("Stoped TCP Listener")
	return t.listener.Close()
}

// Consume implement Transport interface. It returns a read-only channel
// that will be used to recieve messages from other peer on the network
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcChan
}

// startAcceptLoop waits for new connections. In case of a new connection will
// call handleConnection with the new net.Conn in a goroutine and will
// wait for another connection
func (t *TCPTransport) startAcceptLoop() {
	for {

		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			log.Println("Stopped accepting new connecton")
			return
		}
		if err != nil {
			log.Printf("Tcp connection error: %s\n", err)
		}

		go t.handleConnection(conn, false)
	}
}

// handleConnection calls NewTCpPeer to get a new peer for the recieved net.Conn
// Calls HandshakeFunc and OnPeer for that peer. And waits
// for any incoming data on that connection 
func (t *TCPTransport) handleConnection(conn net.Conn, outbound bool) {
	var err error
	defer func() {
		fmt.Printf("Dropping connnection: %s\n", err)
		conn.Close()
	}()
	peer := NewTCPPeer(conn, outbound)

	if err = t.HandshakeFunc(peer); err != nil {
		fmt.Printf("Tcp error : %s\n", err)
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// Read loop
	for {
		rpc := RPC{}
		// Incoming data will be decoded and stored in rpc.Payload
		err := t.Decoder.Decode(conn, &rpc)
		if err != nil {
			fmt.Printf("Tcp read error : %s\n", err)
			return
		}

		rpc.From = conn.RemoteAddr().String()

		// If the incomming data is a stream, then don't handle it and 
		// waits for the stream to resolve 
		if rpc.Stream {
			peer.wg.Add(1)

			fmt.Printf(
				"[%s] incoming stream, waiting ....\n",
				conn.RemoteAddr())

			peer.wg.Wait()

			fmt.Printf(
				"[%s] stream closed, resuming read loop ....\n",
				conn.RemoteAddr())

			continue
		}

		// Passed the recieved rpc to rpcChan
		t.rpcChan <- rpc
	}
}

// Addr implemnts Transport and return listening address of TCPTransport
func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}
