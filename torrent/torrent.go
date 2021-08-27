package torrent

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"github.com/johneliades/flash/client"
	"github.com/johneliades/flash/message"
	"github.com/johneliades/flash/peer"
	"os"
	"time"
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

// MaxBlockSize is the largest number of bytes a request can ask for
const MaxBlockSize = 16384

// MaxBacklog is the number of unfulfilled requests a client can have in its pipeline
const MaxBacklog = 5

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

type pieceProgress struct {
	index      int
	client     *client.Client
	buf        []byte
	downloaded int
	requested  int
	backlog    int
}

func (torrent *Torrent) startDownload(peer peer.Peer, workQueue chan *pieceWork, results chan *pieceResult) {
	c, err := client.New(peer, torrent.PeerID, torrent.InfoHash)

	if err == nil {
		println(peer.String() + " - " + Green + "Success" + Reset)
	} else {
		println(peer.String() + Red + " - " + err.Error() + Reset)
	}

	for pw := range workQueue {
		if !c.BitField.HasPiece(pw.index) {
			workQueue <- pw // Put piece back on the queue
			continue
		}

		buf, err := attemptDownloadPiece(c, pw)
		if err != nil {
			//println(Red + "Exiting" + Reset, err)
			workQueue <- pw // Put piece back on the queue
			return
		}

		hash := sha1.Sum(buf)
		if !bytes.Equal(hash[:], pw.hash[:]) {
			fmt.Printf(Red+"Piece #%d failed integrity check, retrying.\n"+Reset, pw.index)
			workQueue <- pw // Put piece back on the queue
			continue
		}

		//		c.SendHave(pw.index)
		results <- &pieceResult{pw.index, buf}
	}
}

func (state *pieceProgress) readMessage() error {
	msg, err := message.Read(state.client.Conn)
	if err != nil {
		return err
	}

	if msg == nil { // keep-alive
		return nil
	}

	switch msg.ID {
	case message.Unchoke:
		state.client.Choked = false
	case message.Choke:
		state.client.Choked = true
	case message.Have:
		index, err := message.ParseHave(msg)
		if err != nil {
			return err
		}
		state.client.BitField.SetPiece(index)
	case message.Piece:
		n, err := message.ParsePiece(state.index, state.buf, msg)
		if err != nil {
			return err
		}
		state.downloaded += n
		state.backlog--
	}
	return nil
}

func attemptDownloadPiece(c *client.Client, pw *pieceWork) ([]byte, error) {
	state := pieceProgress{
		index:  pw.index,
		client: c,
		buf:    make([]byte, pw.length),
	}

	// Setting a deadline helps get unresponsive peers unstuck.
	// 30 seconds is more than enough time to download a 262 KB piece
	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{}) // Disable the deadline

	for state.downloaded < pw.length {
		// If unchoked, send requests until we have enough unfulfilled requests
		if !state.client.Choked {
			for state.backlog < MaxBacklog && state.requested < pw.length {
				blockSize := MaxBlockSize
				// Last block might be shorter than the typical block
				if pw.length-state.requested < blockSize {
					blockSize = pw.length - state.requested
				}

				payload := make([]byte, 12)
				binary.BigEndian.PutUint32(payload[0:4], uint32(pw.index))
				binary.BigEndian.PutUint32(payload[4:8], uint32(state.requested))
				binary.BigEndian.PutUint32(payload[8:12], uint32(blockSize))

				req := &message.Message{ID: message.Request, Payload: payload}

				_, err := c.Conn.Write(req.Serialize())
				if err != nil {
					return nil, err
				}
				state.backlog++
				state.requested += blockSize
			}
		}

		err := state.readMessage()
		if err != nil {
			return nil, err
		}
	}

	return state.buf, nil
}

func (torrent *Torrent) Download(path string) {
	if len(torrent.Files) == 0 {
		f, err := os.Create(torrent.Name)
		if err != nil {
			println("create file error")
			return
		}
		defer f.Close()
	} else {
		//print("\nhad multiple files\n")
	}

	file, _ := os.OpenFile(torrent.Name, os.O_RDWR, 0660)
	defer file.Close()

	workQueue := make(chan *pieceWork, len(torrent.PieceHashes))
	results := make(chan *pieceResult)
	for index, hash := range torrent.PieceHashes {
		begin := index * torrent.PieceLength
		end := begin + torrent.PieceLength
		if end > torrent.Length {
			end = torrent.Length
		}
		// Wrong for multiple files
		workQueue <- &pieceWork{index, hash, end - begin}
	}

	for _, peer := range torrent.Peers {
		go torrent.startDownload(peer, workQueue, results)
	}

	donePieces := 0

	for donePieces < len(torrent.PieceHashes) {
		res := <-results

		donePieces++

		if len(torrent.Files) == 0 {
			_, err := file.Seek(int64(res.index*torrent.PieceLength), 0)
			if err != nil {
				fmt.Printf(Red+"%v"+Reset, err)
				return
			}

			bytesWritten, err := file.Write(res.buf)
			if err != nil || bytesWritten != len(res.buf) {
				fmt.Printf(Red+"%v"+Reset, err)
				return
			}
		} else {
			//print("\nhad multiple files\n")
		}

		percent := float64(donePieces) / float64(len(torrent.PieceHashes)) * 100
		
		print("\r")
		for i := 0; i <= 100; i++ {
			print(" ")
		}

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

	print("\r")
	for i := 0; i <= 100; i++ {
		print(" ")
	}

	print("\r|" + Green)

	for i := 0; i <= 50; i++ {
		print("=")
	}
	print(Reset + "| 100%")
	for i := 0; i <= 25; i++ {
		print(" ")
	}

	close(workQueue)

	//	return buf
}
