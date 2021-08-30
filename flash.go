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
	torrent, err := torrent_file.Open("torrents/netrunner-desktop-2101-64bit.iso.torrent")

	//multiple files
	//torrent, err := torrent_file.Open("torrents/AFC634F60782AE4EA51D2BBFF506479F613CF761.torrent")

	//multiple files
	//torrent, err := torrent_file.Open("torrents/2A7312B8CE79E9633821B1B43FF8143849D6742F.torrent")
	check(err)

	torrent.Download("downloads")
}
