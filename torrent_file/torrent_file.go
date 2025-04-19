package torrent_file

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/johneliades/flash/peer"
	"github.com/johneliades/flash/torrent"
	"github.com/marksamman/bencode"
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

	announce := ""
	if val, ok := data["announce"].(string); ok {
		announce = val
	}

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
			t.length += int(file_dict["length"].(int64))
		}
	} else {
		//single file
		t.length = int(bencodeInfo["length"].(int64))
	}

	return t
}

func (t *torrentFile) getPeers(tracker string, peers chan *peer.Peer, wg *sync.WaitGroup,
	peerID string, port int) {

	var body []byte

	defer wg.Done()

	if strings.HasPrefix(tracker, "http") {
		// ============== HTTP Tracker ==============

		base, ok := url.Parse(tracker)
		if ok != nil {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		params := url.Values{}
		params.Add("info_hash", string(t.infoHash[:]))
		// peer_id must be 20 bytes
		params.Add("peer_id", peerID)
		params.Add("port", strconv.Itoa(port))
		params.Add("uploaded", "0")
		params.Add("downloaded", "0")
		params.Add("left", strconv.Itoa(t.length))
		params.Add("compact", "1")

		base.RawQuery = params.Encode()

		// communicate with tracker url

		resp, ok := http.Get(base.String())
		if ok != nil {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		defer resp.Body.Close()

		body, ok = io.ReadAll(resp.Body)
		if ok != nil {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		data, ok := bencode.Decode(bytes.NewReader(body[:]))
		if ok != nil {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		// in seconds, for connecting to the tracker again
		//interval := data["interval"].(int64)

		val := data["peers"]
		if val != nil {
			// Wrong if peers are dictionary?
			for _, peer := range peer.Deserialize([]byte(data["peers"].(string))) {
				peers <- &peer
			}
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Green + "Success" + Reset)
			}
			return
		}
	} else if strings.HasPrefix(tracker, "udp") {
		// ============== UDP Tracker ==============
		// https://libtorrent.org/udp_tracker_protocol.html#authentication
		// http://xbtt.sourceforge.net/udp_tracker_protocol.html

		conn, ok := net.Dial("udp", tracker[len("udp://"):len(tracker)-len("/announce")])
		if ok != nil {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		conn.SetDeadline(time.Now().Add(1 * time.Second))
		defer conn.SetDeadline(time.Time{})

		transaction := rand.Uint32()

		buf := make([]byte, 16) // 8 + 4 + 4
		binary.BigEndian.PutUint64(buf, uint64(0x41727101980))
		binary.BigEndian.PutUint32(buf[8:], uint32(0))
		binary.BigEndian.PutUint32(buf[12:], uint32(transaction))
		_, ok = conn.Write(buf)
		if ok != nil {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		buf_res := make([]byte, 16) // 8 + 4 + 4

		ok = binary.Read(conn, binary.BigEndian, &buf_res)
		if ok != nil {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		action := binary.BigEndian.Uint32(buf_res[:4])
		if action != 0 {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		transaction_id := binary.BigEndian.Uint32(buf_res[4:8])
		if transaction != transaction_id {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		connection_id := binary.BigEndian.Uint64(buf_res[8:])

		transaction = rand.Uint32()
		key := rand.Uint32()

		buf = make([]byte, 98)
		binary.BigEndian.PutUint64(buf, connection_id)
		binary.BigEndian.PutUint32(buf[8:], uint32(1))
		binary.BigEndian.PutUint32(buf[12:], uint32(transaction))
		copy(buf[16:], t.infoHash[:])
		copy(buf[36:], peerID[:])
		binary.BigEndian.PutUint64(buf[56:], 0)
		binary.BigEndian.PutUint64(buf[64:], uint64(t.length))
		binary.BigEndian.PutUint64(buf[72:], 0)
		binary.BigEndian.PutUint32(buf[80:], 0)
		binary.BigEndian.PutUint32(buf[84:], 0)
		binary.BigEndian.PutUint32(buf[88:], key)
		copy(buf[92:], []byte(strconv.Itoa(-1)))
		binary.BigEndian.PutUint16(buf[96:], uint16(port))

		_, ok = conn.Write(buf)
		if ok != nil {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		// Wrong cause it blocks if there are less than 15 clients
		// and ignores the rest if there are more
		//buf_res = make([]byte, 20+15*6)
		//ok = binary.Read(conn, binary.BigEndian, &buf_res)
		//but uses big endian

		buf_res = make([]byte, 20+30*6) // 20 header and 30 clients of 6 bytes each
		_, ok = bufio.NewReader(conn).Read(buf_res)
		if ok != nil {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		action = binary.BigEndian.Uint32(buf_res[:4])
		if action != 1 {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		transaction_id = binary.BigEndian.Uint32(buf_res[4:8])
		if transaction != transaction_id {
			if torrent.Debug {
				println("\rTrying tracker: " + tracker + " - " + Red + ok.Error() + Reset)
			}
			return
		}

		interval := binary.BigEndian.Uint32(buf_res[8:12])
		leechers := binary.BigEndian.Uint32(buf_res[12:16])
		seeders := binary.BigEndian.Uint32(buf_res[16:20])

		_ = interval
		_ = leechers
		_ = seeders

		buf_res = buf_res[20:]

		for _, peer := range peer.Deserialize(buf_res) {
			peers <- &peer
		}
		if torrent.Debug {
			println("\rTrying tracker: " + tracker + " - " + Green + "Success" + Reset)
		}
		return
	}

	if torrent.Debug {
		println("\rTrying tracker: " + tracker + " - " + Red + "Faulty tracker" + Reset)
	}
}

func Open(path string, debug bool, data ...[]byte) (torrent.Torrent, error) {
    var reader io.Reader

    if len(data) > 0 && len(data[0]) > 0 {
        // If data is passed, use it
        reader = bytes.NewReader(data[0])
    } else {
        // If data is not passed, read from file
        file, err := os.Open(path)
        if err != nil {
            return torrent.Torrent{}, err
        }
        defer file.Close()
        reader = file
    }

    torrent.Debug = debug

    rand.Seed(time.Now().UnixNano())
    var peerID [20]byte
    _, err := rand.Read(peerID[:])
    if err != nil {
        return torrent.Torrent{}, err
    }

    var tracker string
    peers := make(chan *peer.Peer)
    wg := &sync.WaitGroup{}

    t := btoTorrentStruct(reader)
    for i := 0; i < len(t.announceList); i++ {
        tracker = t.announceList[i]

        wg.Add(1)
        go t.getPeers(tracker, peers, wg, string(peerID[:]), 3000)
    }

    go func() {
        wg.Wait()
        close(peers)
    }()

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