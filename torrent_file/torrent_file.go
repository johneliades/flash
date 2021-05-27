package torrent_file

import (
	"crypto/sha1"
	"io"
	"os"
	"fmt"
	"github.com/marksamman/bencode"
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

func Open(path string) {
	file, ok := os.Open(path)
	check(ok)

	torrent := btoTorrentStruct(file)
	peers := torrent.GetPeers(3000)
	fmt.Printf("%v", peers)
}