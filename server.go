package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/ShivankSharma070/DistributedFileStorage/p2p"
)

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
	gob.Register(MessageDeleteFile{})
}

// FileServerOpts holds configuration options for creating a new FileServer
type FileServerOpts struct {
	// ID is the unique identifier for this file server node
	ID string

	// EncryptionKey is the key used for encrypting files when 
	// storing/retrieving on other connected peers
	EncryptionKey []byte

	StorageRoot string
	PathTransformFunc PathTransformFunc
	Transport p2p.Transport

	// bootstrapNodes is a list of addresses to connect to on startup
	bootstrapNodes []string
}

// FileServer is the main server that handles file storage and retrieval
// across a P2P network. It manages local storage, peer connections,
// and message handling for file operations
type FileServer struct {
	FileServerOpts

	// peerLock protects the peers map during concurrent access
	peerLock sync.Mutex
	// peers is a map of remote peer addresses to their connection
	peers map[string]p2p.Peer

	store *Store
	// quitCh is used to signal the server to stop
	quitCh chan struct{}
}

// NewFileServer creates and returns a new FileServer instance with the
// provided options. If no ID is provided, one will be generated automatically
func NewFileServer(opts FileServerOpts) *FileServer {
	serverOpts := StoreOpts{
		PathTransformFunc: opts.PathTransformFunc,
		root:              opts.StorageRoot,
	}

	if len(opts.ID) == 0 {
		opts.ID = generateID()
	}

	return &FileServer{
		store:          NewStore(serverOpts),
		FileServerOpts: opts,
		quitCh:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

// broadcast sends a message to all connected peers in the network
func (s *FileServer) broadcast(msg *Message) error {
	buff := new(bytes.Buffer)
	if err := gob.NewEncoder(buff).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buff.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// Message is the generic message structure used for P2P communication.
// The Payload field contains the actual message data (e.g., MessageStoreFile,
// MessageGetFile, or MessageDeleteFile)
type Message struct {
	Payload any
}

// MessageDeleteFile is the payload for requesting file deletion across peers
type MessageDeleteFile struct {
	ID  string
	Key string
}

// MessageStoreFile is the payload for announcing a new file storage to peers
type MessageStoreFile struct {
	ID   string
	Key  string
	Size int64
}

// MessageGetFile is the payload for requesting a file from another peer
type MessageGetFile struct {
	ID  string
	Key string
}

// Delete removes the file with the given key from the local store and
// broadcasts the deletion request to all connected peers
func (s *FileServer) Delete(key string) error {
	if !s.store.Has(s.ID, key) {
		return fmt.Errorf("[%s] Don't have file %s locally, not checking on other peers", s.Transport.Addr(), key)
	}
	err := s.store.Delete(s.ID, key)
	if err != nil {
		return err
	}

	log.Printf("[%s] file %s deleted succesfullly, requesting other peers", s.Transport.Addr(), key)

	msg := &Message{
		Payload: MessageDeleteFile{
			Key: hashKey(key),
			ID:  s.ID,
		},
	}

	if err := s.broadcast(msg); err != nil {
		return err
	}

	return nil
}

// Get retrieves a file with the given key. First checks local storage,
// and if not found, broadcasts a request to peers and waits for response
func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(s.ID, key) {
		_, buf, err := s.store.Read(s.ID, key)
		if err != nil {
			return nil, err
		}
		return buf, nil
	}
	log.Printf(
		"[%s] Dont have file locally, fetching from network",
		s.Transport.Addr())

	msg := &Message{
		Payload: MessageGetFile{
			Key: hashKey(key),
			ID:  s.ID,
		},
	}

	if err := s.broadcast(msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 100)

	for _, peer := range s.peers {
		var fSize int64
		binary.Read(peer, binary.LittleEndian, &fSize)

		n, err := s.store.writeDecrypt(s.EncryptionKey, s.ID, key, io.LimitReader(peer, fSize))
		if err != nil {
			return nil, err
		}
		fmt.Printf(
			"[%s] Received %d bytes over the network.\n",
			s.Transport.Addr(),
			n)
		peer.CloseStream()
	}

	_, buf, err := s.store.Read(s.ID, key)
	return buf, err
}

// Store writes the file content from the provided io.Reader to local storage
// and broadcasts it to all connected peers
func (s *FileServer) Store(key string, r io.Reader) error {
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := s.store.Write(s.ID, key, tee)
	if err != nil {
		return err
	}

	msg := &Message{
		Payload: MessageStoreFile{
			Key:  hashKey(key),
			Size: size + 16,
			ID:   s.ID,
		},
	}

	if err := s.broadcast(msg); err != nil {
		return err
	}

	time.Sleep(5 * time.Millisecond)

	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	// Mulitwriter enables writting to mulitple peers, for loop cannot be used
	// here as reader becomes unreadable once it is read
	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})
	n, err := copyEncrypt(s.EncryptionKey, fileBuffer, mw)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] Broadcasted %d bytes \n", s.Transport.Addr(), n)

	return nil
}

