package yugaware_client_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/yugabyte/yb-tools/integration/util"
	"github.com/yugabyte/yb-tools/integration/util/provider"
	"github.com/yugabyte/yb-tools/integration/util/storage"
	ywflags "github.com/yugabyte/yb-tools/pkg/flag"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var (
	Options   YWTestOptions
	ywContext *util.YWTestContext
	Provider  provider.Provider
	Storage   storage.StorageProvider

	flags  *pflag.FlagSet
	logger logr.Logger
	logs   *observer.ObservedLogs

	failed = false
)

type YWTestOptions struct {
	Hostname             string `mapstructure:"hostname"`
	DialTimeout          int    `mapstructure:"dialtimeout"`
	SkipHostVerification bool   `mapstructure:"skiphostverification"`
	CACert               string `mapstructure:"cacert"`
	ClientCert           string `mapstructure:"client_cert"`
	ClientKey            string `mapstructure:"client_key"`
	APIToken             string `mapstructure:"api_token"`

	ProviderType string `mapstructure:"provider_type,omitempty"`
	ProviderName string `mapstructure:"provider_name,omitempty"`

	StorageProviderType string `mapstructure:"storage_provider_type"`
	StorageProviderName string `mapstructure:"storage_provider_name"`

	TestUniverseName string   `mapstructure:"test_universe_name"`
	Regions          []string `mapstructure:"regions"`
	InstanceType     string   `mapstructure:"instance_type"`

	SkipCleanup bool `mapstructure:"skip_cleanup"`
	SkipLogs    bool `mapstructure:"skip_logs"`
}

func (o *YWTestOptions) Validate() error {
	if o.ProviderName == "" {
		return fmt.Errorf("YW_TEST_PROVIDER_NAME is not set")
	}

	if o.StorageProviderName == "" {
		return fmt.Errorf("YW_TEST_STORAGE_PROVIDER_NAME is not set")
	}

	// TODO: how should the tests set the regions?
	if len(o.Regions) == 0 {
		return fmt.Errorf("YW_TEST_REGIONS is not set")
	}

	if o.InstanceType == "" {
		return fmt.Errorf("YW_TEST_INSTANCE_TYPE is not set")
	}

	return nil
}

func init() {
}

func RegisterTestFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("testflags", pflag.ExitOnError)

	flags.StringVar(&Options.Hostname, "hostname", "localhost:8080", "hostname of yugaware")
	flags.IntVar(&Options.DialTimeout, "dialtimeout", 60, "number of seconds for dial timeouts")
	flags.BoolVar(&Options.SkipHostVerification, "skiphostverification", false, "skip tls host verification")
	flags.StringVar(&Options.CACert, "cacert", "", "the path to the CA certificate")
	flags.StringVar(&Options.ClientCert, "client-cert", "", "the path to the client certificate")
	flags.StringVar(&Options.ClientKey, "client-key", "", "the path to the client key file")
	flags.StringVar(&Options.APIToken, "api-token", "", "api token for yugaware session")

	// Provider Flags
	flags.StringVar(&Options.ProviderType, "provider-type", "", "provider to use for tests")
	flags.StringVar(&Options.ProviderName, "provider-name", "itest-provider", "name of provider to use for itests")

	// Storage provider flags
	flags.StringVar(&Options.StorageProviderType, "storage-provider-type", "", "provider to use for tests")
	flags.StringVar(&Options.StorageProviderName, "storage-provider-name", "itest-storage", "name of provider to use for itests")

	// Test universe flags
	flags.StringVar(&Options.TestUniverseName, "test-universe-name", "ybtools-itest", "name of universe to create for tests")
	flags.StringSliceVar(&Options.Regions, "regions", []string{}, "regions to use for the test universe")
	flags.StringVar(&Options.InstanceType, "instance-type", "", "instance type to use for test universe")

	flags.BoolVar(&Options.SkipCleanup, "skip-cleanup", false, "skip test cleanup")
	flags.BoolVar(&Options.SkipLogs, "skip-logs", false, "skip dumping yugaware logs")

	ywflags.MarkFlagsRequired([]string{"api-token"}, flags)

	return flags
}

var _ = BeforeSuite(func() {
	// TODO: make it possible to actually set these flags as flags, rather than environment variables
	ctx := context.Background()
	var err error

	logger, logs = NewLogObserver()

	// Use the same environment variables as the yugaware-client cli utility
	viper.SetEnvPrefix("YW_TEST")
	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	flags = RegisterTestFlags()

	ywflags.BindFlags(flags)

	err = ywflags.MergeConfigFile(logger, &Options)
	Expect(err).NotTo(HaveOccurred())

	err = ywflags.ValidateRequiredFlags(flags)
	Expect(err).NotTo(HaveOccurred())

	err = Options.Validate()
	Expect(err).NotTo(HaveOccurred())

	Provider, err = provider.GetProvider(logger, flags, Options.ProviderType)
	Expect(err).NotTo(HaveOccurred())

	Storage, err = storage.GetStorageProvider(logger, flags, Options.StorageProviderType)
	Expect(err).NotTo(HaveOccurred())

	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("connecting to host %s", Options.Hostname))
	ywContext = util.NewYugawareTestContext(ctx, logger, Options.Hostname, Options.DialTimeout, Options.SkipHostVerification, Options.CACert, Options.ClientCert, Options.ClientKey, Options.APIToken)
})

var _ = AfterEach(func() {
	if ywContext != nil && CurrentGinkgoTestDescription().Failed {
		if !Options.SkipLogs {
			fmt.Print("\ntest failed, attempting to dump yugaware logs...\n\n")
			ywContext.DumpYugawareLogs()
		}
	}

	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = AfterSuite(func() {
	if !Options.SkipCleanup && ywContext != nil {
		ywContext.CleanupUniverse(Options.TestUniverseName)
	}
})

func TestYugawareClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "YugawareClient Integration Suite")
}

func NewLogObserver() (logr.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)

	ocore := zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return core
	})

	zc := zap.NewProductionConfig()

	z, err := zc.Build(ocore)
	Expect(err).NotTo(HaveOccurred())

	logger := zapr.NewLogger(z).WithName("testlog")
	return logger, logs
}

func CreateTestUniverseIfNotExists() *models.UniverseResp {
	return ywContext.CreateUniverseIfNotExists(Options.TestUniverseName, Options.ProviderName, Options.InstanceType, false, Options.Regions...)
}

func GetTestUniverse() *models.UniverseResp {
	return ywContext.GetUniverse(Options.TestUniverseName)
}
