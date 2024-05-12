package io

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const testFilename = "../testdata/indepedant_declaration.txt"

type readSeekSizer interface {
	io.ReadSeeker
	Size() int64
}

func verifySeek(t *testing.T, seeker io.Seeker, offset, expectPosition int64,
	whence int) {
	n, err := seeker.Seek(offset, whence)
	require.Nil(t, err)
	require.Equal(t, expectPosition, n, "offset: %d, whence: %d", offset, whence)
}

func testReadSeekerSeek(t *testing.T, prs readSeekSizer) {
	// this is normal steps to get file length
	// 1. seek 0 at current, and check current offset
	// 2. seek 0 from start, and ensure from the beginning
	// 3. seek 0 from current, and check if it is also from the beginning
	// 4. seek 0 from end, and get file length
	// 5. seek 0 from current, and check if it is same size as step 4
	// 6. seek 0 from start, and ensure from the beginning
	// 7. seek 0 from current, and check if it is also from the beginning
	verifySeek(t, prs, 0, 0, io.SeekCurrent)
	verifySeek(t, prs, 0, 0, io.SeekStart)
	verifySeek(t, prs, 0, 0, io.SeekCurrent)
	verifySeek(t, prs, 0, prs.Size(), io.SeekEnd)
	verifySeek(t, prs, 0, prs.Size(), io.SeekCurrent)
	verifySeek(t, prs, 0, 0, io.SeekStart)
	verifySeek(t, prs, 0, 0, io.SeekCurrent)

	// random seek
	// 1. seek to the half length from start, then seek 1 byte more from current
	// then seek -2 byte from current
	verifySeek(t, prs, prs.Size()/2, prs.Size()/2, io.SeekStart)
	verifySeek(t, prs, 1, prs.Size()/2+1, io.SeekCurrent)
	verifySeek(t, prs, -2, prs.Size()/2-1, io.SeekCurrent)

	// 2. seek to the half length from end, then seek 1 byte more from current
	// then seek -2 byte from current
	verifySeek(t, prs, -1*prs.Size()/2, prs.Size()/2, io.SeekEnd)
	verifySeek(t, prs, 1, prs.Size()/2+1, io.SeekCurrent)
	verifySeek(t, prs, -2, prs.Size()/2-1, io.SeekCurrent)

	// negative seek
	// 1. seek ahead of beginning of file
	verifySeek(t, prs, -1, 0, io.SeekStart)
	verifySeek(t, prs, -1-prs.Size(), 0, io.SeekCurrent)
	verifySeek(t, prs, -1-prs.Size(), 0, io.SeekEnd)

	// 2. seek after the end of file
	verifySeek(t, prs, prs.Size()+1, prs.Size(), io.SeekStart)
	verifySeek(t, prs, prs.Size()+1, prs.Size(), io.SeekCurrent)
	verifySeek(t, prs, 1, prs.Size(), io.SeekEnd)

	// 3. seek into middle, then seek above beginning
	verifySeek(t, prs, prs.Size()/2, prs.Size()/2, io.SeekStart)

	verifySeek(t, prs, -1, 0, io.SeekStart)
	verifySeek(t, prs, -1-prs.Size(), 0, io.SeekCurrent)
	verifySeek(t, prs, -1-prs.Size(), 0, io.SeekEnd)

	// 4. seek into middle, then seek after end
	verifySeek(t, prs, prs.Size()/2, prs.Size()/2, io.SeekStart)

	verifySeek(t, prs, prs.Size()+1, prs.Size(), io.SeekStart)
	verifySeek(t, prs, prs.Size()+1, prs.Size(), io.SeekCurrent)
	verifySeek(t, prs, 1, prs.Size(), io.SeekEnd)
}

func TestFilePartReadSeekerSeek(t *testing.T) {
	fpart, err := os.Open(testFilename)
	require.Nil(t, err)
	defer fpart.Close()

	prs := NewFilePartReadSeeker(fpart, 0, 1000)

	n, err := prs.Seek(0, io.SeekStart)
	require.Nil(t, err)
	require.Equal(t, int64(0), n)

	testReadSeekerSeek(t, prs)

	// create another seeker in the middle, but reuse current position
	prs = NewFilePartReadSeeker(fpart, 1000, 2000)
	testReadSeekerSeek(t, prs)

	// create the 3rd seeker in the middle, and seek original file to the beginning
	// above steps should have same result
	n, err = fpart.Seek(0, io.SeekStart)
	require.Nil(t, err)
	require.Equal(t, int64(0), n)

	prs = NewFilePartReadSeeker(fpart, 2000, 3000)
	testReadSeekerSeek(t, prs)
}

