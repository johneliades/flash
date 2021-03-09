package main

import (
	"fmt"
	"io"
	"os"
	"github.com/marksamman/bencode"
)

type torrentFile struct {
	announce     string
	pieces       string
	pieceLength  int

	//used in single file only, it is the single file's length
	length       int 

	// when torrent has a single file this is the file name
	// when multiple files this is the directory
	name         string 

	//list of file lengths and paths, used only when multiple files
	files []struct { 
		length    int
		path      []string
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func BtoTorrent(file_bytes io.Reader) (torrentFile) {
	data, err := bencode.Decode(file_bytes)
	check(err)

	announce := data["announce"].(string)

	bencodeInfo := data["info"].(map[string]interface{})
	pieces := bencodeInfo["pieces"].(string)
	pieceLength := int(bencodeInfo["piece length"].(int64))
	name := bencodeInfo["name"].(string)

	torrent := torrentFile{}
	if _, ok := bencodeInfo["files"]; ok {
		//multiple files

		torrent = torrentFile{announce:announce, pieces:pieces, pieceLength:pieceLength, name:name}
		for _, element := range bencodeInfo["files"].([]interface{}) {
			file_dict := element.(map[string]interface{})
			temp_length := int(file_dict["length"].(int64))
			var temp_path []string
			for _, path := range file_dict["path"].([]interface{}) {
				temp_path = append(temp_path, path.(string))
			}

			torrent.files = append(torrent.files, struct {
					length    int
					path      []string
				}{
					temp_length,
					temp_path,
				})
		}		
	} else {
		//single file
		length := int(bencodeInfo["length"].(int64))
		torrent = torrentFile{announce:announce, pieces:pieces, pieceLength:pieceLength, length:length, name:name}
	}

	return torrent
}

func main() {
//	file, err := os.Open("netrunner-desktop-2101-64bit.iso.torrent")
	file, err := os.Open("AFC634F60782AE4EA51D2BBFF506479F613CF761.torrent")
	check(err)

	torrent := BtoTorrent(file)

	fmt.Printf("%v", torrent.files)
}