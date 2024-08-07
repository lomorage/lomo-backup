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
	wg *sync.WaitGroup, ch chan FileCallback) error {
	processItem := func(path string, entry fs.DirEntry) error {
		info, err := entry.Info()
		if err != nil {
			return err
		}

		// ignore symbol link file
		if info.Mode()&fs.ModeSymlink != 0 {
			return nil
		}

		wg.Add(1)
		ch <- FileCallback{Path: path, Info: info}
		return nil
	}
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
			return processItem(path, entry)
		}

		_, ignore := ignoreFiles[entry.Name()]
		if ignore {
			return nil
		}

		return processItem(path, entry)
	})
	if err != nil {
		return err
	}

	return nil
}
