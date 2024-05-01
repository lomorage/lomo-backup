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
	scanUsage = "[directory to scan]"
	db        *dbx.DB

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
					ArgsUsage: "[iso filename. if empty, filename will be <oldest file name>--<latest filename>.iso]",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "iso-size,s",
							Usage: "Size of each ISO file. KB=1000 Byte",
							Value: "5G",
						},
						cli.StringFlag{
							Name:  "store-dir,p",
							Usage: "Directory to store the ISOs. It's urrent directory by default",
						},
					},
				},
				{
					Name:   "list",
					Action: listISO,
					Usage:  "List all created iso files",
				},
				{
					Name:      "dump",
					Action:    dumpISO,
					Usage:     "Dump and print all files/directories in given ISO in tree",
					ArgsUsage: "[iso filename]",
				},
				{
					Name:   "upload",
					Action: uploadISOs,
					Usage:  "Upload specified or all iso files",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:   "awsAccessKeyID",
							Usage:  "aws Access Key ID",
							EnvVar: "AWS_ACCESS_KEY_ID",
						},
						cli.StringFlag{
							Name:   "awsSecretAccessKey",
							Usage:  "aws Secret Access Key",
							EnvVar: "AWS_SECRET_ACCESS_KEY",
						},
						cli.StringFlag{
							Name:   "awsBucketRegion",
							Usage:  "aws Bucket Region",
							EnvVar: "AWS_DEFAULT_REGION",
						},
						cli.StringFlag{
							Name:  "awsBucketName",
							Usage: "awsBucketName",
							Value: "lomorage",
						},
						cli.StringFlag{
							Name:  "part-size,p",
							Usage: "Size of each upload partition. KB=1000 Byte",
							Value: "6M",
						},
						cli.IntFlag{
							Name:  "nthreads,n",
							Usage: "Number of parallel multi part upload",
							Value: 3,
						},
						cli.BoolFlag{
							Name:  "save-parts,s",
							Usage: "Save multiparts locally for debug",
						},
					},
				},
			},
		},
		{
			Name:  "Upload",
			Usage: "Upload packed ISO files or individual files",
			Subcommands: cli.Commands{
				{
					Name:   "iso",
					Action: uploadISOs,
					Usage:  "Upload specified or all iso files",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:   "awsAccessKeyID",
							Usage:  "aws Access Key ID",
							EnvVar: "AWS_ACCESS_KEY_ID",
						},
						cli.StringFlag{
							Name:   "awsSecretAccessKey",
							Usage:  "aws Secret Access Key",
							EnvVar: "AWS_SECRET_ACCESS_KEY",
						},
						cli.StringFlag{
							Name:   "awsBucketRegion",
							Usage:  "aws Bucket Region",
							EnvVar: "AWS_DEFAULT_REGION",
						},
						cli.StringFlag{
							Name:  "awsBucketName",
							Usage: "awsBucketName",
							Value: "lomorage",
						},
						cli.StringFlag{
							Name:  "part-size,p",
							Usage: "Size of each upload partition. KB=1000 Byte",
							Value: "6M",
						},
						cli.IntFlag{
							Name:  "nthreads,n",
							Usage: "Number of parallel multi part upload",
							Value: 3,
						},
						cli.BoolFlag{
							Name:  "save-parts,s",
							Usage: "Save multiparts locally for debug",
						},
					},
				},
				{
					Name:   "files",
					Action: uploadFiles,
					Usage:  "Upload individual files",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "table-view, t",
							Usage: "List all directories in table",
						},
					},
				},
			},
		},
		{
			Name:  "list",
			Usage: "List scanned files related commands",
			Subcommands: cli.Commands{
				{
					Name:   "bigfiles",
					Action: listBigfiles,
					Usage:  "List big files",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "file-size,s",
							Usage: "Minimum file size in the list result. KB=1000 Byte",
							Value: "50M",
						},
					},
				},
				{
					Name:   "dirs",
					Action: listScanedDirs,
					Usage:  "List all scanned directories",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "table-view, t",
							Usage: "List all directories in table",
						},
					},
				},
				{
					Name:      "parts",
					Action:    calculatePartHash,
					Usage:     "Calculate given files base64 hash",
					ArgsUsage: "[filename]",
				},
				{
					Name:   "gdrive",
					Action: listFilesInGDrive,
					Usage:  "List files in google drive",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "cred",
							Usage: "Google cloud oauth credential json file",
							Value: "gdrive-credentials.json",
						},
						cli.StringFlag{
							Name:  "token",
							Usage: "Token file to access google cloud",
							Value: "gdrive-token.json",
						},
					},
				},
			},
		},
		{
			Name:  "util",
			Usage: "Various tools",
			Subcommands: cli.Commands{
				{
					Name:      "parts",
					Action:    calculatePartHash,
					Usage:     "Calculate given files base64 hash",
					ArgsUsage: "[filename]",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "part-size,p",
							Usage: "Size of each upload partition. KB=1000 Byte",
							Value: "6M",
						},
						cli.IntFlag{
							Name:  "part-number,pn",
							Usage: "The number of part to calculate",
						},
					},
				},
				{
					Name:   "list-inprogress-upload",
					Action: listUploadingItems,
					Usage:  "List uploading tasks in AWS",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "local, l",
							Usage: "List all uploads stored in local DB only",
						},
						cli.StringFlag{
							Name:   "awsAccessKeyID",
							Usage:  "aws Access Key ID",
							EnvVar: "AWS_ACCESS_KEY_ID",
						},
						cli.StringFlag{
							Name:   "awsSecretAccessKey",
							Usage:  "aws Secret Access Key",
							EnvVar: "AWS_SECRET_ACCESS_KEY",
						},
						cli.StringFlag{
							Name:   "awsBucketRegion",
							Usage:  "aws Bucket Region",
							EnvVar: "AWS_DEFAULT_REGION",
						},
						cli.StringFlag{
							Name:  "awsBucketName",
							Usage: "awsBucketName",
							Value: "lomorage",
						},
					},
				},
				{
					Name:      "abort-upload",
					Action:    abortUpload,
					Usage:     "Abort in progress upload",
					ArgsUsage: "[upload key] [upload ID]",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:   "awsAccessKeyID",
							Usage:  "aws Access Key ID",
							EnvVar: "AWS_ACCESS_KEY_ID",
						},
						cli.StringFlag{
							Name:   "awsSecretAccessKey",
							Usage:  "aws Secret Access Key",
							EnvVar: "AWS_SECRET_ACCESS_KEY",
						},
						cli.StringFlag{
							Name:   "awsBucketRegion",
							Usage:  "aws Bucket Region",
							EnvVar: "AWS_DEFAULT_REGION",
						},
						cli.StringFlag{
							Name:  "awsBucketName",
							Usage: "awsBucketName",
							Value: "lomorage",
						},
					},
				},
				{
					Name:   "gcloud-auth",
					Action: gcloudAuth,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "cred",
							Usage: "Google cloud oauth credential json file",
							Value: "gdrive-credentials.json",
						},
						cli.StringFlag{
							Name:  "token",
							Usage: "Token file to access google cloud",
							Value: "gdrive-token.json",
						},
						cli.StringFlag{
							Name:  "redirect-path",
							Usage: "Redirect path defined in credentials.json",
							Value: "/",
						},
						cli.IntFlag{
							Name:  "redirect-port",
							Usage: "Redirect port defined in credentials.json",
							Value: 80,
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Errorf(err.Error())
		return
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
