# flash

A BitTorrent client made in golang as a Diploma Thesis.

![Image of crawler](https://github.com/johneliades/flash/blob/main/preview.png)

## How to use

Sample main:
```
package main

import (
	"github.com/johneliades/flash/torrent_file"
	"fmt"
)

func main() {
	torrent, err := torrent_file.Open("torrent_path", false)
	if err != nil {
		fmt.Errorf("%v", err)
	}
	torrent.Download("downloads")
}
```
Where "torrent_path" is the path to the .torrent file, "false"  
deactivates the debugging information and "downloads" is the name 
of the folder the downloaded files go in.

By running:

```
go mod init [module path]
go mod tidy
```

Golang downloads the flash library automatically.

## Author

**Eliades John** - *Developer* - [Github](https://github.com/johneliades)
