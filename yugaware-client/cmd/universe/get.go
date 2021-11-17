package universe

import (
	"fmt"
	"github.com/yugabyte/yb-tools/pkg/format"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func GetCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get (UNIVERSE_NAME|UNIVERSE_UUID)",
		Short: "Show universe info",
		Long:  `Show universe info`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).Setup()
			if err != nil {
				return err
			}

			// Positional argument
			universeIdentifier := args[0]

			return getCmd(ctx, universeIdentifier)
		},
	}

	return cmd
}

func getCmd(ctx *cmdutil.YWClientContext, universeIdentifier string) error {
	universe, err := ctx.Client.GetUniverseByIdentifier(universeIdentifier)
	if err != nil {
		return err
	}
	if universe == nil {
		return fmt.Errorf("universe does not exist: %s", universeIdentifier)
	}

	table := &format.Output{
		JSONObject: universe,
		OutputType: ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
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
