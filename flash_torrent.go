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
	torrent := torrent_file.Open("torrents/netrunner-desktop-2101-64bit.iso.torrent")
	//torrent_file.Open("torrents/AFC634F60782AE4EA51D2BBFF506479F613CF761.torrent")
	//torrent_file.Open("torrents/2A7312B8CE79E9633821B1B43FF8143849D6742F.torrent")

	torrent.Download(".")
}
