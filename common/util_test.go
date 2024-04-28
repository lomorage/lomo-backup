package common

import (
	"encoding/base64"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFilename = "./testdata/indepedant_declaration.txt"

// hex hash is calculated by "sha256sum indepedant_declaration.txt"
const expectHexHash = "4cfd75f4c8aa09ff1992e493999fd1c0333da4393fffd9d207b60c6dc2516263"

// base64 hash is calculated by "cat indepedant_declaration.txt | openssl dgst -binary -sha256 | openssl base64 -A"
const expectBase64Hash = "TP119MiqCf8ZkuSTmZ/RwDM9pDk//9nSB7YMbcJRYmM="

// split file using command "split --bytes=1000 --numeric-suffixes=1 --suffix-length=1 ./indepedant_declaration.txt test"
// for i in `seq 1 9`; do sha256sum test$i; done | awk -F' ' '{print $1}'
var expectHexHashMultiparts = []string{
	"966b6a905a846ca688fd1f9e08882c43ec52bff8852f69139d5c720b7361d30f",
	"18e05d53abcd5c1267a762347bb6bf0ac5948524932cd4c544914372832547e1",
	"648d21f3ba5b237581f691d68e2f596dd36bfc687bbf41cca4e2c8f744e5e309",
	"fd3f7646b387a922b7a9912200c703381dee8f41ba2fb56f6d9f1e4c6c9304eb",
	"19183c6d710f85e3c0d36e5c1fbf0e1fb42b832a1471a745c11aa8d9fed85f79",
	"5d350af5e1f3e149e1f6cefe595e18a71951b306f00ef304177b7999745f4c3b",
	"42664de22a93b15140c8e882f7d5deb9a5fb29e937e6c0bef81fedea50a250f9",
	"a5747c2ec4cedc33709a361318ddf658a5afa7a12c889e7315bf0adaa82c0086",
	"eb05dc46b7f61ed556a96ee44fd68e5ebe8012926901c56c5578b7f363134360",
}

// for i in `seq 1 9`; do cat test$i | openssl dgst -binary -sha256 | openssl base64 -A; echo;done
var expectBase64HashMultiparts = []string{
	"lmtqkFqEbKaI/R+eCIgsQ+xSv/iFL2kTnVxyC3Nh0w8=",
	"GOBdU6vNXBJnp2I0e7a/CsWUhSSTLNTFRJFDcoMlR+E=",
	"ZI0h87pbI3WB9pHWji9ZbdNr/Gh7v0HMpOLI90Tl4wk=",
	"/T92RrOHqSK3qZEiAMcDOB3uj0G6L7VvbZ8eTGyTBOs=",
	"GRg8bXEPhePA025cH78OH7QrgyoUcadFwRqo2f7YX3k=",
	"XTUK9eHz4Unh9s7+WV4YpxlRswbwDvMEF3t5mXRfTDs=",
	"QmZN4iqTsVFAyOiC99XeuaX7Kek35sC++B/t6lCiUPk=",
	"pXR8LsTO3DNwmjYTGN32WKWvp6EsiJ5zFb8K2qgsAIY=",
	"6wXcRrf2HtVWqW7kT9aOXr6AEpJpAcVsVXi382MTQ2A=",
}

func TestCalculateHash(t *testing.T) {
	assert := assert.New(t)

	hash, err := CalculateHash(testFilename)

	// assert for nil (good for errors)
	assert.Nil(err)

	// assert equality
	// calculated by command "sha256sum indepedant_declaration.txt"
	assert.Equal(expectHexHash, CalculateHashHex(hash), "they should be equal")

	// calculated by command "cat indepedant_declaration.txt | openssl dgst -binary -sha256 | openssl base64 -A"
	assert.Equal(expectBase64Hash, CalculateHashBase64(hash), "they should be equal")
}

func TestCalculateMultiPartsHash(t *testing.T) {
	assert := assert.New(t)

	partsHash, err := CalculateMultiPartsHash(testFilename, 1000)

	// assert for nil (good for errors)
	assert.Nil(err)

	// assert equality
	hexParts := make([]string, len(partsHash))
	base64Parts := make([]string, len(partsHash))
	for i, p := range partsHash {
		hexParts[i] = CalculateHashHex(p)
		base64Parts[i] = CalculateHashBase64(p)
	}
	require.Equal(t, expectHexHashMultiparts, hexParts)

	require.Equal(t, expectBase64HashMultiparts, base64Parts)
}

func verifySeek(t *testing.T, seeker io.Seeker, offset, expectPosition int64,
	whence int) {
	n, err := seeker.Seek(offset, whence)
	require.Nil(t, err)
	require.Equal(t, expectPosition, n)
}

func testFilePartReadSeekerSeek(t *testing.T, prs *FilePartReadSeeker) {
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

	// this is normal steps to get file length
	// 1. seek 0 at current, and check current offset
	// 2. seek 0 from start, and ensure from the beginning
	// 3. seek 0 from current, and check if it is also from the beginning
	// 4. seek 0 from end, and get file length
	// 5. seek 0 from current, and check if it is same size as step 4
	// 6. seek 0 from start, and ensure from the beginning
	// 7. seek 0 from current, and check if it is also from the beginning
	testFilePartReadSeekerSeek(t, prs)

	// create another seeker in the middle, but reuse current position
	prs = NewFilePartReadSeeker(fpart, 1000, 2000)
	testFilePartReadSeekerSeek(t, prs)

	// create the 3rd seeker in the middle, and seek original file to the beginning
	// above steps should have same result
	n, err = fpart.Seek(0, io.SeekStart)
	require.Nil(t, err)
	require.Equal(t, int64(0), n)

	prs = NewFilePartReadSeeker(fpart, 2000, 3000)
	testFilePartReadSeekerSeek(t, prs)
}

func verifyRead(t *testing.T, expectReader, reader io.Reader, len int) {
	expectBuffer := make([]byte, len)
	expectSize, err := expectReader.Read(expectBuffer)
	require.Nil(t, err)

	buffer := make([]byte, len)
	size, err := reader.Read(buffer)
	require.Nil(t, err)

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

	fpart, err := os.Open(testFilename)
	require.Nil(t, err)
	defer fpart.Close()

	prs := NewFilePartReadSeeker(fpart, 0, 1000)

	verifyReadSeek(t, expectFile, prs, 500, 500, 500, io.SeekStart)
	verifyReadSeek(t, expectFile, prs, 500, -500, -500, io.SeekCurrent)

	prs = NewFilePartReadSeeker(fpart, 1000, 2000)
	verifyReadSeek(t, expectFile, prs, 500, 1500, 500, io.SeekStart)
	verifyReadSeek(t, expectFile, prs, 500, -500, -500, io.SeekCurrent)
}

func TestConcatAndCalculateBase64Hash(t *testing.T) {
	parts := [][]byte{}
	for _, p := range []string{
		"lzeb6gPr4raiM1LG0ZNF2OOtdoUCRu+6ewNA0Qir4sI=",
		"lZh9FyuGrF/0Vbw8CBtSFVMX04SgRnLbrPX9BYpQRNg=",
	} {
		b, err := base64.StdEncoding.DecodeString(p)
		require.Nil(t, err)
		parts = append(parts, b)
	}

	hash, err := ConcatAndCalculateBase64Hash(parts)
	require.Nil(t, err)
	require.Equal(t, "NnO4DPqD+RLUyOycER1BKbzMv6+APV72KGFvLBNay8c=", hash)
}
