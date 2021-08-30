package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/johneliades/flash/client"
	"github.com/johneliades/flash/message"
	"github.com/johneliades/flash/peer"
	"path/filepath"
	"strconv"
	"os"
	"time"
	"reflect"
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

// maxBacklog is the number of unfulfilled requests a client can have in its pipeline
var maxBacklog int = 5

type File struct {
	Length int
	Path   []string
}

type Torrent struct {
	Peers       chan *peer.Peer
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

func (torrent *Torrent) startPeer(peer peer.Peer, workQueue chan *pieceWork, results chan *pieceResult) {
	c, err := client.New(peer, torrent.PeerID, torrent.InfoHash)

	if err == nil {
		println(peer.String() + " - " + Green + "Success" + Reset)
	} else {
		println(peer.String() + Red + " - " + err.Error() + Reset)
		return
	}

	c.SendUnchoke()
	c.SendInterested()

	for pw := range workQueue {
		if !c.BitField.HasPiece(pw.index) {
			workQueue <- pw // Put piece back on the queue
			continue
		}

		buf, err := getPiece(c, pw)
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

		c.SendHave(pw.index)
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

func getPiece(c *client.Client, pw *pieceWork) ([]byte, error) {
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
			for state.backlog < maxBacklog && state.requested < pw.length {
				blockSize := MaxBlockSize
				// Last block might be shorter than the typical block
				if pw.length-state.requested < blockSize {
					blockSize = pw.length - state.requested
				}

				err := c.SendRequest(pw.index, state.requested, blockSize)
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

func ByteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

func (torrent *Torrent) Download(downloadLocation string) {
	var fileArray []os.File
	var fileSingle os.File

	if _, err := os.Stat(downloadLocation); os.IsNotExist(err) {
		err = os.Mkdir(downloadLocation, 0755)
		if err != nil {
			fmt.Printf(Red+"%v"+Reset, err)
		}
	}

	if len(torrent.Files) == 0 {
		// Single file in torrent
		f, err := os.Create(filepath.Join(downloadLocation, torrent.Name))
		if err != nil {
			fmt.Printf(Red+"%v"+Reset, err)
			return
		}
		fileSingle = *f
		defer f.Close()
	} else {
		// Multiple files in torrent
		
		// Creation of core directory
		if _, err := os.Stat(filepath.Join(downloadLocation, torrent.Name)); os.IsNotExist(err) {
			err := os.Mkdir(filepath.Join(downloadLocation, torrent.Name), 0755)
			if err != nil {
				fmt.Printf(Red+"%v"+Reset, err)
			}
		}

		// Get each "file" in the torrent
		for _, file := range torrent.Files {
			// create the nested directories
			path := filepath.Join(downloadLocation, torrent.Name)
			for _, f := range file.Path[:len(file.Path)-1] {
				path = filepath.Join(path, f) 
			}

			if _, err := os.Stat(path); os.IsNotExist(err) {
				err := os.MkdirAll(path, 0755)
				if err != nil {
					fmt.Printf(Red+"%v"+Reset, err)
				}
			}
			
			// and then the file
			f, err := os.Create(filepath.Join(path, file.Path[len(file.Path)-1]))
			if err != nil {
				println("create file error")
				return
			}
			fileArray = append(fileArray, *f)

			defer f.Close()
		}
	}

	numPieces := 0

	workQueue := make(chan *pieceWork, len(torrent.PieceHashes))
	results := make(chan *pieceResult)
	for index, hash := range torrent.PieceHashes {
		begin := index * torrent.PieceLength
		end := begin + torrent.PieceLength
		if end > torrent.Length {
			end = torrent.Length
		}
		numPieces++
		workQueue <- &pieceWork{index, hash, end - begin}
	}

	// Connect to each peer once
	var peersUsed []peer.Peer

	SKIP:
	for peer := range torrent.Peers {
		for _, v := range peersUsed {
			if reflect.DeepEqual(v, *peer) {
				continue SKIP
			}
		}

		go torrent.startPeer(*peer, workQueue, results)
		peersUsed = append(peersUsed, *peer)
	}

	println(Green + "Download started" + Reset)

	donePieces := 0
    	
    start := time.Now()

	for donePieces < len(torrent.PieceHashes) {
		res := <-results

		donePieces++
		
		rate := float64(torrent.PieceLength) / time.Since(start).Seconds()
		if (rate/1024 < 20) {
			maxBacklog = int(rate/1024 + 2);
		} else {
			maxBacklog = int(rate/1024/5 + 18);
		}

		if len(torrent.Files) == 0 {
			_, err := fileSingle.Seek(int64(res.index*torrent.PieceLength), 0)
			if err != nil {
				fmt.Printf(Red+"%v"+Reset, err)
				return
			}

			bytesWritten, err := fileSingle.Write(res.buf)
			if err != nil || bytesWritten != len(res.buf) {
				fmt.Printf(Red+"%v"+Reset, err)
				return
			}
		} else {
			//print("\nhad multiple files\n")
		}

		start = time.Now()

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

		fmt.Printf("%0.2f%% - #%s, %d left, %v", percent, Green + strconv.Itoa(res.index) + Reset,
			numPieces - donePieces, ByteCountIEC(int64(rate)))
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
