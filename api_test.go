package main

import "testing"

func TestPickBestMedia(t *testing.T) {
	media := func(formats ...string) []songMedia {
		out := make([]songMedia, len(formats))
		for i, f := range formats {
			out[i].Format = f
		}
		return out
	}
	cases := []struct {
		formats []string
		want    string
	}{
		{[]string{"MP3_128", "MP3_320", "FLAC"}, "FLAC"},
		{[]string{"MP3_128", "MP3_320"}, "MP3_320"},
		{[]string{"MP3_128"}, "MP3_128"},
		{[]string{"UNKNOWN", "mp3_320"}, "mp3_320"},
	}
	for _, c := range cases {
		if got := pickBestMedia(media(c.formats...)).Format; got != c.want {
			t.Errorf("pickBestMedia(%v) = %q, want %q", c.formats, got, c.want)
		}
	}
}