func verifyRead(t *testing.T, expectReader, reader io.Reader, len int) {
	expectBuffer := make([]byte, len)
	expectSize, err := expectReader.Read(expectBuffer)
	require.Nil(t, err)

	buffer := make([]byte, len)
	size, err := reader.Read(buffer)
	require.Nil(t, err, "read lengh: %d", size)

	require.Equal(t, expectSize, size)
	require.Equal(t, expectBuffer, buffer)
}

func TestFilePartReadSeekerRead(t *testing.T) {
	expectFile, err := os.Open(testFilename)
	require.Nil(t, err)
	defer expectFile.Close()

	fpart, err := os.Open(testFilename)
	require.Nil(t, err)
	defer fpart.Close()

	prs := NewFilePartReadSeeker(fpart, 0, 1000)
	verifyRead(t, expectFile, prs, 1000)

	prs = NewFilePartReadSeeker(fpart, 1000, 2000)
	verifyRead(t, expectFile, prs, 1000)

	// seek the multipart reader back to start, and read from middle, then compare
	n, err := fpart.Seek(0, io.SeekStart)
	require.Nil(t, err)
	require.EqualValues(t, 0, n)

	prs = NewFilePartReadSeeker(fpart, 2000, 3000)
	verifyRead(t, expectFile, prs, 500)
}

func verifyReadSeek(t *testing.T, expectReadSeeker, readSeeker io.ReadSeeker,
	len, expectOffset, offset, whence int) {
	_, err := expectReadSeeker.Seek(int64(expectOffset), whence)
	require.Nil(t, err)

	_, err = readSeeker.Seek(int64(offset), whence)
	require.Nil(t, err)

	verifyRead(t, expectReadSeeker, readSeeker, len)
}

func TestFilePartReadSeekerReadSeek(t *testing.T) {
	expectFile, err := os.Open(testFilename)
	require.Nil(t, err)
	defer expectFile.Close()

	stat, err := expectFile.Stat()
	require.Nil(t, err)
	size := int(stat.Size())

	fpart, err := os.Open(testFilename)
	require.Nil(t, err)
	defer fpart.Close()

	prs := NewFilePartReadSeeker(fpart, 0, 1000)

	verifyReadSeek(t, expectFile, prs, 500, 500, 500, io.SeekStart)
	verifyReadSeek(t, expectFile, prs, 500, -500, -500, io.SeekCurrent)

	// read from end
	seekOffset := 10
	readerSeekLen := -1 * (1000 - seekOffset)
	actualFileSeekLen := -1 * (size - seekOffset)
	verifyReadSeek(t, expectFile, prs, 100, actualFileSeekLen, readerSeekLen, io.SeekEnd)

	seekOffset = 800
	readerSeekLen = -1 * (1000 - seekOffset)
	actualFileSeekLen = -1 * (size - seekOffset)
	verifyReadSeek(t, expectFile, prs, 100, actualFileSeekLen, readerSeekLen, io.SeekEnd)

	// now read until end
	verifyReadSeek(t, expectFile, prs, 101, -1, -1, io.SeekCurrent)

	// next read should be EOF
	buf := make([]byte, 2)
	_, err = prs.Read(buf)
	require.EqualValues(t, io.EOF, err)

	prs = NewFilePartReadSeeker(fpart, 1000, 2000)
	verifyReadSeek(t, expectFile, prs, 500, 1500, 500, io.SeekStart)
	verifyReadSeek(t, expectFile, prs, 400, -500, -500, io.SeekCurrent)

	// read from end
	seekOffset = 10
	readerSeekLen = -1 * (1000 - seekOffset)
	actualFileSeekLen = -1 * (size - 1000 - seekOffset)
	verifyReadSeek(t, expectFile, prs, 100, actualFileSeekLen, readerSeekLen, io.SeekEnd)

	seekOffset = 800
	readerSeekLen = -1 * (1000 - seekOffset)
	actualFileSeekLen = -1 * (size - 1000 - seekOffset)
	verifyReadSeek(t, expectFile, prs, 100, actualFileSeekLen, readerSeekLen, io.SeekEnd)

	// now read until end
	verifyReadSeek(t, expectFile, prs, 101, -1, -1, io.SeekCurrent)

	// next read should be EOF
	_, err = prs.Read(buf)
	require.EqualValues(t, io.EOF, err)
}
