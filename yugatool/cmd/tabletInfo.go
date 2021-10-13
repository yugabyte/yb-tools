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
	"github.com/yugabyte/yb-tools/yugatool/api/yb/cdc"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/consensus"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/tserver"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"github.com/yugabyte/yb-tools/yugatool/pkg/cmdutil"
	"google.golang.org/protobuf/encoding/prototext"
)

func TabletInfoCmd(ctx *cmdutil.YugatoolContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tablet_info TABLET_UUID",
		Short: "Get tablet consensus info and state",
		Long:  `Get tablet consensus info and state`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).Setup()
			if err != nil {
				return err
			}
			defer ctx.Client.Close()

			// Positional argument
			tablet := args[0]

			return tabletInfo(ctx.Client, tablet)
		},
	}

	return cmd
}

func tabletInfo(c *client.YBClient, tablet string) error {
	for _, host := range c.TServersHostMap {
		tablets, err := host.TabletServerService.ListTablets(&tserver.ListTabletsRequestPB{})
		if err != nil {
			return err
		}

		for _, t := range tablets.GetStatusAndSchema() {
			if t.GetTabletStatus().GetTabletId() == tablet {
				latestEntryOpID, err := host.CDCService.GetLatestEntryOpId(&cdc.GetLatestEntryOpIdRequestPB{
					TabletId: []byte(tablet),
				})
				if err != nil {
					return err
				}

				pb := consensus.GetConsensusStateRequestPB{
					DestUuid: host.Status.GetNodeInstance().GetPermanentUuid(),
					TabletId: []byte(t.GetTabletStatus().GetTabletId()),
					Type:     consensus.ConsensusConfigType_CONSENSUS_CONFIG_COMMITTED.Enum(),
				}
				consensusState, err := host.ConsensusService.GetConsensusState(&pb)
				if err != nil {
					return err
				}
				fmt.Printf("============================================================================\n"+
					" %s (UUID %s)\n"+
					"============================================================================\n"+
					"%s\n"+
					"%s\n"+
					"%s\n",
					host.Status.GetBoundRpcAddresses(), host.Status.GetNodeInstance().GetPermanentUuid(),
					prototext.Format(t.GetTabletStatus()),
					prototext.Format(consensusState),
					prototext.Format(latestEntryOpID),
				)
			}
		}
	}
	return nil
}
