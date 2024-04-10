package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/lomorage/lomo-backup/common/dbx"
	"github.com/lomorage/lomo-backup/common/scan"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	usage       = "[directory to scan]"
	scanRootDir string
	db          *dbx.DB
	dirs        map[string]int
	lock        *sync.Mutex
)

func main() {
	app := cli.NewApp()

	app.Usage = "Backup files to remote storage with 2 stage approach"
	app.Email = "support@lomorage.com"
	app.Action = serve
	app.ArgsUsage = usage
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "iso-size,s",
			Usage: "Size of each ISO file",
			Value: 1000000000,
		},
		cli.StringFlag{
			Name:  "db",
			Usage: "Filename of DB",
			Value: "lomob.db",
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Errorf(err.Error())
	}
}

func serve(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		return errors.New("usage: lomob " + usage)
	}
	scanRootDir = ctx.Args()[0]

	var err error
	db, err = dbx.OpenDB(ctx.GlobalString("db"))
	if err != nil {
		return err
	}

	err = db.Prepare()
	if err != nil {
		return err
	}

	dirs = make(map[string]int)
	lock = &sync.Mutex{}

	return scan.Directory(scanRootDir, handleScan)
}

func handleScan(path string, info os.FileInfo) (err error) {
	fmt.Printf("path %s: file %s\n", path, info.Name())

	dir := strings.TrimSuffix(path, info.Name())
	// check dir is inserted or not before
	lock.Lock()
	dirID, ok := dirs[dir]
	if ok {
		goto unlock
	}
	dirID, err = db.GetDirIDByPath(dir)
	if err != nil {
		if !db.IsErrNoRow(err) {
			goto unlock
		}
		dirID, err = db.InsertDir(dir)
		if err != nil {
			goto unlock
		}
	}
	dirs[dir] = dirID

unlock:
	lock.Unlock()
	return
}
