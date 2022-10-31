package storage

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"github.com/yugabyte/yb-tools/integration/util"
	ywflags "github.com/yugabyte/yb-tools/pkg/flag"
)

type StorageProvider interface {
	Type() string
	RegisterFlags(flags *pflag.FlagSet)
	ValidateFlags() error
	Configure(ctx *util.YWTestContext, storageName string) error
	ConfigureIfNotExists(ctx *util.YWTestContext, storageName string) error
}

func GetStorageProvider(logger logr.Logger, flags *pflag.FlagSet, providerType string) (StorageProvider, error) {
	var storageProviders []StorageProvider

	storageProviders = append(storageProviders, &PreconfiguredStorage{})
	storageProviders = append(storageProviders, &GCSStorage{})

	for _, storageProvider := range storageProviders {
		storageProvider.RegisterFlags(flags)
	}
	ywflags.BindFlags(flags)

	for _, options := range storageProviders {
		err := ywflags.MergeConfigFile(logger, options)
		if err != nil {
			return nil, err
		}
	}

	return validateStorageProvider(storageProviders, providerType)
}

func validateStorageProvider(storageProviders []StorageProvider, storageType string) (StorageProvider, error) {
	var provider StorageProvider
	for _, storageProvider := range storageProviders {
		if storageType == storageProvider.Type() {
			err := storageProvider.ValidateFlags()
			if err != nil {
				return nil, err
			}

			provider = storageProvider
		}
	}

	if provider == nil {
		var supported []string
		for _, s := range storageProviders {
			supported = append(supported, s.Type())
		}
		return nil, fmt.Errorf(`YW_TEST_STORAGE_PROVIDER_TYPE is "%s" currently supported: %s`, storageType, supported)
	}

	return provider, nil
}

type PreconfiguredStorage struct {
}

var _ StorageProvider = &PreconfiguredStorage{}

func (s *PreconfiguredStorage) Type() string {
	return "preconfigured"
}

func (s *PreconfiguredStorage) RegisterFlags(_ *pflag.FlagSet) {}

func (s *PreconfiguredStorage) ValidateFlags() error {
	return nil
}

func (s *PreconfiguredStorage) Configure(_ *util.YWTestContext, providerName string) error {
	return fmt.Errorf(`YW_TEST_STORAGE_PROVIDER_NAME "%s" does not exist. when using YW_TEST_STORAGE_PROVIDER_TYPE "%s" the storage provider must be pre-configured.`, providerName, s.Type())
}

func (s *PreconfiguredStorage) ConfigureIfNotExists(ctx *util.YWTestContext, storageName string) error {
	return configureIfNotExists(s, ctx, storageName)
}

func configureIfNotExists(storage StorageProvider, ctx *util.YWTestContext, storageName string) error {
	// Do not create storage if it already exists
	p, err := ctx.GetStorageConfigByIdentifier(storageName)
	if err != nil {
		return err
	}

	if p != nil {
		return nil
	}

	return storage.Configure(ctx, storageName)
}
