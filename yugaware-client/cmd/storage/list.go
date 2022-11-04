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

package storage

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/customer_configuration"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func ListCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List storage providers",
		Long:  `List storage providers`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := ctx.WithCmd(cmd).Setup()
			if err != nil {
				return err
			}

			return list(ctx)
		},
	}
}

func list(ctx *cmdutil.YWClientContext) error {
	configParams := customer_configuration.NewGetListOfCustomerConfigParams().
		WithCUUID(ctx.Client.CustomerUUID())

	configs, err := ctx.Client.PlatformAPIs.CustomerConfiguration.GetListOfCustomerConfig(configParams, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}

	table := &format.Output{
		OutputMessage: "Storage Providers",
		JSONObject:    configs.GetPayload(),
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "NAME", JSONPath: "$.configName"},
			{Name: "UUID", JSONPath: "$.configUUID"},
			{Name: "TYPE", JSONPath: "$.name"},
			{Name: "CUSTOMER_UUID", JSONPath: "$.customerUUID"},
			{Name: "STATE", JSONPath: "$.state"},
		},
		Filter: fmt.Sprintf(`@.type == "%s"`, models.CustomerConfigTypeSTORAGE),
	}
	return table.Print()
}
