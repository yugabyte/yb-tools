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

package universe

import (
	"github.com/spf13/cobra"
	cmdutil "github.com/yugabyte/yb-tools/yugaware-client/cmd/util"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/universe_management"
)

func ListCmd(ctx *cmdutil.CommandContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Yugabyte universes",
		Long:  `List Yugabyte universes`,
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
	log.V(1).Info("fetching universes")
	params := universe_management.NewListUniversesParams().
		WithContext(ctx).
		WithCUUID(ywc.CustomerUUID())
	universes, err := ywc.PlatformAPIs.UniverseManagement.ListUniverses(params, ywc.SwaggerAuth)
	if err != nil {
		return err
	}

	table := &cmdutil.OutputFormatter{
		OutputMessage: "Universe List",
		JSONObject:    universes.GetPayload(),
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []cmdutil.Column{
			{Name: "NAME", JSONPath: "$.name"},
			{Name: "UNIVERSE_UUID", JSONPath: "$.universeUUID"},
			{Name: "CLOUD", JSONPath: "$.universeDetails.clusters[0].placementInfo.cloudList[0].code"},
			{Name: "REGIONS", JSONPath: "$.universeDetails.clusters[0].regions[*].code"},
			{Name: "INSTANCE_TYPE", JSONPath: "$.universeDetails.clusters[0].userIntent.instanceType"},
			{Name: "RF", JSONPath: "$.universeDetails.clusters[0].userIntent.replicationFactor"},
			{Name: "NODE_COUNT", JSONPath: "$.universeDetails.clusters[0].userIntent.numNodes"},
			{Name: "AVAILABILITY_ZONES", JSONPath: "$.universeDetails.clusters[0].regions[*].zones[*].code"},
			{Name: "VERSION", JSONPath: "$.universeDetails.clusters[0].userIntent.ybSoftwareVersion"},
		},
	}
	return table.Print()
}
