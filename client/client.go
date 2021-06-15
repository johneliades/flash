package client

import (
	"bytes"
	"fmt"
	"github.com/johneliades/flash_torrent/handshake"
	"github.com/johneliades/flash_torrent/peer"
	"net"
	"time"
)

type Client struct {
	conn     net.Conn
	choked   bool
	bitField []byte
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

//	conn.SetDeadline(time.Now().Add(5 * time.Second))
//	defer conn.SetDeadline(time.Time{})

	/*
		msg, ok := message.Read(conn)
		if ok != nil {
			panic(ok)
		}

		if msg.ID != message.MsgbitField {
			err := fmt.Errorf("Expected bitField but got ID %d", msg.ID)
			return nil, err
		}

		bf = msg.Payload

		return &Client{
			conn:     conn,
			choked:   true,
			bitField: []byte,
			peer:     peer,
			infoHash: infoHash,
			peerID:   peerID,
		}, nil */

	return nil, ok
}
