package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

// downloadTrack resolves a stream URL for the song, downloads it, and writes
// tags + embedded cover. Tagging and cover failures are logged as warnings (the
// audio file is still usable); an error is returned only when the track itself
// cannot be fetched. Shared by the CLI and server, album and playlist paths so
// their behaviour cannot drift.
func downloadTrack(song resSongInfoData, album resAlbum, config configuration) error {
	// streamId identifies the recording actually being streamed. It normally
	// matches song, but diverges when we fall back, and the Blowfish key is
	// derived from it, so it must follow the stream rather than the metadata.
	streamId := song.SngId
	selectedFormat, songUrl, err := resolveSongUrl(song.TrackToken, config)
	if err != nil && song.Fallback != nil && song.Fallback.TrackToken != "" {
		fbFormat, fbUrl, fbErr := resolveSongUrl(song.Fallback.TrackToken, config)
		if fbErr == nil {
			log.Printf("\"%s\" has no streaming rights, using fallback recording %s",
				song.SngTitle, song.Fallback.SngId)
			selectedFormat, songUrl, streamId, err = fbFormat, fbUrl, song.Fallback.SngId, nil
		}
	}
	if err != nil {
		return fmt.Errorf("no playable format for \"%s\" by %s from \"%s\": %w",
			song.SngTitle, song.ArtName, song.AlbTitle, err)
	}

	songPath := getSongPath(song, album, config, selectedFormat)
	coverFilePath := path.Dir(songPath) + "/cover.jpg"

	if err = ensureSongDirectoryExists(songPath, coverURL(album, song)); err != nil {
		return fmt.Errorf("preparing directory for \"%s\": %w", song.SngTitle, err)
	}

	bytesWritten, err := downloadSong(songUrl, songPath, streamId, 0, config)
	if err != nil {
		return fmt.Errorf("downloading \"%s\": %w", song.SngTitle, err)
	}
	log.Printf("Wrote %d bytes: %s", bytesWritten, songPath)

	if strings.ToUpper(selectedFormat) == "FLAC" {
		if err = addTags(song, songPath, album); err != nil {
			log.Printf("Warning: failed to add tags to \"%s\": %v", song.SngTitle, err)
		}
		if _, statErr := os.Stat(coverFilePath); statErr == nil {
			if err = addCover(songPath, coverFilePath); err != nil {
				log.Printf("Warning: failed to add cover to \"%s\": %v", song.SngTitle, err)
			}
		}
	} else if err = addID3Tags(song, songPath, coverFilePath, album); err != nil {
		log.Printf("Warning: failed to add ID3 tags to \"%s\": %v", song.SngTitle, err)
	}

	log.Printf("Downloaded \"%s\" as %s", song.SngTitle, selectedFormat)
	return nil
}

// downloadPlaylistTrack resolves full song info and album metadata for a
// playlist entry, then downloads it. Album metadata is best-effort: a single
// unavailable album yields incomplete tags rather than aborting the track.
func downloadPlaylistTrack(track resTrack, config configuration) error {
	songInfo, err := getSongInfo(track.Id, config)
	if err != nil {
		return fmt.Errorf("getting song info for track %d: %w", track.Id, err)
	}
	song := songInfo.Data

	album, err := getAlbum(song.AlbId, config)
	if err != nil {
		log.Printf("Warning: failed to get album info for \"%s\", tags will be incomplete: %v", song.SngTitle, err)
		album = resAlbum{}
	}

	return downloadTrack(song, album, config)
}

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
			album.NbDiscs = computeNbDiscs(albumInfo.Songs.Data)
		}

		failed := 0
		for _, song := range albumInfo.Songs.Data {
			if err := downloadTrack(song, album, config); err != nil {
				failed++
				msg := fmt.Sprintf("skipping track: %v\n", err)
				log.Print(msg)
				logFile.Write([]byte(msg))
			}
		}
		total := len(albumInfo.Songs.Data)
		if failed == total {
			log.Print("Album download failed: " + albumId + "\n\n")
			logFile.Write([]byte("Album download failed: " + albumId + "\n"))
			continue album_loop
		}
		var result string
		if failed > 0 {
			result = fmt.Sprintf("Album download completed with %d/%d tracks unavailable: %s\n", failed, total, albumId)
		} else {
			result = "Album download succeeded: " + albumId + "\n"
		}
		log.Print(result + "\n")
		logFile.Write([]byte(result))
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

		failed := 0
		for _, track := range tracks.Data {
			if err := downloadPlaylistTrack(track, config); err != nil {
				failed++
				msg := fmt.Sprintf("skipping track: %v\n", err)
				log.Print(msg)
				logFile.Write([]byte(msg))
			}
		}
		total := len(tracks.Data)
		if total > 0 && failed == total {
			log.Print("Playlist download failed: " + playlistId + "\n\n")
			logFile.Write([]byte("Playlist download failed: " + playlistId + "\n"))
			continue playlist_loop
		}
		var result string
		if failed > 0 {
			result = fmt.Sprintf("Playlist download completed with %d/%d tracks unavailable: %s\n", failed, total, playlistId)
		} else {
			result = "Playlist download succeeded: " + playlistId + "\n"
		}
		log.Print(result + "\n")
		logFile.Write([]byte(result))
	}
}
