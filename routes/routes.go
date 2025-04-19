package routes

import (
	"fmt"
	"math"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/johneliades/flash/torrent"
	"github.com/johneliades/flash/torrent_file"
)

var activeTorrents map[string]*torrent.TorrentStatus

func RegisterRoutes(r *gin.Engine) {
	activeTorrents = make(map[string]*torrent.TorrentStatus)

	r.POST("/start-download", func(c *gin.Context) {
		file, err := c.FormFile("torrent")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tmpFile, err := os.CreateTemp("", "*.torrent")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		if err := c.SaveUploadedFile(file, tmpFile.Name()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		fmt.Printf("Starting download for torrent: %s\n", tmpFile.Name())

		torrent, err := torrent_file.Open(tmpFile.Name(), false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		activeTorrents[file.Filename] = &torrent.Status
		torrent.Download("") 

		c.JSON(http.StatusOK, gin.H{"message": "Download started"})
	})

	// /download-progress route: Use the torrent file name to track progress
	r.GET("/download-progress", func(c *gin.Context) {
		torrentName := c.DefaultQuery("torrent_name", "")
		
		if torrent, exists := activeTorrents[torrentName]; exists {
			// Create a response object in the expected format
			response := gin.H{
				"name":           torrentName,
				"progress":       math.Round(torrent.Progress*100) / 100, // Round to two decimal places
				"downloadSpeed":  torrent.DownSpeed,
				"size":           torrent.Size,
			}

			// Send the response back to the frontend
			c.JSON(http.StatusOK, response)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Torrent not found"})
		}
	})
}