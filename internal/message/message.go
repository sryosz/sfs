package message

const (
	IncomingMsg    = 0x1
	IncomingStream = 0x2
)

type PeerMsg struct {
	From    string
	Payload []byte
}
