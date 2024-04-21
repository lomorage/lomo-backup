package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"github.com/lomorage/lomo-backup/clients"
	"github.com/urfave/cli"
)

const maxPartSize = int64(5 * 1024 * 1024)

func uploadISO(ctx *cli.Context) error {
	accessKeyID := ctx.String("awsAccessKeyID")
	secretAccessKey := ctx.String("awsSecretAccessKey")
	bucketRegion := ctx.String("awsBucketRegion")
	bucketName := ctx.String("awsBucketName")
	fmt.Println(accessKeyID)
	fmt.Println(secretAccessKey)
	fmt.Println(bucketRegion)
	fmt.Println(bucketName)

	cli, err := clients.NewUpload(accessKeyID, secretAccessKey, bucketRegion, clients.AWS)
	if err != nil {
		return err
	}

	file, err := os.Open("test.jpg")
	if err != nil {
		return err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	size := fileInfo.Size()
	buffer := make([]byte, size)
	fileType := http.DetectContentType(buffer)
	file.Read(buffer)

	path := "/media/" + file.Name()

	request, err := cli.CreateMultipartUpload(bucketName, path, fileType)
	if err != nil {
		return err
	}
	fmt.Println("Created multipart upload request")

	var curr, partLength int64
	var remaining = size
	var completedParts []clients.CompletePart
	partNumber := 1
	for curr = 0; remaining != 0; curr += partLength {
		if remaining < maxPartSize {
			partLength = remaining
		} else {
			partLength = maxPartSize
		}
		etag, err := cli.Upload(int64(partNumber), int64(partLength), request, bytes.NewReader(buffer[curr:curr+partLength]), "")
		if err != nil {
			return err
		}
		remaining -= partLength
		partNumber++
		completedParts = append(completedParts, clients.CompletePart{
			PartNo: int64(partNumber),
			Etag:   etag,
		})
	}

	return cli.CompleteMultipartUpload(request, completedParts, "")
}
