package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/lomorage/lomo-backup/clients"
	"github.com/lomorage/lomo-backup/common"
	"github.com/lomorage/lomo-backup/common/crypto"
	"github.com/lomorage/lomo-backup/common/datasize"
	lomohash "github.com/lomorage/lomo-backup/common/hash"
	lomoio "github.com/lomorage/lomo-backup/common/io"
	"github.com/lomorage/lomo-backup/common/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	binContentType  = "application/octet-stream"
	textContentType = "text/plain"
)

func mkIsoMetadataFilename(isoFilename string) string {
	return isoFilename + ".meta.txt"
}

func validateISO(isoFilename string) (*os.File, *types.ISOInfo, error) {
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

	hash, err := lomohash.CalculateHashFile(isoFilename)
	if err != nil {
		return nil, nil, err
	}
	hashHex := lomohash.CalculateHashHex(hash)
	if hashHex != iso.HashLocal {
		return nil, nil, errors.Errorf("Hash in DB is %s, but got %s", iso.HashLocal, hashHex)
	}
	return f, iso, nil
}

func prepareUploadParts(isoFilename string, partSize int, calHash bool) (*os.File, *types.ISOInfo, []*types.PartInfo, error) {
	isoFile, isoInfo, err := validateISO(isoFilename)
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
	var (
		numParts      int
		partsChecksum [][]byte
	)
	if calHash {
		partsChecksum, err = lomohash.CalculateMultiPartsHash(isoFilename, partSize)
		if err != nil {
			return nil, nil, nil, err
		}
		numParts = len(partsChecksum)
	} else {
		numParts = isoInfo.Size/partSize + 1
	}

	partLength := 0
	remaining := isoInfo.Size
	parts = make([]*types.PartInfo, numParts)
	for i := 0; i < numParts; i++ {
		if remaining < partSize {
			partLength = remaining
		} else {
			partLength = partSize
		}
		parts[i] = &types.PartInfo{
			PartNo: i + 1,
			Size:   partLength,
		}
		if partsChecksum != nil {
			parts[i].HashLocal = lomohash.CalculateHashHex(partsChecksum[i])
			parts[i].HashRemote = lomohash.CalculateHashBase64(partsChecksum[i])
		}
		remaining -= partLength
	}

	err = db.InsertIsoParts(isoInfo.ID, parts)
	if err != nil {
		return nil, nil, nil, err
	}
	if !calHash {
		return isoFile, isoInfo, parts, nil
	}

	isoInfo.HashRemote, err = lomohash.ConcatAndCalculateBase64Hash(partsChecksum)
	if err != nil {
		return nil, nil, nil, err
	}
	return isoFile, isoInfo, parts, db.UpdateIsoRemoteHash(isoInfo.ID, isoInfo.HashRemote)
}

func prepareUploadRequest(cli *clients.AWSClient, region, bucket, storageClass string,
	isoInfo *types.ISOInfo, force bool) (*clients.UploadRequest, error) {
	isoFilename := filepath.Base(isoInfo.Name)
	remoteInfo, err := cli.HeadObject(bucket, isoFilename)
	if err != nil {
		return nil, err
	}
	if !force && remoteInfo != nil {
		if remoteInfo.Size != isoInfo.Size {
			return nil, errors.Errorf("%s exists in cloud and its size is %d, but provided file size is %d",
				isoFilename, remoteInfo.Size, isoInfo.Size)
		}
		if isoInfo.HashRemote != "" {
			remoteHash := strings.Split(remoteInfo.HashRemote, "-")[0]
			if remoteHash != isoInfo.HashRemote {
				return nil, errors.Errorf("%s exists in cloud and its checksum is %s, but provided ccommonhecksum is %s",
					isoFilename, remoteHash, isoInfo.HashRemote)
			}
		}
		// no need upload, return nil upload request
		return nil, nil
	}

	// not exist but previous upload not finish, so reuse previous upload
	if isoInfo.Region == region && isoInfo.Bucket == bucket && isoInfo.UploadID != "" &&
		isoInfo.UploadKey != "" {
		return &clients.UploadRequest{
			ID:     isoInfo.UploadID,
			Bucket: bucket,
			Key:    isoInfo.UploadKey,
		}, nil
	}

	// create new upload
	request, err := cli.CreateMultipartUpload(bucket, isoFilename, binContentType, storageClass)
	if err != nil {
		return nil, err
	}

	isoInfo.Region = region
	isoInfo.Bucket = request.Bucket
	isoInfo.UploadKey = request.Key
	isoInfo.UploadID = request.ID

	return request, db.UpdateIsoUploadInfo(isoInfo)
}

