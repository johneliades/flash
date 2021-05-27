package main

import (
	"github.com/johneliades/flash_torrent/torrent_file"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	torrent_file.Open("torrents/netrunner-desktop-2101-64bit.iso.torrent")
	//torrent_file.Open("torrents/AFC634F60782AE4EA51D2BBFF506479F613CF761.torrent")
}
