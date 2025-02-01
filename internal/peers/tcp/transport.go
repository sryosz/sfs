package tcptransport

import (
	"errors"
	"fmt"
	"net"
	"sync"

	sfsencoding "github.com/sryosz/sharing-file-system/internal/encoding"
	"github.com/sryosz/sharing-file-system/internal/message"
	"github.com/sryosz/sharing-file-system/internal/peers"
)

type TCPPeer struct {
	net.Conn
	Outbound bool
	wg       *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{conn, outbound, &sync.WaitGroup{}}
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Write(b)
	return err
}

func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

type TCPTransportOpts struct {
	ListenAddr string
	Decoder    sfsencoding.Decoder
	//Handshake func
	OnPeer func(peers.Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	Ch       chan message.PeerMsg
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{TCPTransportOpts: opts, Ch: make(chan message.PeerMsg, 1024)}
}

func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

func (t *TCPTransport) Consume() <-chan message.PeerMsg {
	return t.Ch
}

func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)

	return nil
}

func (t *TCPTransport) Listen() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	go t.Accept()

	fmt.Printf("TCP transport listening on port: %s\n", t.listener.Addr())

	return nil
}

func (t *TCPTransport) Accept() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			} else {
				fmt.Println("TCP accept error: ", err.Error())
			}
		}

		go t.handleConn(conn, false)
	}
}

type HandshakeFunc func(peers.Peer) error

func NOPHandshakeFunc(peers.Peer) error { return nil }

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error

	defer func() {
		fmt.Printf("dropping peer connection: %s", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, outbound)

	// handshake ???
	if t.OnPeer != nil {
		if err := t.OnPeer(peer); err != nil {
			return
		}
	}

	msg := message.PeerMsg{}

	for {
		if err := t.Decoder.Decode(conn, &msg); err != nil {
			fmt.Printf("Failed to decode msg: %s", err.Error())
		}

		msg.From = conn.RemoteAddr().String()

		// if msg.Stream {
		peer.wg.Add(1)
		fmt.Println("Incoming stream. Waiting")
		t.Ch <- msg
		peer.wg.Wait()
		fmt.Println("Stream is done")
		// }

	}
}