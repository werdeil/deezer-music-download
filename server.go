package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

// ProgressEvent is a single NDJSON line streamed to the extension during a
// download so the progress bar can advance track by track.
type ProgressEvent struct {
	Type    string `json:"type"`              // "progress" | "done" | "error"
	Current int    `json:"current,omitempty"` // tracks processed so far
	Total   int    `json:"total,omitempty"`   // total tracks
	Title   string `json:"title,omitempty"`   // current track title
	Success bool   `json:"success,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// writeEvent encodes one ProgressEvent as an NDJSON line and flushes it so the
// client receives it immediately.
func writeEvent(w http.ResponseWriter, flusher http.Flusher, ev ProgressEvent) {
	json.NewEncoder(w).Encode(ev)
	flusher.Flush()
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

	// Ensure album.NbDiscs is set: the public API does not expose nb_discs,
	// so compute it from the scraped song list (matches the CLI album path).
	if album.NbDiscs == 0 {
		album.NbDiscs = computeNbDiscs(albumSongs.Songs.Data)
	}

	// Download all tracks, streaming progress to the client
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondWithError(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	total := len(albumSongs.Songs.Data)
	log.Printf("Downloading album: %s (%d tracks)", album.Title, total)
	downloadCount := 0
	for i, song := range albumSongs.Songs.Data {
		log.Printf("[%02d/%02d] %s - %s", i+1, total, album.Title, song.SngTitle)
		if err := downloadTrack(song, album, tempConfig); err != nil {
			log.Printf("Failed: %v", err)
		} else {
			downloadCount++
		}
		writeEvent(w, flusher, ProgressEvent{Type: "progress", Current: i + 1, Total: total, Title: song.SngTitle})
	}

	message := fmt.Sprintf("Downloaded %d/%d tracks from album: %s", downloadCount, total, album.Title)
	log.Print(message)
	writeEvent(w, flusher, ProgressEvent{Type: "done", Success: true, Message: message})
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

	// Download all tracks, streaming progress to the client
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondWithError(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	total := len(playlistSongs.Data)
	log.Printf("Downloading playlist: %s (%d tracks)", playlist.Title, total)
	downloadCount := 0
	for i, track := range playlistSongs.Data {
		log.Printf("[%02d/%02d] %s", i+1, total, track.Title)
		if err := downloadPlaylistTrack(track, tempConfig); err != nil {
			log.Printf("Failed: %v", err)
		} else {
			downloadCount++
		}
		writeEvent(w, flusher, ProgressEvent{Type: "progress", Current: i + 1, Total: total, Title: track.Title})
	}

	message := fmt.Sprintf("Downloaded %d/%d tracks from playlist: %s", downloadCount, total, playlist.Title)
	log.Print(message)
	writeEvent(w, flusher, ProgressEvent{Type: "done", Success: true, Message: message})
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
