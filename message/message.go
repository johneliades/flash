package message

import (
	"encoding/binary"
	"io"
)

const (
	choke         uint8 = 0
	unchoke       uint8 = 1
	interested    uint8 = 2
	notInterested uint8 = 3
	have          uint8 = 4
	BitField      uint8 = 5
	request       uint8 = 6
	piece         uint8 = 7
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
