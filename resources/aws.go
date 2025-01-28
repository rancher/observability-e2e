package resources

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
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
	// Load the config if not provided
	if config == nil {
		// Load default config
		config = &localConfig.BackupRestoreConfig{}
		err := utils.LoadConfigIntoStruct("BackupRestoreConfigurationFileKey", config)
		if err != nil {
			return nil, fmt.Errorf("failed to load default config: %v", err)
		}
	}

	// Create a new AWS session using the region from the config
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.S3Region),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Create and return the S3 client
	client := s3.New(sess)
	return &S3Client{client: client}, nil
}

// CreateBucket creates the S3 bucket with the specified name and region
func (s *S3Client) CreateBucket(bucketName string, region string) error {
	// Create the S3 bucket
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

// DeleteBucket deletes the S3 bucket
func (s *S3Client) DeleteBucket(bucketName string) error {
	_, err := s.client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete bucket '%s': %v", bucketName, err)
	}
	return nil
}
