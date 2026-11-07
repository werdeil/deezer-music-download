package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizePath(t *testing.T) {
	cases := map[string]string{
		"Simple":          "Simple",
		"AC/DC":           "AC-DC",
		`a<b>c:d"e|f?g*h`: "a-b-c-d-e-f-g-h",
		`back\slash`:      "back-slash",
		"trailing/":       "trailing",
		"Hello: World?":   "Hello- World-",
	}
	for in, want := range cases {
		if got := SanitizePath(in); got != want {
			t.Errorf("SanitizePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractYear(t *testing.T) {
	cases := map[string]string{
		"2021-05-14": "2021",
		"1999":       "1999",
		" 2005-01 ":  "2005",
		"":           "",
		"abc":        "",
		"20a1-01-01": "",
		"0000-00-00": "0000",
	}
	for in, want := range cases {
		if got := extractYear(in); got != want {
			t.Errorf("extractYear(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGetTitle(t *testing.T) {
	if got := getTitle(resSongInfoData{SngTitle: "Song"}); got != "Song" {
		t.Errorf("got %q", got)
	}
	if got := getTitle(resSongInfoData{SngTitle: "Song", Version: "(Live)"}); got != "Song (Live)" {
		t.Errorf("got %q", got)
	}
}

func TestGetArtistSortsNames(t *testing.T) {
	song := resSongInfoData{Artists: []resSongInfoArtist{{ArtName: "Zed"}, {ArtName: "Alpha"}, {ArtName: "Mid"}}}
	if got := getArtist(song); got != "Alpha, Mid, Zed" {
		t.Errorf("getArtist = %q", got)
	}
	if got := getArtist(resSongInfoData{}); got != "" {
		t.Errorf("getArtist(empty) = %q", got)
	}
}

func TestComputeNbDiscs(t *testing.T) {
	songs := []resSongInfoData{{DiskNumber: "1"}, {DiskNumber: "3"}, {DiskNumber: ""}, {DiskNumber: "x"}, {DiskNumber: "2"}}
	if got := computeNbDiscs(songs); got != 3 {
		t.Errorf("computeNbDiscs = %d, want 3", got)
	}
	if got := computeNbDiscs(nil); got != 0 {
		t.Errorf("computeNbDiscs(nil) = %d, want 0", got)
	}
}

func TestCoverURL(t *testing.T) {
	if got := coverURL(resAlbum{CoverXl: "https://x/cover.jpg"}, resSongInfoData{AlbPicture: "abc"}); got != "https://x/cover.jpg" {
		t.Errorf("public API cover should win, got %q", got)
	}
	got := coverURL(resAlbum{}, resSongInfoData{AlbPicture: "abc123"})
	if !strings.Contains(got, "/images/cover/abc123/1000x1000") {
		t.Errorf("fallback cover URL = %q", got)
	}
	if got := coverURL(resAlbum{}, resSongInfoData{}); got != "" {
		t.Errorf("no cover expected, got %q", got)
	}
}

func TestGetAlbumGenres(t *testing.T) {
	var album resAlbum
	if err := json.Unmarshal([]byte(`{"label":"Some Label"}`), &album); err != nil {
		t.Fatal(err)
	}
	if got := getAlbumGenres(album); got != "Some Label" {
		t.Errorf("label fallback = %q", got)
	}
	if err := json.Unmarshal([]byte(`{"genres":{"data":[{"name":"Rock"},{"name":"  "},{"name":"Pop"}]}}`), &album); err != nil {
		t.Fatal(err)
	}
	if got := getAlbumGenres(album); got != "Rock, Pop" {
		t.Errorf("genres = %q", got)
	}
	if got := getAlbumGenres(resAlbum{}); got != "" {
		t.Errorf("empty album = %q", got)
	}
}

func TestGetComposer(t *testing.T) {
	var song resSongInfoData
	if got := getComposer(song); got != "" {
		t.Errorf("no contributors = %q", got)
	}
}

func TestGetSongPath(t *testing.T) {
	cfg := configuration{DestDir: "/music"}
	var album resAlbum
	album.Artist.Name = "AC/DC"
	song := resSongInfoData{TrackNumber: "3", AlbTitle: "Back: In Black", SngTitle: "Hells Bells?"}

	want := "/music/AC-DC/AC-DC - Back- In Black/03 - Hells Bells-.flac"
	if got := getSongPath(song, album, cfg, "FLAC"); got != want {
		t.Errorf("FLAC path = %q, want %q", got, want)
	}
	if got := getSongPath(song, album, cfg, "MP3_320"); !strings.HasSuffix(got, ".mp3") {
		t.Errorf("MP3_320 path should end in .mp3: %q", got)
	}
	if got := getSongPath(song, album, cfg, "mp3_128"); !strings.HasSuffix(got, ".mp3") {
		t.Errorf("mp3_128 path should end in .mp3: %q", got)
	}
}

func TestGetSongPathPanicsOnInvalidTrackNumber(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic for non-numeric track number")
		}
	}()
	getSongPath(resSongInfoData{TrackNumber: "abc"}, resAlbum{}, configuration{}, "FLAC")
}
