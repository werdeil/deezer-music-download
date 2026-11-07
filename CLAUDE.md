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

# Tests (offline, no Deezer account needed)
go test -race ./...

# Lint (config in .golangci.yml)
golangci-lint run ./...

# Chrome extension checks
node .github/scripts/check-extension.mjs chrome-extension
(cd chrome-extension && npx eslint@10 --config ../.github/scripts/eslint.config.mjs .)
```

Tests cover the pure helpers (paths, FLAC STREAMINFO and Vorbis tagging, Blowfish key/decryption with dummy keys, format selection) and the server's request validation. They never call Deezer.

Lint conventions: close read-only resources (response bodies, files opened for reading) with `closeQuietly` (util.go); for files being written, propagate the `Close` error. Write to the run log with `appendLog`.

## CI/CD (`.github/`)

- `workflows/ci.yml` — gofmt, `go vet`, `go mod tidy` diff, `go test -race`, golangci-lint (the whole codebase must be clean), cross-builds (linux/darwin/windows × amd64/arm64), and the extension job (manifest/locale checks, ESLint, zip artifact).
- `workflows/security.yml` — govulncheck (also weekly) and gitleaks secret scan.
- `dependabot.yml` — weekly updates for Go modules and GitHub Actions.
- `workflows/pull-request.yml` + `labeler.yml` — labels each PR by area from the changed files (`api`, `download`, `tags`, `server`, `cli`, `extension`, `i18n`, `tests`, `dependencies`, `docs`, `ci`) and by type from a Conventional Commits title (`feat`, `fix`, `refactor`, `docs`, `chore`). It also requires PR titles to follow Conventional Commits: `type(scope): summary`, type among `feat`, `fix`, `refactor`, `docs`, `chore`, lowercase after the colon, no trailing period. Runs on `pull_request_target`, so it uses the workflow and config from `main` and never runs the PR's code.
- `release.yml` — groups the generated release notes by label: Features (`feat`), Fixes (`fix`), Dependencies, Other changes.

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
8. **`orchestrator.go`** — `processAlbums` and `processPlaylists`: iterate tracks, resolve a stream URL via `resolveSongUrl` (api.go), which requests `FLAC`, `MP3_320` and `MP3_128` in a single `get_url` call and keeps the best format served, then download + tag. For FLAC, the real resolution (16/24-bit, sample rate) is read from the STREAMINFO block (`readFlacStreamInfo` in tags.go) and logged.
9. **`server.go`** — HTTP server for the Chrome extension. Endpoints: `GET /health`, `POST /download-album`, `POST /download-playlist`. Accepts `{"id": "...", "license_token": "..."}`.

### Chrome extension (`chrome-extension/`)

`manifest.json` → `background.js` intercepts `get_url` network requests to capture the `license_token` automatically → `popup.js`/`popup.html` let the user trigger downloads and configure the server URL. `content.js` extracts the album/playlist ID from the current URL.
