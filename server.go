package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

// API Request/Response types
type DownloadRequest struct {
	ID           string `json:"id"`
	LicenseToken string `json:"license_token"`
}

type DownloadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// StartServer starts the HTTP API server for Chrome extension
func StartServer(port string, config configuration) {
	// Set up CORS middleware
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}

	// Health check endpoint
	http.HandleFunc("/health", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))

	// Download album endpoint
	http.HandleFunc("/download-album", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handleDownloadAlbum(w, r, config)
	}))

	// Download playlist endpoint
	http.HandleFunc("/download-playlist", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handleDownloadPlaylist(w, r, config)
	}))

	log.Printf("API Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleDownloadAlbum(w http.ResponseWriter, r *http.Request, config configuration) {
	w.Header().Set("Content-Type", "application/json")

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.ID) == 0 {
		respondWithError(w, "ID is required", http.StatusBadRequest)
		return
	}

	// Create temporary config with the provided license token, or use the one from config.toml
	tempConfig := config
	if len(req.LicenseToken) > 0 {
		// Use provided token from extension (captured from Deezer)
		tempConfig.LicenseToken = req.LicenseToken
	} else if len(config.LicenseToken) == 0 {
		// No token provided and no token in config
		respondWithError(w, "License token is required. Either provide it or refresh in the extension.", http.StatusBadRequest)
		return
	}
	// else: use config.LicenseToken (already in tempConfig)

	// Get album info
	album, err := getAlbum(req.ID, tempConfig)
	if err != nil {
		respondWithError(w, "Failed to get album info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get album songs with more details
	albumSongs, err := getAlbumSongs(req.ID, tempConfig)
	if err != nil {
		respondWithError(w, "Failed to get album songs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Download all tracks
	total := len(albumSongs.Songs.Data)
	log.Printf("Downloading album: %s (%d tracks)", album.Title, total)
	downloadCount := 0
	for i, song := range albumSongs.Songs.Data {
		log.Printf("[%02d/%02d] %s - %s", i+1, total, album.Title, song.SngTitle)
		if err := downloadSingleTrackFromSong(song, album, tempConfig); err != nil {
			log.Printf("Failed: %v", err)
			continue
		}
		downloadCount++
	}

	message := fmt.Sprintf("Downloaded %d/%d tracks from album: %s", downloadCount, total, album.Title)
	log.Print(message)
	respondWithSuccess(w, message)
}

func handleDownloadPlaylist(w http.ResponseWriter, r *http.Request, config configuration) {
	w.Header().Set("Content-Type", "application/json")

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.ID) == 0 {
		respondWithError(w, "ID is required", http.StatusBadRequest)
		return
	}

	// Create temporary config with the provided license token, or use the one from config.toml
	tempConfig := config
	if len(req.LicenseToken) > 0 {
		// Use provided token from extension (captured from Deezer)
		tempConfig.LicenseToken = req.LicenseToken
	} else if len(config.LicenseToken) == 0 {
		// No token provided and no token in config
		respondWithError(w, "License token is required. Either provide it or refresh in the extension.", http.StatusBadRequest)
		return
	}
	// else: use config.LicenseToken (already in tempConfig)

	// Get playlist info
	playlist, err := getPlaylist(req.ID, tempConfig)
	if err != nil {
		respondWithError(w, "Failed to get playlist info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get playlist songs
	playlistSongs, err := getPlaylistSongs(req.ID, tempConfig)
	if err != nil {
		respondWithError(w, "Failed to get playlist songs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Download all tracks
	total := len(playlistSongs.Data)
	log.Printf("Downloading playlist: %s (%d tracks)", playlist.Title, total)
	downloadCount := 0
	for i, track := range playlistSongs.Data {
		log.Printf("[%02d/%02d] %s", i+1, total, track.Title)
		if err := downloadSingleTrackFromPlaylist(track, tempConfig); err != nil {
			log.Printf("Failed: %v", err)
			continue
		}
		downloadCount++
	}

	message := fmt.Sprintf("Downloaded %d/%d tracks from playlist: %s", downloadCount, total, playlist.Title)
	log.Print(message)
	respondWithSuccess(w, message)
}

func downloadSingleTrackFromSong(song resSongInfoData, album resAlbum, config configuration) error {
	// Try multiple formats
	formats := []string{"FLAC", "MP3_320", "MP3_256", "MP3_128"}
	var selectedFormat string
	var songUrl string

	var lastErr error
	for _, f := range formats {
		songUrlData, err := getSongUrlData(song.TrackToken, f, config)
		if err != nil {
			lastErr = err
			continue
		}
		songUrlTry, err := getSongUrl(songUrlData, f)
		if err != nil {
			lastErr = err
			continue
		}
		selectedFormat = f
		songUrl = songUrlTry
		break
	}

	if selectedFormat == "" {
		return fmt.Errorf("no available formats for track %s (%v)", song.SngTitle, lastErr)
	}

	// Build file path
	songPath := getSongPath(song, album, config, selectedFormat)
	songDir := path.Dir(songPath)

	// Ensure directory exists
	err := ensureSongDirectoryExists(songPath, album.CoverXl)
	if err != nil {
		return err
	}

	// Download song
	bytesWritten, err := downloadSong(songUrl, songPath, song.SngId, 0, config)
	if err != nil {
		return err
	}
	log.Printf("  → %s (Wrote %d bytes: %s)", selectedFormat, bytesWritten, songPath)

	// Add tags
	if strings.ToUpper(selectedFormat) == "FLAC" {
		err = addTags(song, songPath, album)
		if err != nil {
			log.Printf("Warning: failed to add tags: %v", err)
		}
		coverFilePath := songDir + "/cover.jpg"
		if _, statErr := os.Stat(coverFilePath); statErr == nil {
			if err = addCover(songPath, coverFilePath); err != nil {
				log.Printf("Warning: failed to add cover: %v", err)
			}
		}
	} else {
		coverFilePath := songDir + "/cover.jpg"
		err = addID3Tags(song, songPath, coverFilePath, album)
		if err != nil {
			log.Printf("Warning: failed to add ID3 tags: %v", err)
		}
	}

	return nil
}

func downloadSingleTrackFromPlaylist(track resTrack, config configuration) error {
	// Get full song info
	songInfo, err := getSongInfo(track.Id, config)
	if err != nil {
		return fmt.Errorf("failed to get song info: %w", err)
	}

	song := songInfo.Data

	// Get album metadata for tagging
	album, err := getAlbum(song.AlbId, config)
	if err != nil {
		log.Printf("Warning: failed to get album info for %s, tags will be incomplete: %v", song.SngTitle, err)
		album = resAlbum{}
	}

	// Try multiple formats
	formats := []string{"FLAC", "MP3_320", "MP3_256", "MP3_128"}
	var selectedFormat string
	var songUrl string

	var lastErr error
	for _, f := range formats {
		songUrlData, err := getSongUrlData(song.TrackToken, f, config)
		if err != nil {
			lastErr = err
			continue
		}
		songUrlTry, err := getSongUrl(songUrlData, f)
		if err != nil {
			lastErr = err
			continue
		}
		selectedFormat = f
		songUrl = songUrlTry
		break
	}

	if selectedFormat == "" {
		return fmt.Errorf("no available formats for track %s (%v)", song.SngTitle, lastErr)
	}

	// Build file path using getSongPath for consistency with album downloads
	songPath := getSongPath(song, album, config, selectedFormat)
	songDir := path.Dir(songPath)

	// Use album cover URL, fall back to song-level picture
	coverUrl := album.CoverXl
	if coverUrl == "" {
		coverUrl = song.AlbPicture
	}

	// Ensure directory exists
	err = ensureSongDirectoryExists(songPath, coverUrl)
	if err != nil {
		return err
	}

	// Download song
	bytesWritten, err := downloadSong(songUrl, songPath, song.SngId, 0, config)
	if err != nil {
		return err
	}
	log.Printf("  → %s (Wrote %d bytes: %s)", selectedFormat, bytesWritten, songPath)

	// Add tags
	if strings.ToUpper(selectedFormat) == "FLAC" {
		err = addTags(song, songPath, album)
		if err != nil {
			log.Printf("Warning: failed to add tags: %v", err)
		}
		coverFilePath := songDir + "/cover.jpg"
		if _, statErr := os.Stat(coverFilePath); statErr == nil {
			if err = addCover(songPath, coverFilePath); err != nil {
				log.Printf("Warning: failed to add cover: %v", err)
			}
		}
	} else {
		coverFilePath := songDir + "/cover.jpg"
		err = addID3Tags(song, songPath, coverFilePath, album)
		if err != nil {
			log.Printf("Warning: failed to add ID3 tags: %v", err)
		}
	}

	return nil
}

func respondWithSuccess(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(DownloadResponse{
		Success: true,
		Message: message,
	})
}

func respondWithError(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(DownloadResponse{
		Success: false,
		Error:   errorMsg,
	})
}
