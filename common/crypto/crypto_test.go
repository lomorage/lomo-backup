package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	key, err := hex.DecodeString("6368616e676520746869732070617373")
	require.Nil(t, err)

	plaintext := []byte("some plaintext very very long -----")
	iv := make([]byte, aes.BlockSize)
	_, err = io.ReadFull(rand.Reader, iv)
	require.Nil(t, err)

	buf := bytes.NewBuffer(plaintext)
	en, err := NewEncryptor(buf, key, iv)
	require.Nil(t, err)

	testBuffer := make([]byte, len(plaintext))

	n, err := en.Read(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, len(iv))
	require.EqualValues(t, iv, testBuffer[:len(iv)])

	n, err = en.Read(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, len(plaintext))

	// test buffer should not be equal to plaintext
	require.NotEqualValues(t, plaintext, testBuffer)

	// now try decrypt
	decyptBuf := &bytes.Buffer{}
	de, err := NewDecryptor(decyptBuf, key, iv)
	require.Nil(t, err)

	n, err = de.Write(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, len(plaintext))
	require.EqualValues(t, plaintext, decyptBuf.Bytes())
}
