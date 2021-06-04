package peer

import (
	"net"
	"encoding/binary"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func ExtractPeers(peersBinary []byte) []Peer {
	// Extract peers from binary

	numPeers := len(peersBinary)/ 6
	if len(peersBinary)%6 != 0 {
		panic("Received malformed peers")
	}
	peers := make([]Peer, numPeers)
	for i, offset := 0, 0; i < numPeers; i, offset = i+1, i * 6 {
		peers[i].IP = net.IP(peersBinary[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16([]byte(peersBinary[offset+4 : offset+6]))
	}

	return peers
}
