package scan

import (
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"sync"
)

// Directory is to scan given root directory, and build DB tree
func Directory(rootDir string, cb func(string, os.FileInfo) error) error {
	var wg sync.WaitGroup

	// Function to scan a directory recursively
	scanDirectory := func(directory string) {
		defer wg.Done()

		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			return cb(path, info)
		})
		if err != nil {
			logrus.Warnf("Error scanning directory %s: %s", directory, err)
		}
		return
	}

	// Launch a goroutine for each directory
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			wg.Add(1)
			go scanDirectory(path)
			return nil
		}
		return cb(path, info)
	})
	if err != nil {
		return err
	}

	// Wait for all goroutines to finish
	wg.Wait()

	return nil
}
