package main

import "encoding/json"

// Data models extracted from the original main.go

type configuration struct {
	Arl          string `toml:"arl"`
	LicenseToken string `toml:"license_token"`
	DestDir      string `toml:"dest_dir"`
	Iv           string `toml:"iv"`
	PreKey       string `toml:"pre_key"`
}

type resTrackAlbum struct {
	Id          int64  `json:"id"`
	Title       string `json:"title"`
	Cover       string `json:"cover"`
	CoverSmall  string `json:"cover_small"`
	CoverMedium string `json:"cover_medium"`
	CoverBig    string `json:"cover_big"`
	CoverXl     string `json:"cover_xl"`
	Md5Image    string `json:"md5_image"`
	Tracklist   string `json:"tracklist"`
	Type        string `json:"type"`
}

type resTrackArtist struct {
	Id            int64  `json:"id"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	PictureSmall  string `json:"picture_small"`
	PictureMedium string `json:"picture_medium"`
	PictureBig    string `json:"picture_big"`
	PictureXl     string `json:"picture_xl"`
	Tracklist     string `json:"tracklist"`
	Type          string `json:"type"`
}

type resTrack struct {
	Id                    int64          `json:"id"`
	Readable              bool           `json:"readable"`
	Title                 string         `json:"title"`
	Link                  string         `json:"link"`
	Duration              int            `json:"duration"`
	Rank                  int            `json:"rank"`
	ExplicitLyrics        bool           `json:"explicit_lyrics"`
	ExplicitContentLyrics int            `json:"explicit_content_lyrics"`
	ExplicitContentCover  int            `json:"explicit_content_cover"`
	Md5Image              string         `json:"md5_image"`
	TimeAdd               int64          `json:"time_add"`
	Album                 resTrackAlbum  `json:"album"`
	Artist                resTrackArtist `json:"artist"`
	Type                  string         `json:"type"`
}

type resTracks struct {
	Data  []resTrack `json:"data"`
	Total int        `json:"total"`
}

type resSongInfoArtist struct {
	ArtId             string      `json:"ART_ID"`
	RoleId            string      `json:"ROLE_ID"`
	ArtistsSongsOrder string      `json:"ARTISTS_SONGS_ORDER"`
	ArtName           string      `json:"ART_NAME"`
	ArtistIsDummy     bool        `json:"ARTIST_IS_DUMMY"`
	ArtPicture        string      `json:"ART_PICTURE"`
	Rank              string      `json:"RANK"`
	Locales           interface{} `json:"LOCALES"`
	Type              string      `json:"__TYPE__"`
}

type resSongInfoMedia struct {
	Type string `json:"TYPE"`
	Href string `json:"HREF"`
}

type resSongInfoRights struct {
	StreamAdsAvailable bool   `json:"STREAM_ADS_AVAILABLE"`
	StreamAds          string `json:"STREAM_ADS"`
	StreamSubAvailable bool   `json:"STREAM_SUB_AVAILABLE"`
	StreamSub          string `json:"STREAM_SUB"`
}

type CustomContributors struct {
	Data []resSongInfoContributors
}

func (c *CustomContributors) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &c.Data); err == nil {
		return nil
	}

	var single resSongInfoContributors
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}

	c.Data = []resSongInfoContributors{single}
	return nil
}

type resSongInfoContributors struct {
	MainArtist     []string `json:"main_artist"`
	Composer       []string `json:"composer"`
	Featuring      []string `json:"featuring"`
	Narrator       []string `json:"narrator"`
	MusicPublisher []string `json:"music_publisher"`
}

type resSongInfoExplicitTrackContent struct {
	ExplicitLyricsStatus int `json:"EXPLICIT_LYRICS_STATUS"`
	ExplicitCoverStatus  int `json:"EXPLICIT_COVER_STATUS"`
}

type resSongInfoAvailableCountries struct {
	StreamAds     []string      `json:"STREAM_ADS"`
	StreamSubOnly []interface{} `json:"STREAM_SUB_ONLY"`
}

type resSongInfoData struct {
	SngId                string                          `json:"SNG_ID"`
	ProductTrackId       string                          `json:"PRODUCT_TRACK_ID"`
	UploadId             int                             `json:"UPLOAD_ID"`
	SngTitle             string                          `json:"SNG_TITLE"`
	ArtId                string                          `json:"ART_ID"`
	ProviderId           string                          `json:"PROVIDER_ID"`
	ArtName              string                          `json:"ART_NAME"`
	ArtistIsDummy        bool                            `json:"ARTIST_IS_DUMMY"`
	Artists              []resSongInfoArtist             `json:"ARTISTS"`
	AlbId                string                          `json:"ALB_ID"`
	AlbTitle             string                          `json:"ALB_TITLE"`
	Type                 int                             `json:"TYPE"`
	Md5Origin            string                          `json:"MD5_ORIGIN"`
	Video                bool                            `json:"VIDEO"`
	Duration             string                          `json:"DURATION"`
	AlbPicture           string                          `json:"ALB_PICTURE"`
	ArtPicture           string                          `json:"ART_PICTURE"`
	RankSng              string                          `json:"RANK_SNG"`
	FilesizeAac64        string                          `json:"FILESIZE_AAC_64"`
	FilesizeMp364        string                          `json:"FILESIZE_MP3_64"`
	FilesizeMp3128       string                          `json:"FILESIZE_MP3_128"`
	FilesizeMp3320       string                          `json:"FILESIZE_MP3_320"`
	FilesizeFlac         string                          `json:"FILESIZE_FLAC"`
	Filesize             string                          `json:"FILESIZE"`
	Gain                 string                          `json:"GAIN"`
	MediaVersion         string                          `json:"MEDIA_VERSION"`
	DiskNumber           string                          `json:"DISK_NUMBER"`
	TrackNumber          string                          `json:"TRACK_NUMBER"`
	TrackToken           string                          `json:"TRACK_TOKEN"`
	TrackTokenExpire     int                             `json:"TRACK_TOKEN_EXPIRE"`
	Version              string                          `json:"VERSION"`
	Media                []resSongInfoMedia              `json:"MEDIA"`
	ExplicitLyrics       string                          `json:"EXPLICIT_LYRICS"`
	Rights               resSongInfoRights               `json:"RIGHTS"`
	Isrc                 string                          `json:"ISRC"`
	HierarchicalTitle    string                          `json:"HIERARCHICAL_TITLE"`
	SngContributors      CustomContributors              `json:"SNG_CONTRIBUTORS"` // ✅ Type personnalisé
	LyricsId             int                             `json:"LYRICS_ID"`
	ExplicitTrackContent resSongInfoExplicitTrackContent `json:"EXPLICIT_TRACK_CONTENT"`
	Copyright            string                          `json:"COPYRIGHT"`
	PhysicalReleaseDate  string                          `json:"PHYSICAL_RELEASE_DATE"`
	SMod                 int                             `json:"S_MOD"`
	SPremium             int                             `json:"S_PREMIUM"`
	DateStartPremium     string                          `json:"DATE_START_PREMIUM"`
	DateStart            string                          `json:"DATE_START"`
	Status               int                             `json:"STATUS"`
	UserId               int                             `json:"USER_ID"`
	URLRewriting         string                          `json:"URL_REWRITING"`
	SngStatus            string                          `json:"SNG_STATUS"`
	AvailableCountries   resSongInfoAvailableCountries   `json:"AVAILABLE_COUNTRIES"`
	UpdateDate           string                          `json:"UPDATE_DATE"`
	Type0                string                          `json:"__TYPE__"`
	DigitalReleaseDate   string                          `json:"DIGITAL_RELEASE_DATE"`
}

type resSongInfoIsrcData struct {
	ArtName            string            `json:"ART_NAME"`
	ArtId              string            `json:"ART_ID"`
	AlbPicture         string            `json:"ALB_PICTURE"`
	AlbId              string            `json:"ALB_ID"`
	AlbTitle           string            `json:"ALB_TITLE"`
	Duration           string            `json:"DURATION"`
	DigitalReleaseDate string            `json:"DIGITAL_RELEASE_DATE"`
	Rights             resSongInfoRights `json:"RIGHTS"`
	LyricsId           int               `json:"LYRICS_ID"`
	Type               string            `json:"__TYPE__"`
}

type resSongInfoIsrc struct {
	Data  []resSongInfoIsrcData `json:"data"`
	Count int                   `json:"count"`
	Total int                   `json:"total"`
}

type resSongInfoRelatedAlbumsData struct {
	ArtName            string            `json:"ART_NAME"`
	ArtId              string            `json:"ART_ID"`
	AlbPicture         string            `json:"ALB_PICTURE"`
	AlbId              string            `json:"ALB_ID"`
	AlbTitle           string            `json:"ALB_TITLE"`
	Duration           string            `json:"DURATION"`
	DigitalReleaseDate string            `json:"DIGITAL_RELEASE_DATE"`
	Rights             resSongInfoRights `json:"RIGHTS"`
	LyricsId           int               `json:"LYRICS_ID"`
	Type               string            `json:"__TYPE__"`
}

type resSongInfoRelatedAlbums struct {
	Data  []resSongInfoRelatedAlbumsData `json:"data"`
	Count int                            `json:"count"`
	Total int                            `json:"total"`
}

type resSongInfo struct {
	Data          resSongInfoData          `json:"DATA"`
	Isrc          resSongInfoIsrc          `json:"ISRC"`
	RelatedAlbums resSongInfoRelatedAlbums `json:"RELATED_ALBUMS"`
}

type resSongUrl struct {
	Data []struct {
		Errors []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
		Media []struct {
			Cipher struct {
				Type string `json:"type"`
			} `json:"cipher"`
			Exp       int    `json:"exp"`
			Format    string `json:"format"`
			MediaType string `json:"media_type"`
			Nbf       int    `json:"nbf"`
			Sources   []struct {
				Provider string `json:"provider"`
				Url      string `json:"url"`
			} `json:"sources"`
		} `json:"media"`
	} `json:"data"`
}

type resAlbumInfo struct {
	Songs struct {
		Data          []resSongInfoData `json:"data"`
		Count         int               `json:"count"`
		Total         int               `json:"total"`
		FilteredCount int               `json:"filtered_count"`
	} `json:"SONGS"`
}

type resAlbumGenres struct {
	Data []struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
		Type    string `json:"type"`
	} `json:"data"`
}

type resAlbumContributor struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Link          string `json:"link"`
	Share         string `json:"share"`
	Picture       string `json:"picture"`
	PictureSmall  string `json:"picture_small"`
	PictureMedium string `json:"picture_medium"`
	PictureBig    string `json:"picture_big"`
	PictureXl     string `json:"picture_xl"`
	Radio         bool   `json:"radio"`
	Tracklist     string `json:"tracklist"`
	Type          string `json:"type"`
	Role          string `json:"role"`
}

type resAlbumArtist struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	PictureSmall  string `json:"picture_small"`
	PictureMedium string `json:"picture_medium"`
	PictureBig    string `json:"picture_big"`
	PictureXl     string `json:"picture_xl"`
	Tracklist     string `json:"tracklist"`
	Type          string `json:"type"`
}

type resAlbumTracks struct {
	Data []struct {
		ID                    int    `json:"id"`
		Readable              bool   `json:"readable"`
		Title                 string `json:"title"`
		TitleShort            string `json:"title_short"`
		TitleVersion          string `json:"title_version"`
		Link                  string `json:"link"`
		Duration              int    `json:"duration"`
		Rank                  int    `json:"rank"`
		ExplicitLyrics        bool   `json:"explicit_lyrics"`
		ExplicitContentLyrics int    `json:"explicit_content_lyrics"`
		ExplicitContentCover  int    `json:"explicit_content_cover"`
		Preview               string `json:"preview"`
		Md5Image              string `json:"md5_image"`
		Artist                struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			Tracklist string `json:"tracklist"`
			Type      string `json:"type"`
		} `json:"artist"`
		Album struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			Cover       string `json:"cover"`
			CoverSmall  string `json:"cover_small"`
			CoverMedium string `json:"cover_medium"`
			CoverBig    string `json:"cover_big"`
			CoverXl     string `json:"cover_xl"`
			Md5Image    string `json:"md5_image"`
			Tracklist   string `json:"tracklist"`
			Type        string `json:"type"`
		} `json:"album"`
		Type string `json:"type"`
	} `json:"data"`
}

type resAlbum struct {
	ID                    int                   `json:"id"`
	Title                 string                `json:"title"`
	Upc                   string                `json:"upc"`
	Link                  string                `json:"link"`
	Share                 string                `json:"share"`
	Cover                 string                `json:"cover"`
	CoverSmall            string                `json:"cover_small"`
	CoverMedium           string                `json:"cover_medium"`
	CoverBig              string                `json:"cover_big"`
	CoverXl               string                `json:"cover_xl"`
	Md5Image              string                `json:"md5_image"`
	GenreID               int                   `json:"genre_id"`
	Genres                resAlbumGenres        `json:"genres"`
	Label                 string                `json:"label"`
	NbTracks              int                   `json:"nb_tracks"`
	NbDiscs               int                   `json:"nb_discs"`
	Duration              int                   `json:"duration"`
	Fans                  int                   `json:"fans"`
	ReleaseDate           string                `json:"release_date"`
	RecordType            string                `json:"record_type"`
	Available             bool                  `json:"available"`
	Tracklist             string                `json:"tracklist"`
	ExplicitLyrics        bool                  `json:"explicit_lyrics"`
	ExplicitContentLyrics int                   `json:"explicit_content_lyrics"`
	ExplicitContentCover  int                   `json:"explicit_content_cover"`
	Contributors          []resAlbumContributor `json:"contributors"`
	Artist                resAlbumArtist        `json:"artist"`
	Type                  string                `json:"type"`
	Tracks                resAlbumTracks        `json:"tracks"`
}

type resPlaylist struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Picture   string    `json:"picture"`
	PictureXl string    `json:"picture_xl"`
	Tracks    resTracks `json:"tracks"`
}

type resPing struct {
	Error   []string `json:"error"`
	Results struct {
		Session         string `json:"SESSION"`
		UserId          int    `json:"USER_ID"`
		Checkform       string `json:"CHECKFORM"`
		ServerTimestamp int    `json:"SERVER_TIMESTAMP"`
	} `json:"results"`
}
