package io

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func getCryptoStream(t *testing.T, key, iv []byte) cipher.Stream {
	block, err := aes.NewCipher(key)
	require.Nil(t, err)

	return cipher.NewCTR(block, iv)
}

func verifyCryptoReadOnce(t *testing.T, r *CryptoStreamReader, step, l int, expectData []byte) {
	if step == 0 {
		return
	}
	buf := make([]byte, step)
	offset := 0
	for offset < l {
		n, err := r.Read(buf)
		require.Nil(t, err, "offset %d", offset)
		expectReadLen := step
		if step > l {
			expectReadLen = l
		} else if offset+step > l {
			expectReadLen = l - offset
		}
		require.EqualValues(t, expectReadLen, n, "offset %d", offset)
		require.EqualValues(t, expectData[offset:offset+expectReadLen], buf[:expectReadLen], "offset %d", offset)
		offset += n
	}
	// check partial file
	if offset == l || offset-step >= l {
		return
	}
	offset -= step
	// check last part
	n, err := r.Read(buf)
	require.Nil(t, err)
	require.EqualValues(t, l-offset, n, "offset %d", offset)
	require.EqualValues(t, expectData[offset:l], buf[:n])
}

//nolint:unparam
func verifyCryptoRead(t *testing.T, r *CryptoStreamReader, stepNonce, stepBuf, nl, l int,
	expectData, expectHashOrig, expectHashEncryt []byte) {
	verifyCryptoReadOnce(t, r, stepNonce, nl, expectData)

	verifyCryptoReadOnce(t, r, stepBuf, l, expectData[nl:])

	require.EqualValues(t, r.GetHashOrig(), expectHashOrig)
	require.EqualValues(t, r.GetHashEncrypt(), expectHashEncryt)
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

	h := sha256.New()
	h.Write(data)
	expectHashOrig := h.Sum(nil)

	h = sha256.New()
	h.Write(nonce)
	h.Write(data)
	expectHashEncrypt := h.Sum(nil)

	buf := bytes.NewReader(data)
	r, err := NewCryptoStreamReader(buf, nonce, nil)
	require.Nil(t, err)

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

	require.EqualValues(t, expectHashOrig, r.GetHashOrig())
	require.EqualValues(t, expectHashEncrypt, r.GetHashEncrypt())

	// even step read
	// reset buffer
	buf = bytes.NewReader(data)
	r, err = NewCryptoStreamReader(buf, nonce, nil)
	require.Nil(t, err)
	// try small buffer
	verifyCryptoRead(t, r, 2, 2, nl, l, append(nonce, data...), expectHashOrig, expectHashEncrypt)

	buf = bytes.NewReader(data)
	r, err = NewCryptoStreamReader(buf, nonce, nil)
	require.Nil(t, err)
	// different buffer
	verifyCryptoRead(t, r, nl/2, l/2, nl, l, append(nonce, data...), expectHashOrig, expectHashEncrypt)

	buf = bytes.NewReader(data)
	r, err = NewCryptoStreamReader(buf, nonce, nil)
	require.Nil(t, err)
	// first read is small buffer, and big buffer at succeeding
	verifyCryptoRead(t, r, nl/2, l, nl, l, append(nonce, data...), expectHashOrig, expectHashEncrypt)

	// more complex test. return buffer is different
	// case 1: verify the length of read buffer > 1/2*l, but < l
	buf = bytes.NewReader(data)
	r, err = NewCryptoStreamReader(buf, nonce, nil)
	require.Nil(t, err)
	verifyCryptoRead(t, r, nl/2+1, l, nl, l, append(nonce, data...), expectHashOrig, expectHashEncrypt)

	// case 2: verify the length of read buffer > 1/2*nl, but < nl
	buf = bytes.NewReader(data)
	r, err = NewCryptoStreamReader(buf, nonce, nil)
	require.Nil(t, err)
	verifyCryptoRead(t, r, nl, l/2+1, nl, l, append(nonce, data...), expectHashOrig, expectHashEncrypt)

	// case 3: each step is half plus 1
	buf = bytes.NewReader(data)
	r, err = NewCryptoStreamReader(buf, nonce, nil)
	require.Nil(t, err)
	verifyCryptoRead(t, r, nl/2+1, l/2+1, nl, l, append(nonce, data...), expectHashOrig, expectHashEncrypt)
}

