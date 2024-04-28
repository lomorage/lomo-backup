package clients

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lomorage/lomo-backup/common"
	"github.com/lomorage/lomo-backup/common/datasize"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	maxPartSize = int64(5 * 1024 * 1024)
	maxRetries  = 1
)

var (
	checksumAlgorithm = s3.ChecksumAlgorithmSha256
)

const (
	AWS   = 1
	Minio = 2
)

type UploadRequest struct {
	ID     *string
	Bucket *string
	Key    *string
}

type CompletePart struct {
	PartNo   int64
	Etag     string
	Checksum string
}

type UploadClient interface {
	CreateMultipartUpload(bucket, remotePath, fileType string) (*UploadRequest, error)
	Upload(partNo, length int64, request *UploadRequest, reader io.ReadSeeker,
		checksum string) (string, error)
	CompleteMultipartUpload(request *UploadRequest, parts []CompletePart, checksum string) error
	AbortMultipartUpload(request *UploadRequest) error
}

type Upload struct {
	client UploadClient
}

func NewUpload(keyID, key, region string, svc int) (*Upload, error) {
	aclient, err := newAWSClient(keyID, key, region)
	if err != nil {
		return nil, err
	}
	return &Upload{client: aclient}, nil
}

func (up *Upload) CreateMultipartUpload(bucket, remotePath, fileType string) (*UploadRequest, error) {
	return up.client.CreateMultipartUpload(bucket, remotePath, fileType)
}

func (up *Upload) Upload(partNo, length int64, request *UploadRequest, reader io.ReadSeeker,
	checksum string) (string, error) {
	return up.client.Upload(partNo, length, request, reader, checksum)
}

func (up *Upload) CompleteMultipartUpload(request *UploadRequest, parts []CompletePart, checksum string) error {
	return up.client.CompleteMultipartUpload(request, parts, checksum)
}

func (up *Upload) AbortMultipartUpload(request *UploadRequest) error {
	return up.client.AbortMultipartUpload(request)
}

type awsClient struct {
	region string
	svc    *s3.S3
}

func newAWSClient(keyID, key, region string) (*awsClient, error) {
	creds := credentials.NewStaticCredentials(keyID, key, "")
	_, err := creds.Get()
	if err != nil {
		return nil, err
	}
	cfg := aws.NewConfig().WithRegion(region).WithCredentials(creds)
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return &awsClient{region: region, svc: s3.New(sess, cfg)}, nil
}

func (ac *awsClient) CreateMultipartUpload(bucket, remotePath, fileType string) (*UploadRequest, error) {
	// create bucket if not exist
	_, err := ac.svc.HeadBucket(&s3.HeadBucketInput{Bucket: &bucket})
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != "NotFound" {
			return nil, err
		}
		logrus.Infof("Bucket %s does't exist in %s, creating", bucket, ac.region)

		_, err = ac.svc.CreateBucket(&s3.CreateBucketInput{Bucket: &bucket})
		if err != nil {
			return nil, err
		}
		logrus.Infof("Bucket %s is created in %s", bucket, ac.region)
	} else {
		logrus.Infof("Bucket %s exists in %s", bucket, ac.region)
	}

	object, err := ac.svc.HeadObject(&s3.HeadObjectInput{
		Bucket:       &bucket,
		Key:          &remotePath,
		ChecksumMode: aws.String(s3.ChecksumModeEnabled),
	})
	if err == nil {
		errString := fmt.Sprintf("%s exists in region %s, bucket %s.",
			remotePath, ac.region, bucket)
		if object.ContentLength != nil {
			errString += " Its size is " + datasize.ByteSize(*object.ContentLength).HR() + "."
		}
		content, _ := json.MarshalIndent(object, "", "  ")
		logrus.Debugf("%s at region %s, bucket %s head object: %s",
			remotePath, ac.region, bucket, string(content))
		if object.ChecksumSHA256 != nil {
			errString += " Its base64 encoded sha256 value is " + *object.ChecksumSHA256 + "."
		}
		return nil, errors.Errorf(errString)
	} else if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
		logrus.Infof("%s does't exist in region %s, bucket %s, multipart uploading",
			remotePath, ac.region, bucket)
	} else {
		return nil, err
	}

	input := &s3.CreateMultipartUploadInput{
		Bucket:            &bucket,
		Key:               &remotePath,
		ContentType:       &fileType,
		ChecksumAlgorithm: &checksumAlgorithm,
	}

	resp, err := ac.svc.CreateMultipartUpload(input)
	if err != nil {
		return nil, err
	}

	common.LogDebugObject("CreateMultipartUploadReply", resp)
	return &UploadRequest{
		ID:     resp.UploadId,
		Bucket: resp.Bucket,
		Key:    resp.Key,
	}, nil
}

func (ac *awsClient) Upload(partNo, length int64, request *UploadRequest, reader io.ReadSeeker,
	checksum string) (string, error) {
	partInput := &s3.UploadPartInput{
		Body:              reader,
		Bucket:            request.Bucket,
		Key:               request.Key,
		PartNumber:        &partNo,
		UploadId:          request.ID,
		ContentLength:     &length,
		ChecksumAlgorithm: &checksumAlgorithm,
		ChecksumSHA256:    &checksum,
	}

	common.LogDebugObject("UploadPartInput", partInput)

	var retErr error
	for retry := 1; retry <= maxRetries; retry++ {
		logrus.Infof("#%d retry uploading part %d for %s", retry, partNo, *request.Key)
		uploadResult, err := ac.svc.UploadPart(partInput)
		if err == nil {
			if uploadResult.ETag == nil {
				return "", nil
			}

			common.LogDebugObject("UploadPartReply", uploadResult)

			return *uploadResult.ETag, nil
		}
		if retErr == nil {
			retErr = err
		} else {
			retErr = errors.Wrapf(retErr, "%d err: %s", retry, err)
		}
	}
	return "", retErr
}

func (ac *awsClient) CompleteMultipartUpload(request *UploadRequest, parts []CompletePart, checksum string) error {
	completedParts := make([]*s3.CompletedPart, len(parts))
	for i, p := range parts {
		completedParts[i] = &s3.CompletedPart{
			PartNumber:     aws.Int64(p.PartNo),
			ETag:           aws.String(p.Etag),
			ChecksumSHA256: aws.String(p.Checksum),
		}
	}

	completeInput := &s3.CompleteMultipartUploadInput{
		Bucket:         request.Bucket,
		Key:            request.Key,
		UploadId:       request.ID,
		ChecksumSHA256: &checksum,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}

	common.LogDebugObject("CompleteMultipartUploadInput", completeInput)

	resp, err := ac.svc.CompleteMultipartUpload(completeInput)

	common.LogDebugObject("CompleteMultipartUploadReply", resp)

	return err
}

func (ac *awsClient) AbortMultipartUpload(request *UploadRequest) error {
	abortInput := &s3.AbortMultipartUploadInput{
		Bucket:   request.Bucket,
		Key:      request.Key,
		UploadId: request.ID,
	}

	common.LogDebugObject("AbortMultipartUpload", abortInput)
	_, err := ac.svc.AbortMultipartUpload(abortInput)
	return err
}
