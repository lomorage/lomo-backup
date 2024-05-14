package clients

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lomorage/lomo-backup/common"
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

/*
type UploadClient interface {
	GetObject(bucket, remotePath string) (*types.ISOInfo, error)
	PutObject(bucket, remotePath, checksum, fileType string, reader io.ReadSeeker) error
	CreateMultipartUpload(bucket, remotePath, fileType string) (*UploadRequest, error)
	Upload(partNo, length int64, request *UploadRequest, reader io.ReadSeeker,
		checksum string) (string, error)
	CompleteMultipartUpload(request *UploadRequest, parts []*types.PartInfo, checksum string) error
	ListMultipartUploads(bucket string) ([]*UploadRequest, error)
	AbortMultipartUpload(request *UploadRequest) error
}

func NewAws(keyID, key, region string, svc int) (UploadClient, error) {
	return newAWSClient(keyID, key, region)
}
*/

type AWSClient struct {
	region string
	svc    *s3.S3
}

func NewAWSClient(keyID, key, region string) (*AWSClient, error) {
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
	return &AWSClient{region: region, svc: s3.New(sess, cfg)}, nil
}

func (ac *AWSClient) ListMultipartUploads(bucket string) ([]*UploadRequest, error) {
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

func (ac *AWSClient) HeadObject(bucket, remotePath string) (*types.ISOInfo, error) {
	object, err := ac.svc.HeadObject(&s3.HeadObjectInput{
		Bucket:       &bucket,
		Key:          &remotePath,
		ChecksumMode: aws.String(s3.ChecksumModeEnabled),
	})
	if err == nil {
		info := &types.ISOInfo{}
		if object.ContentLength != nil {
			info.Size = int(*object.ContentLength)
		}
		if object.ChecksumSHA256 != nil {
			info.HashBase64 = *object.ChecksumSHA256
		}
		common.LogDebugObject("HeadObjectReply", info)
		return info, nil
	} else if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
		return nil, nil
	} else {
		return nil, err
	}
}

func (ac *AWSClient) createBucketIfNotExist(bucket string) error {
	// create bucket if not exist
	_, err := ac.svc.HeadBucket(&s3.HeadBucketInput{Bucket: &bucket})
	if err == nil {
		logrus.Infof("Bucket %s exists in %s", bucket, ac.region)
		return nil
	}
	if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != "NotFound" {
		return err
	}
	logrus.Infof("Bucket %s does't exist in %s, creating", bucket, ac.region)

	_, err = ac.svc.CreateBucket(&s3.CreateBucketInput{Bucket: &bucket})
	if err != nil {
		return err
	}
	logrus.Infof("Bucket %s is created in %s", bucket, ac.region)
	return nil
}

func (ac *AWSClient) PutObject(bucket, remotePath, checksum, fileType string, reader io.ReadSeeker) error {
	err := ac.createBucketIfNotExist(bucket)
	if err != nil {
		return err
	}
	input := &s3.PutObjectInput{
		Body:              reader,
		Bucket:            aws.String(bucket),
		Key:               aws.String(remotePath),
		ContentType:       aws.String(fileType),
		ChecksumAlgorithm: &checksumAlgorithm,
		ChecksumSHA256:    aws.String(checksum),
	}
	_, err = ac.svc.PutObject(input)
	return err
}

func (ac *AWSClient) CreateMultipartUpload(bucket, remotePath, fileType string) (*UploadRequest, error) {
	err := ac.createBucketIfNotExist(bucket)
	if err != nil {
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

func (ac *AWSClient) Upload(partNo, length int64, request *UploadRequest, reader io.ReadSeeker,
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

func (ac *AWSClient) CompleteMultipartUpload(request *UploadRequest, parts []*types.PartInfo, checksum string) error {
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

func (ac *AWSClient) AbortMultipartUpload(request *UploadRequest) error {
	abortInput := &s3.AbortMultipartUploadInput{
		Bucket:   &request.Bucket,
		Key:      &request.Key,
		UploadId: &request.ID,
	}

	common.LogDebugObject("AbortMultipartUpload", abortInput)
	_, err := ac.svc.AbortMultipartUpload(abortInput)
	return err
}

func (ac *AWSClient) GetObject(ctx context.Context, bucket, remotePath string, writer io.Writer) (int64, error) {
	err := ac.createBucketIfNotExist(bucket)
	if err != nil {
		return 0, err
	}

	result, err := ac.svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return 0, err
	}
	defer result.Body.Close()

	return io.Copy(writer, result.Body)
}