func validateISOMetafile(metaFilename string, tree []byte) error {
	meta, err := os.Open(metaFilename)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// create the file now
		return os.WriteFile(metaFilename, tree, 0644)
	}
	defer meta.Close()

	info, err := meta.Stat()
	if err != nil {
		return err
	}

	if info.Size() != int64(len(tree)) {
		// recreate the metafile
		logrus.Warnf("Existing meta file %s's size is %d while expecting %d. Recreating",
			metaFilename, info.Size(), len(tree))
		return os.WriteFile(metaFilename, tree, 0644)
	}

	content, err := io.ReadAll(meta)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(content, tree) {
		return nil
	}
	logrus.Warnf("Existing meta file %s has different content. Recreating", metaFilename)

	return os.WriteFile(metaFilename, tree, 0644)
}

func uploadISOMetafile(cli *clients.AWSClient, bucket, storageClass, isoFilename, masterKey string) error {
	// TODO: create meta file if it is zero or not exist
	tree, err := genTreeInIso(isoFilename)
	if err != nil {
		return err
	}

	treeBuf := []byte(tree)

	metaFilename := mkIsoMetadataFilename(isoFilename)
	err = validateISOMetafile(metaFilename, treeBuf)
	if err != nil {
		return nil
	}

	if masterKey == "" {
		fmt.Printf("Uploading un-encrypted metadata file %s\n", metaFilename)

		return uploadRawFileToS3(cli, bucket, storageClass, metaFilename, textContentType)
	}

	fmt.Printf("Uploading encrypted metadata file %s\n", metaFilename)

	tmpFileName, err := uploadEncryptFileToS3(cli, bucket, storageClass, metaFilename, masterKey)
	if err != nil {
		return err
	}
	return os.Remove(tmpFileName)
}

func uploadRawParts(cli *clients.AWSClient, region, bucket, storageClass, isoFilename string,
	partSize int, saveParts, force bool) error {
	isoFile, isoInfo, parts, err := prepareUploadParts(isoFilename, partSize, true)
	if err != nil {
		return err
	}
	defer isoFile.Close()

	request, err := prepareUploadRequest(cli, region, bucket, storageClass, isoInfo, force)
	if err != nil {
		return err
	}
	if request == nil {
		fmt.Printf("%s is already in region %s, bucket %s, no need upload again !\n",
			isoFilename, region, bucket)
		return nil
	}

	var start, end int64
	var failParts []int
	for i, p := range parts {
		if i == 0 {
			end = int64(p.Size)
		} else {
			start = end
			end += int64(p.Size)
		}

		if p.Status == types.PartUploaded {
			logrus.Infof("%s's part %d was uploaded successfully, skip new upload", isoFilename, p.PartNo)
			continue
		}

		logrus.Infof("Uploading %s's part %d [%d, %d]", isoFilename, p.PartNo, start, end)

		var readSeeker io.ReadSeeker
		prs := lomoio.NewFilePartReadSeeker(isoFile, start, end)
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
			readSeeker = lomoio.NewReadSeekSaver(partFile, prs)
		} else {
			readSeeker = prs
		}

		p.Etag, err = cli.Upload(int64(p.PartNo), int64(p.Size), request, readSeeker, p.HashRemote)
		if err != nil {
			failParts = append(failParts, p.PartNo)
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
		logrus.Infof("Uploading %s's part %d is done!", isoFilename, p.PartNo)
	}

	if len(failParts) != 0 {
		return errors.Errorf("Parts %v failed to upload", failParts)
	}
	err = cli.CompleteMultipartUpload(request, parts, isoInfo.HashRemote)
	if err != nil {
		logrus.Warnf("Upload %s fail: %s", isoFilename, err)
		return err
	}
	fmt.Printf("%s is uploaded to region %s, bucket %s successfully!\n",
		isoFilename, region, bucket)

	return db.UpdateIsoStatus(isoInfo.ID, types.IsoUploaded)
}

