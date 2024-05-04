package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lomorage/lomo-backup/common/gcloud"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func uploadFiles(ctx *cli.Context) error {
	err := initDB(ctx.GlobalString("db"))
	if err != nil {
		return err
	}

	err = initLogLevel(ctx.GlobalInt("log-level"))
	if err != nil {
		return err
	}

	client, err := gcloud.CreateClient(&gcloud.Config{
		CredFilename:  ctx.String("cred"),
		TokenFilename: ctx.String("token"),
	})
	if err != nil {
		return fmt.Errorf("unable to retrieve Drive client: %v", err)
	}
	if err != nil {
		return fmt.Errorf("unable to retrieve Drive client: %v", err)
	}

	uploadRootFolder := ctx.String("folder")
	exist, uploadRootFolderID, err := client.GetAndCreateFileIDIfNotExist(uploadRootFolder, "", nil,
		gcloud.FileMetadata{
			ModTime: time.Now(),
		})
	if err != nil {
		return err
	}
	if !exist {
		logrus.Infof("Root folder '%s' does not exist, created", uploadRootFolder)
	}

	scanRootDirs, err := db.ListScanRootDirs()
	if err != nil {
		return err
	}

	fileInfos, err := db.ListFilesNotInISO()
	if err != nil {
		return err
	}

	// root folder is
	type dirInfoInCloud struct {
		folderID       string
		parentFolderID string
	}
	existingDirsInCloud := map[string]dirInfoInCloud{
		"": {folderID: uploadRootFolderID},
	}
	for _, f := range fileInfos {
		scanRoot, ok := scanRootDirs[f.DirID]
		if !ok {
			return fmt.Errorf("unable to find scan root directory whose ID is %d", f.DirID)
		}

		origFolder := scanRoot

		// flatten scan root dir so as to have only one folder
		// find its folder ID in google cloud, and create one if not exist, then add into local map
		scanRootFolderInCloud := flattenScanRootDir(strings.Trim(scanRoot, string(os.PathSeparator)))
		info, ok := existingDirsInCloud[scanRootFolderInCloud]
		if !ok {
			stat, err := os.Stat(origFolder)
			if err != nil {
				return err
			}

			info = dirInfoInCloud{parentFolderID: uploadRootFolderID}
			ok, info.folderID, err = client.GetAndCreateFileIDIfNotExist(scanRootFolderInCloud, uploadRootFolderID, nil,
				gcloud.FileMetadata{ModTime: stat.ModTime()})
			if err != nil {
				return err
			}
			if !ok {
				logrus.Infof("Folder '%s' doesn not exist, created", scanRootFolderInCloud)
			}
			existingDirsInCloud[scanRootFolderInCloud] = info
		}
		scanRootFolderIDInCloud := info.folderID

		// check all directories' existence in cloud, and create if not exist
		dir, filename := filepath.Split(f.Name)
		dir = strings.Trim(dir, string(os.PathSeparator))
		folderKey := scanRootFolderInCloud
		parentID := scanRootFolderIDInCloud
		for _, p := range strings.Split(dir, string(os.PathSeparator)) {
			origFolder = filepath.Join(origFolder, p)
			folderKey += "/" + p
			info, ok = existingDirsInCloud[folderKey]
			if !ok {
				stat, err := os.Stat(origFolder)
				if err != nil {
					return err
				}
				info = dirInfoInCloud{parentFolderID: parentID}
				ok, info.folderID, err = client.GetAndCreateFileIDIfNotExist(p, parentID, nil,
					gcloud.FileMetadata{ModTime: stat.ModTime()})
				if err != nil {
					return err
				}
				if !ok {
					logrus.Infof("Folder '%s' does not exist, created with ID '%s'", folderKey, info.folderID)
				}
				existingDirsInCloud[folderKey] = info
			}
			parentID = info.folderID
		}

		fullLocalPath := filepath.Join(scanRoot, f.Name)

		// reuse folder ID if it is in map already
		reader, err := os.Open(fullLocalPath)
		if err != nil {
			return err
		}
		logrus.Infof("Uploading: %s into %s (%s):%s\n", fullLocalPath, folderKey, parentID, filename)
		stat, err := reader.Stat()
		if err != nil {
			return err
		}
		_, err = client.CreateFile(filename, parentID, reader, gcloud.FileMetadata{
			ModTime: stat.ModTime(),
			Hash:    f.Hash,
		})
		if err != nil {
			return err
		}
		logrus.Infof("Uploading success")
		err = reader.Close()
		if err != nil {
			logrus.Warnf("Close %s: %s", fullLocalPath, err)
		}
	}

	return nil
}

func flattenScanRootDir(dir string) string {
	return strings.ReplaceAll(dir, string(os.PathSeparator), "_")
}
