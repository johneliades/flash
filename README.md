# flash

A BitTorrent client made in golang as a Diploma Thesis.

![Image of crawler](https://github.com/johneliades/flash/blob/main/preview.png)

## Clone

Clone the repository locally by entering the following command:
```
git clone https://github.com/johneliades/flash.git
```
Or by clicking on the green "Clone or download" button on top and then 
decompressing the zip.

## Run

After installing golang, you can run the example:

```
go run flash.go
```

## How to use the golang library

After including the flash library:
```
import (
	"github.com/johneliades/flash/torrent_file"
)
```

You can download any torrent file like this:
```
torrent, err := torrent_file.Open("path", false)
torrent.Download("downloads")
```
where "path" is the path to the .torrent file, "false" is if
debugging information should be shown and "downloads" is 
the name of the folder the files go in.

## Author

**Eliades John** - *Developer* - [Github](https://github.com/johneliades)
