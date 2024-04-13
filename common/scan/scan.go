package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// Directory is to scan given root directory, and build DB tree
func Directory(rootDir string, cb func(string, os.FileInfo) error) error {
	var wg sync.WaitGroup

	// Launch a goroutine for each directory
	err := filepath.WalkDir(rootDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			//wg.Add(1)
			//go scanDirectory(path)
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = cb(path, info)
			if err != nil {
				logrus.Warnf("Error handling file %s: %s", path, err)
			}
		}()

		return nil
	})
	if err != nil {
		return err
	}

	// Wait for all goroutines to finish
	wg.Wait()

	return nil
}
