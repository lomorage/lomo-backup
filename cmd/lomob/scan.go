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
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	scanRootDir   string
	scanRootDirID int
	dirs          map[string]int
)

func scanDir(ctx *cli.Context) (err error) {
	if len(ctx.Args()) != 1 {
		return errors.New("usage: lomob " + scanUsage)
	}
	scanRootDir, err = filepath.Abs(ctx.Args()[0])
	if err != nil {
		return err
	}

	nthreads := ctx.Int("threads")

	err = initLogLevel(ctx.GlobalInt("log-level"))
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

	ignoreFiles := make(map[string]struct{})

	for _, ignore := range strings.Split(ctx.String("ignore-files"), ",") {
		ignoreFiles[ignore] = struct{}{}
	}

	ignoreDirs := make(map[string]struct{})

	for _, ignore := range strings.Split(ctx.String("ignore-dirs"), ",") {
		ignoreDirs[ignore] = struct{}{}
	}

	dirs = make(map[string]int)
	lock = &sync.Mutex{}

	threads := make(chan scan.FileCallback, nthreads)
	go func() {
		for {
			cb := <-threads
			err = handleScan(cb.Path, cb.Info)
			if err != nil {
				logrus.Warnf("Error handling file %s: %s", cb.Path, err)
			}
		}
	}()

	return scan.Directory(scanRootDir, ignoreFiles, ignoreDirs, threads)
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
	dir := strings.TrimSuffix(path, info.Name())
	dir = strings.TrimPrefix(dir, scanRootDir)
	dir = strings.Trim(dir, string(filepath.Separator))

	logrus.Debugf("Start scan %s", path)
	defer logrus.Debugf("Finish scan %s", path)

	dirID, err := selectOrInsertDir(dir)
	if err != nil {
		return err
	}

	return selectOrInsertFile(dirID, path, info)
}
