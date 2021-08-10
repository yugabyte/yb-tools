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

	"github.com/blang/vfs"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/consensus"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/tserver"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	"github.com/yugabyte/yb-tools/yugatool/cmd/util"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"google.golang.org/protobuf/encoding/prototext"
)

// clusterInfoCmd represents the clusterInfo command
var clusterInfoCmd = &cobra.Command{
	Use:   "cluster_info -m master-1[:port],master-2[:port]...",
	Short: "Export cluster information",
	Long:  `Export cluster information`,
	RunE:  clusterInfo,
}

func init() {
	rootCmd.AddCommand(clusterInfoCmd)
}

// TODO: right now this is just a dumping ground for running random RPC calls... Getting cluster info should be
//       a real command at some point
func clusterInfo(cmd *cobra.Command, args []string) error {
	log, err := util.GetLogger("cluster_info", debug)
	if err != nil {
		return err
	}

	hosts, err := util.ValidateHostnameList(masterAddresses, client.DefaultMasterPort)
	if err != nil {
		return err
	}

	c := &client.YBClient{
		Log: log.WithName("client"),
		Fs:  vfs.OS(),
		Config: &config.UniverseConfigPB{
			Masters:        hosts,
			TimeoutSeconds: &dialTimeout,
			TlsOpts: &config.TlsOptionsPB{
				SkipHostVerification: &skipHostVerification,
				CaCertPath:           &caCert,
				CertPath:             &clientCert,
				KeyPath:              &clientKey,
			},
		},
	}

	err = c.Connect()
	if err != nil {
		return err
	}
	defer c.Close()

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
				Type:     common.ConsensusConfigType_CONSENSUS_CONFIG_COMMITTED.Enum(),
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
