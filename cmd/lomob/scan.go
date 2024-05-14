package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lomorage/lomo-backup/common/dbx"
	lomohash "github.com/lomorage/lomo-backup/common/hash"
	"github.com/lomorage/lomo-backup/common/scan"
	"github.com/lomorage/lomo-backup/common/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type scanDirInfo struct {
	id      int
	modTime *time.Time
}

var (
	scanRootDir   string
	scanRootDirID int
	dirs          map[string]scanDirInfo
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

	dirs = make(map[string]scanDirInfo)
	lock = &sync.Mutex{}

	ch := make(chan scan.FileCallback, nthreads)
	go func() {
		for {
			cb := <-ch
			err = handleScan(cb.Path, cb.Info)
			if err != nil {
				logrus.Warnf("Error handling file %s: %s", cb.Path, err)
			}
		}
	}()

	return scan.Directory(scanRootDir, ignoreFiles, ignoreDirs, ch)
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

	info, err := os.Stat(scanRootDir)
	if err != nil {
		return err
	}
	t := info.ModTime()
	scanRootDirID, err = db.InsertDir(scanRootDir, dbx.SuperScanRootDirID, &t)
	return err
}

func selectOrInsertDir(dir string, modTime *time.Time) (dirID int, err error) {
	// check dir is inserted or not before
	lock.Lock()
	defer lock.Unlock()

	info, ok := dirs[dir]
	if ok {
		if info.modTime != nil || modTime == nil {
			return info.id, nil
		}
		//fmt.Printf("update dir %s: %v\n", dir, modTime)
		err := db.UpdateDirModTime(info.id, *modTime)
		if err != nil {
			return 0, err
		}
		info.modTime = modTime
		dirs[dir] = info
		return info.id, nil
	}
	id, err := db.GetDirIDByPathAndRootID(dir, scanRootDirID)
	if err != nil {
		return
	}
	if id != nil {
		dirID = *id
	} else {
		//fmt.Printf("insert dir %s: %v\n", dir, modTime)
		dirID, err = db.InsertDir(dir, scanRootDirID, modTime)
		if err != nil {
			return
		}
	}
	dirs[dir] = scanDirInfo{id: dirID, modTime: modTime}
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

	hash, err := lomohash.CalculateHashFile(path)
	if err != nil {
		return err
	}

	fi = &types.FileInfo{
		DirID:   dirID,
		Name:    info.Name(),
		Size:    int(info.Size()),
		ModTime: info.ModTime(),
		Hash:    lomohash.CalculateHashHex(hash),
	}

	_, err = db.InsertFile(fi)
	return err
}

func handleScan(path string, info os.FileInfo) error {
	if info.IsDir() {
		dir := strings.TrimPrefix(path, scanRootDir)
		dir = strings.Trim(dir, string(filepath.Separator))
		//logrus.Infof("Start scan %s: %s", path, dir)
		t := info.ModTime()
		_, err := selectOrInsertDir(dir, &t)
		return err
	}

	dir := strings.TrimSuffix(path, info.Name())
	dir = strings.TrimPrefix(dir, scanRootDir)
	dir = strings.Trim(dir, string(filepath.Separator))

	logrus.Debugf("Start scan file %s", path)
	defer logrus.Debugf("Finish scan file %s", path)

	// not sure mod time,thus pass nill, and let other routine to update the data
	dirID, err := selectOrInsertDir(dir, nil)
	if err != nil {
		return err
	}

	return selectOrInsertFile(dirID, path, info)
}
