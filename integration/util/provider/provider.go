package provider

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"github.com/yugabyte/yb-tools/integration/util"
	ywflags "github.com/yugabyte/yb-tools/pkg/flag"
)

type Provider interface {
	Type() string
	RegisterFlags(flags *pflag.FlagSet)
	ValidateFlags() error
	Configure(ctx *util.YWTestContext, providerName string) error
	ConfigureIfNotExists(ctx *util.YWTestContext, providerName string) error
}

func GetProvider(logger logr.Logger, flags *pflag.FlagSet, providerType string) (Provider, error) {
	var providers []Provider

	providers = append(providers, &PreconfiguredProvider{})
	providers = append(providers, &KubernetesProvider{})

	for _, storageProvider := range providers {
		storageProvider.RegisterFlags(flags)
	}
	ywflags.BindFlags(flags)

	for _, options := range providers {
		err := ywflags.MergeConfigFile(logger, options)
		if err != nil {
			return nil, err
		}
	}

	return validateProvider(providers, providerType)
}

func validateProvider(providers []Provider, providerType string) (Provider, error) {
	var provider Provider
	for _, p := range providers {
		if providerType == p.Type() {
			err := p.ValidateFlags()
			if err != nil {
				return nil, err
			}

			provider = p
		}
	}

	if provider == nil {
		var supported []string
		for _, s := range providers {
			supported = append(supported, s.Type())
		}
		return nil, fmt.Errorf(`YW_TEST_PROVIDER_TYPE is "%s" currently supported: %s`, providerType, supported)
	}

	return provider, nil
}

type PreconfiguredProvider struct{}

func (s *PreconfiguredProvider) ConfigureIfNotExists(ctx *util.YWTestContext, providerName string) error {
	return configureIfNotExists(s, ctx, providerName)
}

var _ Provider = &PreconfiguredProvider{}

func (s *PreconfiguredProvider) Type() string {
	return "preconfigured"
}

func (s *PreconfiguredProvider) RegisterFlags(_ *pflag.FlagSet) {}

func (s *PreconfiguredProvider) ValidateFlags() error {
	return nil
}

func (s *PreconfiguredProvider) Configure(_ *util.YWTestContext, providerName string) error {
	return fmt.Errorf(`YW_TEST_PROVIDER_NAME "%s" does not exist: when using YW_TEST_PROVIDER_TYPE "%s" the cloud provider must be pre-configured`, providerName, s.Type())
}

func configureIfNotExists(provider Provider, ctx *util.YWTestContext, providerName string) error {
	// Do not create storage if it already exists
	p, err := ctx.GetProviderByIdentifier(providerName)
	if err != nil {
		return err
	}

	if p != nil {
		return nil
	}

	return provider.Configure(ctx, providerName)
}
