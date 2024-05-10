package main

import (
	"fmt"
	"strconv"

	"github.com/lomorage/lomo-backup/common/gcloud"
	"github.com/urfave/cli"
	"github.com/xlab/treeprint"
)

func listFilesInGDrive(ctx *cli.Context) error {
	client, err := gcloud.CreateDriveClient(&gcloud.Config{
		CredFilename:  ctx.String("cred"),
		TokenFilename: ctx.String("token"),
	})
	if err != nil {
		return fmt.Errorf("unable to retrieve Drive client: %v", err)
	}

	folder := ctx.String("folder")
	folderID, _, err := client.GetFileID(folder, "")
	if err != nil {
		return err
	}

	if folderID == "" {
		fmt.Printf("Folder '%s' not found.\n", folder)
		return nil
	} else {
		fmt.Printf("File Name: %s, File ID: %s\n", folder, folderID)
	}
	rootNode := treeprint.NewWithRoot(folder)
	err = listFileTreeInGDrive(client, rootNode, folderID)
	if err != nil {
		return err
	}
	fmt.Println(rootNode.String())
	return nil
}

func listFileTreeInGDrive(client *gcloud.DriveClient, currNode treeprint.Tree, folderID string) error {
	folders, files, err := client.ListFiles(folderID)
	if err != nil {
		return err
	}
	for _, folder := range folders {
		t := folder.ModTime

		childNode := currNode.AddMetaBranch(
			fmt.Sprintf("\t%02d/%02d/%d", t.Month(), t.Day(), t.Year()), folder.Path)
		err = listFileTreeInGDrive(client, childNode, folder.RefID)
		if err != nil {
			return err
		}
	}
	for _, file := range files {
		t := file.ModTime
		hashOrigin := file.Hash
		if len(hashOrigin) > 6 {
			hashOrigin = hashOrigin[:6]
		}

		hashEncrypt := file.HashEncrypt
		if len(hashEncrypt) > 6 {
			hashEncrypt = hashEncrypt[:6]
		}
		currNode.AddMetaNode(fmt.Sprintf("\t%12s\t%02d/%02d/%d\t%s\t%s", strconv.Itoa(file.Size),
			t.Month(), t.Day(), t.Year(), hashOrigin, hashEncrypt), file.Name)
	}
	return nil
}
