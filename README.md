# flash

A BitTorrent client made in golang as a Diploma Thesis.

![Image of crawler](https://github.com/johneliades/flash/blob/main/preview.png)

## How to use

Sample main:
```
package main

import (
	"github.com/johneliades/flash"
	"fmt"
)

func main() {
	err := flash.DownloadTorrent("torrent_path", "downloads", false)
	if err != nil {
		fmt.Errorf("%v", err)
	}
}
```
Where "torrent_path" is the path to the .torrent file, "false" deactivates the 
debugging information and "downloads" is the name of the folder the downloaded 
files go in.

By running:

```
go mod init main
go mod tidy
go run main.go
```

Golang downloads the flash library automatically and starts the torrent download.

## Author

**Eliades John** - *Developer* - [Github](https://github.com/johneliades)
