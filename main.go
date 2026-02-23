package main

import (
	"log"

	"github.com/ShivankSharma070/DistributedFileStorage/p2p"
)

func makeServer(listenAddr string, node ...string) *FileServer {

	tcp_opts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// TODO: OnPeer
	}

	tcp_transport := p2p.NewTCPTransport(tcp_opts)
	opts := FileServerOpts{
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcp_transport,
		bootstrapNodes:    node,
	}

	return NewFileServer(opts)

}

func main() {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(s1.Start())
	}()

	if err := s2.Start(); err != nil {
		log.Fatal(err)
	}
}
