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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/flag"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/pkg/util"
	"github.com/yugabyte/yb-tools/yugaware-client/entity/yugaware"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/customer_configuration"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/session_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func RegisterCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
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

func registerYugaware(ctx *cmdutil.YWClientContext, options *RegisterOptions) error {
	countParams := session_management.NewCustomerCountParams()
	countResponse, err := ctx.Client.PlatformAPIs.SessionManagement.CustomerCount(countParams)
	if err != nil {
		return err
	}

	if *countResponse.GetPayload().Count != int32(0) {
		ctx.Log.Info("yugaware is already registered")
		return nil
	}

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

	ctx.Log.Info("registration completed", "response", response)

	ctx.Log.V(1).Info("logging in")
	_, err = ctx.Client.Login(&yugaware.LoginRequest{
		Email:    options.Email,
		Password: options.Password,
	})
	if err != nil {
		return fmt.Errorf("unable to log in: %w", err)
	}

	if options.GenerateAPIToken {
		return generateAPIKey(ctx)
	}

	return nil
}

func generateAPIKey(ctx *cmdutil.YWClientContext) error {
	ctx.Log.V(1).Info("generating api key")

	// This requires a dummy value to work around https://yugabyte.atlassian.net/browse/PLAT-2076
	params := customer_configuration.NewGenerateAPITokenParams().WithCUUID(ctx.Client.CustomerUUID()).WithDummy(&models.DummyBody{})
	apiResponse, err := ctx.Client.PlatformAPIs.CustomerConfiguration.GenerateAPIToken(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return fmt.Errorf("unable to generate API Token: %w", err)
	}

	table := &format.Output{
		JSONObject: apiResponse.GetPayload(),
		OutputType: ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "API_TOKEN", JSONPath: "$.apiToken"},
			{Name: "CUSTOMER_UUID", JSONPath: "$.customerUUID"},
			{Name: "USER_UUID", JSONPath: "$.userUUID"},
		},
	}

	return table.Println()
}

var _ cmdutil.CommandOptions = &RegisterOptions{}

type RegisterOptions struct {
	Environment      string `mapstructure:"environment,omitempty"`
	FullName         string `mapstructure:"full_name,omitempty"`
	Email            string `mapstructure:"email,omitempty"`
	Password         string `mapstructure:"password,omitempty"`
	GenerateAPIToken bool   `mapstructure:"generate_api_token,omitempty"`
}

func (o *RegisterOptions) Validate(_ *cmdutil.YWClientContext) error {
	return o.validatePassword()
}

func (o *RegisterOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVar(&o.Environment, "environment", "", `The environment Yugaware is deployed`)
	flags.StringVar(&o.FullName, "full-name", "", "The full name of the admin user")
	flags.StringVar(&o.Email, "email", "", "The email address of the admin user")
	flags.StringVar(&o.Password, "password", "", "The password to register")
	flags.BoolVar(&o.GenerateAPIToken, "generate-api-token", false, "Generate an api token")

	flag.MarkFlagsRequired([]string{"environment", "full-name", "email"}, flags)
}

func (o *RegisterOptions) validatePassword() error {
	var err error
	if o.Password == "" {
		o.Password, err = util.PasswordPrompt()
	}
	return err
}
