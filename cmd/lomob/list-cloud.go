package main

import (
	"fmt"

	"github.com/lomorage/lomo-backup/common/gcloud"
	"github.com/urfave/cli"
)

func listFilesInGDrive(ctx *cli.Context) error {
	client, err := gcloud.CreateClient(&gcloud.Config{
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
	} else {
		fmt.Printf("File Name: %s, File ID: %s\n", folder, folderID)
	}
	return nil
}
