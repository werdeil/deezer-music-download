package main

import (
	"os"
	"strconv"

	id3v2 "github.com/bogem/id3v2"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
)

func extractFlacComment(f *flac.File) (*flacvorbis.MetaDataBlockVorbisComment, int, bool, error) {
	var err error
	var cmt *flacvorbis.MetaDataBlockVorbisComment
	var cmtIdx int
	found := false
	for idx, meta := range f.Meta {
		if meta.Type == flac.VorbisComment {
			cmt, err = flacvorbis.ParseFromMetaDataBlock(*meta)
			cmtIdx = idx
			found = true
			if err != nil {
				return nil, 0, false, err
			}
		}
	}
	return cmt, cmtIdx, found, nil
}

func addCover(songPath string, coverPath string) error {
	coverData, err := os.ReadFile(coverPath)
	if err != nil {
		return err
	}

	f, err := flac.ParseFile(songPath)
	if err != nil {
		return err
	}

	picture, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover,
		"Front cover", coverData, "image/jpeg")
	if err != nil {
		return err
	}

	picturemeta := picture.Marshal()
	f.Meta = append(f.Meta, &picturemeta)
	f.Save(songPath)
	return nil
}

// addID3Tags writes ID3v2 tags and embedded cover to an MP3 file.
func addID3Tags(song resSongInfoData, mp3Path string, coverPath string, album resAlbum) error {
	var tag *id3v2.Tag
	var err error

	tag, err = id3v2.Open(mp3Path, id3v2.Options{Parse: true})
	if err != nil {
		tag = id3v2.NewEmptyTag()
	}
	defer tag.Close()

	title := getTitle(song)
	artist := getArtist(song)
	composer := getComposer(song)

	genre := getAlbumGenres(album)

	tag.SetTitle(title)
	tag.SetAlbum(song.AlbTitle)
	tag.SetArtist(artist)
	tag.AddTextFrame(tag.CommonID("Album artist"), tag.DefaultEncoding(), album.Artist.Name)
	if composer != "" {
		tag.AddTextFrame(tag.CommonID("Composer"), tag.DefaultEncoding(), composer)
	}
	if genre != "" {
		tag.AddTextFrame(tag.CommonID("Content type"), tag.DefaultEncoding(), genre)
	}
	if song.TrackNumber != "" {
		trckValue := song.TrackNumber
		if album.NbTracks > 0 {
			trckValue = song.TrackNumber + "/" + strconv.Itoa(album.NbTracks)
		}
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), tag.DefaultEncoding(), trckValue)
	} else if album.NbTracks > 0 {
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), tag.DefaultEncoding(), strconv.Itoa(album.NbTracks))
	}
	if song.DiskNumber != "" {
		discValue := song.DiskNumber
		if album.NbDiscs > 0 {
			discValue = song.DiskNumber + "/" + strconv.Itoa(album.NbDiscs)
		}
		tag.AddTextFrame(tag.CommonID("Part of a set"), tag.DefaultEncoding(), discValue)
	} else if album.NbDiscs > 0 {
		tag.AddTextFrame(tag.CommonID("Part of a set"), tag.DefaultEncoding(), strconv.Itoa(album.NbDiscs))
	}
	if song.Copyright != "" {
		tag.AddTextFrame(tag.CommonID("Copyright message"), tag.DefaultEncoding(), song.Copyright)
	}
	// Prefer album release year, fallback to song physical release date
	year := extractYear(album.ReleaseDate)
	if year == "" {
		year = extractYear(song.PhysicalReleaseDate)
	}
	if year != "" {
		tag.SetYear(year)
	}
	if song.Isrc != "" {
		tag.AddTextFrame("TSRC", tag.DefaultEncoding(), song.Isrc)
	}

	if _, err := os.Stat(coverPath); err == nil {
		picBytes, err := os.ReadFile(coverPath)
		if err == nil {
			pf := id3v2.PictureFrame{
				Encoding:    tag.DefaultEncoding(),
				MimeType:    "image/jpeg",
				PictureType: id3v2.PTFrontCover,
				Description: "Cover",
				Picture:     picBytes,
			}
			tag.AddAttachedPicture(pf)
		}
	}

	if err := tag.Save(); err != nil {
		return err
	}
	return nil
}

func addTags(song resSongInfoData, path string, album resAlbum) error {
	var err error

	f, err := flac.ParseFile(path)
	if err != nil {
		return err
	}

	cmts, idx, found, err := extractFlacComment(f)
	if err != nil {
		return err
	}
	if cmts == nil {
		cmts = flacvorbis.New()
	}

	title := getTitle(song)
	artist := getArtist(song)
	composer := getComposer(song)

	cmts.Add("TITLE", title)
	cmts.Add("ALBUM", song.AlbTitle)
	cmts.Add("ARTIST", artist)
	cmts.Add("ALBUMARTIST", album.Artist.Name)
	cmts.Add("COMPOSER", composer)
	cmts.Add("TRACKNUMBER", song.TrackNumber)
	if album.NbTracks > 0 {
		cmts.Add("TRACKTOTAL", strconv.Itoa(album.NbTracks))
	}
	cmts.Add("DISCNUMBER", song.DiskNumber)
	if album.NbDiscs > 0 {
		cmts.Add("DISCTOTAL", strconv.Itoa(album.NbDiscs))
	}
	cmts.Add("COPYRIGHT", song.Copyright)
	// Add genre (from album) to Vorbis comments
	genre := getAlbumGenres(album)
	if genre != "" {
		cmts.Add("GENRE", genre)
	}
	// Prefer album release year, fallback to song physical release date
	year := extractYear(album.ReleaseDate)
	if year == "" {
		year = extractYear(song.PhysicalReleaseDate)
	}
	if year != "" {
		cmts.Add("DATE", year)
	} else {
		// keep original value if no year could be extracted
		cmts.Add("DATE", song.PhysicalReleaseDate)
	}
	cmts.Add("ISRC", song.Isrc)
	cmtsmeta := cmts.Marshal()
	if found {
		f.Meta[idx] = &cmtsmeta
	} else {
		f.Meta = append(f.Meta, &cmtsmeta)
	}

	f.Save(path)

	return nil
}
