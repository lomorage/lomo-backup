package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
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

	filename := "test.jpg"
	if false {
		id := "ateDM93HzZ5RP4W5XRgu46y25HL94QVRV22Pw2tjnCRgboMyC77rXFAV6GI7GxPpHpInwlGXO4O1bn7s0OPJSA--"
		request := &clients.UploadRequest{Bucket: &bucketName, Key: &filename, ID: &id}
		completedParts := []clients.CompletePart{
			{
				PartNo: 1,
				Etag:   "8d200fe757145c820391ead3eead64df",
			},
			{
				PartNo: 2,
				Etag:   "0e39272785889d574dfd80ec4e917603",
			},
			{
				PartNo: 3,
				Etag:   "314f47063d2b446de1381964c1426e88",
			},
			{
				PartNo: 4,
				Etag:   "8d4bfb75dae26829acedc8b388a3ae7c",
			},
			{
				PartNo: 5,
				Etag:   "3743c9b614cf97fe594ac8420ef1ced8",
			},
		}
		return cli.CompleteMultipartUpload(request, completedParts, "")
	}

	file, err := os.Open(filename)
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

	request, err := cli.CreateMultipartUpload(bucketName, file.Name(), fileType)

	if err != nil {
		return err
	}

	fmt.Println("Created multipart upload request")

	var curr, partLength int64
	var remaining = size
	var completedParts []clients.CompletePart
	partNumber := 1

	full := sha256.New()
	for curr = 0; remaining != 0; curr += partLength {
		if remaining < maxPartSize {
			partLength = remaining
		} else {
			partLength = maxPartSize
		}
		buf := buffer[curr : curr+partLength]
		h := sha256.New()
		h.Write(buf)
		full.Write(buf)

		checksum := base64.StdEncoding.EncodeToString(h.Sum(nil))
		etag, err := cli.Upload(int64(partNumber), int64(partLength), request, bytes.NewReader(buf), checksum)
		if err != nil {
			fmt.Println(err)
			return cli.AbortMultipartUpload(request)
		}
		remaining -= partLength
		completedParts = append(completedParts, clients.CompletePart{
			PartNo:   int64(partNumber),
			Etag:     etag,
			Checksum: checksum,
		})
		partNumber++
	}

	err = cli.CompleteMultipartUpload(request, completedParts, base64.StdEncoding.EncodeToString(full.Sum(nil)))
	if err == nil {
		return nil
	}

	fmt.Println(err)
	return cli.AbortMultipartUpload(request)
}
