package storage

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/yugabyte/yb-tools/integration/util"
)

type S3Storage struct {
	S3StorageBucket      string `mapstructure:"s3_storage_bucket"`
	S3StorageAWSHostBase string `mapstructure:"s3_storage_aws_host_base,omitempty"`

	S3StorageUseIAMRole bool `mapstructure:"s3_storage_use_iam_role,omitempty"`

	S3StorageAWSAccessKey string `mapstructure:"s3_storage_aws_access_key,omitempty"`
	S3StorageAWSSecretKey string `mapstructure:"s3_storage_aws_secret_key,omitempty"`
}

var _ StorageProvider = &S3Storage{}

func (s *S3Storage) Type() string {
	return "S3"
}

func (s *S3Storage) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&s.S3StorageBucket, "s3-storage-bucket", "", "storage bucket to use for gcs storage")
	flags.StringVar(&s.S3StorageAWSHostBase, "s3-storage-aws-host-base", "", "host of S3 bucket (defaults to s3.amazonaws.com)")

	flags.BoolVar(&s.S3StorageUseIAMRole, "s3-storage-use-iam-role", false, "whether to use instance's IAM role for S3 backup.")

	flags.StringVar(&s.S3StorageAWSAccessKey, "s3-storage-aws-access-key", "", "AWS access key for S3 backup")
	flags.StringVar(&s.S3StorageAWSSecretKey, "s3-storage-aws-secret-key", "", "AWS secret key for S3 backup")
}

func (s *S3Storage) ValidateFlags() error {
	s3StorageProviderError := func(err error) error {
		return fmt.Errorf("unable to validate S3 storage provider: %w", err)
	}

	if s.S3StorageBucket == "" {
		return s3StorageProviderError(errors.New("YW_TEST_S3_STORAGE_BUCKET is not set"))
	}

	if !s.S3StorageUseIAMRole && (s.S3StorageAWSSecretKey == "" || s.S3StorageAWSAccessKey == "") {
		return s3StorageProviderError(errors.New("must specify YW_TEST_S3_STORAGE_AWS_SECRET_KEY and YW_TEST_S3_STORAGE_AWS_ACCESS_KEY or YW_TEST_S3_STORAGE_USE_IAM_ROLE"))
	}

	return nil
}

func (s *S3Storage) Configure(ctx *util.YWTestContext, providerName string) error {
	command := []string{"storage", "create", "s3", providerName, "--bucket", s.S3StorageBucket}

	if s.S3StorageAWSAccessKey != "" {
		command = append(command, "--aws-access-key", s.S3StorageAWSAccessKey)
	}

	if s.S3StorageAWSSecretKey != "" {
		command = append(command, "--aws-secret-key", s.S3StorageAWSSecretKey)
	}

	if s.S3StorageUseIAMRole {
		command = append(command, "--use-iam-role")
	}

	if s.S3StorageAWSHostBase != "" {
		command = append(command, "--aws-host-base", s.S3StorageAWSHostBase)
	}

	_, err := ctx.RunYugawareCommand(command...)
	return err
}

func (s *S3Storage) ConfigureIfNotExists(ctx *util.YWTestContext, storageName string) error {
	return configureIfNotExists(s, ctx, storageName)
}
