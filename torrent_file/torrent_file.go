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

type torrentFile struct {
	announce     string
	announceList []string
	infoHash     [20]byte
	pieces       [][20]byte
	pieceLength  int

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
	torrent := torrentFile{}
	torrent = torrentFile{
		announce:     announce,
		announceList: announceList,
		infoHash:     infoHash,
		pieces:       pieces,
		pieceLength:  pieceLength,
		name:         name,
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

		conn.SetDeadline(time.Now().Add(5 * time.Second))
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
		buf_res = make([]byte, 20 + 10*6) // 20 header and 10 clients of 6 bytes each
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

func Open(path string) torrentFile {
	file, ok := os.Open(path)
	if ok != nil {
		panic(ok)
	}

	return btoTorrentStruct(file)
}

func (torrent *torrentFile) Download(path string) {
	rand.Seed(time.Now().UnixNano())

	var peerID [20]byte
	_, ok := rand.Read(peerID[:])
	if ok != nil {
		panic(ok)
	}

	//	peer_string := "git:johneliades-" + string(peerID[:])

	print("\n")

	var tracker string
	var peers []peer.Peer

	for i := 0; i < len(torrent.announceList); i++ {
		tracker = torrent.announceList[i]
		print("Trying tracker: " + tracker)
		peers, ok = torrent.getPeers(tracker, string(peerID[:]), 3000)
		if ok == nil {
			print(" - Success\n")
			break
		}
		print(" - " + ok.Error() + "\n")
	}

	peers = unique(peers)
	fmt.Printf("\nFound peers: %v\n\n", peers)

	for i := 0; i < len(peers); i++ {
		print("Connecting: " + peers[i].String())
		_, ok = client.New(peers[i], peerID, torrent.infoHash)
		if ok == nil {
			print(" - Success\n")
		} else {
			print(" - " + ok.Error() + "\n")
		}
	}
}
