package sfsencoding

import (
	"io"

	"github.com/sryosz/sharing-file-system/internal/message"
)

type Decoder interface{
	Decode(r io.Reader, msg *message.PeerMsg) error
}

type DefaultDecoder struct{}

func (d *DefaultDecoder) Decode(r io.Reader, msg *message.PeerMsg) error{
	buf := make([]byte, 1)
	_, err := r.Read(buf)
	if err != nil {
		return err
	}
	
	if buf[0] == message.IncomingStream{
		msg.Stream = true
		return nil
	}

	buf = make([]byte, 1028)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}

	msg.Payload = buf[:n]

	return nil
}

