package torrent_file

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"github.com/johneliades/flash_torrent/peer"
	"github.com/johneliades/flash_torrent/torrent"
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

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"
)

type torrentFile struct {
	announce     string
	announceList []string
	infoHash     [20]byte
	pieceHashes  [][20]byte
	pieceLength  int

	//used in single file only, it is the single file's length
	length int

	// when torrent has a single file this is the file name
	// when multiple files this is the directory
	name string

	//list of file lengths and paths, used only when multiple files
	files []torrent.File
}

func btoTorrentStruct(file_bytes io.Reader) torrentFile {
	data, ok := bencode.Decode(file_bytes)
	if ok != nil {
		panic(ok)
	}

	announce := data["announce"].(string)

	var announceList []string
	for _, tracker := range data["announce-list"].([]interface{}) {
		announceList = append(announceList, tracker.([]interface{})[0].(string))
	}

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
	t := torrentFile{}
	t = torrentFile{
		announce:     announce,
		announceList: announceList,
		infoHash:     infoHash,
		pieceHashes:  pieces,
		pieceLength:  pieceLength,
		name:         name,
	}

	if _, ok := bencodeInfo["files"]; ok {
		//multiple files

		for _, element := range bencodeInfo["files"].([]interface{}) {
			file_dict := element.(map[string]interface{})
			var temp_path []string
			for _, path := range file_dict["path"].([]interface{}) {
				temp_path = append(temp_path, path.(string))
			}

			t.files = append(t.files, torrent.File{
				int(file_dict["length"].(int64)),
				temp_path,
			})
		}
	} else {
		//single file
		t.length = int(bencodeInfo["length"].(int64))
	}

	return t
}

