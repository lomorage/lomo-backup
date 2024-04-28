package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/lomorage/lomo-backup/clients"
	"github.com/lomorage/lomo-backup/common"
	"github.com/lomorage/lomo-backup/common/types"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const isoContentType = "application/octet-stream"

func validateISO(isoFilename, metaFilename string) (*types.ISOInfo, error) {
	// create meta file if it is zero or not exist
	info, err := os.Stat(isoFilename)
	if err != nil {
		return nil, err
	}

	iso, err := db.GetIsoByName(isoFilename)
	if err != nil {
		return nil, err
	}

	if info.Size() != int64(iso.Size) {
		return nil, errors.Errorf("Size in DB is %d, but got %d", iso.Size, info.Size())
	}

	hash, err := common.CalculateHash(isoFilename)
	if err != nil {
		return nil, err
	}
	hashHex := common.CalculateHashHex(hash)
	hashBase64 := common.CalculateHashBase64(hash)
	if hashHex != iso.HashHex {
		return nil, errors.Errorf("Hash in DB is %s, but got %s", iso.HashHex, info.Size())
	}
	if hashBase64 != iso.HashBase64 {
		return nil, errors.Errorf("Hash in DB is %s, but got %s", iso.HashBase64, info.Size())
	}
	return iso, nil
}

func validateUploadParts(isoID int) ([]*types.PartInfo, error) {
	/*
		// create meta file if it is zero or not exist
		info, err := os.Stat(isoFilename)
		if err != nil {
			return nil, nil, err
		}

		iso, err := db.GetIsoByName(isoFilename)
		if err != nil {
			return nil, nil, err
		}

		parts, err := db.GetPartsByIsoID(iso.ID)
		if err != nil {
			return nil, nil, err
		}
	*/
	return nil, nil
}

func uploadISO(ctx *cli.Context) error {
	err := initLogLevel(ctx.GlobalInt("log-level"))
	if err != nil {
		return err
	}

	accessKeyID := ctx.String("awsAccessKeyID")
	secretAccessKey := ctx.String("awsSecretAccessKey")
	bucketRegion := ctx.String("awsBucketRegion")
	bucketName := ctx.String("awsBucketName")
	saveParts := ctx.Bool("save-parts")
	fmt.Println(accessKeyID)
	fmt.Println(secretAccessKey)
	fmt.Println(bucketRegion)
	fmt.Println(bucketName)
	const partSize = 5 * 1024 * 1024

	cli, err := clients.NewUpload(accessKeyID, secretAccessKey, bucketRegion, clients.AWS)
	if err != nil {
		return err
	}

	filename := "test.jpg"
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	/*	_, fullChecksum, err := common.CalculateHash(filename)
		if err != nil {
			return err
		}
	*/
	partsChecksum, err := common.CalculateMultiPartsHash(filename, partSize)
	if err != nil {
		return err
	}
	fullChecksum, err := common.ConcatAndCalculateBase64Hash(partsChecksum)
	if err != nil {
		return err
	}

	var (
		curr, partLength int64
		remaining        = int64(info.Size())
		completedParts   []clients.CompletePart
		partNumber       = 1
	)
	request, err := cli.CreateMultipartUpload(bucketName, f.Name(), isoContentType)
	if err != nil {
		return err
	}

	/*buffer, err := io.ReadAll(f)
	if err != nil {
		return err
	}*/
	for curr = 0; remaining != 0; curr += partLength {
		if remaining < partSize {
			partLength = remaining
		} else {
			partLength = partSize
		}

		var readSeeker io.ReadSeeker
		prs := common.NewFilePartReadSeeker(f, curr, curr+partLength)
		//prs := bytes.NewReader(buffer[curr : curr+partLength])
		if saveParts {
			partFile, err := os.Create(filename + ".part" + strconv.Itoa(partNumber))
			if err != nil {
				e := cli.AbortMultipartUpload(request)
				if e != nil {
					fmt.Printf("abort request %v: %s\n", *request, e)
				}
				return err
			}
			defer partFile.Close()
			readSeeker = common.NewReadSeekSaver(partFile, prs)
		} else {
			readSeeker = prs
		}

		hash := common.CalculateHashBase64(partsChecksum[partNumber-1])
		etag, err := cli.Upload(int64(partNumber), int64(partLength), request, readSeeker, hash)
		if err != nil {
			fmt.Println(err)
			return cli.AbortMultipartUpload(request)
		}
		remaining -= partLength
		completedParts = append(completedParts, clients.CompletePart{
			PartNo:   int64(partNumber),
			Etag:     etag,
			Checksum: hash,
		})
		partNumber++
	}

	err = cli.CompleteMultipartUpload(request, completedParts, fullChecksum)
	if err == nil {
		fmt.Printf("%s is uploaded to region %s, bucket %s successfully!\n", filename, bucketRegion, bucketName)
		return nil
	}

	fmt.Println(err)
	return cli.AbortMultipartUpload(request)
}
