package main

import (
	"fmt"
	"github.com/johneliades/flash_torrent/torrent_decode"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	file, ok := os.Open("torrents/netrunner-desktop-2101-64bit.iso.torrent")
	//	file, ok := os.Open("torrents/AFC634F60782AE4EA51D2BBFF506479F613CF761.torrent")
	check(ok)

	torrent := torrent_decode.BtoTorrentStruct(file)
	trackerResponse := torrent.GetTrackerResponse(3000)

	fmt.Printf("%v", trackerResponse)

}
