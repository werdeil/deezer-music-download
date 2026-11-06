package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFlacHeader writes a minimal FLAC file: the "fLaC" marker followed by a
// STREAMINFO block with the given sample rate, channels and bit depth.
func writeFlacHeader(t *testing.T, blockType byte, sampleRate, channels, bits int) string {
	t.Helper()
	d := make([]byte, 34)
	d[10] = byte(sampleRate >> 12)
	d[11] = byte(sampleRate >> 4)
	d[12] = byte(sampleRate<<4) | byte((channels-1)<<1) | byte((bits-1)>>4)
	d[13] = byte((bits - 1) << 4)
	data := append([]byte("fLaC"), 0x80|blockType, 0, 0, 34)
	data = append(data, d...)
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
