package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lomorage/lomo-backup/common/crypto"
	"github.com/lomorage/lomo-backup/common/gcloud"
	"github.com/lomorage/lomo-backup/common/hash"
	"github.com/lomorage/lomo-backup/common/types"
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

	masterKey := ctx.String("encrypt-key")
	if masterKey == "" {
		masterKey, err = getMasterKey()
		if err != nil {
			return err
		}
	}

	client, err := gcloud.CreateDriveClient(&gcloud.Config{
		CredFilename:  ctx.String("cred"),
		TokenFilename: ctx.String("token"),
		RefreshToken:  true,
	})
	if err != nil {
		return fmt.Errorf("unable to retrieve Drive client: %v", err)
	}

	uploadRootFolder := ctx.String("folder")
	exist, uploadRootFolderID, err := client.GetAndCreateFileIDIfNotExist(uploadRootFolder, "", nil, time.Now())
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

	fileInfos, err := db.ListFilesNotInISOAndCloud()
	if err != nil {
		return err
	}

	if len(fileInfos) == 0 {
		fmt.Println("No files need to be uploaded to google drive")
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
				stat.ModTime())
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
				ok, info.folderID, err = client.GetAndCreateFileIDIfNotExist(p, parentID, nil, stat.ModTime())
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
		file, err := os.Open(fullLocalPath)
		if err != nil {
			return err
		}

		stat, err := file.Stat()
		if err != nil {
			return err
		}

		encryptKey, iv, err := genEncryptKeyAndSalt([]byte(masterKey))
		if err != nil {
			return err
		}

		encryptor, err := crypto.NewEncryptor(file, encryptKey, iv)
		if err != nil {
			return err
		}

		logrus.Infof("Uploading: %s into %s (%s):%s\n", fullLocalPath, folderKey, parentID, filename)

		fileID, err := client.CreateFile(filename, parentID, encryptor, stat.ModTime())
		if err != nil {
			return err
		}
		logrus.Infof("Uploading success")
		err = file.Close()
		if err != nil {
			logrus.Warnf("Close %s: %s", fullLocalPath, err)
		}

		hashEnc := hash.CalculateHashHex(encryptor.GetHash())
		err = db.UpdateFileIsoIDAndEncHash(types.IsoIDCloud, f.ID, hashEnc)
		if err != nil {
			return err
		}

		// add encrypt hash as part of the file's metadata
		err = client.UpdateFileMetadata(fileID, map[string]string{
			types.MetadataKeyHashOrig:    f.Hash,
			types.MetadataKeyHashEncrypt: hashEnc,
		})
		if err != nil {
			return err
		}
	}

	fmt.Printf("%d files are uploaded to google drive\n", len(fileInfos))

	return nil
}

func flattenScanRootDir(dir string) string {
	return strings.ReplaceAll(dir, string(os.PathSeparator), "_")
}