func uploadEncryptParts(cli *clients.AWSClient, region, bucket, storageClass, isoFilename, masterKey string,
	partSize int, saveParts, force bool) error {
	isoFile, isoInfo, parts, err := prepareUploadParts(isoFilename, partSize, false)
	if err != nil {
		return err
	}
	defer isoFile.Close()

	decoded, err := hex.DecodeString(isoInfo.HashLocal)
	if err != nil {
		return err
	}
	if len(decoded) < crypto.SaltLen() {
		return errors.Errorf("invalid hash length '%d', less than '%d'", len(decoded), crypto.SaltLen())
	}

	salt := decoded[:crypto.SaltLen()]

	// Derive key from passphrase using Argon2
	// TODO: Using IV as salt for simplicity, change to different salt?
	encryptKey := crypto.DeriveKeyFromMasterKey([]byte(masterKey), salt)

	// iso size need add salt block size so as to compare with remote size
	isoInfo.Size += crypto.SaltLen()
	isoInfo.HashRemote = ""
	request, err := prepareUploadRequest(cli, region, bucket, storageClass, isoInfo, force)
	if err != nil {
		return err
	}
	if request == nil {
		fmt.Printf("%s is already in region %s, bucket %s, no need upload again !\n",
			isoFilename, region, bucket)
		return nil
	}

	partsHash := [][]byte{}

	var (
		start, end int64
		failParts  []int
		encryptor  *crypto.Encryptor
		prs        *lomoio.FilePartReadSeeker
	)
	for i, p := range parts {
		// add salt len for the last part
		if i == len(parts)-1 {
			p.Size += crypto.SaltLen()
		}

		if i == 0 {
			end = int64(p.Size - crypto.SaltLen())
		} else {
			start = end
			end += int64(p.Size)
		}

		if p.Status == types.PartUploaded {
			logrus.Infof("%s's part %d was uploaded successfully, skip new upload", isoFilename, p.PartNo)
			h, err := lomohash.DecodeHashBase64(p.HashRemote)
			if err != nil {
				return errors.Wrapf(err, "while decode part %d's base64 hash %s", i+1, p.HashRemote)
			}
			partsHash = append(partsHash, h)
			continue
		}

		logrus.Infof("Uploading %s's part %d [%d, %d]", isoFilename, p.PartNo, start, end)

		// create a local tmpfile and save intermittent part
		tmpFile, err := os.CreateTemp("", "part")
		if err != nil {
			return err
		}
		tmpFilename := tmpFile.Name()
		defer os.Remove(tmpFilename)
		defer tmpFile.Close()

		if prs == nil {
			prs = lomoio.NewFilePartReadSeeker(isoFile, start, end)
		} else {
			prs.SetStartEnd(start, end)
		}

		hr := sha256.New()
		mw := io.MultiWriter(hr, tmpFile)

		if encryptor == nil {
			encryptor, err = crypto.NewEncryptor(prs, encryptKey, salt, false)
			if err != nil {
				return err
			}
			n, err := mw.Write(salt)
			if err != nil {
				return err
			}
			if n != len(salt) {
				return fmt.Errorf("write %d byte salt while expecting %d", n, len(salt))
			}
		}

		n, err := io.Copy(mw, encryptor)
		if err != nil {
			return err
		}

		if n != end-start {
			return fmt.Errorf("write %d byte salt while expecting %d btw [%d, %d]", n, end-start, start, end)
		}

		hrData := hr.Sum(nil)
		p.SetHashRemote(hrData)

		// seek to beginning for upload
		_, err = tmpFile.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}

		p.Etag, err = cli.Upload(int64(p.PartNo), int64(p.Size), request, tmpFile, p.HashRemote)
		if err != nil {
			failParts = append(failParts, p.PartNo)
			logrus.Infof("Upload %s's part number %d:%s", isoFilename, p.PartNo, err)
			err = db.UpdatePartStatus(p.IsoID, p.PartNo, types.PartUploadFailed)
			if err != nil {
				logrus.Infof("Update %s's part number %d status %s:%s", isoFilename, p.PartNo,
					types.PartUploadFailed, err)
			}
			continue
		}
		partsHash = append(partsHash, hrData)
		err = db.UpdatePartEtagAndStatusHash(p.IsoID, p.PartNo, p.Etag, p.HashLocal, p.HashRemote, types.PartUploaded)
		if err != nil {
			logrus.Infof("Update %s's part number %d status %s:%s", isoFilename, p.PartNo,
				types.PartUploaded, err)
		}
		logrus.Infof("Uploading %s's part %d is done!", isoFilename, p.PartNo)
		if saveParts {
			err = tmpFile.Close()
			if err != nil {
				return err
			}
			err = os.Rename(tmpFilename, isoFilename+".part"+strconv.Itoa(i+1))
			if err != nil {
				return err
			}
		}
	}

	if len(failParts) != 0 {
		return errors.Errorf("Parts %v failed to upload", failParts)
	}

	isoInfo.HashRemote, err = lomohash.ConcatAndCalculateBase64Hash(partsHash)
	if err != nil {
		return errors.Wrapf(err, "while encode iso base64 hash %v", partsHash)
	}
	err = cli.CompleteMultipartUpload(request, parts, isoInfo.HashRemote)
	if err != nil {
		logrus.Warnf("Upload %s fail: %s", isoFilename, err)
		return err
	}
	fmt.Printf("%s is uploaded to region %s, bucket %s successfully!\n",
		isoFilename, region, bucket)

	return db.UpdateIsoStatusRemoteHash(isoInfo.ID, isoInfo.HashRemote, types.IsoUploaded)
}

