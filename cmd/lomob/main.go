package main

import (
	"os"
	"sync"

	"github.com/lomorage/lomo-backup/common/dbx"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	usage = "[directory to scan]"
	db    *dbx.DB

	lock *sync.Mutex
)

func main() {
	app := cli.NewApp()

	app.Usage = "Backup files to remote storage with 2 stage approach"
	app.Email = "support@lomorage.com"
	app.Action = scanDir
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

func initDB(dbname string) (err error) {
	db, err = dbx.OpenDB(dbname)
	return err
}
