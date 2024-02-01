package flash

import (
	"github.com/johneliades/flash/torrent_file"
)

func DownloadTorrent(path string, destination string, debug_flag bool) error {
    torrent, err := torrent_file.Open(path, false)
    if err != nil {
        return err
    }
    torrent.Download(destination)
    return nil
}