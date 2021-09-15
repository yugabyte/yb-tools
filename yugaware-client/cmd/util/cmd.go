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

package util

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client"
)

type CommandOptions interface {
	AddFlags(cmd *cobra.Command)
	Validate(ctx *CommandContext) error
}

var _ CommandOptions = &GlobalOptions{}

type GlobalOptions struct {
	Debug                bool   `mapstructure:"debug"`
	Output               string `mapstructure:"output"`
	Hostname             string `mapstructure:"hostname"`
	DialTimeout          int    `mapstructure:"dial_timeout"`
	SkipHostVerification bool   `mapstructure:"skip_host_verification"`
	CACert               string `mapstructure:"ca_cert"`
	ClientCert           string `mapstructure:"client_cert"`
	ClientKey            string `mapstructure:"client_key"`
}

func (o *GlobalOptions) AddFlags(cmd *cobra.Command) {
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
}

func (o *GlobalOptions) Validate(_ *CommandContext) error {
	return nil
}

func mergeConfigFile(ctx *CommandContext) error {
	ctx.Log.V(1).Info("using viper config", "config", viper.AllSettings())

	err := viper.Unmarshal(ctx.GlobalOptions)
	if err != nil {
		return err
	}
	ctx.Log.V(1).Info("unmarshalled global options", "config", ctx.GlobalOptions)

	if ctx.CommandOptions != nil {
		err = viper.Unmarshal(ctx.CommandOptions)
		if err != nil {
			return err
		}
		ctx.Log.V(1).Info("unmarshalled command options", "config", ctx.CommandOptions)
	}

	return nil
}

type CommandContext struct {
	context.Context
	Log            logr.Logger
	Cmd            *cobra.Command
	GlobalOptions  *GlobalOptions
	CommandOptions CommandOptions

	Client *client.YugawareClient
}

func NewCommandContext() *CommandContext {
	return &CommandContext{
		Context: context.Background(),
	}
}

func (ctx *CommandContext) WithGlobalOptions(options *GlobalOptions) *CommandContext {
	ctx.GlobalOptions = options
	return ctx
}

func (ctx *CommandContext) WithCmd(cmd *cobra.Command) *CommandContext {
	ctx.Cmd = cmd
	return ctx
}

func (ctx *CommandContext) WithOptions(options CommandOptions) *CommandContext {
	ctx.CommandOptions = options
	return ctx
}

func (ctx *CommandContext) Setup() error {
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

	ctx.Client, err = ConnectToYugaware(ctx)
	if err != nil {
		return setupError(err)
	}

	err = ValidateRequiredFlags(ctx.Cmd.Flags())
	if err != nil {
		return err
	}
	ctx.Cmd.SilenceUsage = true

	return nil
}

func (ctx *CommandContext) complete() error {
	BindFlags(ctx.Cmd.Flags())
	err := mergeConfigFile(ctx)
	if err != nil {
		return err
	}

	return nil
}

func ConnectToYugaware(ctx *CommandContext) (*client.YugawareClient, error) {
	return client.New(ctx, ctx.Log, ctx.GlobalOptions.Hostname).
		TLSOptions(&client.TLSOptions{
			SkipHostVerification: ctx.GlobalOptions.SkipHostVerification,
			CaCertPath:           ctx.GlobalOptions.CACert,
			CertPath:             ctx.GlobalOptions.ClientCert,
			KeyPath:              ctx.GlobalOptions.ClientKey,
		}).TimeoutSeconds(ctx.GlobalOptions.DialTimeout).Connect()
}

func ShowHelp(cmd *cobra.Command, _ []string) {
	_ = cmd.Help()
}

func BindFlags(flags *pflag.FlagSet) {
	replacer := strings.NewReplacer("-", "_")

	flags.VisitAll(func(flag *pflag.Flag) {
		if err := viper.BindPFlag(replacer.Replace(flag.Name), flag); err != nil {
			panic("unable to bind flag " + flag.Name + ": " + err.Error())
		}
	})
}

func ValidateRequiredFlags(flags *pflag.FlagSet) error {
	var required []*pflag.Flag
	flags.VisitAll(func(flag *pflag.Flag) {
		if len(flag.Annotations[YugawareClientMarkFlagRequired]) > 0 {
			required = append(required, flag)
		}
	})

	var missing []string
	for _, flag := range required {
		replacer := strings.NewReplacer("-", "_")
		vname := replacer.Replace(flag.Name)
		// The flag was specified in the command line
		if flag.Changed {
			continue
		}

		value := viper.Get(vname)
		if value != nil {
			if !reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
				continue
			}
		}

		missing = append(missing, flag.Name)
	}

	if len(missing) > 0 {
		return fmt.Errorf("required flag(s) %v not set", missing)
	}
	return nil
}

const (
	YugawareClientMarkFlagRequired = "yugaware_client_flag_required_annotation"
)

func MarkFlagRequired(name string, flags *pflag.FlagSet) {
	err := flags.SetAnnotation(name, YugawareClientMarkFlagRequired, []string{"true"})
	if err != nil {
		panic("could not mark flag required: " + err.Error())
	}
}

func MarkFlagsRequired(names []string, flags *pflag.FlagSet) {
	for _, name := range names {
		MarkFlagRequired(name, flags)
	}
}
