package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	for _, text := range []string{
		"a",
		"small one",
		"some plaintext longer than length of block size -----",
	} {
		for _, hasHeader := range []bool{true, false} {
			testEncryptDecrypt(t, []byte(text), hasHeader)
		}
	}
}

func genExpectEncryptHash(t *testing.T, plaintext, key, iv []byte, inclIv bool) []byte {
	stream, err := newCipherStream(key, iv)
	require.Nil(t, err)

	buf := make([]byte, len(plaintext))
	stream.XORKeyStream(buf, plaintext)

	h := sha256.New()
	if inclIv {
		_, err = h.Write(append(iv, buf...))
	} else {
		_, err = h.Write(buf)
	}
	require.Nil(t, err)
	return h.Sum(nil)
}

func testEncryptDecrypt(t *testing.T, plaintext []byte, hasHeader bool) {
	key, err := hex.DecodeString("6368616e676520746869732070617373")
	require.Nil(t, err)

	iv := make([]byte, aes.BlockSize)
	_, err = io.ReadFull(rand.Reader, iv)
	require.Nil(t, err)

	expectHashEncrypt := genExpectEncryptHash(t, plaintext, key, iv, hasHeader)

	buf := bytes.NewReader(plaintext)
	en, err := NewEncryptor(buf, key, iv, hasHeader)
	require.Nil(t, err)

	// make buffer larger
	testBuffer := make([]byte, 100)
	if hasHeader {
		n, err := en.Read(testBuffer)
		require.Nil(t, err)
		require.EqualValues(t, n, len(iv), plaintext)
		require.EqualValues(t, iv, testBuffer[:len(iv)])
	}
	n, err := en.Read(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, len(plaintext))

	// verify hash
	h := sha256.New()
	h.Write(plaintext)
	require.EqualValues(t, h.Sum(nil), en.GetHashOrig(), string(plaintext))
	require.EqualValues(t, expectHashEncrypt, en.GetHashEncrypt())

	testBuffer = testBuffer[:n]

	// test buffer should not be equal to plaintext
	require.NotEqualValues(t, plaintext, testBuffer)

	// now try decrypt
	decyptBuf := &bytes.Buffer{}
	de, err := NewDecryptor(decyptBuf, key, iv)
	require.Nil(t, err)

	n, err = de.Write(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, len(plaintext), n)
	require.EqualValues(t, plaintext, decyptBuf.Bytes())
}