func uploadISO(accessKeyID, accessKey, region, bucket, storageClass, isoFilename, masterKey string,
	partSize int, saveParts, force bool) error {
	cli, err := clients.NewAWSClient(accessKeyID, accessKey, region)
	if err != nil {
		return err
	}

	// check metadata file firstly
	err = uploadISOMetafile(cli, bucket, storageClass, isoFilename, masterKey)
	if err != nil {
		return err
	}

	if force {
		err = db.ResetISOUploadInfo(isoFilename)
		if err != nil {
			return err
		}
	}
	if masterKey == "" {
		return uploadRawParts(cli, region, bucket, storageClass, isoFilename, partSize, saveParts, force)
	}
	return uploadEncryptParts(cli, region, bucket, storageClass, isoFilename, masterKey, partSize, saveParts, force)
}

func uploadISOs(ctx *cli.Context) error {
	ps, err := datasize.ParseString(ctx.String("part-size"))
	if err != nil {
		return err
	}
	partSize := int(ps)
	if partSize < 5*1024*1024 {
		return errors.New("part size must be larger than 5*1024*1024=5242880")
	}
	if partSize%crypto.SaltLen() != 0 || (partSize-crypto.SaltLen())%crypto.SaltLen() != 0 {
		return errors.Errorf("part size must be able to divided by salt length '%d'", crypto.SaltLen())
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
	force := ctx.Bool("force")

	if len(ctx.Args()) == 0 {
		return errors.New("Please supply one iso file name at least, or -a to upload all files not uploaded")
	}

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

	for _, isoFilename := range ctx.Args() {
		err = uploadISO(accessKeyID, secretAccessKey, region, bucket, storageClass,
			isoFilename, masterKey, partSize, saveParts, force)
		if err != nil {
			return err
		}
	}
	return nil
}

func listUploadingItems(ctx *cli.Context) error {
	accessKeyID := ctx.String("awsAccessKeyID")
	secretAccessKey := ctx.String("awsSecretAccessKey")
	region := ctx.String("awsBucketRegion")
	bucket := ctx.String("awsBucketName")

	cli, err := clients.NewAWSClient(accessKeyID, secretAccessKey, region)
	if err != nil {
		return err
	}

	requests, err := cli.ListMultipartUploads(bucket)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', tabwriter.TabIndent)
	defer writer.Flush()

	fmt.Fprint(writer, "UploadKey\tUploadID\tUploadTime\n")
	for _, r := range requests {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", r.Key, r.ID,
			common.FormatTime(r.Time.Local()))
	}
	return nil
}

