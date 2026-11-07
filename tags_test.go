package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-flac/go-flac"
)

// writeFlacHeader writes a minimal FLAC file: the "fLaC" marker, a STREAMINFO
// block with the given sample rate, channels and bit depth, then dummy audio.
func writeFlacHeader(t *testing.T, blockType byte, sampleRate, channels, bits int) string {
	t.Helper()
	d := make([]byte, 34)
	d[10] = byte(sampleRate >> 12)
	d[11] = byte(sampleRate >> 4)
	d[12] = byte(sampleRate<<4) | byte((channels-1)<<1) | byte((bits-1)>>4)
	d[13] = byte((bits - 1) << 4)
	data := append([]byte("fLaC"), 0x80|blockType, 0, 0, 34)
	data = append(data, d...)
	// Some dummy audio frame bytes: go-flac expects data after the metadata.
	data = append(data, 0xff, 0xf8, 0x69, 0x08, 0x00, 0x00, 0x00, 0x00)
	path := filepath.Join(t.TempDir(), "test.flac")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadFlacStreamInfo(t *testing.T) {
	cases := []struct {
		rate, channels, bits int
		str                  string
	}{
		{44100, 2, 16, "16-bit/44.1kHz"},
		{96000, 2, 24, "24-bit/96kHz"},
		{192000, 1, 24, "24-bit/192kHz"},
	}
	for _, c := range cases {
		si, err := readFlacStreamInfo(writeFlacHeader(t, 0, c.rate, c.channels, c.bits))
		if err != nil {
			t.Fatalf("%s: %v", c.str, err)
		}
		if si.SampleRate != c.rate || si.Channels != c.channels || si.BitsPerSample != c.bits {
			t.Errorf("got %+v, want rate=%d channels=%d bits=%d", si, c.rate, c.channels, c.bits)
		}
		if si.String() != c.str {
			t.Errorf("String() = %q, want %q", si.String(), c.str)
		}
	}
}

func TestReadFlacStreamInfoErrors(t *testing.T) {
	dir := t.TempDir()

	notFlac := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(notFlac, make([]byte, 64), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readFlacStreamInfo(notFlac); err == nil {
		t.Error("expected an error for a non-FLAC file")
	}

	short := filepath.Join(dir, "short.flac")
	if err := os.WriteFile(short, []byte("fLaC"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readFlacStreamInfo(short); err == nil {
		t.Error("expected an error for a truncated file")
	}

	if _, err := readFlacStreamInfo(writeFlacHeader(t, 4, 44100, 2, 16)); err == nil {
		t.Error("expected an error when the first block is not STREAMINFO")
	}

	if _, err := readFlacStreamInfo(filepath.Join(dir, "missing.flac")); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestAddTagsWritesVorbisComments(t *testing.T) {
	path := writeFlacHeader(t, 0, 44100, 2, 16)
	song := resSongInfoData{SngTitle: "Song", Version: "(Live)", AlbTitle: "Album", TrackNumber: "2",
		DiskNumber: "1", PhysicalReleaseDate: "2020-03-04", Isrc: "FR1234567890",
		Artists: []resSongInfoArtist{{ArtName: "B"}, {ArtName: "A"}}}
	var album resAlbum
	album.Artist.Name = "A"
	album.NbTracks = 10

	if err := addTags(song, path, album); err != nil {
		t.Fatalf("addTags: %v", err)
	}

	f, err := flac.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cmts, _, found, err := extractFlacComment(f)
	if err != nil || !found {
		t.Fatalf("no Vorbis comment block (found=%v, err=%v)", found, err)
	}
	want := map[string]string{
		"TITLE": "Song (Live)", "ARTIST": "A, B", "ALBUM": "Album", "TRACKNUMBER": "2",
		"TRACKTOTAL": "10", "DATE": "2020", "ISRC": "FR1234567890",
	}
	for key, value := range want {
		got, err := cmts.Get(key)
		if err != nil || len(got) != 1 || got[0] != value {
			t.Errorf("%s = %v (err %v), want %q", key, got, err, value)
		}
	}
	if si, err := readFlacStreamInfo(path); err != nil || si.SampleRate != 44100 {
		t.Errorf("STREAMINFO damaged after tagging: %+v, %v", si, err)
	}
}

func TestAddTagsReportsSaveError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	path := writeFlacHeader(t, 0, 44100, 2, 16)
	// Read-only file: parsing works but saving the tags must fail.
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	if err := addTags(resSongInfoData{}, path, resAlbum{}); err == nil {
		t.Error("expected addTags to report the save error")
	}
}
