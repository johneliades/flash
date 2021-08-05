package client

import (
	"bytes"
	"fmt"
	"github.com/johneliades/flash/handshake"
	"github.com/johneliades/flash/message"
	"github.com/johneliades/flash/peer"
	"net"
	"time"
)

// A Bitfield represents the pieces that a peer has
type bitfield []byte

// HasPiece tells if a bitfield has a particular index set
func (bf bitfield) HasPiece(index int) bool {
	byteIndex := index / 8
	offset := index % 8
	if byteIndex < 0 || byteIndex >= len(bf) {
		return false
	}
	return bf[byteIndex]>>(7-offset)&1 != 0
}

// SetPiece sets a bit in the bitfield
func (bf bitfield) SetPiece(index int) {
	byteIndex := index / 8
	offset := index % 8

	// silently discard invalid bounded index
	if byteIndex < 0 || byteIndex >= len(bf) {
		return
	}
	bf[byteIndex] |= 1 << (7 - offset)
}

type Client struct {
	Conn     net.Conn
	Choked   bool
	BitField bitfield
	peer     peer.Peer
	infoHash [20]byte
	peerID   [20]byte
}

func New(peer peer.Peer, peerID, infoHash [20]byte) (*Client, error) {
	conn, ok := net.DialTimeout("tcp", peer.String(), 10*time.Second)
	if ok != nil {
		return &Client{}, ok
	}

	req := handshake.New(infoHash, peerID)

	_, ok = conn.Write(req.Serialize())
	if ok != nil {
		return &Client{}, ok
	}

	conn.SetDeadline(time.Now().Add(10 * time.Second))
	defer conn.SetDeadline(time.Time{})

	res, ok := handshake.Read(conn)
	if ok != nil {
		return &Client{}, ok
	}

	if !bytes.Equal(res.InfoHash[:], infoHash[:]) {
		return nil, fmt.Errorf("Expected infohash %x but got %x", res.InfoHash, infoHash)
	}

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	msg, ok := message.Read(conn)
	if ok != nil {
		return &Client{}, ok
	}

	if msg.ID != message.BitField {
		return nil, fmt.Errorf("Expected bitField's id but got: %d", msg.ID)
	}

	return &Client{
		Conn:     conn,
		Choked:   true,
		BitField: msg.Payload,
		peer:     peer,
		infoHash: infoHash,
		peerID:   peerID,
	}, nil
}
