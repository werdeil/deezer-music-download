package main

import (
	"bytes"
	"crypto/cipher"
	"encoding/hex"
	"testing"

	"golang.org/x/crypto/blowfish" //nolint:staticcheck // Deezer streams use Blowfish; the tests must too.
)

// Dummy values: the real pre_key/iv are never needed by the tests.
var testConfig = configuration{
	PreKey: "0123456789abcdef",
	Iv:     "0001020304050607",
}

func TestCalcBfKey(t *testing.T) {
	key := calcBfKey([]byte("123456"), testConfig)
	if len(key) != 16 {
		t.Fatalf("key length = %d, want 16", len(key))
	}
	if !bytes.Equal(key, calcBfKey([]byte("123456"), testConfig)) {
		t.Error("key derivation is not deterministic")
	}
	if bytes.Equal(key, calcBfKey([]byte("654321"), testConfig)) {
		t.Error("different song IDs produced the same key")
	}
	other := testConfig
	other.PreKey = "fedcba9876543210"
	if bytes.Equal(key, calcBfKey([]byte("123456"), other)) {
		t.Error("different pre_key produced the same key")
	}
}

func TestBlowfishDecryptRoundTrip(t *testing.T) {
	key := calcBfKey([]byte("42"), testConfig)
	plain := bytes.Repeat([]byte("deezer!!"), 2048/8)

	iv, _ := hex.DecodeString(testConfig.Iv)
	c, err := blowfish.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := make([]byte, len(plain))
	cipher.NewCBCEncrypter(c, iv).CryptBlocks(encrypted, plain)

	got, err := blowfishDecrypt(encrypted, key, testConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Error("decrypted data does not match the original")
	}
}

func TestBlowfishDecryptInvalidIv(t *testing.T) {
	bad := testConfig
	bad.Iv = "not-hex"
	if _, err := blowfishDecrypt(make([]byte, 8), make([]byte, 16), bad); err == nil {
		t.Error("expected an error for a non-hex iv")
	}
}
