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
	"os"

	. "github.com/icza/gox/gox"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/flag"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/customer_configuration"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func GCSCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	options := &GCSOptions{}
	cmd := &cobra.Command{
		Use:   "gcs CONFIGURATION_NAME --credentials-file <file> --bucket gs://<bucket name>",
		Short: "Create GCS backup configuration",
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

			return makeGCS(ctx, options)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

var _ cmdutil.CommandOptions = &GCSOptions{}

type GCSOptions struct {
	Name string // Positional Arg

	Bucket          string `mapstructure:"bucket,omitempty"`
	CredentialsFile string `mapstructure:"credentials_file,omitempty"`

	bucket      *url.URL
	credentials []byte
}

func (o *GCSOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&o.Bucket, "bucket", "", "gcs bucket for backups")
	flags.StringVar(&o.CredentialsFile, "credentials-file", "", "gcs service account credentials")

	flag.MarkFlagsRequired([]string{"bucket", "credentials-file"}, flags)
}

func (o *GCSOptions) Validate(_ *cmdutil.YWClientContext) error {
	err := o.validateBucket()
	if err != nil {
		return err
	}
	return o.validateCredentialsFile()
}

func (o *GCSOptions) validateBucket() error {
	u, err := url.Parse(o.Bucket)
	if err != nil {
		return err
	}

	if u.Scheme == "" {
		u.Scheme = "gs"
	}

	if u.Scheme != "gs" {
		return fmt.Errorf(`failed to validate GCS bucket scheme "%s" expected "gs"`, u.Scheme)
	}

	o.bucket = u

	return nil
}

func (o *GCSOptions) validateCredentialsFile() error {
	credentialsFileError := func(err error) error {
		return fmt.Errorf(`failed to validate credentials file "%s": %w`, o.CredentialsFile, err)
	}
	_, err := os.Stat(o.CredentialsFile)
	if err != nil {
		return credentialsFileError(err)
	}

	o.credentials, err = os.ReadFile(o.CredentialsFile)
	if err != nil {
		return credentialsFileError(err)
	}

	return nil
}

type GCSStorageConfig struct {
	BackupLocation     string `json:"BACKUP_LOCATION"`
	GCSCredentialsJSON string `json:"GCS_CREDENTIALS_JSON"`
}

func (o *GCSOptions) getGCSStorageConfigParams(ctx *cmdutil.YWClientContext) *customer_configuration.CreateCustomerConfigParams {

	return customer_configuration.NewCreateCustomerConfigParams().
		WithCUUID(ctx.Client.CustomerUUID()).
		WithConfig(&models.CustomerConfig{
			Type:       NewString("STORAGE"),
			Name:       NewString("GCS"),
			ConfigName: &o.Name,
			Data: &GCSStorageConfig{
				BackupLocation:     o.bucket.String(),
				GCSCredentialsJSON: string(o.credentials),
			},
		})
}

func makeGCS(ctx *cmdutil.YWClientContext, options *GCSOptions) error {
	params := options.getGCSStorageConfigParams(ctx)

	config, err := ctx.Client.PlatformAPIs.CustomerConfiguration.CreateCustomerConfig(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}

	table := &format.Output{
		OutputMessage: "Created GCS Storage Provider",
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
