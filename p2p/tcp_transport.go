package p2p

import (
	"fmt"
	"net"
	"sync"
)

type TCPTransport struct {
	listenAddr string
	listener   net.Listener

	mu sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(listenAddr string) *TCPTransport {
	return &TCPTransport{
		listenAddr: listenAddr,
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.listenAddr)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()
	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("Tcp connection error: %s\n", err)
		}

		go t.handleConnection(conn)
	}	
}

func (t *TCPTransport) handleConnection(conn net.Conn) {
	fmt.Printf("Got new Connection: %+v\n", conn)
	defer conn.Close()
}
 
