package server

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	// "github.com/sryosz/sharing-file-system/internal/message"
	"github.com/sryosz/sharing-file-system/internal/message"
	"github.com/sryosz/sharing-file-system/internal/peers"
	"github.com/sryosz/sharing-file-system/internal/storage"
)

type ServerOpts struct {
	ID          string
	StorageRoot string
	PathFunc    storage.PathFunc
	peers.Transport
	Nodes []string
}

type Server struct {
	ServerOpts

	peers map[string]peers.Peer
	mu    sync.Mutex

	storage *storage.Storage

	quitCh chan struct{}
}

type Message struct {
	Payload any
}

type StoreMessage struct {
	Key  string
	Size int64
}

type GetMessage struct {
	Key string
}

func NewServer(opts ServerOpts) *Server {
	storeOpts := storage.StorageOPTS{
		Root:     opts.StorageRoot,
		PathFunc: opts.PathFunc,
	}

	if len(opts.ID) == 0 {
		opts.ID = opts.Transport.Addr()
	}

	return &Server{
		ServerOpts: opts,
		mu:         sync.Mutex{},
		peers:      make(map[string]peers.Peer),
		storage:    storage.NewStorage(storeOpts),
		quitCh:     make(chan struct{}),
	}
}

func (s *Server) handleMessage(from string, msg *Message) error {
	switch m := msg.Payload.(type) {
	case StoreMessage:
		return s.handleStoreMessage(from, &m)
	case GetMessage:
		return s.handleGetMessage(from, &m)
	}

	return nil
}

func (s *Server) handleStoreMessage(from string, msg *StoreMessage) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	n, err := s.storage.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk\n", s.Transport.Addr(), n)

	peer.CloseStream()

	return nil
}

func (s *Server) handleGetMessage(from string, msg *GetMessage) error {
	if !s.storage.Has(msg.Key) {
		return fmt.Errorf("no such file on disk")
	}

	fmt.Printf("search for file (%s)\n", msg.Key)

	p, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("not found in map of peers")
	}

	fileSize, r, err := s.storage.Read(msg.Key)
	if err != nil {
		return err
	}

	p.Send([]byte{message.IncomingStream})
	binary.Write(p,  binary.LittleEndian, &fileSize)

	n, err := io.Copy(p, r)
	if err != nil {
		return err
	}

	fmt.Printf("Read and written %d bytes from disk to the peer\n", n)

	return nil
}

func (s *Server) Get(key string) (io.Reader, error) {
	if s.storage.Has(key) {
		_, r, err := s.storage.Read(key)
		return r, err
	}

	log.Println("cant find data locally. Fetching from the network")

	msg := &Message{
		Payload: &GetMessage{
			Key: key,
		},
	}

	if err := s.broadcast(msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Second * 2)

	for _, p := range s.peers {
		var fileSize int64
		binary.Read(p, binary.LittleEndian, &fileSize)

		n, err := s.storage.Write(key, io.LimitReader(p, fileSize))
		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] Received and written %d bytes to the disk\n", p.RemoteAddr(), n)

		p.CloseStream()
	}

	_, r, err := s.storage.Read(key)

	return r, err
}

func (s *Server) Store(key string, r io.Reader) error {
	buf := new(bytes.Buffer)
	teeReader := io.TeeReader(r, buf)

	size, err := s.storage.Write(key, teeReader)
	if err != nil {
		return err
	}

	m := &Message{
		Payload: StoreMessage{
			Key:  key,
			Size: size,
		},
	}

	if err := s.broadcast(m); err != nil {
		return err
	}

	time.Sleep(2 * time.Second)

	for _, p := range s.peers {
		p.Send([]byte{message.IncomingStream})
		n, err := p.Write(buf.Bytes())
		if err != nil {
			return err
		}

		fmt.Printf("[%s] received and written bytes to disk: %d\n", p.RemoteAddr().String(), n)
	}

	return nil
}

func (s *Server) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, p := range s.peers {
		p.Send([]byte{message.IncomingMsg})
		if err := p.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) connectToNodes() {
	for _, addr := range s.Nodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			fmt.Printf("[%s]: Trying to connect to the remote node: [%s]\n", s.Addr(), addr)
			err := s.Dial(addr)
			if err != nil {
				fmt.Printf("[%s]: Failed to connect: %s\n", addr, err.Error())
			}
			fmt.Printf("[%s]: Successfully connected to the remote node [%s]\n", s.Addr(), addr)
		}(addr)
	}
}

type Payload struct {
	Data []byte
}

func (s *Server) loop() {
	defer func() {
		// log that shit
		s.Transport.Close()
	}()

	for {
		select {
		case peerMsg := <-s.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(peerMsg.Payload)).Decode(&msg); err != nil {
				log.Panic(err)
			}
			if err := s.handleMessage(peerMsg.From, &msg); err != nil {
				log.Panic(err)
			}
		case <-s.quitCh:
			return
		}
	}
}

func (s *Server) OnPeer(p peers.Peer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.peers[p.RemoteAddr().String()] = p

	fmt.Printf("connected with remote %s\n", p.RemoteAddr())

	return nil
}

func (s *Server) Run() {
	fmt.Printf("[%s]: Starting server\n", s.ServerOpts.Addr())

	err := s.Transport.Listen()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	s.connectToNodes()

	s.loop()
}

func (s *Server) Stop() {
	close(s.quitCh)
}

func init() {
	gob.Register(StoreMessage{})
	gob.Register(GetMessage{})
}
