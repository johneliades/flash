package torrent_file

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"github.com/johneliades/flash_torrent/client"
	"github.com/johneliades/flash_torrent/peer"
	"github.com/marksamman/bencode"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type torrentFile struct {
	announce    string
	infoHash    [20]byte
	pieces      [][20]byte
	pieceLength int

	singleFile bool

	//used in single file only, it is the single file's length
	length int

	// when torrent has a single file this is the file name
	// when multiple files this is the directory
	name string

	//list of file lengths and paths, used only when multiple files
	files []struct {
		length int
		path   []string
	}
}

func btoTorrentStruct(file_bytes io.Reader) torrentFile {
	data, err := bencode.Decode(file_bytes)
	check(err)

	announce := data["announce"].(string)

	bencodeInfo := data["info"].(map[string]interface{})
	pieceStr := bencodeInfo["pieces"].(string)
	pieceLength := int(bencodeInfo["piece length"].(int64))
	name := bencodeInfo["name"].(string)

	//sha1 hash of bencoded info
	buf := bencode.Encode(bencodeInfo)
	infoHash := sha1.Sum(buf)

	//split string of hashes in [][20]byte
	pieces := [][20]byte{}
	var l, r int
	var temp [20]byte
	for l, r = 0, 20; r <= len(pieceStr); l, r = r, r+20 {
		copy(temp[:], pieceStr[l:r])
		pieces = append(pieces, [20]byte(temp))
	}

	//common fields
	torrent := torrentFile{}
	torrent = torrentFile{
		announce:    announce,
		infoHash:    infoHash,
		pieces:      pieces,
		pieceLength: pieceLength,
		name:        name,
	}

	if _, ok := bencodeInfo["files"]; ok {
		//multiple files

		torrent.singleFile = false

		for _, element := range bencodeInfo["files"].([]interface{}) {
			file_dict := element.(map[string]interface{})
			var temp_path []string
			for _, path := range file_dict["path"].([]interface{}) {
				temp_path = append(temp_path, path.(string))
			}

			torrent.files = append(torrent.files, struct {
				length int
				path   []string
			}{
				int(file_dict["length"].(int64)),
				temp_path,
			})
		}
	} else {
		//single file
		torrent.length = int(bencodeInfo["length"].(int64))
		torrent.singleFile = true
	}

	return torrent
}

func (torrent *torrentFile) getPeers(peerID string, port int) []peer.Peer {
	var body []byte

	if strings.HasPrefix(torrent.announce, "http") {
		// ============== HTTP Tracker ==============

		base, ok := url.Parse(torrent.announce)
		check(ok)

		params := url.Values{}
		params.Add("info_hash", string(torrent.infoHash[:]))
		// peer_id must be 20 bytes
		params.Add("peer_id", peerID)
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

		conn, ok := net.Dial("udp", torrent.announce[len("udp://"):len(torrent.announce)-len("/announce")])
		check(ok)

		b1 := make([]byte, 8)
		b2 := make([]byte, 4)
		b3 := make([]byte, 4)

		binary.BigEndian.PutUint64(b1, uint64(0x41727101980))
		binary.BigEndian.PutUint32(b2, uint32(0))
		binary.BigEndian.PutUint32(b3, uint32(10000))

		buf := []byte{}
		buf = append(buf, b1...)
		buf = append(buf, b2...)
		buf = append(buf, b3...)

		_, ok = conn.Write(buf)
		check(ok)

		//receiver
		var action int64
		ok = binary.Read(conn, binary.BigEndian, &action)
		check(ok)
	
		print("\n")
		print(action)
		print("\n")
	//	panic("UDP tracker not supported yet.")
	}

	data, err := bencode.Decode(bytes.NewReader(body[:]))
	check(err)

	// in seconds, for connecting to the tracker again
	//interval := data["interval"].(int64)

	return peer.Deserialize([]byte(data["peers"].(string)))
}

func Open(path string) torrentFile {
	file, ok := os.Open(path)
	check(ok)

	return btoTorrentStruct(file)
}

func (torrent *torrentFile) Download(path string) {
	rand.Seed(time.Now().UnixNano())

	var peerID [20]byte
	_, ok := rand.Read(peerID[:])
	check(ok)

	//	peer_string := "git:johneliades-" + string(peerID[:])

	peers := torrent.getPeers(string(peerID[:]), 3000)

	_, ok = client.New(peers[0], peerID, torrent.infoHash)
	check(ok)

	fmt.Printf("%v", peers)
}
