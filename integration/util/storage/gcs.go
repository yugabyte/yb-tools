package storage

import (
	"errors"
	"fmt"
	"os"

	"github.com/blang/vfs"
	"github.com/spf13/pflag"
	"github.com/yugabyte/yb-tools/integration/util"
)

type GCSStorage struct {
	GCSStorageBucket             string `mapstructure:"gcs_storage_bucket"`
	GCSStorageServiceAccountFile string `mapstructure:"gcs_storage_service_account_file"`
}

var _ StorageProvider = &GCSStorage{}

func (s *GCSStorage) Type() string {
	return "GCS"
}

func (s *GCSStorage) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&s.GCSStorageBucket, "gcs-storage-bucket", "", "storage bucket to use for gcs storage")
	flags.StringVar(&s.GCSStorageServiceAccountFile, "gcs-storage-service-account-file", "", "path to service account to use for gcs storage")
}

func (s *GCSStorage) ValidateFlags() error {
	gcsStorageProviderError := func(err error) error {
		return fmt.Errorf("unable to validate GCS storage provider: %w", err)
	}

	if s.GCSStorageServiceAccountFile == "" {
		return gcsStorageProviderError(errors.New("YW_TEST_GCS_STORAGE_SERVICE_ACCOUNT_FILE is not set"))
	}

	if s.GCSStorageBucket == "" {
		return gcsStorageProviderError(errors.New("YW_TEST_GCS_STORAGE_BUCKET is not set"))
	}

	return nil
}

func (s *GCSStorage) Configure(ctx *util.YWTestContext, providerName string) error {
	saFileLocation := "/storage_sa.json"
	sa, err := os.ReadFile(s.GCSStorageServiceAccountFile)
	if err != nil {
		return err
	}

	err = vfs.WriteFile(ctx.Fs, saFileLocation, sa, 0777)
	if err != nil {
		return err
	}

	_, err = ctx.RunYugawareCommand("storage", "create", "gcs", providerName, "--bucket", s.GCSStorageBucket, "--credentials-file", saFileLocation)
	return err
}

func (s *GCSStorage) ConfigureIfNotExists(ctx *util.YWTestContext, storageName string) error {
	return configureIfNotExists(s, ctx, storageName)
}
