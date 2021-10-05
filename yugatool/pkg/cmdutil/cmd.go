package cmdutil

import (
	"context"
	"fmt"

	"github.com/blang/vfs"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/flag"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
)

type CommandOptions interface {
	AddFlags(cmd *cobra.Command)
	Validate() error
}

type YugatoolContext struct {
	context.Context
	Log            logr.Logger
	Cmd            *cobra.Command
	GlobalOptions  *GlobalOptions
	CommandOptions CommandOptions
	Fs             vfs.Filesystem

	Client *client.YBClient
}

func NewCommandContext() *YugatoolContext {
	return &YugatoolContext{
		Context: context.Background(),
		Fs:      vfs.OS(),
	}
}

func (ctx *YugatoolContext) WithGlobalOptions(options *GlobalOptions) *YugatoolContext {
	ctx.GlobalOptions = options
	return ctx
}

func (ctx *YugatoolContext) WithCmd(cmd *cobra.Command) *YugatoolContext {
	ctx.Cmd = cmd
	return ctx
}

func (ctx *YugatoolContext) WithOptions(options CommandOptions) *YugatoolContext {
	ctx.CommandOptions = options
	return ctx
}

func (ctx *YugatoolContext) Setup() error {
	if ctx.Cmd == nil ||
		ctx.GlobalOptions == nil {
		panic("command context is not set")
	}

	setupError := func(err error) error {
		return fmt.Errorf("failed to setup %s command: %w", ctx.Cmd.Name(), err)
	}

	var err error

	ctx.Log, err = GetLogger(ctx.Cmd.Name(), ctx.GlobalOptions.Debug)
	if err != nil {
		return setupError(err)
	}

	err = ctx.complete()
	if err != nil {
		return setupError(err)
	}

	err = flag.ValidateRequiredFlags(ctx.Cmd.Flags())
	if err != nil {
		return err
	}

	err = ctx.GlobalOptions.Validate()
	if err != nil {
		return err
	}

	if ctx.CommandOptions != nil {
		err = ctx.CommandOptions.Validate()
		if err != nil {
			return err
		}
	}

	ctx.Cmd.SilenceUsage = true

	err = ctx.Connect()
	if err != nil {
		return setupError(err)
	}

	return nil
}

func (ctx *YugatoolContext) Connect() error {
	c, err := ConnectToYugabyte(ctx)
	ctx.Client = c
	return err
}

func (ctx *YugatoolContext) complete() error {
	flag.BindFlags(ctx.Cmd.Flags())

	err := flag.MergeConfigFile(ctx.Log, ctx.GlobalOptions)
	if err != nil {
		return err
	}

	if ctx.CommandOptions != nil {
		err := flag.MergeConfigFile(ctx.Log, ctx.CommandOptions)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ CommandOptions = &GlobalOptions{}

type GlobalOptions struct {
	Debug                bool   `mapstructure:"debug"`
	Output               string `mapstructure:"output"`
	DialTimeout          int64  `mapstructure:"dial_timeout"`
	MasterAddresses      string `mapstructure:"master_addresses"`
	CACert               string `mapstructure:"cacert"`
	ClientCert           string `mapstructure:"client_cert"`
	ClientKey            string `mapstructure:"client_key"`
	SkipHostVerification bool   `mapstructure:"skiphostverification"`

	hosts []*common.HostPortPB
}

func (o *GlobalOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.PersistentFlags()
	flags.BoolVar(&o.Debug, "debug", false, "debug mode")
	flags.StringVarP(&o.Output, "output", "o", "table", "Output options as one of: [table, json, yaml]")
	flags.StringVarP(&o.MasterAddresses, "master-addresses", "m", "", "comma-separated list of YB Master server addresses (minimum of one)")
	flags.Int64Var(&o.DialTimeout, "dial-timeout", 10, "number of seconds for dial timeouts")
	flags.BoolVar(&o.SkipHostVerification, "skiphostverification", false, "skip tls host verification")
	flags.StringVarP(&o.CACert, "cacert", "c", "", "the path to the CA certificate")
	flags.StringVar(&o.ClientCert, "client-cert", "", "the path to the client certificate")
	flags.StringVar(&o.ClientKey, "client-key", "", "the path to the client key file")

	flag.MarkFlagRequired("master-addresses", flags)
}

func (o *GlobalOptions) Validate() error {
	hosts, err := ValidateHostnameList(o.MasterAddresses, client.DefaultMasterPort)
	if err != nil {
		return err
	}
	o.hosts = hosts
	return nil
}

func (o *GlobalOptions) Hosts() []*common.HostPortPB {
	return o.hosts
}

func ConnectToYugabyte(ctx *YugatoolContext) (*client.YBClient, error) {
	c := &client.YBClient{
		Log: ctx.Log.WithName("client"),
		Fs:  ctx.Fs,
		Config: &config.UniverseConfigPB{
			Masters:        ctx.GlobalOptions.Hosts(),
			TimeoutSeconds: &ctx.GlobalOptions.DialTimeout,
			TlsOpts: &config.TlsOptionsPB{
				SkipHostVerification: &ctx.GlobalOptions.SkipHostVerification,
				CaCertPath:           &ctx.GlobalOptions.CACert,
				CertPath:             &ctx.GlobalOptions.ClientCert,
				KeyPath:              &ctx.GlobalOptions.ClientKey,
			},
		},
	}

	return c, c.Connect()
}
