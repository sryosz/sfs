package peers

import (
	"net"
	"github.com/sryosz/sharing-file-system/internal/message"
)

type Peer interface {
	Send([]byte) error
	CloseStream()
	net.Conn
}

type Transport interface {
	Addr() string
	Dial(string) error
	Listen() error
	Accept()
	Consume() <-chan message.PeerMsg
	Close() error
}

