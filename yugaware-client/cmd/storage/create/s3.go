/*
Copyright © 2021-2022 Yugabyte Support

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

package create

import (
	"fmt"
	"net/url"

	. "github.com/icza/gox/gox"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/flag"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/customer_configuration"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func S3Cmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	options := &S3Options{}
	cmd := &cobra.Command{
		Use:   "s3 CONFIGURATION_NAME --bucket gs://<bucket name>",
		Short: "Create S3 backup configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}

			options.Name = args[0]

			err = options.Validate(ctx)
			if err != nil {
				return err
			}

			return makeS3(ctx, options)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

var _ cmdutil.CommandOptions = &S3Options{}

type S3Options struct {
	Name string // Positional Arg

	Bucket      string `mapstructure:"bucket,omitempty"`
	AWSHostBase string `mapstructure:"aws_host_base,omitempty"`

	UseIAMRole bool `mapstructure:"use_iam_role,omitempty"`

	AWSAccessKey string `mapstructure:"aws_access_key,omitempty"`
	AWSSecretKey string `mapstructure:"aws_secret_key,omitempty"`

	bucket *url.URL
}

func (o *S3Options) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&o.Bucket, "bucket", "", "s3 bucket for backups")
	flags.StringVar(&o.AWSHostBase, "aws-host-base", "", "host of S3 bucket (defaults to s3.amazonaws.com)")

	flags.BoolVar(&o.UseIAMRole, "use-iam-role", false, "whether to use instance's IAM role for S3 backup.")

	flags.StringVar(&o.AWSAccessKey, "aws-access-key", "", "AWS access key for S3 backup")
	flags.StringVar(&o.AWSSecretKey, "aws-secret-key", "", "AWS secret key for S3 backup")

	flag.MarkFlagsRequired([]string{"bucket"}, flags)
}

func (o *S3Options) Validate(_ *cmdutil.YWClientContext) error {
	err := o.validateIAMRole()
	if err != nil {
		return err
	}

	return o.validateBucket()
}

func (o *S3Options) validateIAMRole() error {
	if o.UseIAMRole {
		if o.AWSAccessKey != "" || o.AWSSecretKey != "" {
			return errors.New(`unable to validate IAM role: flags "aws-access-key" and "aws-secret-key" must not be specified in "use-iam-role" mode`)
		}
		return nil
	}

	if o.AWSAccessKey == "" {
		return errors.New(`unable to validate AWS keys: "aws-access-key" was not specified`)
	}
	if o.AWSSecretKey == "" {
		return errors.New(`unable to validate AWS keys: "aws-secret-key" was not specified`)
	}

	return nil
}
func (o *S3Options) validateBucket() error {
	u, err := url.Parse(o.Bucket)
	if err != nil {
		return err
	}

	if u.Scheme == "" {
		u.Scheme = "s3"
	}

	if u.Scheme != "s3" {
		return fmt.Errorf(`failed to validate S3 bucket scheme "%s" expected "s3"`, u.Scheme)
	}

	o.bucket = u

	return nil
}

type S3StorageConfig struct {
	BackupLocation     string  `json:"BACKUP_LOCATION"`
	AWSHostBase        *string `json:"AWS_HOST_BASE,omitempty"`
	IAMInstanceProfile *string `json:"IAM_INSTANCE_PROFILE,omitempty"`
	AWSAccessKeyID     *string `json:"AWS_ACCESS_KEY_ID,omitempty"`
	AWSSecretAccessKey *string `json:"AWS_SECRET_ACCESS_KEY,omitempty"`
}

func (o *S3Options) getS3StorageConfigParams(ctx *cmdutil.YWClientContext) *customer_configuration.CreateCustomerConfigParams {
	s3Config := &S3StorageConfig{
		BackupLocation: o.bucket.String(),
	}

	if o.AWSHostBase != "" {
		s3Config.AWSHostBase = NewString(o.AWSHostBase)
	}

	if o.UseIAMRole {
		s3Config.IAMInstanceProfile = NewString("true")
	}

	if o.AWSAccessKey != "" {
		s3Config.AWSAccessKeyID = NewString(o.AWSAccessKey)
	}

	if o.AWSSecretKey != "" {
		s3Config.AWSSecretAccessKey = NewString(o.AWSSecretKey)
	}

	return customer_configuration.NewCreateCustomerConfigParams().
		WithCUUID(ctx.Client.CustomerUUID()).
		WithConfig(&models.CustomerConfig{
			Type:       NewString("STORAGE"),
			Name:       NewString("S3"),
			ConfigName: &o.Name,
			Data:       s3Config,
		})
}

func makeS3(ctx *cmdutil.YWClientContext, options *S3Options) error {
	params := options.getS3StorageConfigParams(ctx)

	config, err := ctx.Client.PlatformAPIs.CustomerConfiguration.CreateCustomerConfig(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}

	table := &format.Output{
		OutputMessage: "Created S3 Storage Provider",
		JSONObject:    config.GetPayload(),
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "UUID", JSONPath: "$.configUUID"},
			{Name: "NAME", JSONPath: "$.configName"},
			{Name: "BUCKET", JSONPath: "$.data.BACKUP_LOCATION"},
			{Name: "TYPE", JSONPath: "$.name"},
			{Name: "STATE", JSONPath: "$.state"},
		},
	}

	return table.Print()
}
