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

package provider

import (
	"github.com/spf13/cobra"
	cmdutil "github.com/yugabyte/yb-tools/yugaware-client/cmd/util"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/cloud_providers"
)

func ListCmd(ctx *cmdutil.CommandContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List yugaware providers",
		Long:  `List yugaware providers`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := ctx.WithCmd(cmd).Setup()
			if err != nil {
				return err
			}

			return list(ctx)
		},
	}
}

func list(ctx *cmdutil.CommandContext) error {
	log := ctx.Log
	ywc := ctx.Client
	log.V(1).Info("fetching providers")
	params := cloud_providers.NewGetListOfProvidersParams().
		WithContext(ctx).
		WithCUUID(ywc.CustomerUUID())
	providers, err := ywc.PlatformAPIs.CloudProviders.GetListOfProviders(params, ywc.SwaggerAuth)
	if err != nil {
		return err
	}

	table := &cmdutil.OutputFormatter{
		OutputMessage: "Provider List",
		JSONObject:    providers.GetPayload(),
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []cmdutil.Column{
			{Name: "NAME", JSONPath: "$.name"},
			{Name: "CODE", JSONPath: "$.code"},
			{Name: "UUID", JSONPath: "$.uuid"},
			{Name: "CUSTOMER_UUID", JSONPath: "$.customerUUID"},
		},
	}
	return table.Print()
}
