package main

import (
	"errors"
	"fmt"
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

	defaultBucket = "lomorage"
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
					Usage: "List of ignored files, separated by comma",
					Value: ".DS_Store,._.DS_Store,Thumbs.db",
				},
				cli.StringFlag{
					Name:  "ignore-dirs, in",
					Usage: "List of ignored directories, separated by comma",
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
							Value: defaultBucket,
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
						cli.BoolFlag{
							Name:  "no-encrypt",
							Usage: "not do any encryption, and upload raw files",
						},
						cli.BoolFlag{
							Name:  "force",
							Usage: "force to upload from scratch and not reuse previous upload info",
						},
						cli.StringFlag{
							Name:   "encrypt-key, k",
							Usage:  "Master key to encrypt current upload file",
							EnvVar: "LOMOB_MASTER_KEY",
						},
						cli.StringFlag{
							Name:  "storage-class",
							Usage: "The  type  of storage to use for the object. Valid choices are: DEEP_ARCHIVE | GLACIER | GLACIER_IR | INTELLIGENT_TIERING | ONE-ZONE_IA | REDUCED_REDUNDANCY | STANDARD | STANDARD_IA.",
							Value: "STANDARD",
						},
					},
				},
			},
		},
		{
			Name:  "upload",
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
							Value: defaultBucket,
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
						cli.BoolFlag{
							Name:  "no-encrypt",
							Usage: "not do any encryption, and upload raw files",
						},
						cli.BoolFlag{
							Name:  "force",
							Usage: "force to upload from scratch and not reuse previous upload info",
						},
						cli.StringFlag{
							Name:   "encrypt-key, k",
							Usage:  "Master key to encrypt current upload file",
							EnvVar: "LOMOB_MASTER_KEY",
						},
						cli.StringFlag{
							Name:  "storage-class",
							Usage: "The  type  of storage to use for the object. Valid choices are: DEEP_ARCHIVE | GLACIER | GLACIER_IR | INTELLIGENT_TIERING | ONE-ZONE_IA | REDUCED_REDUNDANCY | STANDARD | STANDARD_IA.",
							Value: "STANDARD",
						},
					},
				},
				{
					Name:   "files",
					Action: uploadFilesToGdrive,
					Usage:  "Upload individual files not in ISO to google drive",
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
							Name:  "folder",
							Usage: "Folders to list",
							Value: defaultBucket,
						},
						cli.StringFlag{
							Name:   "encrypt-key, k",
							Usage:  "Master key to encrypt current upload file",
							EnvVar: "LOMOB_MASTER_KEY",
						},
					},
				},
			},
		},
		{
			Name:  "restore",
			Usage: "Restore encrypted files cloud",
			Subcommands: cli.Commands{
				{
					Name:      "aws",
					Action:    restoreAwsFile,
					Usage:     "Restore ISO files in AWS drive",
					ArgsUsage: "[iso file name] [output file name]",
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
							Value: defaultBucket,
						},
						cli.StringFlag{
							Name:   "encrypt-key, k",
							Usage:  "Master key to encrypt current upload file",
							EnvVar: "LOMOB_MASTER_KEY",
						},
					},
				},
				{
					Name:      "gdrive",
					Action:    restoreGdriveFile,
					Usage:     "Restore files in google drive",
					ArgsUsage: "[encrypted file name in fullpath] [output file name]",
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
							Name:   "encrypt-key, k",
							Usage:  "Master key to encrypt current upload file",
							EnvVar: "LOMOB_MASTER_KEY",
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
						cli.StringFlag{
							Name:  "folder",
							Usage: "Folders to list",
							Value: defaultBucket,
						},
					},
				},
				{
					Name:   "files",
					Action: listFilesNotInIso,
					Usage:  "List all files not packed in ISO including the ones uploaded in google drive",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "no-cloud",
							Usage: "List all files not in google drive or packed in ISO",
						},
					},
				},
				{
					Name:   "isos",
					Action: listISO,
					Usage:  "List all created iso files",
				},
			},
		},
		{
			Name:  "util",
			Usage: "Various tools",
			Subcommands: cli.Commands{
				{
					Name:      "encrypt",
					Action:    encryptCmd,
					Usage:     "Encrypt local file",
					ArgsUsage: "Usage: [input filename] [[output filename]]. If output filename is not given, it will be <intput filename>.enc",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:   "encrypt-key, k",
							Usage:  "Master key to encrypt current upload file",
							EnvVar: "LOMOB_MASTER_KEY",
						},
					},
				},
				{
					Name:      "decrypt",
					Action:    decryptLocalFile,
					Usage:     "Encrypt local file",
					ArgsUsage: "[filename]",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:   "encrypt-key, k",
							Usage:  "Master key to encrypt current upload file",
							EnvVar: "LOMOB_MASTER_KEY",
						},
						cli.StringFlag{
							Name:  "output, o",
							Usage: "Saved file name",
						},
					},
				},
				{
					Name:      "parts",
					Action:    calculatePartHash,
					Usage:     "Calculate given files base64 hash without encryption",
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
							Value: defaultBucket,
						},
					},
				},
				{
					Name:      "abort-upload",
					Action:    abortUpload,
					Usage:     "Abort in progress upload. If upload ID is not provided, it will delete all upload for given key",
					ArgsUsage: "[upload key] [[upload ID]]",
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
							Value: defaultBucket,
						},
					},
				},
				{
					Name:      "upload-s3",
					Action:    uploadFilesToS3,
					Usage:     "Upload individual file into S3 with on-the-fly encryption",
					ArgsUsage: "[local file name]",
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
							Value: defaultBucket,
						},
						cli.BoolFlag{
							Name:  "no-encrypt",
							Usage: "not do any encryption, and upload raw files",
						},
						cli.StringFlag{
							Name:   "encrypt-key, k",
							Usage:  "Master key to encrypt current upload file. If it is empty, means no encryption is needed",
							EnvVar: "LOMOB_MASTER_KEY",
						},
						cli.StringFlag{
							Name:  "storage-class",
							Usage: "The  type  of storage to use for the object. Valid choices are: DEEP_ARCHIVE | GLACIER | GLACIER_IR | INTELLIGENT_TIERING | ONE-ZONE_IA | REDUCED_REDUNDANCY | STANDARD | STANDARD_IA.",
							Value: "STANDARD",
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
				{
					Name:   "gcloud-auth-refresh",
					Action: gcloudTokenRefresh,
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
				{
					Name:      "check-header",
					Action:    checkHeader,
					Usage:     "Check if the header of encrypt file follows the convention",
					ArgsUsage: "[original file name] [encrypt file name]",
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

func getAWSStorageClass(ctx *cli.Context) (string, error) {
	c := ctx.String("storage-class")
	switch c {
	case "DEEP_ARCHIVE":
		fallthrough
	case "GLACIER":
		fallthrough
	case "GLACIER_IR":
		fallthrough
	case "REDUCED_REDUNDANCY":
		fallthrough
	case "INTELLIGENT_TIERING":
		fallthrough
	case "ONEZONE_IA":
		fallthrough
	case "STANDARD":
		fallthrough
	case "STANDARD_IA":
		return c, nil
	}
	return "", fmt.Errorf("Invalid storage class: %s", c)
}
