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
	ywflags "github.com/yugabyte/yb-tools/pkg/flag"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type YWTestOptions struct {
	Hostname             string `mapstructure:"hostname"`
	DialTimeout          int    `mapstructure:"dialtimeout"`
	SkipHostVerification bool   `mapstructure:"skiphostverification"`
	CACert               string `mapstructure:"cacert"`
	ClientCert           string `mapstructure:"client_cert"`
	ClientKey            string `mapstructure:"client_key"`
	APIToken             string `mapstructure:"api_token"`

	Provider         string   `mapstructure:"provider,omitempty"`
	Regions          []string `mapstructure:"regions,omitempty"`
	InstanceType     string   `mapstructure:"instance_type,omitempty"`
	TestUniverseName string   `mapstructure:"test_universe_name"`

	SkipCleanup bool `mapstructure:"skip_cleanup"`
}

var (
	options YWTestOptions

	ywContext *util.YWContext

	logger logr.Logger
	logs   *observer.ObservedLogs

	flags *pflag.FlagSet

	failed = false
)

func init() {
	// TODO: make it possible to actually set these flags as flags, rather than environment variables
	flags = pflag.NewFlagSet("testflags", pflag.ExitOnError)

	flags.StringVar(&options.Hostname, "hostname", "localhost:8080", "hostname of yugaware")
	flags.IntVar(&options.DialTimeout, "dialtimeout", 60, "number of seconds for dial timeouts")
	flags.BoolVar(&options.SkipHostVerification, "skiphostverification", false, "skip tls host verification")
	flags.StringVar(&options.CACert, "cacert", "", "the path to the CA certificate")
	flags.StringVar(&options.ClientCert, "client-cert", "", "the path to the client certificate")
	flags.StringVar(&options.ClientKey, "client-key", "", "the path to the client key file")
	flags.StringVar(&options.APIToken, "api-token", "", "api token for yugaware session")

	flags.StringVar(&options.Provider, "provider", "", "provider to use for tests")
	flags.StringVar(&options.InstanceType, "instance-type", "", "instance type to use for tests")
	flags.StringArrayVar(&options.Regions, "regions", nil, "regions to use for tests")
	flags.StringVar(&options.TestUniverseName, "test-universe-name", "ybtools-itest", "name of universe to create for tests")

	flags.BoolVar(&options.SkipCleanup, "skip-cleanup", false, "skip test cleanup")

	ywflags.BindFlags(flags)
	ywflags.MarkFlagsRequired([]string{"api-token", "provider", "regions", "instance-type"}, flags)
}

var _ = BeforeSuite(func() {
	ctx := context.Background()
	var err error

	logger, logs = NewLogObserver()

	// Use the same environment variables as the yugaware-client cli utility
	viper.SetEnvPrefix("YW")
	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err = ywflags.MergeConfigFile(logger, &options)
	Expect(err).NotTo(HaveOccurred())

	err = ywflags.ValidateRequiredFlags(flags)
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("connecting to host %s", options.Hostname))
	ywContext = util.NewYugawareContext(ctx, logger, options.Hostname, options.DialTimeout, options.SkipHostVerification, options.CACert, options.ClientCert, options.ClientKey, options.APIToken)
})

var _ = AfterEach(func() {
	if ywContext != nil && CurrentGinkgoTestDescription().Failed {
		fmt.Print("\ntest failed, attempting to dump yugaware logs...\n\n")
		ywContext.DumpYugawareLogs()
	}

	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = AfterSuite(func() {
	if !options.SkipCleanup && ywContext != nil {
		ywContext.CleanupUniverse(options.TestUniverseName)
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
	return ywContext.CreateUniverseIfNotExists(options.TestUniverseName, options.Provider, options.InstanceType, false, options.Regions...)
}

func GetTestUniverse() *models.UniverseResp {
	return ywContext.GetUniverse(options.TestUniverseName)
}
