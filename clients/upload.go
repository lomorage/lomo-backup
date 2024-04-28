package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lomorage/lomo-backup/common"
	"github.com/lomorage/lomo-backup/common/datasize"
	"github.com/lomorage/lomo-backup/common/types"
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
	S3    = 1
	Minio = 2
)

type UploadRequest struct {
	Time   time.Time
	ID     string
	Bucket string
	Key    string
}

type UploadClient interface {
	ListMultipartUploads(bucket string) ([]*UploadRequest, error)

	CreateMultipartUpload(bucket, remotePath, fileType string) (*UploadRequest, error)
	Upload(partNo, length int64, request *UploadRequest, reader io.ReadSeeker,
		checksum string) (string, error)
	CompleteMultipartUpload(request *UploadRequest, parts []*types.PartInfo, checksum string) error
	AbortMultipartUpload(request *UploadRequest) error
}

func NewUpload(keyID, key, region string, svc int) (UploadClient, error) {
	return newAWSClient(keyID, key, region)
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

func (ac *awsClient) ListMultipartUploads(bucket string) ([]*UploadRequest, error) {
	output, err := ac.svc.ListMultipartUploads(&s3.ListMultipartUploadsInput{
		Bucket: &bucket,
	})
	if err != nil {
		return nil, err
	}

	requests := make([]*UploadRequest, len(output.Uploads))
	for i, upload := range output.Uploads {
		requests[i] = &UploadRequest{
			Time: *upload.Initiated,
			ID:   *upload.UploadId,
			Key:  *upload.Key,
		}
	}
	return requests, nil
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
	if resp.UploadId == nil {
		return nil, errors.New("received empty upload ID")
	}
	if resp.Bucket == nil {
		return nil, errors.New("received empty bucket")
	}
	if resp.Key == nil {
		return nil, errors.New("received empty key")
	}
	return &UploadRequest{
		ID:     *resp.UploadId,
		Bucket: *resp.Bucket,
		Key:    *resp.Key,
	}, nil
}

func (ac *awsClient) Upload(partNo, length int64, request *UploadRequest, reader io.ReadSeeker,
	checksum string) (string, error) {
	partInput := &s3.UploadPartInput{
		Body:              reader,
		Bucket:            &request.Bucket,
		Key:               &request.Key,
		PartNumber:        &partNo,
		UploadId:          &request.ID,
		ContentLength:     &length,
		ChecksumAlgorithm: &checksumAlgorithm,
		ChecksumSHA256:    &checksum,
	}

	common.LogDebugObject("UploadPartInput", partInput)

	var retErr error
	for retry := 1; retry <= maxRetries; retry++ {
		logrus.Infof("#%d retry uploading part %d for %s", retry, partNo, request.Key)
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

func (ac *awsClient) CompleteMultipartUpload(request *UploadRequest, parts []*types.PartInfo, checksum string) error {
	completedParts := make([]*s3.CompletedPart, len(parts))
	for i, p := range parts {
		completedParts[i] = &s3.CompletedPart{
			PartNumber:     aws.Int64(int64(p.PartNo)),
			ETag:           aws.String(p.Etag),
			ChecksumSHA256: aws.String(p.HashBase64),
		}
	}

	completeInput := &s3.CompleteMultipartUploadInput{
		Bucket:         &request.Bucket,
		Key:            &request.Key,
		UploadId:       &request.ID,
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
		Bucket:   &request.Bucket,
		Key:      &request.Key,
		UploadId: &request.ID,
	}

	common.LogDebugObject("AbortMultipartUpload", abortInput)
	_, err := ac.svc.AbortMultipartUpload(abortInput)
	return err
}
