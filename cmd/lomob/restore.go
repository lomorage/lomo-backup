package main

import (
	"crypto/aes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lomorage/lomo-backup/common/crypto"
	"github.com/lomorage/lomo-backup/common/gcloud"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func restoreGdriveFile(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		return errors.New("please provide one encrypted filename with fullpath")
	}

	client, err := gcloud.CreateDriveClient(&gcloud.Config{
		CredFilename:  ctx.String("cred"),
		TokenFilename: ctx.String("token"),
		RefreshToken:  true,
	})
	if err != nil {
		return fmt.Errorf("unable to retrieve Drive client: %v", err)
	}

	var dst io.Writer
	if ctx.String("output") == "" {
		dst = os.Stdout
	} else {
		f, err := os.Create(ctx.String("output"))
		if err != nil {
			return err
		}
		defer f.Close()
		dst = f
	}

	uploadRootFolder := ctx.String("folder")
	fid, _, err := client.GetFileID(uploadRootFolder, "")
	if err != nil {
		return err
	}
	parts := strings.Split(ctx.Args()[0], "/")
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
	encryptKey := deriveKeyFromMasterKey([]byte(masterKey), iv)

	decryptor, err := crypto.NewDecryptor(dst, encryptKey, iv)
	if err != nil {
		return err
	}

	_, err = io.Copy(decryptor, readCloser)
	return err
}
