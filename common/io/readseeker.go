package io

import (
	"fmt"
	"io"
	"os"

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
				n -= prs.start
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
		n, err = prs.f.Seek(prs.end+offset, io.SeekStart)
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
