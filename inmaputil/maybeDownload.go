package inmaputil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cavaliercoder/grab"
	"github.com/mholt/archiver"
)

// download checks if the input is an existing file locally.
// If not, it checks if the file is a URL.
// If it's a URL, it downloads the file and decompresses it if needed.
// It then returns the path to the Shp file.
// c, if not nil, is a channel across which logging messages will be sent.
func maybeDownload(path string, c chan string) string {
	// Check if local file exists. If it does, return the given path.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path
	}

	// If the input doesn't start with these prefixes, then return the input.
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		return path
	}

	// Prepare a temporary directory for the downloads.
	tempDir, err := ioutil.TempDir("", "inmap")
	if err != nil {
		panic(fmt.Errorf("inmaputil: failed creating temporary download directory: %v", err))
		return path
	}

	// Start the downloading client.
	client := grab.NewClient()
	req, err := grab.NewRequest(tempDir, path)
	if err != nil {
		panic(fmt.Errorf("Failed creating downloading Request."))
	}

	c <- fmt.Sprintf("Downloading ... %v\n", req.URL())
	resp := client.Do(req)

	// Start a ticker to print out progress.
	t := time.NewTicker(1000 * time.Millisecond)
	defer t.Stop()

Loop:
	for {
		select {
		case <-t.C:
			c <- fmt.Sprintf("   transferred %v / %v bytes", resp.BytesComplete(), resp.Size)
		case <-resp.Done:
			break Loop
		}
	}

	if err := resp.Err(); err != nil {
		c <- fmt.Sprintf("Download failed: %v", err)
		return path
	}

	c <- fmt.Sprintf("Download saved to ./%v \n", resp.Filename)

	// Get the file name of the downloaded file, excluding file extension
	// and use that as the folder name to extract into.
	folderName := strings.Split(resp.Filename, ".")[0]

	// Check if the downloaded file is compressed.
	// If so, decompress it according to its extension.
	switch filepath.Ext(resp.Filename) {
	case ".zip":
		c <- ("Decompressing...\n")
		err := archiver.Zip.Open(resp.Filename, folderName)
		if err != nil {
			c <- fmt.Sprintf("Decompressing failed: %v", err)
			return resp.Filename
		}
	case ".tar.gz":
		c <- ("Decompressing...\n")
		err := archiver.TarGz.Open(resp.Filename, folderName)
		if err != nil {
			c <- fmt.Sprintf("Decompressing failed: %v", err)
			return resp.Filename
		}
	case ".tar":
		c <- ("Decompressing...\n")
		err := archiver.Tar.Open(resp.Filename, folderName)
		if err != nil {
			c <- fmt.Sprintf("Decompressing failed: %v", err)
			return resp.Filename
		}
	default:
		// If the file is not compressed, return the path to the downloaded file.
		return resp.Filename
	}

	// Check if there is a file with the ".shp" extension in the extracted directory.
	// If so, return the path to it.
	foundShp := ""

	err = filepath.Walk(folderName, func(path string, info os.FileInfo,
		err error) error {
		if err != nil {
			panic(fmt.Errorf("There was a problem reading extracted file. "))
		}
		if foundShp == "" {
			foundShp = path
		}
		if !info.IsDir() && strings.HasSuffix(path, ".shp") {
			foundShp = path
		}
		return nil
	})

	if err != nil {
		c <- fmt.Sprintf("Download failed: %v", err)
		return resp.Filename
	} else {
		return foundShp
	}

	// Return the downloaded file name if no Shp file found.
	return resp.Filename
}
