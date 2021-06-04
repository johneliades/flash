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
	"crypto/sha1"
	"os"
	"fmt"
	"github.com/marksamman/bencode"
	"github.com/johneliades/flash_torrent/peer"
)

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

func check(e error) {
	if e != nil {
		panic(e)
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

func (torrent *torrentFile) GetPeers(port int) []peer.Peer {
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

	// in seconds, for connecting to the tracker again
	//interval := data["interval"].(int64)

	return peer.ExtractPeers([]byte(data["peers"].(string)))
}

func Open(path string) torrentFile {
	file, ok := os.Open(path)
	check(ok)

	return btoTorrentStruct(file)
}

func (torrent *torrentFile) Download(path string) {
	peers := torrent.GetPeers(3000)
	fmt.Printf("%v", peers)
}