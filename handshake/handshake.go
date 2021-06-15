package handshake

import (
	"fmt"
	"io"
)

type Handshake struct {
	pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

func New(infoHash, peerID [20]byte) *Handshake {
	return &Handshake{
		pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

func (handshake *Handshake) Serialize() []byte {
	buf := []byte{}
	buf = append(buf, byte(len(handshake.pstr)))
	buf = append(buf, handshake.pstr...)
	buf = append(buf, make([]byte, 8)...)
	buf = append(buf, handshake.InfoHash[:]...)
	buf = append(buf, handshake.PeerID[:]...)
	return buf
}

func Read(reader io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)
	_, ok := io.ReadFull(reader, lengthBuf)
	if ok != nil {
		return &Handshake{}, ok
	}

	pstrlen := int(lengthBuf[0])
	if pstrlen == 0 {
		return nil, fmt.Errorf("pstrlen was zero")
	}

	handshakeResponse := make([]byte, 48+pstrlen)
	_, ok = io.ReadFull(reader, handshakeResponse)
	if ok != nil {
		return &Handshake{}, ok
	}

	var infoHash, peerID [20]byte
	copy(infoHash[:], handshakeResponse[pstrlen+8:pstrlen+8+20])
	copy(peerID[:], handshakeResponse[pstrlen+8+20:])

	h := Handshake{
		pstr:     string(handshakeResponse[0:pstrlen]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	return &h, nil
}
