package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"text/tabwriter"

	"github.com/lomorage/lomo-backup/clients"
	"github.com/lomorage/lomo-backup/common"
	"github.com/lomorage/lomo-backup/common/datasize"
	"github.com/lomorage/lomo-backup/common/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const isoContentType = "application/octet-stream"

func mkIsoMetadataFilename(isoFilename string) string {
	return isoFilename + ".meta.txt"
}

func validateISO(isoFilename, metaFilename string) (*os.File, *types.ISOInfo, error) {
	f, err := os.Open(isoFilename)
	if err != nil {
		return nil, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}

	iso, err := db.GetIsoByName(isoFilename)
	if err != nil {
		return nil, nil, err
	}

	if info.Size() != int64(iso.Size) {
		return nil, nil, errors.Errorf("Size in DB is %d, but got %d", iso.Size, info.Size())
	}

	hash, err := common.CalculateHash(isoFilename)
	if err != nil {
		return nil, nil, err
	}
	hashHex := common.CalculateHashHex(hash)
	if hashHex != iso.HashHex {
		return nil, nil, errors.Errorf("Hash in DB is %s, but got %s", iso.HashHex, hash)
	}

	// TODO: create meta file if it is zero or not exist
	return f, iso, nil
}

func prepareUpload(isoFilename string, partSize int) (*os.File, *types.ISOInfo, []*types.PartInfo, error) {
	isoFile, isoInfo, err := validateISO(isoFilename, mkIsoMetadataFilename(isoFilename))
	if err != nil {
		return nil, nil, nil, err
	}

	parts, err := db.GetPartsByIsoID(isoInfo.ID)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(parts) != 0 {
		return isoFile, isoInfo, parts, nil
	}
	partsChecksum, err := common.CalculateMultiPartsHash(isoFilename, partSize)
	if err != nil {
		return nil, nil, nil, err
	}

	partLength := 0
	remaining := isoInfo.Size
	parts = make([]*types.PartInfo, len(partsChecksum))
	for i, p := range partsChecksum {
		if remaining < partSize {
			partLength = remaining
		} else {
			partLength = partSize
		}
		parts[i] = &types.PartInfo{
			PartNo:     i + 1,
			Size:       partLength,
			HashHex:    common.CalculateHashHex(p),
			HashBase64: common.CalculateHashBase64(p),
		}
	}

	err = db.InsertIsoParts(isoInfo.ID, parts)
	if err != nil {
		return nil, nil, nil, err
	}
	isoInfo.HashBase64, err = common.ConcatAndCalculateBase64Hash(partsChecksum)
	if err != nil {
		return nil, nil, nil, err
	}
	return isoFile, isoInfo, parts, db.UpdateIsoBase64Hash(isoInfo.ID, isoInfo.HashBase64)
}

func uploadISO(accessKeyID, accessKey, region, bucket, isoFilename string,
	partSize int, saveParts bool) error {
	isoFile, isoInfo, parts, err := prepareUpload(isoFilename, partSize)
	if err != nil {
		return err
	}

	cli, err := clients.NewUpload(accessKeyID, accessKey, region, clients.S3)
	if err != nil {
		return err
	}

	request, err := cli.CreateMultipartUpload(bucket, filepath.Base(isoFilename), isoContentType)
	if err != nil {
		return err
	}

	isoInfo.Region = region
	isoInfo.Bucket = request.Bucket
	isoInfo.UploadKey = request.Key
	isoInfo.UploadID = request.ID

	err = db.UpdateIsoUploadInfo(isoInfo)
	if err != nil {
		return err
	}

	var start, end int64
	for i, p := range parts {
		if p.Status == types.PartUploaded {
			logrus.Infof("%s's part %d was uploaded successfully, skip new upload", isoFilename, p.PartNo)
			continue
		}
		if i == 0 {
			end = int64(p.Size)
		} else {
			start = end
			end += int64(p.Size)
		}
		var readSeeker io.ReadSeeker
		prs := common.NewFilePartReadSeeker(isoFile, start, end)
		if saveParts {
			partFile, err := os.Create(isoFilename + ".part" + strconv.Itoa(p.PartNo))
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

		p.Etag, err = cli.Upload(int64(p.PartNo), int64(p.Size), request, readSeeker, p.HashBase64)
		if err != nil {
			logrus.Infof("Upload %s's part number %d:%s", isoFilename, p.PartNo, err)
			err = db.UpdatePartStatus(p.IsoID, p.PartNo, types.PartUploadFailed)
			if err != nil {
				logrus.Infof("Update %s's part number %d status %s:%s", isoFilename, p.PartNo,
					types.PartUploadFailed, err)
			}
			continue
		}
		err = db.UpdatePartEtagAndStatus(p.IsoID, p.PartNo, p.Etag, types.PartUploaded)
		if err != nil {
			logrus.Infof("Update %s's part number %d status %s:%s", isoFilename, p.PartNo,
				types.PartUploaded, err)
		}
	}

	// make it fail
	isoInfo.HashBase64 = ""
	err = cli.CompleteMultipartUpload(request, parts, isoInfo.HashBase64)
	if err == nil {
		logrus.Warnf("Upload %s fail: %s", isoFilename, err)
	} else {
		fmt.Printf("%s is uploaded to region %s, bucket %s successfully!\n",
			isoFilename, region, bucket)
	}

	return err
}

func uploadISOs(ctx *cli.Context) error {
	partSize, err := datasize.ParseString(ctx.String("part-size"))
	if err != nil {
		return err
	}
	if partSize < 5*1024*1024 {
		return errors.New("part size must be larger than 5*1024*1024=5242880")
	}

	err = initDB(ctx.GlobalString("db"))
	if err != nil {
		return err
	}

	err = initLogLevel(ctx.GlobalInt("log-level"))
	if err != nil {
		return err
	}

	accessKeyID := ctx.String("awsAccessKeyID")
	secretAccessKey := ctx.String("awsSecretAccessKey")
	region := ctx.String("awsBucketRegion")
	bucket := ctx.String("awsBucketName")
	saveParts := ctx.Bool("save-parts")

	if len(ctx.Args()) == 0 {
		return errors.New("Please supply one iso file name at least, or -a to upload all files not uploaded")
	}

	for _, isoFilename := range ctx.Args() {
		err = uploadISO(accessKeyID, secretAccessKey, region, bucket,
			isoFilename, int(partSize), saveParts)
		if err != nil {
			return err
		}
	}
	return nil
}

func listBackups(ctx *cli.Context) error {
	accessKeyID := ctx.String("awsAccessKeyID")
	secretAccessKey := ctx.String("awsSecretAccessKey")
	region := ctx.String("awsBucketRegion")
	bucket := ctx.String("awsBucketName")

	cli, err := clients.NewUpload(accessKeyID, secretAccessKey, region, clients.S3)
	if err != nil {
		return err
	}

	requests, err := cli.ListMultipartUploads(bucket)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', tabwriter.TabIndent)
	defer writer.Flush()

	fmt.Fprint(writer, "Key\tUploadID\tUploadTime\n")
	for _, r := range requests {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", r.Key, r.ID,
			common.FormatTime(r.Time.Local()))
	}
	return nil
}
