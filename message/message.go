package message

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	Choke         uint8 = 0
	Unchoke       uint8 = 1
	Interested    uint8 = 2
	notInterested uint8 = 3
	Have          uint8 = 4
	BitField      uint8 = 5
	Request       uint8 = 6
	Piece         uint8 = 7
	cancel        uint8 = 8
)

type Message struct {
	ID      uint8
	Payload []byte
}

// <length prefix><message ID><payload>
func (message *Message) Serialize() []byte {
	if message == nil {
		return make([]byte, 4)
	}

	length := uint32(len(message.Payload) + 1) // +1 for id

	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(message.ID)
	copy(buf[5:], message.Payload)

	return buf
}

func Read(reader io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(reader, lengthBuf)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lengthBuf)

	// keep-alive message
	if length == 0 {
		return nil, nil
	}

	messageBuf := make([]byte, length)
	_, err = io.ReadFull(reader, messageBuf)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:      uint8(messageBuf[0]),
		Payload: messageBuf[1:],
	}, nil
}

// ParsePiece parses a PIECE message and copies its payload into a buffer
func ParsePiece(index int, buf []byte, msg *Message) (int, error) {
	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	data := msg.Payload[8:]

	if msg.ID != Piece || len(msg.Payload) < 8 || parsedIndex != index ||
		begin >= len(buf) || begin+len(data) > len(buf) {

		return 0, fmt.Errorf("ParsePiece failed")
	}

	copy(buf[begin:], data)
	return len(data), nil
}

func ParseHave(msg *Message) (int, error) {
	if msg.ID != Have || len(msg.Payload) != 4 {
		return 0, fmt.Errorf("ParseHave failed")
	}

	index := int(binary.BigEndian.Uint32(msg.Payload))
	return index, nil
}

func MakeRequest(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: Request, Payload: payload}
}

func MakeHave(index int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))
	return &Message{ID: Have, Payload: payload}
}
