package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lomorage/lomo-backup/common/dbx"
	"github.com/lomorage/lomo-backup/common/scan"
	"github.com/lomorage/lomo-backup/common/types"
	"github.com/urfave/cli"
)

var (
	scanRootDir   string
	scanRootDirID int
	dirs          map[string]int
)

func scanDir(ctx *cli.Context) (err error) {
	if len(ctx.Args()) != 1 {
		return errors.New("usage: lomob " + usage)
	}
	scanRootDir, err = filepath.Abs(ctx.Args()[0])
	if err != nil {
		return err
	}

	err = initDB(ctx.GlobalString("db"))
	if err != nil {
		return err
	}

	err = selectOrInsertScanRootDir()
	if err != nil {
		return err
	}

	dirs = make(map[string]int)
	lock = &sync.Mutex{}

	return scan.Directory(scanRootDir, handleScan)
}

func selectOrInsertScanRootDir() error {
	id, err := db.GetDirIDByPathAndRootID(scanRootDir, dbx.SuperScanRootDirID)
	if err != nil {
		return err
	}
	if id != nil {
		scanRootDirID = *id
		return nil
	}
	scanRootDirID, err = db.InsertDir(scanRootDir, dbx.SuperScanRootDirID)
	return err
}

func selectOrInsertDir(dir string) (dirID int, err error) {
	// check dir is inserted or not before
	var ok bool

	lock.Lock()
	defer lock.Unlock()

	dirID, ok = dirs[dir]
	if ok {
		return
	}
	id, err := db.GetDirIDByPathAndRootID(dir, scanRootDirID)
	if err != nil {
		return
	}
	if id != nil {
		dirID = *id
	} else {
		dirID, err = db.InsertDir(dir, scanRootDirID)
		if err != nil {
			return
		}
	}
	dirs[dir] = dirID

	return
}

func selectOrInsertFile(dirID int, path string, info os.FileInfo) error {
	fi, err := db.GetFileByNameAndDirID(info.Name(), dirID)
	if err != nil {
		return err
	}
	if fi != nil {
		// skip as already in db
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	fi = &types.FileInfo{
		DirID:   dirID,
		Name:    info.Name(),
		Size:    int(info.Size()),
		ModTime: info.ModTime(),
		Hash:    fmt.Sprintf("%x", h.Sum(nil)),
	}

	_, err = db.InsertFile(fi)
	return err
}

func handleScan(path string, info os.FileInfo) (err error) {
	fmt.Printf("path %s: file %s\n", path, info.Name())

	dir := strings.TrimSuffix(path, string(filepath.Separator)+info.Name())
	dir = strings.TrimPrefix(dir, scanRootDir+string(filepath.Separator))

	dirID, err := selectOrInsertDir(dir)
	if err != nil {
		return err
	}

	return selectOrInsertFile(dirID, path, info)
}
