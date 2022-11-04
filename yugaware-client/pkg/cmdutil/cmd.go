/*
Copyright © 2021 Yugabyte Support

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmdutil

import (
	"context"
	"fmt"

	"github.com/blang/vfs"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/flag"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client"
)

type CommandOptions interface {
	AddFlags(cmd *cobra.Command)
	Validate(ctx *YWClientContext) error
}

var _ CommandOptions = &YWGlobalOptions{}

type YWGlobalOptions struct {
	Debug                bool   `mapstructure:"debug"`
	Output               string `mapstructure:"output"`
	Hostname             string `mapstructure:"hostname"`
	DialTimeout          int    `mapstructure:"dialtimeout"`
	SkipHostVerification bool   `mapstructure:"skiphostverification"`
	CACert               string `mapstructure:"cacert"`
	ClientCert           string `mapstructure:"client_cert"`
	ClientKey            string `mapstructure:"client_key"`
	APIToken             string `mapstructure:"api_token"`
}

func (o *YWGlobalOptions) AddFlags(cmd *cobra.Command) {
	// Global configuration flags
	flags := cmd.PersistentFlags()
	flags.BoolVar(&o.Debug, "debug", false, "debug mode")
	flags.StringVarP(&o.Output, "output", "o", "table", "Output options as one of: [table, json, yaml]")
	flags.StringVar(&o.Hostname, "hostname", "localhost:8080", "hostname of yugaware")
	flags.IntVar(&o.DialTimeout, "dialtimeout", 10, "number of seconds for dial timeouts")
	flags.BoolVar(&o.SkipHostVerification, "skiphostverification", false, "skip tls host verification")
	flags.StringVarP(&o.CACert, "cacert", "c", "", "the path to the CA certificate")
	flags.StringVar(&o.ClientCert, "client-cert", "", "the path to the client certificate")
	flags.StringVar(&o.ClientKey, "client-key", "", "the path to the client key file")
	flags.StringVar(&o.APIToken, "api-token", "", "api token for yugaware session")
}

func (o *YWGlobalOptions) Validate(_ *YWClientContext) error {
	return nil
}

type YWClientContext struct {
	context.Context
	Log            logr.Logger
	Cmd            *cobra.Command
	GlobalOptions  *YWGlobalOptions
	CommandOptions CommandOptions
	Fs             vfs.Filesystem

	Client *client.YugawareClient
}

func NewCommandContext() *YWClientContext {
	return &YWClientContext{
		Context: context.Background(),
	}
}

func (ctx *YWClientContext) WithGlobalOptions(options *YWGlobalOptions) *YWClientContext {
	ctx.GlobalOptions = options
	return ctx
}

func (ctx *YWClientContext) WithCmd(cmd *cobra.Command) *YWClientContext {
	ctx.Cmd = cmd
	return ctx
}

func (ctx *YWClientContext) WithOptions(options CommandOptions) *YWClientContext {
	ctx.CommandOptions = options
	return ctx
}

func (ctx *YWClientContext) WithVFS(fs vfs.Filesystem) *YWClientContext {
	ctx.Fs = fs
	return ctx
}

func (ctx *YWClientContext) SetupNoClient() error {
	return ctx.setup(false)
}

func (ctx *YWClientContext) Setup() error {
	return ctx.setup(true)
}

func (ctx *YWClientContext) setup(useClient bool) error {
	if ctx.Cmd == nil ||
		ctx.GlobalOptions == nil {
		panic("command context is not set")
	}

	setupError := func(err error) error {
		return fmt.Errorf("failed to setup %s command: %w", ctx.Cmd.Name(), err)
	}

	var err error

	format.SetOut(ctx.Cmd.OutOrStdout())

	ctx.Log, err = GetLogger(ctx.Cmd.Name(), ctx.GlobalOptions.Debug)
	if err != nil {
		return setupError(err)
	}

	err = ctx.complete()
	if err != nil {
		return setupError(err)
	}

	if useClient {
		ctx.Client, err = ConnectToYugaware(ctx)
		if err != nil {
			return setupError(err)
		}
	}

	err = flag.ValidateRequiredFlags(ctx.Cmd.Flags())
	if err != nil {
		return err
	}
	ctx.Cmd.SilenceUsage = true

	return nil
}

func (ctx *YWClientContext) complete() error {
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

func ConnectToYugaware(ctx *YWClientContext) (*client.YugawareClient, error) {
	return client.New(ctx, ctx.Log, ctx.GlobalOptions.Hostname).
		TLSOptions(&client.TLSOptions{
			SkipHostVerification: ctx.GlobalOptions.SkipHostVerification,
			CaCertPath:           ctx.GlobalOptions.CACert,
			CertPath:             ctx.GlobalOptions.ClientCert,
			KeyPath:              ctx.GlobalOptions.ClientKey,
		}).APIToken(ctx.GlobalOptions.APIToken).
		TimeoutSeconds(ctx.GlobalOptions.DialTimeout).
		Connect()
}
