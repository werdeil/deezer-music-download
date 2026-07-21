# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run directly (no build step required)
go run . album <album_id>
go run . playlist <playlist_id>

# Build binary
go build -o deezer-downloader .

# Run server mode for Chrome extension
./deezer-downloader --server        # default port 8080
./deezer-downloader --server 9090   # custom port

# Dependencies
go mod tidy
```

There are no tests in this project.

## Configuration

The tool reads `~/.config/deezer-music-download/config.toml` (or `$XDG_CONFIG_HOME/deezer-music-download/config.toml`). See `example_config.toml` for the format. Required fields: `arl`, `license_token`, `dest_dir`, `pre_key`, `iv`.

In server mode, `license_token` can be left empty — the Chrome extension captures it from Deezer network requests and sends it per-request.

## Architecture

All Go files are in `package main` with no subdirectories. The data flow is:

1. **`main.go`** — entry point, HTTP request helper (`makeReq`), rate-limiting (500ms between requests), and Deezer session headers/cookie injection.
2. **`config.go`** — reads and validates `config.toml` via TOML.
3. **`models.go`** — all JSON/TOML struct definitions. Two API surfaces coexist: the public Deezer REST API (`api.deezer.com`) and the private internal API scraped from `window.__DZR_APP_STATE__` embedded in page HTML.
4. **`api.go`** — all Deezer API calls. Album/song data comes from scraping `window.__DZR_APP_STATE__` out of page HTML; track URLs come from `media.deezer.com/v1/get_url` using a `TrackToken`.
5. **`download.go`** — stream download with Blowfish-CBC decryption. Every third 2048-byte block is encrypted; the key is derived per-track from `MD5(songId) XOR preKey`.
6. **`tags.go`** — writes metadata: Vorbis comments + embedded cover for FLAC (`go-flac`), ID3v2 tags + embedded cover for MP3 (`bogem/id3v2`).
7. **`util.go`** — helpers: `getSongPath` (builds `destDir/Artist/Artist - Album/NN - Title.ext`), `SanitizePath`, `getTitle`/`getArtist`/`getComposer`/`getAlbumGenres`.
8. **`orchestrator.go`** — `processAlbums` and `processPlaylists`: iterate tracks, resolve a stream URL via `resolveSongUrl` (api.go), which tries formats `FLAC → MP3_320 → MP3_128`, then download + tag.
9. **`server.go`** — HTTP server for the Chrome extension. Endpoints: `GET /health`, `POST /download-album`, `POST /download-playlist`. Accepts `{"id": "...", "license_token": "..."}`.

### Chrome extension (`chrome-extension/`)

`manifest.json` → `background.js` intercepts `get_url` network requests to capture the `license_token` automatically → `popup.js`/`popup.html` let the user trigger downloads and configure the server URL. `content.js` extracts the album/playlist ID from the current URL.
