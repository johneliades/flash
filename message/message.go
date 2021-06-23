package message

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	Choke         uint8 = 0
	Unchoke       uint8 = 1
	interested    uint8 = 2
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
	if msg.ID != Piece {
		return 0, fmt.Errorf("Expected PIECE (ID %d), got ID %d", Piece, msg.ID)
	}
	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("Payload too short. %d < 8", len(msg.Payload))
	}
	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("Expected index %d, got %d", index, parsedIndex)
	}
	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= len(buf) {
		return 0, fmt.Errorf("Begin offset too high. %d >= %d", begin, len(buf))
	}
	data := msg.Payload[8:]
	if begin+len(data) > len(buf) {
		return 0, fmt.Errorf("Data too long [%d] for offset %d with length %d", len(data), begin, len(buf))
	}
	copy(buf[begin:], data)
	return len(data), nil
}

// ParseHave parses a HAVE message
func ParseHave(msg *Message) (int, error) {
	if msg.ID != Have {
		return 0, fmt.Errorf("Expected HAVE (ID %d), got ID %d", Have, msg.ID)
	}
	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("Expected payload length 4, got length %d", len(msg.Payload))
	}
	index := int(binary.BigEndian.Uint32(msg.Payload))
	return index, nil
}