// OnPeer is called when a new peer connects. It adds the peer to the
// peers map and logs the connection
func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[p.RemoteAddr().String()] = p
	if !p.(*p2p.TCPPeer).Outbound {
		log.Printf(
			"Connected with remote(server - %s): %s\n",
			s.Transport.(*p2p.TCPTransport).ListenAddr,
			p.RemoteAddr())
	}
	return nil
}

// Start begins the file server by starting the transport layer to listen
// for incoming connections, bootstraps the network with known nodes,
// and starts the main message handling loop, it is a blocking function, 
// it will return only when there is a error
func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	if err := s.BootStrapNetwork(); err != nil {
		return err
	}
	s.loop()

	return nil
}

// Quit signals the server to stop by closing the quit channel
func (s *FileServer) Quit() {
	close(s.quitCh)
}

// BootStrapNetwork attempts to connect to all configured bootstrap nodes
// in separate goroutines. Each connection is attempted asynchronously
func (s *FileServer) BootStrapNetwork() error {
	for _, addr := range s.bootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			log.Printf(
				"Server %s is trying to connect to %s\n",
				s.Transport.(*p2p.TCPTransport).ListenAddr,
				addr)
			err := s.Transport.Dial(addr)
			if err != nil {
				log.Printf("Dial Error: %s\n", err)
			}
		}(addr)
	}
	return nil
}

// handleMessage routes incoming messages to the appropriate handler
// based on the payload type (MessageStoreFile, MessageGetFile, or MessageDeleteFile)
func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	case MessageDeleteFile:
		return s.handleMessageDeleteFile(from, v)
	}
	return nil
}

// handleMessageDeleteFile processes a file deletion request from a peer.
// It deletes the file from local storage if it exists
func (s *FileServer) handleMessageDeleteFile(from string, msg MessageDeleteFile) error {
	if !s.store.Has(msg.ID, msg.Key) {
		return fmt.Errorf(
			"[%s] need to delete file %s, but it does not exists on disk",
			s.Transport.Addr(),
			msg.Key,
		)
	}

	err := s.store.Delete(msg.ID, msg.Key)
	if err != nil {
		return fmt.Errorf("[%s] delete error: %s", s.Transport.Addr(), err)
	}

	log.Printf("[%s] file %s deleted succesfullly", s.Transport.Addr(), msg.Key)
	return nil
}

// handleMessageGetFile processes a file retrieval request from a peer.
// If the file exists locally, it sends the file data to the requesting peer
func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.ID, msg.Key) {
		return fmt.Errorf(
			"[%s] need to serve file (%s), but it does not exists on disk",
			s.Transport.Addr(),
			msg.Key)
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("Peer %s could not be found in peer list", from)
	}

	fmt.Printf("[%s] Serve file %s over the network\n", s.Transport.Addr(), msg.Key)
	fSize, r, err := s.store.Read(msg.ID, msg.Key)
	if err != nil {
		return err
	}

	rc, ok := r.(io.ReadCloser)
	if ok {
		defer func(rc io.ReadCloser) {
			fmt.Println("Closing reader")
			rc.Close()
		}(rc)
	}

	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fSize)
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	log.Printf("[%s] Written %d bytes, to the network %s", s.Transport.Addr(), n, from)
	return nil
}

// handleMessageStoreFile processes a file storage request from a peer.
// It reads the file data from the peer and writes it to local storage
func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("Peer %s could not be found in peer list", from)
	}

	n, err := s.store.Write(msg.ID, msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	log.Printf("(%s) -> Written (%d) bytes to disk\n", s.Transport.Addr(), n)
	peer.CloseStream()
	return nil
}

// loop is the main event loop that continuously listens for incoming
// messages from the transport layer and dispatches them for handling
func (s *FileServer) loop() {
	defer func() {
		log.Println("File server stopped due to error or user action.")
		s.Transport.Close()
	}()
	for {
		select {
		case <-s.quitCh:
			return
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("Decoding error: ", err)
			}

			if err := s.handleMessage(rpc.From, &msg); err != nil {
				log.Println("Handle message error: ", err)
			}
		}
	}
}
