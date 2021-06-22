package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/johneliades/flash_torrent/client"
	"log"
	//	"github.com/johneliades/flash_torrent/message"
	"github.com/johneliades/flash_torrent/peer"
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

type File struct {
	Length int
	Path   []string
}

type Torrent struct {
	Peers       []peer.Peer
	PeerID      [20]byte
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
	Files       []File
}

type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	index int
	buf   []byte
}

func (torrent *Torrent) startDownload(peer peer.Peer, workQueue chan *pieceWork, results chan *pieceResult) {
	c, err := client.New(peer, torrent.PeerID, torrent.InfoHash)

	if err == nil {
		log.Printf("\r" + peer.String() + " - " + Green + "Success" + Reset)
	} else {
		log.Printf("\r" + peer.String() + Red + " - " + err.Error() + Reset)
	}

	for pw := range workQueue {
		if !c.BitField.HasPiece(pw.index) {
			workQueue <- pw // Put piece back on the queue
			continue
		}

		// // Download the piece
		// buf, err := attemptDownloadPiece(c, pw)
		// if err != nil {
		// 	log.Println("Exiting", err)
		// 	workQueue <- pw // Put piece back on the queue
		// 	return
		// }

		buf := []byte("Asd")
		_ = sha1.Sum(buf)
		hash := pw.hash[:]
		if !bytes.Equal(hash[:], pw.hash[:]) {
			fmt.Printf("Piece #%d failed integrity check\n", pw.index)
			workQueue <- pw // Put piece back on the queue
			continue
		}

		//		c.SendHave(pw.index)
		results <- &pieceResult{pw.index, buf}
	}
}

func (torrent *Torrent) Download(path string) {
	workQueue := make(chan *pieceWork, len(torrent.PieceHashes))
	results := make(chan *pieceResult)
	for index, hash := range torrent.PieceHashes {
		begin := index * torrent.PieceLength
		end := begin + torrent.PieceLength
		if end > torrent.Length {
			end = torrent.Length
		}
		workQueue <- &pieceWork{index, hash, end - begin}
	}

	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	for _, peer := range torrent.Peers {
		go torrent.startDownload(peer, workQueue, results)
	}

	buf := make([]byte, torrent.Length)
	donePieces := 0

	log.Printf("\n")

	for donePieces < len(torrent.PieceHashes) {
		res := <-results

		begin := res.index * torrent.PieceLength
		end := begin + torrent.PieceLength
		if end > torrent.Length {
			end = torrent.Length
		}
		copy(buf[begin:end], res.buf)
		donePieces++

		percent := float64(donePieces) / float64(len(torrent.PieceHashes)) * 100

		print("\r|" + Cyan)
		for i := 0; i <= 50; i++ {
			if i <= int(percent)/2 {
				print("=")
			} else {
				print(" ")
			}
		}
		print(Reset + "| ")
		fmt.Printf("%0.2f%% - piece #%d complete", percent, res.index)
	}

	print("\r|" + Green)
	for i := 0; i <= 50; i++ {
		print("=")
	}
	print(Reset + "| 100%")
	for i := 0; i <= 25; i++ {
		print(" ")
	}
	print("\n")

	close(workQueue)

	if len(torrent.Files) == 0 {
		print("\nhad single file\n")
	} else {
		print("\nhad multiple files\n")
	}

	//	return buf
}
