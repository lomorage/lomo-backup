package main

import (
	"context"
	"crypto/aes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lomorage/lomo-backup/clients"
	"github.com/lomorage/lomo-backup/common/crypto"
	"github.com/lomorage/lomo-backup/common/gcloud"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func restoreGdriveFile(ctx *cli.Context) error {
	if len(ctx.Args()) != 2 {
		return errors.New("please provide one encrypted filename with fullpath and output filename")
	}

	client, err := gcloud.CreateDriveClient(&gcloud.Config{
		CredFilename:  ctx.String("cred"),
		TokenFilename: ctx.String("token"),
		RefreshToken:  true,
	})
	if err != nil {
		return fmt.Errorf("unable to retrieve Drive client: %v", err)
	}

	src := ctx.Args()[0]
	dst, err := os.Create(ctx.Args()[1])
	if err != nil {
		return err
	}
	defer dst.Close()

	uploadRootFolder := ctx.String("folder")
	fid, _, err := client.GetFileID(uploadRootFolder, "")
	if err != nil {
		return err
	}
	parts := strings.Split(src, "/")
	for _, name := range parts {
		fid, _, err = client.GetFileID(name, fid)
		if err != nil {
			return err
		}
	}
	// final file, decrypt
	readCloser, err := client.GetFile(fid)
	if err != nil {
		return err
	}
	defer readCloser.Close()

	// read nonce
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(readCloser, iv); err != nil {
		return err
	}

	masterKey := ctx.String("encrypt-key")
	if masterKey == "" {
		masterKey, err = getMasterKey()
		if err != nil {
			return err
		}
	}
	encryptKey := crypto.DeriveKeyFromMasterKey([]byte(masterKey), iv)

	decryptor, err := crypto.NewDecryptor(dst, encryptKey, iv)
	if err != nil {
		return err
	}

	_, err = io.Copy(decryptor, readCloser)
	return err
}

func restoreAwsFile(ctx *cli.Context) error {
	if len(ctx.Args()) != 2 {
		return errors.New("please provide one iso filename and output filename")
	}

	accessKeyID := ctx.String("awsAccessKeyID")
	accessKey := ctx.String("awsSecretAccessKey")
	region := ctx.String("awsBucketRegion")
	bucket := ctx.String("awsBucketName")

	src := ctx.Args()[0]
	dst, err := os.Create(ctx.Args()[1])
	if err != nil {
		return err
	}
	defer dst.Close()

	cli, err := clients.NewAWSClient(accessKeyID, accessKey, region)
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

	decryptor := crypto.NewMasterDecryptor(dst, []byte(masterKey))
	_, err = cli.GetObject(context.Background(), bucket, src, decryptor)
	return err
}
