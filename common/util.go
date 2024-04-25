package common

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"time"
)

func CalculateHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func FormatTimeDateOnly(t time.Time) string {
	return t.Format("2006-01-02")
}
