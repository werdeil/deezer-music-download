package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var lastReqTime int64

var REQ_MIN_INTERVAL int64 = 500000000

func makeReq(method, url string, body io.Reader, config configuration) (*http.Response, error) {
	var err error

	tDiff := time.Now().UnixNano() - lastReqTime
	if tDiff < REQ_MIN_INTERVAL {
		time.Sleep(time.Duration(REQ_MIN_INTERVAL-tDiff) * time.Nanosecond)
	}
	lastReqTime = time.Now().UnixNano()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Pragma", "no-cache")
	req.Header.Add("Origin", "https://www.deezer.com")
	req.Header.Add("Accept-Language", "en-US,en;q=0.9")
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/68.0.3440.106 Safari/537.36")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Referer", "https://www.deezer.com/")
	req.Header.Add("DNT", "1")
	cookie := &http.Cookie{
		Name:  "arl",
		Value: config.Arl,
	}
	req.AddCookie(cookie)

	var res *http.Response
	res, err = http.DefaultClient.Do(req)
	for err != nil {
		log.Print("(network hiccup)")
		res, err = http.DefaultClient.Do(req)
	}
	return res, err
}

func printUsage() {
	log.Println("deezer-music-download is a program to freely download Deezer music files.")
	log.Println("")
	log.Println("To download one or more albums:")
	log.Println("\tdeezer-music-download album <album_id> [<album_id>...]")
	log.Println("")
	log.Println("To download one or more playlists:")
	log.Println("\tdeezer-music-download playlist <playlist_id> [<playlist_id>...]")
	log.Println("")
	log.Println("To start API server for Chrome extension:")
	log.Println("\tdeezer-music-download --server [port]")
	log.Println("")
	log.Println("See README for full details.")
}

func main() {
	var err error
	log.SetFlags(0)

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	config, err := getConfig()
	if err != nil {
		log.Fatalf("error reading config file: %s\n", err)
	}

	// Check for server mode
	if command == "--server" {
		port := "8080"
		if len(os.Args) > 2 {
			port = os.Args[2]
		}
		log.Printf("Starting API server on port %s for Chrome extension usage", port)
		StartServer(port, config)
		return
	}

	// CLI mode
	if len(os.Args) < 3 {
		printUsage()
		return
	}

	args := os.Args[2:]

	logFilePath := os.TempDir() + "/deezer-music-download.log"
	logFile, err := os.Create(logFilePath)
	if err != nil {
		log.Fatalf("error creating log file %s: %s\n", logFilePath, err)
	}
	defer logFile.Close()

	switch command {
	case "album":
		processAlbums(args, config, logFile)
	case "playlist":
		processPlaylists(args, config, logFile)
	default:
		printUsage()
		return
	}
}
