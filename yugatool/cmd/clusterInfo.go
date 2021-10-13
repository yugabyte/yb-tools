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
	"github.com/yugabyte/yb-tools/yugatool/api/yb/consensus"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/tserver"
	"github.com/yugabyte/yb-tools/yugatool/pkg/cmdutil"
	"google.golang.org/protobuf/encoding/prototext"
)

func ClusterInfoCmd(ctx *cmdutil.YugatoolContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster_info",
		Short: "Export cluster information",
		Long:  `Export cluster information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).Setup()
			if err != nil {
				return err
			}
			defer ctx.Client.Close()

			return clusterInfo(ctx)
		},
	}

	return cmd
}

func clusterInfo(ctx *cmdutil.YugatoolContext) error {
	c := ctx.Client

	listMasters, err := c.Master.MasterService.ListMasters(&master.ListMastersRequestPB{})
	if err != nil {
		return err
	}
	fmt.Println(prototext.Format(listMasters))

	tabletservers, err := c.Master.MasterService.ListTabletServers(&master.ListTabletServersRequestPB{})
	if err != nil {
		return err
	}
	fmt.Println(prototext.Format(tabletservers))

	masterClusterConfig, err := c.Master.MasterService.GetMasterClusterConfig(&master.GetMasterClusterConfigRequestPB{})
	if err != nil {
		return err
	}
	fmt.Println(prototext.Format(masterClusterConfig))

	fmt.Println(prototext.Format(c.Master.Status))

	for _, host := range c.TServersHostMap {
		fmt.Println(prototext.Format(host.Status))

		tablets, err := host.TabletServerService.ListTablets(&tserver.ListTabletsRequestPB{})
		if err != nil {
			return err
		}

		fmt.Println(prototext.Format(tablets))

		for _, tablet := range tablets.GetStatusAndSchema() {
			pb := consensus.GetConsensusStateRequestPB{
				DestUuid: host.Status.GetNodeInstance().GetPermanentUuid(),
				TabletId: []byte(tablet.GetTabletStatus().GetTabletId()),
				Type:     consensus.ConsensusConfigType_CONSENSUS_CONFIG_COMMITTED.Enum(),
			}
			consensusState, err := host.ConsensusService.GetConsensusState(&pb)
			if err != nil {
				return err
			}
			fmt.Printf("tablet: %s\n%s\n", tablet.GetTabletStatus().GetTabletId(), prototext.Format(consensusState))

		}
	}
	return nil
}
