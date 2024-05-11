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
		testEncryptDecrypt(t, []byte(text))
	}
}

func genExpectHash(t *testing.T, plaintext, key, iv []byte) []byte {
	stream, err := newCipherStream(key, iv)
	require.Nil(t, err)

	buf := make([]byte, len(plaintext))
	stream.XORKeyStream(buf, plaintext)

	h := sha256.New()
	_, err = h.Write(append(iv, buf...))
	require.Nil(t, err)
	return h.Sum(nil)
}

func testEncryptDecrypt(t *testing.T, plaintext []byte) {
	key, err := hex.DecodeString("6368616e676520746869732070617373")
	require.Nil(t, err)

	iv := make([]byte, aes.BlockSize)
	_, err = io.ReadFull(rand.Reader, iv)
	require.Nil(t, err)

	expectHash := genExpectHash(t, plaintext, key, iv)

	buf := bytes.NewReader(plaintext)
	en, err := NewEncryptor(buf, key, iv)
	require.Nil(t, err)

	// make buffer larger
	testBuffer := make([]byte, 100)

	n, err := en.Read(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, len(iv), plaintext)
	require.EqualValues(t, iv, testBuffer[:len(iv)])

	n, err = en.Read(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, len(plaintext))

	// verify hash
	require.EqualValues(t, expectHash, en.GetHash())

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
