# flash

A BitTorrent client made in golang as a Diploma Thesis.

![Image of crawler](https://github.com/johneliades/flash/blob/main/preview.png)

## How to use

Include the flash library in your code:
```
import (
	"github.com/johneliades/flash/torrent_file"
)
```
You can then download any torrent file like this:
```
torrent, err := torrent_file.Open("path", false)
torrent.Download("downloads")
```
where "path" is the path to the .torrent file, "false" is if
debugging information should be shown and "downloads" is 
the name of the folder the files go in.

By running:

```
go mod init [module path]
go mod tidy
```

Golang downloads the flash library automatically.

## Author

**Eliades John** - *Developer* - [Github](https://github.com/johneliades)
