package peer

import (
	"encoding/binary"
	"net"
	"strconv"
)

type Peer struct {
	ip   net.IP
	port uint16
}

func New(ip net.IP, port uint16) *Peer {
	return &Peer{ip, port}
}

func Deserialize(peersBinary []byte) []Peer {
	// Extract peers from binary

	numPeers := len(peersBinary) / 6
	if len(peersBinary)%6 != 0 {
		panic("Peers binary length is not multiple of 6 bytes (4 for ip 2 for port)")
	}

	peers := []Peer{}
	for i, offset := 0, 0; i < numPeers; i, offset = i+1, i*6 {
		newPeer := New(net.IP(peersBinary[offset:offset+4]),
			binary.BigEndian.Uint16([]byte(peersBinary[offset+4:offset+6])))

		peers = append(peers, *newPeer)
	}

	return peers
}

func (peer Peer) String() string {
	return net.JoinHostPort(peer.ip.String(), strconv.Itoa(int(peer.port)))
}
