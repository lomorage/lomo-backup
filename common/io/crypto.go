package io

import (
	"crypto/cipher"
	"crypto/sha256"
	"errors"
	"hash"
	"io"
)

// CryptoStreamReader implements io.Reader interface to act as proxy btw cloud API and local file
// It will also encrypt file on the fly
type CryptoStreamReader struct {
	f           io.ReadSeeker
	fileLen     int64
	nonce       []byte
	nonceLen    int64
	stream      cipher.Stream
	offset      int
	hashOrig    hash.Hash
	hashEncrypt hash.Hash
}

func NewCryptoStreamReader(f io.ReadSeeker, nonce []byte, stream cipher.Stream) (*CryptoStreamReader, error) {
	l, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	return &CryptoStreamReader{
		f: f, fileLen: l,
		nonce: nonce, nonceLen: int64(len(nonce)),
		stream:   stream,
		hashOrig: sha256.New(), hashEncrypt: sha256.New(),
	}, nil
}

func (r *CryptoStreamReader) Size() int64 {
	return r.nonceLen + r.fileLen
}

func (r *CryptoStreamReader) Read(p []byte) (n int, err error) {
	if r.offset < len(r.nonce) {
		// only copy w.nonce to buffer for initial read
		l := len(r.nonce)
		if l > len(p) {
			if l > r.offset+len(p) {
				l = len(p)
			} else {
				l -= r.offset
			}
		}
		copy(p, r.nonce[r.offset:r.offset+l])

		_, err = r.hashEncrypt.Write(p[:l])

		r.offset += l

		return l, err
	}
	if r.stream == nil {
		n, err = r.f.Read(p)
		if err != nil {
			return
		}
		r.offset += n
		_, err = r.hashOrig.Write(p[:n])
		if err != nil {
			return
		}
		if len(r.nonce) == 0 {
			return
		}
		_, err = r.hashEncrypt.Write(p[:n])
		return
	}

	buf := make([]byte, len(p))
	defer func() {
		buf = nil
	}()

	n, err = r.f.Read(buf)
	if err != nil {
		return 0, err
	}

	_, err = r.hashOrig.Write(buf[:n])
	if err != nil {
		return 0, err
	}

	r.offset += n

	r.stream.XORKeyStream(p, buf)

	_, err = r.hashEncrypt.Write(p[:n])
	if err != nil {
		return 0, err
	}
	return
}

func (r *CryptoStreamReader) Seek(offset int64, whence int) (n int64, err error) {
	switch whence {
	case io.SeekStart:
		r.offset = int(offset)
		if offset <= 0 {
			r.offset = 0
			_, err = r.f.Seek(0, whence)
			return 0, err
		}

		if offset > r.nonceLen+r.fileLen {
			// beyond actual file's length, seek to the end of current file
			r.offset = int(r.nonceLen + r.fileLen)
			_, err = r.f.Seek(0, io.SeekEnd)
			return r.nonceLen + r.fileLen, err
		}

		if offset < r.nonceLen {
			_, err = r.f.Seek(0, whence)
			return offset, err
		}

		_, err = r.f.Seek(offset-r.nonceLen, whence)
		return offset, err
	case io.SeekCurrent:
		// get actual file's current position
		n, err = r.f.Seek(0, io.SeekCurrent)
		if err != nil {
			return
		}

		if n+offset <= 0 {
			// roll back to beginning
			r.offset = 0
			n, err = r.f.Seek(0, io.SeekStart)
			return
		}

		if n+offset >= r.fileLen {
			// beyond actual file's length, seek to the end of current file
			r.offset = int(r.nonceLen + r.fileLen)
			_, err = r.f.Seek(0, io.SeekEnd)
			return r.nonceLen + r.fileLen, err
		}

		if n+offset < r.nonceLen {
			r.offset = int(n + offset)
			// rollback actual file to the beginning
			_, err = r.f.Seek(0, io.SeekStart)
			return int64(r.offset), err
		}

		r.offset = len(r.nonce) + int(n+offset)
		_, err = r.f.Seek(n+offset, io.SeekStart)
		return int64(r.offset), err
	case io.SeekEnd:
		r.offset = len(r.nonce) + int(r.fileLen+offset)
		if r.offset <= 0 {
			r.offset = 0
			_, err = r.f.Seek(0, io.SeekStart)
			return 0, err
		}

		if r.offset >= int(r.nonceLen+r.fileLen) {
			// beyond actual file's length, seek to the end of current file
			r.offset = int(r.nonceLen + r.fileLen)
			_, err = r.f.Seek(0, io.SeekEnd)
			return r.nonceLen + r.fileLen, err
		}

		if r.offset < int(r.nonceLen) {
			// rollback actual file to the beginning
			_, err = r.f.Seek(0, io.SeekStart)
			return int64(r.offset), err
		}

		_, err = r.f.Seek(r.fileLen+offset, io.SeekStart)
		return r.nonceLen + r.fileLen + offset, err
	}
	return 0, errors.New("not implemented")
}

func (r *CryptoStreamReader) GetHashOrig() []byte {
	return r.hashOrig.Sum(nil)
}

func (r *CryptoStreamReader) GetHashEncrypt() []byte {
	return r.hashEncrypt.Sum(nil)
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