func (torrent *torrentFile) getPeers(tracker string, peerID string, port int) ([]peer.Peer, error) {
	var body []byte

	if strings.HasPrefix(tracker, "http") {
		// ============== HTTP Tracker ==============

		base, ok := url.Parse(tracker)
		if ok != nil {
			return []peer.Peer{}, ok
		}

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
		if ok != nil {
			return []peer.Peer{}, ok
		}

		defer resp.Body.Close()

		body, ok = io.ReadAll(resp.Body)
		if ok != nil {
			return []peer.Peer{}, ok
		}

		data, ok := bencode.Decode(bytes.NewReader(body[:]))
		if ok != nil {
			return []peer.Peer{}, ok
		}

		// in seconds, for connecting to the tracker again
		//interval := data["interval"].(int64)

		return peer.Deserialize([]byte(data["peers"].(string))), nil
	} else if strings.HasPrefix(tracker, "udp") {
		// ============== UDP Tracker ==============
		// https://libtorrent.org/udp_tracker_protocol.html#authentication
		// http://xbtt.sourceforge.net/udp_tracker_protocol.html

		conn, ok := net.Dial("udp", tracker[len("udp://"):len(tracker)-len("/announce")])
		if ok != nil {
			return []peer.Peer{}, ok
		}

		conn.SetDeadline(time.Now().Add(2 * time.Second))
		defer conn.SetDeadline(time.Time{})

		transaction := rand.Uint32()

		buf := make([]byte, 16) // 8 + 4 + 4
		binary.BigEndian.PutUint64(buf, uint64(0x41727101980))
		binary.BigEndian.PutUint32(buf[8:], uint32(0))
		binary.BigEndian.PutUint32(buf[12:], uint32(transaction))
		_, ok = conn.Write(buf)
		if ok != nil {
			return []peer.Peer{}, ok
		}

		buf_res := make([]byte, 16) // 8 + 4 + 4

		ok = binary.Read(conn, binary.BigEndian, &buf_res)
		if ok != nil {
			return []peer.Peer{}, ok
		}

		action := binary.BigEndian.Uint32(buf_res[:4])
		if action != 0 {
			return []peer.Peer{}, ok
		}

		transaction_id := binary.BigEndian.Uint32(buf_res[4:8])
		if transaction != transaction_id {
			return []peer.Peer{}, ok
		}

		connection_id := binary.BigEndian.Uint64(buf_res[8:])

		transaction = rand.Uint32()
		key := rand.Uint32()

		buf = make([]byte, 98)
		binary.BigEndian.PutUint64(buf, connection_id)
		binary.BigEndian.PutUint32(buf[8:], uint32(1))
		binary.BigEndian.PutUint32(buf[12:], uint32(transaction))
		copy(buf[16:], torrent.infoHash[:])
		copy(buf[36:], peerID[:])
		binary.BigEndian.PutUint64(buf[56:], 0)
		binary.BigEndian.PutUint64(buf[64:], uint64(torrent.length))
		binary.BigEndian.PutUint64(buf[72:], 0)
		binary.BigEndian.PutUint32(buf[80:], 0)
		binary.BigEndian.PutUint32(buf[84:], 0)
		binary.BigEndian.PutUint32(buf[88:], key)
		copy(buf[92:], []byte(strconv.Itoa(-1)))
		binary.BigEndian.PutUint16(buf[96:], uint16(port))

		_, ok = conn.Write(buf)
		if ok != nil {
			return []peer.Peer{}, ok
		}

		// Wrong cause it blocks if there are less than 10 clients
		// and ignores the rest if there are more
		buf_res = make([]byte, 20+15*6) // 20 header and 10 clients of 6 bytes each
		ok = binary.Read(conn, binary.BigEndian, &buf_res)
		if ok != nil {
			return []peer.Peer{}, ok
		}

		action = binary.BigEndian.Uint32(buf_res[:4])
		if action != 1 {
			return []peer.Peer{}, ok
		}

		transaction_id = binary.BigEndian.Uint32(buf_res[4:8])
		if transaction != transaction_id {
			return []peer.Peer{}, ok
		}

		interval := binary.BigEndian.Uint32(buf_res[8:12])
		leechers := binary.BigEndian.Uint32(buf_res[12:16])
		seeders := binary.BigEndian.Uint32(buf_res[16:20])

		_ = interval
		_ = leechers
		_ = seeders

		buf_res = buf_res[20:]

		return peer.Deserialize(buf_res), nil
	}

	return []peer.Peer{}, fmt.Errorf("Faulty tracker")
}

func unique(intSlice []peer.Peer) []peer.Peer {
	keys := make(map[string]bool)
	list := []peer.Peer{}
	for _, entry := range intSlice {
		if _, value := keys[entry.String()]; !value {
			keys[entry.String()] = true
			list = append(list, entry)
		}
	}
	return list
}

func Open(path string) (torrent.Torrent, error) {
	file, err := os.Open(path)
	if err != nil {
		return torrent.Torrent{}, err
	}

	rand.Seed(time.Now().UnixNano())
	var peerID [20]byte
	_, err = rand.Read(peerID[:])
	if err != nil {
		return torrent.Torrent{}, err
	}

	//	peer_string := "git:johneliades-" + string(peerID[:])

	print("\n")

	var tracker string
	var peers []peer.Peer

	t := btoTorrentStruct(file)
	for i := 0; i < len(t.announceList); i++ {
		tracker = t.announceList[i]
		print("Trying tracker: " + tracker)
		peers, err = t.getPeers(tracker, string(peerID[:]), 3000)
		if err == nil {
			println(" - " + Green + "Success" + Reset)
			break
		} else {
			println(" - " + Red + err.Error() + Reset)
		}
	}

	peers = unique(peers)
	fmt.Printf("\nFound peers: %v\n\n", peers)

	if len(peers) == 0 {
		return torrent.Torrent{}, fmt.Errorf("No peers found")
	}

	return torrent.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.infoHash,
		PieceHashes: t.pieceHashes,
		PieceLength: t.pieceLength,
		Length:      t.length,
		Name:        t.name,
		Files:       t.files,
	}, nil
}
