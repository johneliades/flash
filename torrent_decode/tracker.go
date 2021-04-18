package torrent_decode

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (torrent *torrentFile) GetTrackerResponse(port int) string {
	if strings.HasPrefix(torrent.announce, "http") {
		return getHTTPTrackerResponse(torrent, port)
	} else if strings.HasPrefix(torrent.announce, "udp") {
		fmt.Printf("UDP tracker not supported yet.")
	}

	return ""
}

func getHTTPTrackerResponse(torrent *torrentFile, port int) string {
	// create tracker url

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
	body, ok := io.ReadAll(resp.Body)
	check(ok)

	return string(body[:])
}
