package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lomorage/lomo-backup/clients"
	"github.com/lomorage/lomo-backup/common/crypto"
	"github.com/lomorage/lomo-backup/common/gcloud"
	"github.com/lomorage/lomo-backup/common/hash"
	lomohash "github.com/lomorage/lomo-backup/common/hash"
	"github.com/lomorage/lomo-backup/common/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func uploadFilesToGdrive(ctx *cli.Context) error {
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

		encryptor, err := crypto.NewEncryptor(file, encryptKey, iv, true)
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

func uploadFilesToS3(ctx *cli.Context) error {
	accessKeyID := ctx.String("awsAccessKeyID")
	accessKey := ctx.String("awsSecretAccessKey")
	region := ctx.String("awsBucketRegion")
	bucket := ctx.String("awsBucketName")

	storageClass, err := getAWSStorageClass(ctx)
	if err != nil {
		return err
	}

	masterKey := ctx.String("encrypt-key")
	if ctx.Bool("no-encrypt") {
		masterKey = ""
	} else if masterKey == "" {
		masterKey, err = getMasterKey()
		if err != nil {
			return err
		}
	}

	cli, err := clients.NewAWSClient(accessKeyID, accessKey, region)
	if err != nil {
		return err
	}

	for _, name := range ctx.Args() {
		fmt.Printf("Uploading file %s\n", name)

		if masterKey == "" {
			err = uploadRawFileToS3(cli, bucket, storageClass, name, binContentType)
			if err != nil {
				return err
			}
			continue
		}
		tmpFilename, err := uploadEncryptFileToS3(cli, bucket, storageClass, name, masterKey)
		if err != nil {
			return err
		}
		err = os.Remove(tmpFilename)
		if err != nil {
			return err
		}
	}
	return nil
}

func uploadFileToS3(cli *clients.AWSClient, bucket, storageClass, remoteFilename, expectHash, contentType string, expectSize int,
	reader io.ReadSeeker) error {
	remoteInfo, err := cli.HeadObject(bucket, remoteFilename)
	if err != nil {
		return err
	}
	if remoteInfo != nil {
		fmt.Printf("%s is already in bucket %s, no need upload again !\n",
			remoteFilename, bucket)
		return nil
	}
	if remoteInfo != nil {
		recreate := false
		if remoteInfo.Size != expectSize {
			logrus.Warnf("%s exists in cloud and its size is %d, but provided file size is %d",
				remoteFilename, remoteInfo.Size, expectSize)
			recreate = true
		}
		if remoteInfo.HashBase64 != expectHash {
			logrus.Warnf("%s exists in cloud and its checksum is %s, but provided checksum is %s",
				remoteFilename, remoteInfo.HashBase64, expectHash)
			recreate = true
		}
		// no need upload, return nil upload request
		if !recreate {
			fmt.Printf("%s is already in bucket %s, no need upload again !\n",
				remoteFilename, bucket)
			return nil
		}
	}

	err = cli.PutObject(bucket, remoteFilename, expectHash, contentType, storageClass, reader)
	if err != nil {
		fmt.Printf("Uploading metadata file %s fail: %s\n", remoteFilename, err)
	} else {
		fmt.Printf("Upload metadata file %s success!\n", remoteFilename)
	}
	return err
}

func uploadRawFileToS3(cli *clients.AWSClient, bucket, storageClass, filename, contentType string) error {
	h, err := hash.CalculateHashFile(filename)
	if err != nil {
		return err
	}

	hashBase64 := hash.CalculateHashBase64(h)

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	return uploadFileToS3(cli, bucket, storageClass, filepath.Base(filename), hashBase64, contentType, int(stat.Size()), f)
}

// as PutObject requires encryption before input, thus, it has to write into one temp file or memory to get all data
// return tmp filename, and let caller delete
func uploadEncryptFileToS3(cli *clients.AWSClient, bucket, storageClass, filename, masterKey string) (string, error) {
	src, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer src.Close()

	tmpFile, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	tmpFileName := tmpFile.Name()
	defer tmpFile.Close()

	hash, err := encryptLocalFile(src, tmpFile, masterKey, true)
	if err != nil {
		return "", err
	}

	size, err := tmpFile.Seek(0, io.SeekEnd)
	if err != nil {
		return "", err
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}

	return tmpFileName, uploadFileToS3(cli, bucket, storageClass, filepath.Base(filename), lomohash.CalculateHashBase64(hash),
		binContentType, int(size), tmpFile)
}
