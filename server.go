package main

import (
	"fmt"
	"log"

	"github.com/ShivankSharma070/DistributedFileStorage/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	bootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	store  *Store
	quitCh chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	serverOpts := StoreOpts{
		PathTransformFunc: opts.PathTransformFunc,
		root:              opts.StorageRoot,
	}

	return &FileServer{
		store:          NewStore(serverOpts),
		FileServerOpts: opts,
		quitCh:         make(chan struct{}),
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.BootStrapNetwork()
	s.loop()

	return nil
}

func (s *FileServer) Quit() {
	close(s.quitCh)
}

func (s *FileServer) BootStrapNetwork() error {
	for _, addr := range s.bootstrapNodes {
		if len(addr) == 0 {
			continue;
		}
		go func(addr string) {
			err := s.Transport.Dial(addr)
			if err != nil {
				log.Printf("Dial Error: %s\n", err)
			}
		}(addr)
	}
	return nil
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("File server stopped due to user action.")
		s.Transport.Close()
	}()
	for {
		select {
		case <-s.quitCh:
			return
		case msg := <-s.Transport.Consume():
			fmt.Printf("Message : %s\n", msg.Payload)
		}
	}
}
