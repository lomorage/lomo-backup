package common

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type FilePartReadSeeker struct {
	f       *os.File
	start   int64
	end     int64
	current int64
}

func NewFilePartReadSeeker(f *os.File, start, end int64) *FilePartReadSeeker {
	return &FilePartReadSeeker{f: f, start: start, end: end, current: start}
}

func (prs *FilePartReadSeeker) Size() int64 {
	return prs.end - prs.start
}

func (prs *FilePartReadSeeker) Read(p []byte) (n int, err error) {
	currBegin := prs.current
	defer func() {
		logs := fmt.Sprintf("read %s %d bytes, start-%d, end-%d, currBefore-%d, currAfter-%d",
			prs.f.Name(), len(p), prs.start, prs.end, currBegin, prs.current)
		if err != nil {
			logs += ": " + err.Error()
		}
		logrus.Trace(logs)
	}()
	// seek to the current if it is not yet
	curr, err := prs.f.Seek(0, io.SeekCurrent)
	if err != nil {
		return
	}
	if curr != prs.current {
		curr, err = prs.f.Seek(prs.current, io.SeekStart)
		if err != nil {
			return
		}
		if curr != prs.current {
			return 0, fmt.Errorf("fail to seek to offset %d before read", prs.current)
		}
	}

	if prs.current >= prs.end {
		return 0, io.EOF
	}
	currLen := prs.end - prs.current
	if len(p) <= int(currLen) {
		n, err = prs.f.Read(p)
		if err != nil {
			return
		}
		prs.current += int64(n)
		return
	}
	buf := make([]byte, currLen)
	n, err = prs.f.Read(buf)
	if err != nil {
		return
	}
	copy(p, buf)
	prs.current += int64(n)

	return
}

func (prs *FilePartReadSeeker) Seek(offset int64, whence int) (n int64, err error) {
	defer func() {
		logs := fmt.Sprintf("seek %s request %d, %d, reply %d", prs.f.Name(), offset, whence, n)
		if err != nil {
			logs += ": " + err.Error()
		}
		logrus.Trace(logs)
	}()
	// seek to whenonce point, then seek
	switch whence {
	case io.SeekStart:
		if offset <= 0 {
			_, err = prs.f.Seek(prs.start, io.SeekStart)
			n = 0
			prs.current = prs.start
			return
		}
		if offset >= prs.end-prs.start {
			_, err = prs.f.Seek(prs.end, io.SeekStart)
			n = prs.end - prs.start
			prs.current = prs.end
			return
		}
		n, err = prs.f.Seek(prs.start+offset, io.SeekStart)
	case io.SeekCurrent:
		if offset < 0 {
			if prs.current+offset < prs.start {
				_, err = prs.f.Seek(prs.start, io.SeekStart)
				n = 0
				prs.current = prs.start
				return
			}
		} else {
			if prs.current+offset >= prs.end {
				n, err = prs.f.Seek(prs.end, io.SeekStart)
				n -= int64(prs.start)
				prs.current = prs.end
				return
			}
		}
		n, err = prs.f.Seek(prs.current+offset, io.SeekStart)
	case io.SeekEnd:
		if offset >= 0 {
			_, err = prs.f.Seek(prs.end, io.SeekStart)
			n = prs.end - prs.start
			prs.current = prs.end
			return
		}
		if offset*-1 > prs.end-prs.start {
			_, err = prs.f.Seek(prs.start, io.SeekStart)
			n = 0
			prs.current = prs.start
			return
		}
		n, err = prs.f.Seek(prs.start-offset, io.SeekStart)
	default:
		return 0, fmt.Errorf("not implemented")
	}
	n -= prs.start
	prs.current = prs.start + n

	return
}

type ReadSeekSaver struct {
	f  *os.File
	rs io.ReadSeeker
}

func NewReadSeekSaver(f *os.File, rs io.ReadSeeker) *ReadSeekSaver {
	return &ReadSeekSaver{f: f, rs: rs}
}

func (rss *ReadSeekSaver) Read(p []byte) (n int, err error) {
	n, err = rss.rs.Read(p)
	if err != nil {
		return
	}
	var n2 int
	if n == len(p) {
		n2, err = rss.f.Write(p)
	} else {
		n2, err = rss.f.Write(p[:n])
	}
	if err != nil {
		return
	}
	if n != n2 {
		return 0, fmt.Errorf("expect write %d, actual write %d", n, n2)
	}
	return
}

func (rss *ReadSeekSaver) Seek(offset int64, whence int) (n int64, err error) {
	n, err = rss.f.Seek(offset, whence)
	if err != nil {
		return
	}
	_, err2 := rss.rs.Seek(offset, whence)
	if err2 != nil {
		return 0, err2
	}
	return
}

func CalculateHashHex(hash []byte) string {
	return fmt.Sprintf("%x", hash)
}

func CalculateHashBase64(hash []byte) string {
	return base64.StdEncoding.EncodeToString(hash)
}

func CalculateHashBytes(buffer []byte) []byte {
	h := sha256.New()
	h.Write(buffer)
	return h.Sum(nil)
}

func CalculateHash(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// return hex encoding and base 64 encoding
func CalculateMultiPartsHash(path string, partSize int) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	partsHash := [][]byte{}
	var curr, partLength int64
	var remaining = int64(info.Size())
	for curr = 0; remaining != 0; curr += partLength {
		if remaining < int64(partSize) {
			partLength = remaining
		} else {
			partLength = int64(partSize)
		}
		prs := NewFilePartReadSeeker(f, curr, curr+partLength)
		h := sha256.New()
		_, err = io.Copy(h, prs)
		if err != nil {
			return nil, err
		}

		partsHash = append(partsHash, h.Sum(nil))

		remaining -= partLength
	}
	return partsHash, nil
}

func ConcatAndCalculateBase64Hash(parts [][]byte) (string, error) {
	h := sha256.New()
	for _, p := range parts {
		_, err := h.Write(p)
		if err != nil {
			return "", err
		}
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func FormatTimeDateOnly(t time.Time) string {
	return t.Format("2006-01-02")
}

func LogDebugObject(key string, obj any) {
	content, _ := json.Marshal(obj)
	logrus.Debugf("%s: %s", key, string(content))
}
