package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

func processAlbums(args []string, config configuration, logFile *os.File) {
album_loop:
	for idx, albumId := range args {
		log.Printf("[%03d/%03d] Downloading album %s\n", idx+1, len(args), albumId)
		albumInfo, err := getAlbumSongs(albumId, config)
		if err != nil {
			log.Fatalf("error getting album songs: %s\n", err)
		}

		album, err := getAlbum(albumId, config)
		if err != nil {
			log.Fatalf("error getting album: %s\n", err)
		}

		// Ensure album.NbDiscs is set: compute from albumInfo if API didn't provide it
		if album.NbDiscs == 0 {
			maxDisc := 0
			for _, s := range albumInfo.Songs.Data {
				if s.DiskNumber != "" {
					if d, err := strconv.Atoi(s.DiskNumber); err == nil {
						if d > maxDisc {
							maxDisc = d
						}
					}
				}
			}
			if maxDisc > 0 {
				album.NbDiscs = maxDisc
			}
		}

		for _, song := range albumInfo.Songs.Data {
			selectedFormat, songUrl, err := resolveSongUrl(song.TrackToken, config)
			if err != nil {
				msg := fmt.Sprintf("error getting URL for song \"%s\" by %s from \"%s\": %v\n",
					song.SngTitle, song.ArtName, song.AlbTitle, err)
				log.Print(msg)
				logFile.Write([]byte(msg))
				log.Print("Album download failed: " + albumId + "\n\n")
				logFile.Write([]byte("Album download failed: " + albumId + "\n"))
				continue album_loop
			}

			songPath := getSongPath(song, album, config, selectedFormat)
			songDir := path.Dir(songPath)
			coverFilePath := songDir + "/cover.jpg"

			err = ensureSongDirectoryExists(songPath, album.CoverXl)
			if err != nil {
				log.Fatalf("error preparing directory for song: %s\n", err)
			}
			var bytesWritten int64
			bytesWritten, err = downloadSong(songUrl, songPath, song.SngId, 0, config)
			if err != nil {
				log.Fatalf("error downloading song: %s\n", err)
			}
			log.Printf("Wrote %d bytes: %s", bytesWritten, songPath)

			if strings.ToUpper(selectedFormat) == "FLAC" {
				err = addTags(song, songPath, album)
				if err != nil {
					log.Fatalf("error adding tags to song: %s\n", err)
				}
				err = addCover(songPath, coverFilePath)
				if err != nil {
					log.Fatalf("error adding cover image to song: %s\n", err)
				}
			} else {
				err = addID3Tags(song, songPath, coverFilePath, album)
				if err != nil {
					log.Fatalf("error adding ID3 tags to MP3: %s\n", err)
				}
				log.Printf("Downloaded %s as %s and added ID3 tags", song.SngTitle, selectedFormat)
			}
		}
		log.Print("Album download succeeded: " + albumId + "\n\n")
		logFile.Write([]byte("Album download succeeded: " + albumId + "\n"))
	}
}

func processPlaylists(args []string, config configuration, logFile *os.File) {
playlist_loop:
	for idx, playlistId := range args {
		log.Printf("[%03d/%03d] Downloading playlist %s\n", idx+1, len(args), playlistId)
		playlist, err := getPlaylist(playlistId, config)
		if err != nil {
			log.Fatalf("error getting playlist: %s\n", err)
		}

		tracks := playlist.Tracks
		if tracks.Total == 0 || len(tracks.Data) == 0 {
			tracksParsed, err2 := getPlaylistSongs(playlistId, config)
			if err2 == nil {
				tracks = tracksParsed
			} else {
				log.Printf("could not extract playlist tracks from page: %v\n", err2)
			}
		}

		for _, track := range tracks.Data {
			songInfo, err := getSongInfo(track.Id, config)
			if err != nil {
				log.Fatalf("error getting song info: %s\n", err)
			}
			song := songInfo.Data

			album, err := getAlbum(song.AlbId, config)
			if err != nil {
				log.Fatalf("error getting album: %s\n", err)
			}

			selectedFormat, songUrl, err := resolveSongUrl(song.TrackToken, config)
			if err != nil {
				msg := fmt.Sprintf("error getting URL for song \"%s\" by %s from \"%s\": %v\n",
					song.SngTitle, song.ArtName, song.AlbTitle, err)
				log.Print(msg)
				logFile.Write([]byte(msg))
				log.Print("Playlist download failed: " + playlistId + "\n\n")
				logFile.Write([]byte("Playlist download failed: " + playlistId + "\n"))
				continue playlist_loop
			}

			songPath := getSongPath(song, album, config, selectedFormat)
			songDir := path.Dir(songPath)
			coverFilePath := songDir + "/cover.jpg"

			err = ensureSongDirectoryExists(songPath, album.CoverXl)
			if err != nil {
				log.Fatalf("error preparing directory for song: %s\n", err)
			}
			var bytesWritten int64
			bytesWritten, err = downloadSong(songUrl, songPath, song.SngId, 0, config)
			if err != nil {
				log.Fatalf("error downloading song: %s\n", err)
			}
			log.Printf("Wrote %d bytes: %s", bytesWritten, songPath)

			if strings.ToUpper(selectedFormat) == "FLAC" {
				err = addTags(song, songPath, album)
				if err != nil {
					log.Fatalf("error adding tags to song: %s\n", err)
				}
				err = addCover(songPath, coverFilePath)
				if err != nil {
					log.Fatalf("error adding cover image to song: %s\n", err)
				}
			} else {
				err = addID3Tags(song, songPath, coverFilePath, album)
				if err != nil {
					log.Fatalf("error adding ID3 tags to MP3: %s\n", err)
				}
				log.Printf("Downloaded %s as %s and added ID3 tags", song.SngTitle, selectedFormat)
			}
		}
		log.Print("Playlist download succeeded: " + playlistId + "\n\n")
		logFile.Write([]byte("Playlist download succeeded: " + playlistId + "\n"))
	}
}
