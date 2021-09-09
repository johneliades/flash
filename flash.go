package main

import (
	"github.com/johneliades/flash/torrent_file"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	//single file
	torrent, err := torrent_file.Open("torrents/netrunner-desktop-2101-64bit.iso.torrent", false)

	//multiple files
	//torrent, err := torrent_file.Open("torrents/38979EE94106A4F586AA024649B0ABE331F49141.torrent", false)
	check(err)

	torrent.Download("downloads")
}
