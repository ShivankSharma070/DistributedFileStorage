package main

import (
	"bytes"
	"log"
	"strings"
	"time"

	"github.com/ShivankSharma070/DistributedFileStorage/p2p"
)

func makeServer(listenAddr string, node ...string) *FileServer {

	tcp_opts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}

	tcp_transport := p2p.NewTCPTransport(tcp_opts)
	opts := FileServerOpts{
		EncryptionKey:     newEncryptionKey(),
		StorageRoot:       strings.TrimPrefix(listenAddr, ":") + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcp_transport,
		bootstrapNodes:    node,
	}

	s := NewFileServer(opts)
	// Use a onPeer function defined for a new server.
	tcp_transport.OnPeer = s.OnPeer
	return s
}

func main() {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")
	s3 := makeServer(":5000", ":3000", ":4000")

	go func() {
		log.Fatal(s1.Start())
	}()

	time.Sleep(time.Second * 1)

	go func() {
		log.Fatal(s2.Start())
	}()

	time.Sleep(time.Second * 1)

	go func() {
		log.Fatal(s3.Start())
	}()

	time.Sleep(time.Second * 1)

	key := "mysecretkejy"
	buf := bytes.NewReader([]byte("This is some large file"))
	s3.Store(key, buf)
	time.Sleep(15*time.Second)

	if err := s3.Delete("someotherkey"); err != nil {
		log.Fatalf("Error deleting file : %s", err)
	}

	// r, err := s3.Get(key)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// data, err := io.ReadAll(r)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println("Recieved file form server:", string(data))

	select {}
}
