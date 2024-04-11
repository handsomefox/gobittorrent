package bencode

type MessageID byte

const (
	Choke MessageID = iota
	Unchoke
	Interested
	NotInterested
	Have
	Bitfield
	Request
	Piece
	Cancel
)

type PeerMessage struct {
	Length    int32
	MessageID byte
	Payload   []byte
}
