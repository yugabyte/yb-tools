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

package cmd

import (
	"github.com/spf13/cobra"
	cmdutil "github.com/yugabyte/yb-tools/yugaware-client/cmd/util"
	"github.com/yugabyte/yb-tools/yugaware-client/entity/yugaware"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/util"
)

func RegisterCmd(ctx *cmdutil.CommandContext) *cobra.Command {
	options := &RegisterOptions{}

	registerCmd := &cobra.Command{
		Use:   "register --environment <environment> --full-name <full name> --email <email> [--password <password>]",
		Short: "Register a Yugaware server",
		Long:  `Register a Yugaware server`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}

			err = options.Validate(ctx)
			if err != nil {
				return err
			}

			return registerYugaware(ctx, options)
		},
	}
	options.AddFlags(registerCmd)

	return registerCmd
}

func registerYugaware(ctx *cmdutil.CommandContext, options *RegisterOptions) error {
	err := executeRegister(ctx, options)
	if err != nil {

		return err
	}

	ctx.Log.V(1).Info("logging in")
	_, err = ctx.Client.Login(&yugaware.LoginRequest{
		Email:    options.Email,
		Password: options.Password,
	})
	if err != nil {
		return err
	}
	ctx.Log.V(1).Info("logged in")

	return nil
}

func executeRegister(ctx *cmdutil.CommandContext, options *RegisterOptions) error {
	countResponse, err := ctx.Client.CustomerCount()
	if err != nil {
		return err
	}

	if countResponse.Count == 0 {
		ctx.Log.V(1).Info("registering yugaware")
		response, err := ctx.Client.RegisterYugaware(&yugaware.RegisterYugawareRequest{
			Code:            options.Environment,
			Name:            options.FullName,
			Email:           options.Email,
			Password:        options.Password,
			ConfirmPassword: options.Password,
			ConfirmEula:     true,
		})
		if err != nil {
			return err
		}
		ctx.Log.V(1).Info("registered yugaware", "response", response)

		table := &cmdutil.OutputFormatter{
			OutputMessage: "Registration Completed",
			JSONObject:    response,
			OutputType:    ctx.GlobalOptions.Output,
			TableColumns: []cmdutil.Column{
				{Name: "AUTH_TOKEN", JSONPath: "$.authToken"},
				{Name: "CUSTOMER_UUID", JSONPath: "$.customerUUID"},
				{Name: "USER_UUID", JSONPath: "$.userUUID"},
			},
		}

		err = table.Print()
		if err != nil {
			return err
		}

	} else {
		ctx.Log.Info("yugaware is already registered")
	}
	return nil
}

var _ cmdutil.CommandOptions = &RegisterOptions{}

type RegisterOptions struct {
	Environment string `mapstructure:"environment,omitempty"`
	FullName    string `mapstructure:"full_name,omitempty"`
	Email       string `mapstructure:"email,omitempty"`
	Password    string `mapstructure:"password,omitempty"`
}

func (o *RegisterOptions) Validate(_ *cmdutil.CommandContext) error {
	return o.validatePassword()
}

func (o *RegisterOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVar(&o.Environment, "environment", "", `The environment Yugaware is deployed`)
	flags.StringVar(&o.FullName, "full-name", "", "The full name of the admin user")
	flags.StringVar(&o.Environment, "email", "", "The email address of the admin user")
	flags.StringVar(&o.Password, "password", "", "The password to register")

	cmdutil.BindFlags(flags)
	cmdutil.MarkFlagsRequired([]string{"environment", "full-name", "email"}, flags)
}

func (o *RegisterOptions) validatePassword() error {
	var err error
	if o.Password == "" {
		o.Password, err = util.PasswordPrompt()
	}
	return err
}
