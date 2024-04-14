package main

import (
	"errors"
	"os"
	"sync"

	"github.com/lomorage/lomo-backup/common/dbx"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	scanUsage  = "[directory to scan]"
	mkisoUsage = "[local directory to store isos]"
	db         *dbx.DB

	lock *sync.Mutex
)

func main() {
	app := cli.NewApp()

	app.Usage = "Backup files to remote storage with 2 stage approach"
	app.Email = "support@lomorage.com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "db",
			Usage: "Filename of DB",
			Value: "lomob.db",
		},
		cli.IntFlag{
			Name:  "log-level, l",
			Usage: "Log level for processing. 0: Panic, 1: Fatal, 2: Error, 3: Warn, 4: Info, 5: Debug, 6: TraceLevel",
			Value: int(logrus.InfoLevel),
		},
	}
	app.Commands = []cli.Command{
		{
			Name:      "scan",
			Action:    scanDir,
			Usage:     "Scan all files under given directory",
			ArgsUsage: scanUsage,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "ignore-files, if",
					Usage: "List of ignored files, seperated by comman",
					Value: ".DS_Store,._.DS_Store,Thumbs.db",
				},
				cli.StringFlag{
					Name:  "ignore-dirs, in",
					Usage: "List of ignored directories, seperated by comman",
					Value: ".idea,.git,.github",
				},
				cli.IntFlag{
					Name:  "threads, t",
					Usage: "Number of scan threads in parallel",
					Value: 20,
				},
			},
		},
		{
			Name:  "iso",
			Usage: "ISO related commands",
			Subcommands: cli.Commands{
				{
					Name:      "create",
					Action:    mkISO,
					Usage:     "Group scanned files and make iso",
					ArgsUsage: mkisoUsage,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "iso-size,s",
							Usage: "Size of each ISO file",
							Value: "1GB",
						},
					},
				},
				{
					Name:   "list",
					Action: listISO,
					Usage:  "List all files created files",
				},
			},
		},
		{
			Name:  "list",
			Usage: "List scanned files related commands",
			Subcommands: cli.Commands{
				{
					Name:      "bigfiles",
					Action:    listBigfiles,
					Usage:     "List big files",
					ArgsUsage: mkisoUsage,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "file-size,s",
							Usage: "Minimum file size in the list result",
							Value: "50MB",
						},
					},
				},
				{
					Name:   "dirs",
					Action: listScanedDirs,
					Usage:  "List all scanned directories",
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Errorf(err.Error())
	}
}

func initLogLevel(level int) error {
	if level < int(logrus.PanicLevel) || level > int(logrus.TraceLevel) {
		return errors.New("unrecognized log level")
	}

	logrus.SetLevel(logrus.Level(level))
	return nil
}

func initDB(dbname string) (err error) {
	db, err = dbx.OpenDB(dbname)
	return err
}
