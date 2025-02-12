package resources

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	localConfig "github.com/rancher/observability-e2e/tests/helper/config"
	"github.com/rancher/observability-e2e/tests/helper/utils"
)

// S3Client wraps the AWS S3 service client
type S3Client struct {
	client *s3.S3
}

// NewS3Client creates an AWS S3 client from BackupRestoreConfig
func NewS3Client(config *localConfig.BackupRestoreConfig) (*S3Client, error) {
	if config == nil {
		config = &localConfig.BackupRestoreConfig{}
		err := utils.LoadConfigIntoStruct("BackupRestoreConfigurationFileKey", config)
		if err != nil {
			return nil, fmt.Errorf("failed to load default config: %v", err)
		}
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.S3Region),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}

	client := s3.New(sess)
	return &S3Client{client: client}, nil
}

// FileExistsInBucket checks if a file exists in the given S3 bucket
func (s *S3Client) FileExistsInBucket(bucketName, fileName string) (bool, error) {
	_, err := s.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			return false, nil // File does not exist
		}
		return false, fmt.Errorf("error checking if file exists: %v", err)
	}

	return true, nil // File exists
}

// CreateBucket creates the S3 bucket
func (s *S3Client) CreateBucket(bucketName string, region string) error {
	_, err := s.client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(region),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket '%s' in region '%s': %v", bucketName, region, err)
	}
	return nil
}

// DeleteAllObjects deletes all objects in an S3 bucket
func (s *S3Client) DeleteAllObjects(bucketName string) error {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}
	for {
		result, err := s.client.ListObjectsV2(input)
		if err != nil {
			return fmt.Errorf("failed to list objects: %v", err)
		}

		if len(result.Contents) == 0 {
			break
		}

		var objectsToDelete []*s3.ObjectIdentifier
		for _, obj := range result.Contents {
			objectsToDelete = append(objectsToDelete, &s3.ObjectIdentifier{Key: obj.Key})
		}

		_, err = s.client.DeleteObjects(&s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &s3.Delete{
				Objects: objectsToDelete,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to delete objects: %v", err)
		}
	}

	return nil
}

// DeleteBucket deletes the S3 bucket
func (s *S3Client) DeleteBucket(bucketName string) error {
	if err := s.DeleteAllObjects(bucketName); err != nil {
		return err
	}
	_, err := s.client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete bucket '%s': %v", bucketName, err)
	}
	return nil
}
