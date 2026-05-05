package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

func getFavorites(userId string, config configuration) (resTracks, error) {
	url := fmt.Sprintf("https://api.deezer.com/user/%s/tracks?limit=10000000000", userId)
	res, err := makeReq("GET", url, nil, config)
	if err != nil {
		return resTracks{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bytes, _ := io.ReadAll(res.Body)
		bstr := string(bytes)
		if len(bstr) > 200 {
			bstr = bstr[:200] + "..."
		}
		log.Printf("non-200 response body (truncated): %s", bstr)
		return resTracks{}, fmt.Errorf("got status code %d", res.StatusCode)
	}

	var tracks resTracks
	err = json.NewDecoder(res.Body).Decode(&tracks)
	return tracks, err
}

func getSongInfo(id int64, config configuration) (resSongInfo, error) {
	url := fmt.Sprintf("https://www.deezer.com/de/track/%d", id)

	res, err := makeReq("GET", url, nil, config)
	if err != nil {
		return resSongInfo{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bytes, _ := io.ReadAll(res.Body)
		bstr := string(bytes)
		if len(bstr) > 200 {
			bstr = bstr[:200] + "..."
		}
		log.Printf("non-200 response body (truncated): %s", bstr)
		return resSongInfo{}, fmt.Errorf("got status code %d", res.StatusCode)
	}

	bytes, _ := io.ReadAll(res.Body)
	s := string(bytes)

	startMarker := `window.__DZR_APP_STATE__ = `
	endMarker := `</script>`
	startIdx := strings.Index(s, startMarker)
	endIdx := strings.Index(s[startIdx:], endMarker)
	sData := s[startIdx+len(startMarker) : startIdx+endIdx]

	var songInfo resSongInfo
	err = json.NewDecoder(strings.NewReader(sData)).Decode(&songInfo)
	return songInfo, err
}

func getAlbum(albumId string, config configuration) (resAlbum, error) {
	url := fmt.Sprintf("https://api.deezer.com/album/%s", albumId)
	res, err := makeReq("GET", url, nil, config)
	if err != nil {
		return resAlbum{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bytes, _ := io.ReadAll(res.Body)
		bstr := string(bytes)
		if len(bstr) > 200 {
			bstr = bstr[:200] + "..."
		}
		log.Printf("non-200 response body (truncated): %s", bstr)
		return resAlbum{}, fmt.Errorf("got status code %d", res.StatusCode)
	}

	var album resAlbum
	err = json.NewDecoder(res.Body).Decode(&album)
	return album, err
}

func getAlbumSongs(albumId string, config configuration) (resAlbumInfo, error) {
	url := fmt.Sprintf("https://www.deezer.com/de/album/%s", albumId)

	res, err := makeReq("GET", url, nil, config)
	if err != nil {
		return resAlbumInfo{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bytes, _ := io.ReadAll(res.Body)
		bstr := string(bytes)
		if len(bstr) > 200 {
			bstr = bstr[:200] + "..."
		}
		log.Printf("non-200 response body (truncated): %s", bstr)
		return resAlbumInfo{}, fmt.Errorf("got status code %d", res.StatusCode)
	}

	bytes, _ := io.ReadAll(res.Body)
	s := string(bytes)

	startMarker := `window.__DZR_APP_STATE__ = `
	endMarker := `</script>`
	startIdx := strings.Index(s, startMarker)
	endIdx := strings.Index(s[startIdx:], endMarker)
	sData := s[startIdx+len(startMarker) : startIdx+endIdx]

	var albumInfo resAlbumInfo
	err = json.NewDecoder(strings.NewReader(sData)).Decode(&albumInfo)
	// Ignore error, because we're only unmarshaling SONGS
	return albumInfo, nil
}

// getPlaylist fetches playlist metadata and its full track list.
func getPlaylist(playlistId string, config configuration) (resPlaylist, error) {
	// Fetch playlist page like albums to reuse same ARL cookie and parsing
	var playlist resPlaylist
	apiUrl := fmt.Sprintf("https://api.deezer.com/playlist/%s?access_token=%s", playlistId, config.LicenseToken)
	res, err := makeReq("GET", apiUrl, nil, config)
	if err != nil {
		return resPlaylist{}, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		bodyBytes, _ := io.ReadAll(res.Body)
		if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&playlist); err == nil {
			// if API returned tracks, return it
			if playlist.Tracks.Total > 0 || len(playlist.Tracks.Data) > 0 {
				return playlist, nil
			}
		}
		// fall through to webpage parsing below
	}
	// Fallback: fetch playlist page like albums to reuse ARL cookie and parsing
	pageUrl := fmt.Sprintf("https://www.deezer.com/de/playlist/%s", playlistId)
	resPage, err := makeReq("GET", pageUrl, nil, config)
	if err != nil {
		return resPlaylist{}, err
	}
	defer resPage.Body.Close()

	if resPage.StatusCode == 200 {
		bodyBytes, _ := io.ReadAll(resPage.Body)
		s := string(bodyBytes)

		startMarker := `window.__DZR_APP_STATE__ = `
		endMarker := `</script>`
		startIdx := strings.Index(s, startMarker)
		if startIdx >= 0 {
			endIdx := strings.Index(s[startIdx:], endMarker)
			if endIdx >= 0 {
				sData := s[startIdx+len(startMarker) : startIdx+endIdx]
				var generic interface{}
				if err := json.NewDecoder(strings.NewReader(sData)).Decode(&generic); err == nil {
					// try to find playlist title in parsed state
					var foundTitle string
					var walkTitle func(interface{}) bool
					walkTitle = func(n interface{}) bool {
						switch v := n.(type) {
						case map[string]interface{}:
							// common keys: "PLAYLIST" or objects with "TITLE"
							if t, ok := v["PLAYLIST"]; ok {
								if mp, ok2 := t.(map[string]interface{}); ok2 {
									if title, ok3 := mp["TITLE"].(string); ok3 {
										foundTitle = title
										return true
									}
									if title, ok3 := mp["title"].(string); ok3 {
										foundTitle = title
										return true
									}
								}
							}
							if title, ok := v["TITLE"].(string); ok && foundTitle == "" {
								foundTitle = title
								return true
							}
							if title, ok := v["title"].(string); ok && foundTitle == "" {
								foundTitle = title
								return true
							}
							for _, val := range v {
								if walkTitle(val) {
									return true
								}
							}
						case []interface{}:
							for _, el := range v {
								if walkTitle(el) {
									return true
								}
							}
						}
						return false
					}
					walkTitle(generic)
					playlist.Title = foundTitle
				}
			}
		}
		// get tracks using the same page parsing approach
		tracks, err2 := getPlaylistSongs(playlistId, config)
		if err2 == nil {
			playlist.Tracks = tracks
		}
		return playlist, nil
	}

	// get tracks using the same page parsing approach
	tracks, err := getPlaylistSongs(playlistId, config)
	if err == nil {
		playlist.Tracks = tracks
	}

	return playlist, nil
}

// getPlaylistSongs parses the public playlist page and extracts track list
func getPlaylistSongs(playlistId string, config configuration) (resTracks, error) {
	url := fmt.Sprintf("https://www.deezer.com/playlist/%s", playlistId)
	res, err := makeReq("GET", url, nil, config)
	if err != nil {
		return resTracks{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bytes, _ := io.ReadAll(res.Body)
		bstr := string(bytes)
		if len(bstr) > 200 {
			bstr = bstr[:200] + "..."
		}
		log.Printf("non-200 playlist page body (truncated): %s", bstr)

		if res.StatusCode == 404 {
			return resTracks{}, fmt.Errorf("Playlist not found - it may be private or deleted")
		}
		return resTracks{}, fmt.Errorf("HTTP %d - Failed to access playlist", res.StatusCode)
	}

	bytesBody, _ := io.ReadAll(res.Body)
	s := string(bytesBody)

	startMarker := `window.__DZR_APP_STATE__ = `
	endMarker := `</script>`
	startIdx := strings.Index(s, startMarker)
	if startIdx < 0 {
		return resTracks{}, fmt.Errorf("could not find app state in playlist page")
	}
	endIdx := strings.Index(s[startIdx:], endMarker)
	if endIdx < 0 {
		return resTracks{}, fmt.Errorf("could not find script end in playlist page")
	}
	sData := s[startIdx+len(startMarker) : startIdx+endIdx]

	var generic interface{}
	if err := json.NewDecoder(strings.NewReader(sData)).Decode(&generic); err != nil {
		return resTracks{}, err
	}

	// recursive search for an array of track-like objects
	var found []interface{}
	var walk func(interface{}) bool
	walk = func(n interface{}) bool {
		switch v := n.(type) {
		case map[string]interface{}:
			for _, val := range v {
				if walk(val) {
					return true
				}
			}
		case []interface{}:
			if len(v) > 0 {
				if first, ok := v[0].(map[string]interface{}); ok {
					// heuristic: track object should have an "id" or "SNG_ID"
					if _, hasId := first["id"]; hasId {
						found = v
						return true
					}
					if _, hasSng := first["SNG_ID"]; hasSng {
						// album song info uses SNG_ID, convert later
						found = v
						return true
					}
				}
			}
			// continue search inside array elements
			for _, elem := range v {
				if walk(elem) {
					return true
				}
			}
		}
		return false
	}

	walk(generic)
	if found == nil {
		return resTracks{}, fmt.Errorf("no track array found in playlist page")
	}

	// Convert found array into []resTrack robustly (tolerate type variations)
	tracks := make([]resTrack, 0, len(found))
	for _, el := range found {
		m, ok := el.(map[string]interface{})
		if !ok {
			continue
		}
		var t resTrack

		// id (float64 or string)
		if v, ok := m["id"]; ok {
			switch vv := v.(type) {
			case float64:
				t.Id = int64(vv)
			case string:
				if idInt, err := strconv.ParseInt(vv, 10, 64); err == nil {
					t.Id = idInt
				}
			}
		} else if v, ok := m["SNG_ID"]; ok {
			if s, ok2 := v.(string); ok2 {
				if idInt, err := strconv.ParseInt(s, 10, 64); err == nil {
					t.Id = idInt
				}
			}
		}

		// title
		if v, ok := m["title"]; ok {
			if s, ok2 := v.(string); ok2 {
				t.Title = s
			}
		} else if v, ok := m["SNG_TITLE"]; ok {
			if s, ok2 := v.(string); ok2 {
				t.Title = s
			}
		} else if v, ok := m["SngTitle"]; ok {
			if s, ok2 := v.(string); ok2 {
				t.Title = s
			}
		}

		// md5 image
		if v, ok := m["md5_image"]; ok {
			if s, ok2 := v.(string); ok2 {
				t.Md5Image = s
			}
		} else if v, ok := m["MD5_ORIGIN"]; ok {
			if s, ok2 := v.(string); ok2 {
				t.Md5Image = s
			}
		}

		// album title/md5
		if alb, ok := m["album"]; ok {
			if amap, ok2 := alb.(map[string]interface{}); ok2 {
				if at, ok3 := amap["title"]; ok3 {
					if s, ok4 := at.(string); ok4 {
						t.Album.Title = s
					}
				}
				if md, ok3 := amap["md5_image"]; ok3 {
					if s, ok4 := md.(string); ok4 {
						t.Album.Md5Image = s
					}
				}
			}
		} else {
			if v, ok := m["ALB_TITLE"]; ok {
				if s, ok2 := v.(string); ok2 {
					t.Album.Title = s
				}
			}
		}

		// duration (number or string)
		if v, ok := m["duration"]; ok {
			switch vv := v.(type) {
			case float64:
				t.Duration = int(vv)
			case string:
				if d, err := strconv.Atoi(vv); err == nil {
					t.Duration = d
				}
			}
		} else if v, ok := m["DURATION"]; ok {
			if s, ok2 := v.(string); ok2 {
				if d, err := strconv.Atoi(s); err == nil {
					t.Duration = d
				}
			}
		}

		// best-effort artist
		if art, ok := m["artist"]; ok {
			if amap, ok2 := art.(map[string]interface{}); ok2 {
				if idv, ok3 := amap["id"]; ok3 {
					switch iv := idv.(type) {
					case float64:
						t.Artist.Id = int64(iv)
					case string:
						if idInt, err := strconv.ParseInt(iv, 10, 64); err == nil {
							t.Artist.Id = idInt
						}
					}
				}
				if namev, ok3 := amap["name"]; ok3 {
					if s, ok4 := namev.(string); ok4 {
						t.Artist.Name = s
					}
				}
			}
		}

		tracks = append(tracks, t)
	}

	return resTracks{Data: tracks, Total: len(tracks)}, nil
}

func getSongUrlData(trackToken string, format string, config configuration) (resSongUrl, error) {
	url := "https://media.deezer.com/v1/get_url"
	bodyJsonStr := fmt.Sprintf(`{"license_token":"%s","media":[{"type":"FULL","formats":[{"cipher":"BF_CBC_STRIPE","format":"%s"}]}],"track_tokens":["%s"]}`, config.LicenseToken, format, trackToken)
	res, err := makeReq("POST", url, bytes.NewBuffer([]byte(bodyJsonStr)), config)
	if err != nil {
		return resSongUrl{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bytes, _ := io.ReadAll(res.Body)
		bstr := string(bytes)
		if len(bstr) > 200 {
			bstr = bstr[:200] + "..."
		}
		log.Printf("non-200 get_url response (truncated): %s", bstr)
		return resSongUrl{}, fmt.Errorf("got status code %d", res.StatusCode)
	}

	var songUrlData resSongUrl
	err = json.NewDecoder(res.Body).Decode(&songUrlData)

	if len(songUrlData.Data) == 0 {
		return resSongUrl{}, fmt.Errorf("got empty Data array when trying to get song URL")
	}

	if len(songUrlData.Data[0].Errors) > 0 {
		return resSongUrl{}, fmt.Errorf("got error when trying to get song URL: %s", songUrlData.Data[0].Errors[0].Message)
	}

	// If Data exists but Media is empty, treat it as "format not available"
	if len(songUrlData.Data[0].Media) == 0 {
		return resSongUrl{}, fmt.Errorf("no media available for requested format %s", format)
	}
	return songUrlData, err
}

func getPing(config configuration) (resPing, error) {
	url := "https://www.deezer.com/ajax/gw-light.php?method=deezer.ping&input=3&api_version=1.0&api_token"
	res, err := makeReq("GET", url, nil, config)
	if err != nil {
		return resPing{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bytes, _ := io.ReadAll(res.Body)
		bstr := string(bytes)
		if len(bstr) > 200 {
			bstr = bstr[:200] + "..."
		}
		log.Printf("non-200 ping response (truncated): %s", bstr)
		return resPing{}, fmt.Errorf("got status code %d", res.StatusCode)
	}

	var ping resPing
	err = json.NewDecoder(res.Body).Decode(&ping)
	return ping, err
}

func getSongUrl(songUrlData resSongUrl) (string, error) {
	if len(songUrlData.Data) == 0 || len(songUrlData.Data[0].Media) == 0 {
		return "", errors.New("no media available in songUrlData")
	}
	sources := songUrlData.Data[0].Media[0].Sources
	for _, source := range sources {
		if source.Provider == "ak" {
			return source.Url, nil
		}
	}
	return sources[0].Url, nil
}
