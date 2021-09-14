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

type LoginOptions struct {
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

func (o *LoginOptions) Validate(_ *cmdutil.CommandContext) error {
	var err error
	if o.Password == "" {
		o.Password, err = util.PasswordPrompt()
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *LoginOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVarP(&o.Email, "email", "u", "", "yugaware username")
	flags.StringVarP(&o.Password, "password", "p", "", "yugaware password")

	cmdutil.BindFlags(flags)
	cmdutil.MarkFlagRequired("email", flags)
}

var _ cmdutil.CommandOptions = &LoginOptions{}

func LoginCmd(ctx *cmdutil.CommandContext) *cobra.Command {
	options := &LoginOptions{}

	loginCmd := &cobra.Command{
		Use:   "login --email <email> [--password <password>]",
		Short: "Login to a Yugaware server",
		Long:  `Login to a Yugaware server`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}

			err = options.Validate(ctx)
			if err != nil {
				return err
			}

			return loginToYugaware(ctx, options)
		},
	}
	options.AddFlags(loginCmd)

	return loginCmd
}

func loginToYugaware(ctx *cmdutil.CommandContext, loginOpts *LoginOptions) error {
	log := ctx.Log

	_, err := ctx.Client.Login(&yugaware.LoginRequest{
		Email:    loginOpts.Email,
		Password: loginOpts.Password,
	})
	if err != nil {
		return err
	}

	log.Info("login successful")
	return nil
}
