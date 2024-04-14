package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

var errPermissionRegex = regexp.MustCompile(fs.ErrPermission.Error())

// FileCallback is to pass scan file info to caller
type FileCallback struct {
	Path string
	Info os.FileInfo
}

// Directory is to scan given root directory, and build DB tree
func Directory(root string, ignoreFiles, ignoreDirs map[string]struct{},
	ch chan FileCallback) error {
	var wg sync.WaitGroup

	// Launch a goroutine for each directory
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if errPermissionRegex.MatchString(err.Error()) {
				if entry.IsDir() {
					return fs.SkipDir
				}
				// skip this file only
				return nil
			}
			return err
		}
		if entry.IsDir() {
			_, ignore := ignoreDirs[entry.Name()]
			if ignore {
				return fs.SkipDir
			}
			return nil
		}

		_, ignore := ignoreFiles[entry.Name()]
		if ignore {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		// ignore symbol link file
		if info.Mode()&fs.ModeSymlink != 0 {
			return nil
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			ch <- FileCallback{Path: path, Info: info}
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
