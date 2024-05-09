package io

import (
	"crypto/cipher"
	"io"
)

// CryptoStreamReader implements io.Reader interface to act as proxy btw cloud API and local file
// It will also encrypt file on the fly
type CryptoStreamReader struct {
	f      io.Reader
	nonce  []byte
	stream cipher.Stream
	offset int
}

func NewCryptoStreamReader(f io.Reader, nonce []byte, stream cipher.Stream) *CryptoStreamReader {
	return &CryptoStreamReader{f: f, nonce: nonce, stream: stream}
}

func (r *CryptoStreamReader) Read(p []byte) (n int, err error) {
	if r.offset < len(r.nonce) {
		// only copy w.nonce to buffer for initial read
		l := len(r.nonce)
		if l > len(p) {
			l = len(p)
		}
		copy(p, r.nonce[r.offset:r.offset+l])
		r.offset += l
		return l, nil
	}
	if r.stream == nil {
		return r.f.Read(p)
	}

	buf := make([]byte, len(p))
	defer func() {
		buf = nil
	}()

	n, err = r.f.Read(buf)
	if err != nil {
		return 0, err
	}

	r.stream.XORKeyStream(p, buf)

	return n, nil
}

type CryptoStreamWriter struct {
	f      io.Writer
	stream cipher.Stream
}

func NewCryptoStreamWriter(f io.Writer, stream cipher.Stream) *CryptoStreamWriter {
	return &CryptoStreamWriter{f: f, stream: stream}
}

func (w *CryptoStreamWriter) Write(p []byte) (n int, err error) {
	buf := make([]byte, len(p))
	defer func() {
		buf = nil
	}()

	w.stream.XORKeyStream(buf, p)
	return w.f.Write(buf)
}
