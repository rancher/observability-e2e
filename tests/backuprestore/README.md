# Backup and Restore Test Setup Instructions

## Prerequisites
Ensure you have the following prerequisites before proceeding:
- Access to an AWS S3 bucket or an S3-compatible storage service.
- AWS credentials (Access Key and Secret Key) or a Kubernetes secret containing them.
- Go installed (`/usr/local/go/bin/go`).
- The `cattle-config.yaml` configuration file.

## 1. Copy and Update the Configuration

### Copy the Example Configuration File
Run the following command to copy the example configuration file:
```sh
cp tests/helper/yamls/inputBackupRestoreConfig.yaml.example tests/helper/yamls/inputBackupRestoreConfig.yaml
```

### Update the Configuration File
Open `tests/helper/yamls/inputBackupRestoreConfig.yaml` in a text editor and replace the placeholders with your actual values:

| Parameter | Description |
|-----------|-------------|
| `s3BucketName` | Name of your S3 bucket for backups. |
| `s3Region` | AWS region where your S3 bucket is hosted (e.g., `us-west-2`). |
| `s3Endpoint` | S3 endpoint URL (if using a custom S3-compatible service). |
| `accessKey` | Your AWS access key for authentication. |
| `secretKey` | Your AWS secret key for authentication. |
| `credentialSecretName` | Kubernetes secret name containing the credentials for accessing the S3 bucket. |

### Example Configuration
```yaml
s3BucketName: "your-s3-bucket-name"
s3Region: "us-west-2"
s3Endpoint: "s3.us-west-2.amazonaws.com"
credentialSecretName: "aws-credentials"
accessKey: "<YOUR_ACCESS_KEY>"
secretKey: "<YOUR_SECRET_KEY>"
```

## 2. Running Tests

To run the backup and restore tests with detailed logs, execute the following commands:

### Set the Configuration Path
```sh
export CATTLE_TEST_CONFIG=<path to cattle-config.yaml>
```

### Run the Tests
```sh
TEST_LABEL_FILTER=backup-restore /usr/local/go/bin/go test -timeout 60m github.com/rancher/observability-e2e/tests/backuprestore -v -count=1 --ginkgo.v
```

## Notes
- Ensure that the `cattle-config.yaml` file is correctly configured.
- Verify that your AWS credentials have sufficient permissions to access the S3 bucket.
- If using an S3-compatible service, ensure the `s3Endpoint` is correctly set.
- The `--ginkgo.v` flag enables verbose test output for better debugging.

## Troubleshooting
- **Test fails due to missing credentials**: Ensure your `inputBackupRestoreConfig.yaml` is correctly updated and that Kubernetes has the required secret.
- **Invalid S3 endpoint**: Double-check the `s3Endpoint` URL format and ensure the bucket is accessible.
- **Test timeout issues**: Increase the `-timeout` value if needed.

For further support, refer to the [Rancher Observability E2E documentation](https://github.com/rancher/observability-e2e).

