package hash

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	lomoio "github.com/lomorage/lomo-backup/common/io"
)

func CalculateHashHex(hash []byte) string {
	return fmt.Sprintf("%x", hash)
}

func CalculateHashBase64(hash []byte) string {
	return base64.StdEncoding.EncodeToString(hash)
}

func DecpdeHashBase64(hash string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(hash)
}

func CalculateHashBytes(buffer []byte) []byte {
	h := sha256.New()
	h.Write(buffer)
	return h.Sum(nil)
}

func CalculateHashFile(path string) ([]byte, error) {
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
	var remaining = info.Size()
	for curr = 0; remaining != 0; curr += partLength {
		if remaining < int64(partSize) {
			partLength = remaining
		} else {
			partLength = int64(partSize)
		}
		prs := lomoio.NewFilePartReadSeeker(f, curr, curr+partLength)
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
