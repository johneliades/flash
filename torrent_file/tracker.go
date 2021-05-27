package torrent_file

import (
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"bytes"
	"net"
	"encoding/binary"
	"github.com/marksamman/bencode"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func (torrent *torrentFile) GetPeers(port int) []Peer {
	var body []byte

	if strings.HasPrefix(torrent.announce, "http") { 
		// ============== HTTP Tracker ==============

		base, ok := url.Parse(torrent.announce)
		check(ok)

		rand.Seed(time.Now().UnixNano())
		var peerID [4]byte
		_, ok = rand.Read(peerID[:])
		check(ok)

		params := url.Values{}
		params.Add("info_hash", string(torrent.infoHash[:]))
		// peer_id must be 20 bytes
		params.Add("peer_id", "git:johneliades-"+string(peerID[:]))
		params.Add("port", strconv.Itoa(port))
		params.Add("uploaded", "0")
		params.Add("downloaded", "0")
		params.Add("left", strconv.Itoa(torrent.length))
		params.Add("compact", "1")

		base.RawQuery = params.Encode()

		// communicate with tracker url

		resp, ok := http.Get(base.String())
		check(ok)

		defer resp.Body.Close()

		body, ok = io.ReadAll(resp.Body)
		check(ok)
	} else if strings.HasPrefix(torrent.announce, "udp") {
		// ============== UDP Tracker ==============
		panic("UDP tracker not supported yet.")
	}

	data, err := bencode.Decode(bytes.NewReader(body[:]))
	check(err)

	// start ============== Extract peers from binary ==============

	peersBinary := []byte(data["peers"].(string))

	numPeers := len(peersBinary)/ 6
	if len(peersBinary)%6 != 0 {
		panic("Received malformed peers")
	}
	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * 6
		peers[i].IP = net.IP(peersBinary[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16([]byte(peersBinary[offset+4 : offset+6]))
	}
	
	// end ============== Extract peers from binary ==============

	return peers
}
