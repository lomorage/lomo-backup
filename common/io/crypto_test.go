package io

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func getCryptoStream(t *testing.T, key, iv []byte) cipher.Stream {
	block, err := aes.NewCipher(key)
	require.Nil(t, err)

	return cipher.NewCTR(block, iv)
}

func verifyCryptWriterRead(t *testing.T, r *CryptoStreamReader, stepNonce, stepBuf, nl, l int, expectData []byte) {
	// longer buffer
	buf := make([]byte, stepNonce)
	offset := 0
	for i := 0; i < nl; i += stepNonce {
		n, err := r.Read(buf)
		require.Nil(t, err)
		require.EqualValues(t, stepNonce, n, "Iteration %d", i)
		require.EqualValues(t, expectData[offset:offset+stepNonce], buf)
		offset += stepNonce
	}

	// reduce buffer for regular stepNonce
	buf = make([]byte, stepBuf)
	for i := 0; i < l; i += stepBuf {
		n, err := r.Read(buf)
		require.Nil(t, err)
		require.EqualValues(t, stepBuf, n, "Iteration %d", i)
		require.EqualValues(t, expectData[offset:offset+stepBuf], buf)
		offset += stepBuf
	}
}

func TestCryptoStreamReaderBasic(t *testing.T) {
	l := 100
	nl := 16
	data := make([]byte, l)
	nonce := make([]byte, nl)
	for i := 0; i < l; i++ {
		if i < nl {
			nonce[i] = byte(l - i)
		}
		data[i] = byte(i)
	}

	buf := bytes.NewBuffer(data)
	r := NewCryptoStreamReader(buf, nonce, nil)

	// normal read
	testBuffer := make([]byte, l+1)
	n, err := r.Read(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, nl)
	require.EqualValues(t, nonce, testBuffer[:n])

	n, err = r.Read(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, l)
	require.EqualValues(t, data, testBuffer[:n])

	// more complex test
	// reset buffer
	buf = bytes.NewBuffer(data)
	r = NewCryptoStreamReader(buf, nonce, nil)
	// try small buffer
	verifyCryptWriterRead(t, r, 2, 2, nl, l, append(nonce, data...))

	buf = bytes.NewBuffer(data)
	r = NewCryptoStreamReader(buf, nonce, nil)
	// different buffer
	verifyCryptWriterRead(t, r, nl/2, l/2, nl, l, append(nonce, data...))

	buf = bytes.NewBuffer(data)
	r = NewCryptoStreamReader(buf, nonce, nil)
	// first read is small buffer, and big buffer at succeeding
	verifyCryptWriterRead(t, r, nl/2, l, nl, l, append(nonce, data...))
}

func TestCryptoStream(t *testing.T) {
	l := 100
	nl := aes.BlockSize
	data := make([]byte, l)
	nonce := make([]byte, nl)
	for i := 0; i < l; i++ {
		if i < nl {
			nonce[i] = byte(l - i)
		}
		data[i] = byte(i)
	}

	key, _ := hex.DecodeString("6368616e676520746869732070617373")

	stream := getCryptoStream(t, key, nonce)

	buf := bytes.NewBuffer(data)

	r := NewCryptoStreamReader(buf, nonce, stream)

	// normal read
	testBuffer := make([]byte, l+1)
	n, err := r.Read(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, nl)
	require.EqualValues(t, nonce, testBuffer[:n])

	n, err = r.Read(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, l)

	expectStream := getCryptoStream(t, key, nonce)
	expectData := make([]byte, l)
	expectStream.XORKeyStream(expectData, data)

	require.EqualValues(t, expectData, testBuffer[:n])

	// use this test Buffer as input to test writer
	// recreate stream
	stream = getCryptoStream(t, key, nonce)

	bufWrite := &bytes.Buffer{}
	w := NewCryptoStreamWriter(bufWrite, stream)
	n, err = w.Write(expectData)
	require.Nil(t, err)
	require.Equal(t, l, n)
	require.EqualValues(t, data, bufWrite.Bytes())
}
