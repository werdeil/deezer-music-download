package main

import (
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"golang.org/x/crypto/blowfish" //nolint:staticcheck // Deezer encrypts streams with Blowfish; there is no choice of cipher.
)

func calcBfKey(songId []byte, config configuration) []byte {
	preKey := []byte(config.PreKey)
	songIdHash := md5.Sum(songId)
	songIdMd5 := hex.EncodeToString(songIdHash[:])
	key := make([]byte, 16)
	for i := 0; i < 16; i++ {
		key[i] = songIdMd5[i] ^ songIdMd5[i+16] ^ preKey[i]
	}
	return key
}

func blowfishDecrypt(data []byte, key []byte, config configuration) ([]byte, error) {
	iv, err := hex.DecodeString(config.Iv)
	if err != nil {
		return nil, err
	}
	c, err := blowfish.NewCipher(key)
	if err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCDecrypter(c, iv)
	res := make([]byte, len(data))
	cbc.CryptBlocks(res, data)
	return res, nil
}

func ensureSongDirectoryExists(songPath string, coverUrl string) error {
	var err error
	songDir := path.Dir(songPath)
	if _, err = os.Stat(songDir); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(songDir, os.ModePerm); err != nil {
			return err
		}

		textFilePath := songDir + "/info.txt"
		textFileData := []byte("Downloaded from Deezer.\n")
		err = os.WriteFile(textFilePath, textFileData, 0644)
		if err != nil {
			return err
		}

		if len(coverUrl) == 0 {
			log.Println("Skipping cover")
		} else {
			return downloadCover(coverUrl, songDir+"/folder.jpg")
		}
	}
	return nil
}

func downloadCover(coverUrl string, coverFilePath string) (err error) {
	f, err := os.Create(coverFilePath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	res, err := http.Get(coverUrl)
	if err != nil {
		return err
	}
	defer closeQuietly(res.Body)
	if res.StatusCode != 200 {
		return fmt.Errorf("error downloading cover: status %d", res.StatusCode)
	}
	_, err = io.Copy(f, res.Body)
	return err
}

func downloadSong(url string, songPath string, songId string, attempt int, config configuration) (written int64, err error) {
	if attempt >= 10 {
		return 0, fmt.Errorf("giving up downloading song after %d attempts", attempt)
	}

	f, err := os.Create(songPath)
	if err != nil {
		return 0, err
	}
	// The file is written to: a Close error can mean lost data, so report it.
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	res, err := makeReq("GET", url, nil, config)
	if err != nil {
		return 0, err
	}
	defer closeQuietly(res.Body)

	if res.StatusCode != 200 {
		bytes, _ := io.ReadAll(res.Body)
		bstr := string(bytes)
		if len(bstr) > 200 {
			bstr = bstr[:200] + "..."
		}
		log.Printf("non-200 download response (truncated): %s", bstr)
		return 0, fmt.Errorf("got status code %d", res.StatusCode)
	}

	bfKey := calcBfKey([]byte(songId), config)

	// One in every third 2048 byte block is encrypted
	blockSize := 2048
	buf := make([]byte, blockSize)
	i := 0
	nRead := 0
	totalBytes := 0
	breakNextTime := false

outer_loop:
	for {
		nRead = 0
		for nRead < blockSize {
			nNewRead, err := res.Body.Read(buf[nRead:])
			nRead += nNewRead
			totalBytes += nNewRead
			if breakNextTime {
				break outer_loop
			}
			if err == io.EOF {
				breakNextTime = true
				break
			}
			if err != nil && err != io.EOF {
				log.Printf("Error reading body on i=%d: %s\n", i, err)
				log.Println("Retrying")
				time.Sleep(500 * time.Millisecond)
				return downloadSong(url, songPath, songId, attempt+1, config)
			}
		}

		isEncrypted := ((i % 3) == 0)
		isWholeBlock := (nRead == blockSize)

		if isEncrypted && isWholeBlock {
			decBuf, err := blowfishDecrypt(buf, bfKey, config)
			if err != nil {
				return 0, fmt.Errorf("error decrypting: %w", err)
			}
			if _, err := f.Write(decBuf); err != nil {
				return 0, err
			}
		} else {
			if _, err := f.Write(buf[:nRead]); err != nil {
				return 0, err
			}
		}

		i += 1
	}

	return int64(totalBytes), nil
}
