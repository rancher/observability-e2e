package resources

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	localConfig "github.com/rancher/observability-e2e/tests/helper/config"
	"github.com/rancher/observability-e2e/tests/helper/utils"
	e2e "k8s.io/kubernetes/test/e2e/framework"
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
	timeout := 1 * time.Minute
	interval := 5 * time.Second
	start := time.Now()

	for {
		_, err := s.client.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileName),
		})

		if err == nil {
			return true, nil // File exists
		}

		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			//  File not found, retry until timeout
			if time.Since(start) > timeout {
				return false, fmt.Errorf("timeout: file %s not found in bucket %s after %s", fileName, bucketName, timeout)
			}
			e2e.Logf("File %s not found yet, retrying in %s...", fileName, interval)
			time.Sleep(interval)
			continue
		}

		// Some other error
		return false, fmt.Errorf("error checking if file exists: %v", err)
	}
}

// DownloadFile downloads a file from S3 to a local path
func (s *S3Client) DownloadFile(bucketName, fileName, localPath string) error {
	output, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
	})
	if err != nil {
		return fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer output.Body.Close()

	outFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, output.Body)
	if err != nil {
		return fmt.Errorf("failed to copy object to local file: %w", err)
	}

	return nil
}

// ListFilesAndTimeDifference retrieves a list of files from the specified folder in the bucket and calculates time differences
func (s *S3Client) ListFilesAndTimeDifference(bucketName, folderName string) ([]string, error) {
	var fileDetails []string
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(folderName),
	}

	// List objects in the specified folder
	resp, err := s.client.ListObjectsV2(input)
	if err != nil {
		return nil, fmt.Errorf("error listing objects: %v", err)
	}

	// Iterate through the listed objects and calculate the time difference for each file
	for _, object := range resp.Contents {
		// Calculate the time difference
		lastModified := *object.LastModified
		timeDiff := time.Since(lastModified)

		// Format the file details (file name and time difference)
		fileDetails = append(fileDetails, fmt.Sprintf("File: %s, Last Modified: %s, Time Difference: %v", *object.Key, lastModified.Format(time.RFC3339), timeDiff))
	}

	return fileDetails, nil
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