func abortUpload(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		return errors.New("please provide upload key at least")
	}
	accessKeyID := ctx.String("awsAccessKeyID")
	secretAccessKey := ctx.String("awsSecretAccessKey")
	region := ctx.String("awsBucketRegion")
	bucket := ctx.String("awsBucketName")

	cli, err := clients.NewAWSClient(accessKeyID, secretAccessKey, region)
	if err != nil {
		return err
	}

	uploadKey := ctx.Args()[0]
	if len(ctx.Args()) > 1 {
		err = cli.AbortMultipartUpload(&clients.UploadRequest{
			Key:    uploadKey,
			ID:     ctx.Args()[1],
			Bucket: bucket,
		})
		if err != nil {
			return err
		}
		fmt.Println("abort upload success")
		return nil
	}

	requests, err := cli.ListMultipartUploads(bucket)
	if err != nil {
		return errors.Wrap(err, "while listing all multi part uploads")
	}
	if len(requests) == 0 {
		fmt.Println("no in progress multipart upload to abort")
		return nil
	}
	for _, r := range requests {
		if r.Key != uploadKey {
			continue
		}
		err = cli.AbortMultipartUpload(&clients.UploadRequest{
			Key:    uploadKey,
			ID:     r.ID,
			Bucket: bucket,
		})
		if err != nil {
			fmt.Printf("abort upload ID %s: %s\n", r.ID, err)
		} else {
			fmt.Printf("abort upload ID %s success!\n", r.ID)
		}
	}
	return nil
}

func calculatePartHash(ctx *cli.Context) error {
	partSize, err := datasize.ParseString(ctx.String("part-size"))
	if err != nil {
		return err
	}
	filename := ctx.Args()[0]
	partNumber := ctx.Int("part-number")
	if partNumber == 0 {
		parts, err := lomohash.CalculateMultiPartsHash(filename, int(partSize))
		if err != nil {
			return err
		}
		for i, p := range parts {
			fmt.Printf("Part %d: %s\n", i+1, lomohash.CalculateHashBase64(p))
		}

		overall, err := lomohash.ConcatAndCalculateBase64Hash(parts)
		if err != nil {
			return err
		}
		fmt.Printf("Overall: %s\n", overall)
		return nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := os.Stat(filename)
	if err != nil {
		return err
	}

	var curr, partLength int64
	var remaining = info.Size()
	var no = 1
	var prs *lomoio.FilePartReadSeeker
	var h hash.Hash
	for curr = 0; remaining != 0; curr += partLength {
		if remaining < int64(partSize) {
			partLength = remaining
		} else {
			partLength = int64(partSize)
		}

		if partNumber != no {
			goto next
		}

		fmt.Printf("Calculating base64 hash from %d to %d and remaining %d\n",
			curr, curr+partLength, remaining)
		prs = lomoio.NewFilePartReadSeeker(f, curr, curr+partLength)
		h = sha256.New()
		_, err = io.Copy(h, prs)
		if err != nil {
			return err
		}

		fmt.Printf("Part %d: %s\n", partNumber, lomohash.CalculateHashBase64(h.Sum(nil)))
		return nil
	next:
		no++
		remaining -= partLength
	}

	return nil
}
