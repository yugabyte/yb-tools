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
	"github.com/yugabyte/yb-tools/yugatool/api/yb/cdc"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/consensus"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/tserver"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	"github.com/yugabyte/yb-tools/yugatool/cmd/util"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"google.golang.org/protobuf/encoding/prototext"
)

var tablet string

// clusterInfoCmd represents the clusterInfo command
var tabletInfoCmd = &cobra.Command{
	Use:   "tablet_info -m master-1[:port],master-2[:port]... --tablet <tablet>",
	Short: "Get tablet consensus info and state",
	Long:  `Get tablet consensus info and state`,
	RunE:  tabletInfo,
}

func init() {
	rootCmd.AddCommand(tabletInfoCmd)

	flags := tabletInfoCmd.PersistentFlags()
	flags.StringVar(&tablet, "tablet", "", "the tablet to query")

	if err := cobra.MarkFlagRequired(flags, "tablet"); err != nil {
		panic(err)
	}
}

func tabletInfo(cmd *cobra.Command, args []string) error {
	log, err := util.GetLogger("tablet_info", debug)
	if err != nil {
		return err
	}

	hosts, err := util.ValidateMastersFlag(masterAddresses)
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
					Type:     common.ConsensusConfigType_CONSENSUS_CONFIG_COMMITTED.Enum(),
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