func TestCryptoStreamReaderBasicNoNonce(t *testing.T) {
	l := 100
	data := make([]byte, l)
	for i := 0; i < l; i++ {
		data[i] = byte(i)
	}

	h := sha256.New()
	h.Write(data)
	expectHash := h.Sum(nil)

	buf := bytes.NewReader(data)
	r, err := NewCryptoStreamReader(buf, nil, nil)
	require.Nil(t, err)

	// normal read
	testBuffer := make([]byte, l+1)

	n, err := r.Read(testBuffer)
	require.Nil(t, err)
	require.EqualValues(t, n, l)
	require.EqualValues(t, data, testBuffer[:n])

	require.EqualValues(t, expectHash, r.GetHashOrig())
	require.EqualValues(t, sha256.New().Sum(nil), r.GetHashEncrypt())

	// even step read
	// reset buffer
	buf = bytes.NewReader(data)
	r, err = NewCryptoStreamReader(buf, nil, nil)
	require.Nil(t, err)
	// try small buffer
	verifyCryptoRead(t, r, 0, 2, 0, l, data, expectHash, sha256.New().Sum(nil))

	buf = bytes.NewReader(data)
	r, err = NewCryptoStreamReader(buf, nil, nil)
	require.Nil(t, err)
	// different buffer
	verifyCryptoRead(t, r, 0, l/2, 0, l, data, expectHash, sha256.New().Sum(nil))

	// case 2: verify the length of read buffer > 1/2*nl, but < nl
	buf = bytes.NewReader(data)
	r, err = NewCryptoStreamReader(buf, nil, nil)
	require.Nil(t, err)
	verifyCryptoRead(t, r, 0, l/2+1, 0, l, data, expectHash, sha256.New().Sum(nil))
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

	buf := bytes.NewReader(data)
	r, err := NewCryptoStreamReader(buf, nonce, stream)
	require.Nil(t, err)

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

	// test hash
	h := sha256.New()
	h.Write(data)
	expectHashOrig := h.Sum(nil)

	require.EqualValues(t, expectHashOrig, r.GetHashOrig())

	h = sha256.New()
	h.Write(nonce)
	h.Write(expectData)
	expectHashEncrypt := h.Sum(nil)

	require.EqualValues(t, expectHashEncrypt, r.GetHashEncrypt())

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

func TestCryptoStreamNoNonce(t *testing.T) {
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

	buf := bytes.NewReader(data)
	r, err := NewCryptoStreamReader(buf, nil, stream)
	require.Nil(t, err)

	// normal read
	testBuffer := make([]byte, l+1)
	n, err := r.Read(testBuffer)
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

func TestCryptoStreamReaderSeek(t *testing.T) {
	nl := 16
	nonce := make([]byte, nl)
	for i := 0; i < nl; i++ {
		nonce[i] = byte(i)
	}

	f, err := os.Open(testFilename)
	require.Nil(t, err)
	defer f.Close()

	key, _ := hex.DecodeString("6368616e676520746869732070617373")

	stream := getCryptoStream(t, key, nonce)

	r, err := NewCryptoStreamReader(f, nonce, stream)
	require.Nil(t, err)

	testReadSeekerSeek(t, r)
}

func TestCryptoStreamReaderSeekNoNonce(t *testing.T) {
	nl := 16
	nonce := make([]byte, nl)
	for i := 0; i < nl; i++ {
		nonce[i] = byte(i)
	}

	f, err := os.Open(testFilename)
	require.Nil(t, err)
	defer f.Close()

	key, _ := hex.DecodeString("6368616e676520746869732070617373")

	stream := getCryptoStream(t, key, nonce)

	r, err := NewCryptoStreamReader(f, nil, stream)
	require.Nil(t, err)

	testReadSeekerSeek(t, r)
}

func TestCryptoStreamReaderSeekReadNoEncrypt(t *testing.T) {
	nl := 16
	nonce := make([]byte, nl)
	for i := 0; i < nl; i++ {
		nonce[i] = byte(i)
	}

	f, err := os.Open(testFilename)
	require.Nil(t, err)
	defer f.Close()

	r, err := NewCryptoStreamReader(f, nonce, nil)
	require.Nil(t, err)

	expectFile, err := os.Open(testFilename)
	require.Nil(t, err)
	defer expectFile.Close()

	// initial read will return nonce
	buf := make([]byte, 200)
	n, err := r.Read(buf)
	require.Nil(t, err)
	require.EqualValues(t, len(nonce), n)
	require.EqualValues(t, nonce, buf[:n])

	// read from current
	verifyReadSeek(t, expectFile, r, 500, 500, 500, io.SeekCurrent)
	verifyReadSeek(t, expectFile, r, 400, -400, -400, io.SeekCurrent)

	// read from end
	verifyReadSeek(t, expectFile, r, 100, -100, -100, io.SeekEnd)
	verifyReadSeek(t, expectFile, r, 100, -200, -200, io.SeekEnd)

	// now read until end
	verifyReadSeek(t, expectFile, r, 101, -1, -1, io.SeekCurrent)
}

func verifyCryptoStreamRead(t *testing.T, expectReader, reader io.Reader, len int, stream cipher.Stream) {
	expectBuffer := make([]byte, len)
	tmpBuffer := make([]byte, len)
	expectSize, err := expectReader.Read(tmpBuffer)
	require.Nil(t, err)
	stream.XORKeyStream(expectBuffer, tmpBuffer)

	buffer := make([]byte, len)
	size, err := reader.Read(buffer)
	require.Nil(t, err, "read lengh: %d", size)

	require.Equal(t, expectSize, size)
	require.Equal(t, expectBuffer, buffer)
}

func verifyCryptoReadSeek(t *testing.T, expectReadSeeker, readSeeker io.ReadSeeker,
	len, expectOffset, offset, whence int, stream cipher.Stream) {
	_, err := expectReadSeeker.Seek(int64(expectOffset), whence)
	require.Nil(t, err)

	_, err = readSeeker.Seek(int64(offset), whence)
	require.Nil(t, err)

	verifyCryptoStreamRead(t, expectReadSeeker, readSeeker, len, stream)
}

func TestCryptoStreamReaderSeekReadNoEncryptNoNonce(t *testing.T) {
	f, err := os.Open(testFilename)
	require.Nil(t, err)
	defer f.Close()

	r, err := NewCryptoStreamReader(f, nil, nil)
	require.Nil(t, err)

	expectFile, err := os.Open(testFilename)
	require.Nil(t, err)
	defer expectFile.Close()

	// read from current
	verifyReadSeek(t, expectFile, r, 500, 500, 500, io.SeekCurrent)
	verifyReadSeek(t, expectFile, r, 400, -400, -400, io.SeekCurrent)

	// read from end
	verifyReadSeek(t, expectFile, r, 100, -100, -100, io.SeekEnd)
	verifyReadSeek(t, expectFile, r, 100, -200, -200, io.SeekEnd)

	// now read until end
	verifyReadSeek(t, expectFile, r, 101, -1, -1, io.SeekCurrent)
}

func TestCryptoStreamReaderSeekReadEncrypt(t *testing.T) {
	nl := 16
	nonce := make([]byte, nl)
	for i := 0; i < nl; i++ {
		nonce[i] = byte(i)
	}

	f, err := os.Open(testFilename)
	require.Nil(t, err)
	defer f.Close()

	key, _ := hex.DecodeString("6368616e676520746869732070617373")

	stream := getCryptoStream(t, key, nonce)

	r, err := NewCryptoStreamReader(f, nonce, stream)
	require.Nil(t, err)

	expectFile, err := os.Open(testFilename)
	require.Nil(t, err)
	defer expectFile.Close()

	// initial read will return nonce
	buf := make([]byte, 200)
	n, err := r.Read(buf)
	require.Nil(t, err)
	require.EqualValues(t, len(nonce), n)
	require.EqualValues(t, nonce, buf[:n])

	expectStream := getCryptoStream(t, key, nonce)

	// read from current
	verifyCryptoReadSeek(t, expectFile, r, 500, 500, 500, io.SeekCurrent, expectStream)
	verifyCryptoReadSeek(t, expectFile, r, 400, -400, -400, io.SeekCurrent, expectStream)

	// read from end
	verifyCryptoReadSeek(t, expectFile, r, 100, -100, -100, io.SeekEnd, expectStream)
	verifyCryptoReadSeek(t, expectFile, r, 100, -200, -200, io.SeekEnd, expectStream)

	// now read until end
	verifyCryptoReadSeek(t, expectFile, r, 101, -1, -1, io.SeekCurrent, expectStream)
}

func TestCryptoStreamReaderEncryptLargeBuffer(t *testing.T) {
	// use large buffer to read multiple times, and value should be same
	nl := 16
	nonce := make([]byte, nl)
	for i := 0; i < nl; i++ {
		nonce[i] = byte(i)
	}

	f, err := os.Open(testFilename)
	require.Nil(t, err)
	defer f.Close()

	key, _ := hex.DecodeString("6368616e676520746869732070617373")

	stream := getCryptoStream(t, key, nonce)

	prs := NewFilePartReadSeeker(f, 0, 100)
	r, err := NewCryptoStreamReader(prs, nonce, stream)
	require.Nil(t, err)

	expectFile, err := os.Open(testFilename)
	require.Nil(t, err)
	defer expectFile.Close()

	// initial read will return nonce
	buf := make([]byte, 200)
	n, err := r.Read(buf)
	require.Nil(t, err)
	require.EqualValues(t, len(nonce), n)
	require.EqualValues(t, nonce, buf[:n])

	expectStream := getCryptoStream(t, key, nonce)

	verifyCryptoLargeBuffer(t, expectFile, expectStream, 100, r)

	prs.SetStartEnd(100, 200)
	verifyCryptoLargeBuffer(t, expectFile, expectStream, 100, r)

	prs.SetStartEnd(200, 201)
	verifyCryptoLargeBuffer(t, expectFile, expectStream, 1, r)

	prs.SetStartEnd(201, 300)
	verifyCryptoLargeBuffer(t, expectFile, expectStream, 99, r)
}

func verifyCryptoLargeBuffer(t *testing.T, expectReader io.Reader, expectStream cipher.Stream,
	expectLen int, stream *CryptoStreamReader) {
	expectReadBuffer := make([]byte, expectLen)
	expectBuffer := make([]byte, expectLen)
	expectSize, err := expectReader.Read(expectReadBuffer)
	require.Nil(t, err)
	expectStream.XORKeyStream(expectBuffer, expectReadBuffer)

	buffer := make([]byte, expectLen+100)
	size, err := stream.Read(buffer)
	require.Nil(t, err, "read lengh: %d", size)

	require.Equal(t, expectSize, size)
	require.Equal(t, expectBuffer, buffer[:size], "expect len: %d", expectLen)
}

func TestCryptoStreamReaderSeekReadEncryptNoNonce(t *testing.T) {
	nl := 16
	nonce := make([]byte, nl)
	for i := 0; i < nl; i++ {
		nonce[i] = byte(i)
	}

	f, err := os.Open(testFilename)
	require.Nil(t, err)
	defer f.Close()

	key, _ := hex.DecodeString("6368616e676520746869732070617373")

	stream := getCryptoStream(t, key, nonce)

	r, err := NewCryptoStreamReader(f, nil, stream)
	require.Nil(t, err)

	expectFile, err := os.Open(testFilename)
	require.Nil(t, err)
	defer expectFile.Close()

	expectStream := getCryptoStream(t, key, nonce)

	// read from current
	verifyCryptoReadSeek(t, expectFile, r, 500, 500, 500, io.SeekCurrent, expectStream)
	verifyCryptoReadSeek(t, expectFile, r, 400, -400, -400, io.SeekCurrent, expectStream)

	// read from end
	verifyCryptoReadSeek(t, expectFile, r, 100, -100, -100, io.SeekEnd, expectStream)
	verifyCryptoReadSeek(t, expectFile, r, 100, -200, -200, io.SeekEnd, expectStream)

	// now read until end
	verifyCryptoReadSeek(t, expectFile, r, 101, -1, -1, io.SeekCurrent, expectStream)
}
