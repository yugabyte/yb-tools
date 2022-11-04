package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/blang/vfs"
	"github.com/spf13/pflag"
	"github.com/yugabyte/yb-tools/integration/util"
	"github.com/yugabyte/yb-tools/yugaware-client/entity/cli"
)

type KubernetesProvider struct {
	KubernetesProviderKubeconfig         string   `mapstructure:"kubernetes_provider_kubeconfig""`
	KubernetesProviderRegion             string   `mapstructure:"kubernetes_provider_region"`
	KubernetesProviderZones              []string `mapstructure:"kubernetes_provider_zones"`
	KubernetesProviderStorageClass       string   `mapstructure:"kubernetes_provider_storage_class"`
	KubernetesProviderOverridesFile      string   `mapstructure:"kubernetes_provider_overrides_file"`
	KubernetesProviderInstanceType       string   `mapstructure:"kubernetes_provider_instance_type"`
	KubernetesProviderServiceAccountName string   `mapstructure:"kubernetes_provider_service_account_name"`
	KubernetesProviderImageRegistry      string   `mapstructure:"kubernetes_provider_image_registry"`
	KubernetesProviderPullSecretPath     string   `mapstructure:"kubernetes_provider_pull_secret_path"`
}

var _ Provider = &KubernetesProvider{}

func (p *KubernetesProvider) Type() string {
	return "kubernetes"
}

func (p *KubernetesProvider) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&p.KubernetesProviderKubeconfig, "kubernetes-provider-kubeconfig", "", "kubernetes provider kubeconfig location")
	flags.StringVar(&p.KubernetesProviderRegion, "kubernetes-provider-region", "", "kubernetes provider region")
	flags.StringSliceVar(&p.KubernetesProviderZones, "kubernetes-provider-zones", []string{}, "zones provider zones")
	flags.StringVar(&p.KubernetesProviderStorageClass, "kubernetes-provider-storage-class", "", "storage class to use for Kubeconfig")
	flags.StringVar(&p.KubernetesProviderOverridesFile, "kubernetes-provider-overrides-file", "", "kubernetes overrides")
	flags.StringVar(&p.KubernetesProviderInstanceType, "kubernetes-provider-instance-type", "xsmall", "kubernetes provider instance type")
	flags.StringVar(&p.KubernetesProviderServiceAccountName, "kubernetes-provider-service-account-name", "yugabyte-platform-universe-management", "kubernetes provider service account name")
	flags.StringVar(&p.KubernetesProviderImageRegistry, "kubernetes-provider-image-registry", "quay.io", "image registry to use for Yugabyte container images")
	flags.StringVar(&p.KubernetesProviderPullSecretPath, "kubernetes-provider-pull-secret-path", "", "kubernetes provider pull secret path")
	// TODO: test namespaced kubernetes providers
}

func (p *KubernetesProvider) ValidateFlags() error {
	kubernetesProviderError := func(err error) error {
		return fmt.Errorf("unable to validate kubernetes provider: %w", err)
	}

	if p.KubernetesProviderKubeconfig == "" {
		return kubernetesProviderError(errors.New("YW_TEST_KUBERNETES_PROVIDER_KUBECONFIG is not set"))
	}

	if p.KubernetesProviderRegion == "" {
		return kubernetesProviderError(errors.New("YW_TEST_KUBERNETES_PROVIDER_REGION is not set"))
	}

	if len(p.KubernetesProviderZones) == 0 {
		return kubernetesProviderError(errors.New("YW_TEST_KUBERNETES_PROVIDER_ZONES is not set"))
	}

	if p.KubernetesProviderStorageClass == "" {
		return kubernetesProviderError(errors.New("YW_TEST_KUBERNETES_PROVIDER_STORAGE_CLASS is not set"))
	}

	if p.KubernetesProviderInstanceType == "" {
		return kubernetesProviderError(errors.New("YW_TEST_KUBERNETES_PROVIDER_INSTANCE_TYPE is not set"))
	}

	if p.KubernetesProviderServiceAccountName == "" {
		return kubernetesProviderError(errors.New("YW_TEST_KUBERNETES_PROVIDER_SERVICE_ACCOUNT_NAME is not set"))
	}

	if p.KubernetesProviderImageRegistry == "" {
		return kubernetesProviderError(errors.New("YW_TEST_KUBERNETES_PROVIDER_IMAGE_REGISTRY is not set"))
	}

	if p.KubernetesProviderPullSecretPath == "" {
		return kubernetesProviderError(errors.New("YW_TEST_KUBERNETES_PROVIDER_PULL_SECRET_PATH is not set"))
	}

	return nil
}

func (p *KubernetesProvider) Configure(ctx *util.YWTestContext, providerName string) error {
	kubeConfigPath := "/kubeconfig.yaml"
	pullSecretPath := "/pull_secret.yaml"
	providerConfigPath := "/provider_config.yaml"

	kubeConfig, err := os.ReadFile(p.KubernetesProviderKubeconfig)
	if err != nil {
		return err
	}

	err = vfs.WriteFile(ctx.Fs, kubeConfigPath, kubeConfig, 0777)
	if err != nil {
		return err
	}

	if p.KubernetesProviderPullSecretPath != "" {
		pullSecret, err := os.ReadFile(p.KubernetesProviderPullSecretPath)
		if err != nil {
			return err
		}

		err = vfs.WriteFile(ctx.Fs, pullSecretPath, pullSecret, 0777)
		if err != nil {
			return err
		}
	}

	var overrides []byte
	if p.KubernetesProviderOverridesFile != "" {
		overrides, err = os.ReadFile(p.KubernetesProviderOverridesFile)
		if err != nil {
			return err
		}
	}

	// TODO: currently these tests only support a single region
	region := cli.Regions{Code: p.KubernetesProviderRegion}
	for _, zone := range p.KubernetesProviderZones {
		region.ZoneInfo = append(region.ZoneInfo, cli.ZoneInfo{
			Name: zone,
			Config: cli.Config{
				//KubernetesNamespace: "", // TODO: test kubernetes namespaces
				StorageClass: p.KubernetesProviderStorageClass,
				Overrides:    string(overrides),
			},
		})
	}
	regions := []cli.Regions{region}

	providerConfig := cli.KubernetesProvider{
		Name:                providerName,
		KubeconfigPath:      kubeConfigPath,
		ServiceAccountName:  p.KubernetesProviderServiceAccountName,
		ImageRegistry:       p.KubernetesProviderImageRegistry,
		ImagePullSecretPath: pullSecretPath,
		Regions:             regions,
	}

	rawConfig, err := json.Marshal(providerConfig)
	if err != nil {
		return err
	}

	err = vfs.WriteFile(ctx.Fs, providerConfigPath, rawConfig, 0777)
	if err != nil {
		return err
	}

	_, err = ctx.RunYugawareCommand("provider", "create", "kubernetes_provider", "--filename", providerConfigPath)
	if err != nil {
		return err
	}

	return nil
}

func (s *KubernetesProvider) ConfigureIfNotExists(ctx *util.YWTestContext, providerName string) error {
	return configureIfNotExists(s, ctx, providerName)
}
