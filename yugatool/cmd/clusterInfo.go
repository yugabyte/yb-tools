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
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/consensus"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/tserver"
	"github.com/yugabyte/yb-tools/yugatool/pkg/cmdutil"
)

func ClusterInfoCmd(ctx *cmdutil.YugatoolContext) *cobra.Command {
	options := &ClusterInfoOptions{}
	cmd := &cobra.Command{
		Use:   "cluster_info",
		Short: "Export cluster information",
		Long:  `Export cluster information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}
			defer ctx.Client.Close()

			return clusterInfo(ctx, options)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

type ClusterInfoOptions struct {
	TabletReport         bool
	TableFilter          string
	NamespaceFilter      string
	LeadersOnlyFilter    bool
	ShowTombstonedFilter bool
}

func (o *ClusterInfoOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVar(&o.TabletReport, "tablet-report", false, "display tablet information")
	flags.StringVar(&o.TableFilter, "table", "", "in tablet report mode, display only tablets from this table")
	flags.StringVar(&o.NamespaceFilter, "namespace", "", "in tablet report mode, display only tablets from this namespace")
	flags.BoolVar(&o.LeadersOnlyFilter, "leaders-only", false, "in tablet report mode, display only tablet leaders")
	flags.BoolVar(&o.ShowTombstonedFilter, "show-tombstoned", false, "in tablet report mode, display tombstoned tablets")
}

func (o *ClusterInfoOptions) Validate() error {
	return nil
}

var _ cmdutil.CommandOptions = &ClusterInfoOptions{}

type TabletInfo struct {
	Tablet         *tserver.ListTabletsResponsePB_StatusAndSchemaPB `json:"tablet,omitempty"`
	ConsensusState *consensus.GetConsensusStateResponsePB           `json:"consensus_state,omitempty"`
}

func clusterInfo(ctx *cmdutil.YugatoolContext, options *ClusterInfoOptions) error {
	c := ctx.Client

	masterClusterConfig, err := c.Master.MasterService.GetMasterClusterConfig(&master.GetMasterClusterConfigRequestPB{})
	if err != nil {
		return err
	}

	if masterClusterConfig.GetError() != nil {
		return fmt.Errorf("could not get cluster config: %v", masterClusterConfig.GetError())
	}

	clusterConfigReport := format.Output{
		JSONObject: masterClusterConfig.GetClusterConfig(),
		OutputType: ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "CLUSTER_UUID", JSONPath: "$.clusterUuid"},
			{Name: "LIVE_REPLICAS", JSONPath: "$.replicationInfo.liveReplicas.numReplicas"},
			{Name: "ZONES", JSONPath: "$.replicationInfo.liveReplicas.placementBlocks[*].cloudInfo.placementZone"},
			{Name: "VERSION", JSONPath: "$.version"},
		},
	}

	err = clusterConfigReport.Println()
	if err != nil {
		return err
	}

	listMasters, err := c.Master.MasterService.ListMasters(&master.ListMastersRequestPB{})
	if err != nil {
		return err
	}

	if listMasters.Error != nil {
		return fmt.Errorf("could not list masters: %v", listMasters.GetError())
	}

	masterReport := format.Output{
		OutputMessage: "Masters",
		JSONObject:    listMasters.GetMasters(),
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "UUID", Expr: "base64_decode(@.instanceId.permanentUuid)"},
			{Name: "HOST", Expr: "@.registration.privateRpcAddresses[0].host"},
			{Name: "PORT", Expr: "@.registration.privateRpcAddresses[0].port"},
			{Name: "REGION", JSONPath: "$.registration.cloudInfo.placementRegion"},
			{Name: "ZONE", JSONPath: "$.registration.cloudInfo.placementZone"},
			{Name: "ROLE", JSONPath: "$.role"},
		},
	}
	err = masterReport.Println()
	if err != nil {
		return err
	}

	tabletServers, err := c.Master.MasterService.ListTabletServers(&master.ListTabletServersRequestPB{})
	if err != nil {
		return err
	}

	if tabletServers.Error != nil {
		return fmt.Errorf("could not list tablet servers: %v", tabletServers.GetError())
	}

	tabletServerReport := format.Output{
		OutputMessage: "Tablet Servers",
		JSONObject:    tabletServers.GetServers(),
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "UUID", Expr: "base64_decode(@.instance_id.permanentUuid)"},
			{Name: "HOST", JSONPath: "$.registration.common.privateRpcAddresses[0].host"},
			{Name: "PORT", JSONPath: "$.registration.common.privateRpcAddresses[0].port"},
			{Name: "REGION", JSONPath: "$.registration.common.cloudInfo.placementRegion"},
			{Name: "ZONE", JSONPath: "$.registration.common.cloudInfo.placementZone"},
			{Name: "ALIVE", JSONPath: "$.alive"},
			{Name: "READS", Expr: "trunc(@.metrics.readOpsPerSec)"},
			{Name: "WRITES", Expr: "trunc(@.metrics.writeOpsPerSec)"},
			{Name: "HEARTBEAT", Expr: "@.millis_since_heartbeat / 1000"},
			{Name: "UPTIME", Expr: "seconds_pretty(@.metrics.uptimeSeconds)"},
			{Name: "SST_SIZE", Expr: "size_pretty(@.metrics.totalSstFileSize)"},
			{Name: "SST_UNCOMPRESSED", Expr: "size_pretty(@.metrics.uncompressedSstFileSize)"},
			{Name: "SST_FILES", JSONPath: "$.metrics.numSstFiles"},
			{Name: "MEMORY", Expr: "size_pretty(@.metrics.totalRamUsage)"},
		},
	}
	err = tabletServerReport.Println()
	if err != nil {
		return err
	}

	if options.TabletReport {
		type Report struct {
			output *format.Output
			err    error
		}

		wg := &sync.WaitGroup{}
		ch := make(chan Report)

		for _, server := range tabletServers.GetServers() {
			wg.Add(1)
			go func(server *master.ListTabletServersResponsePB_Entry) {
				log := ctx.Log.WithValues("server", server.InstanceId.PermanentUuid)
				defer func() {
					log.V(1).Info("done with waitgroup")
					wg.Done()
				}()
				host, err := c.GetHostByUUID(server.GetInstanceId().GetPermanentUuid())
				if err != nil {
					log.Error(err, "host connection failed")
					return
				}

				tablets, err := host.TabletServerService.ListTablets(&tserver.ListTabletsRequestPB{})
				if err != nil {
					ch <- Report{nil, err}
					return
				}

				var tabletinfos []*TabletInfo
				for _, tablet := range tablets.GetStatusAndSchema() {
					pb := consensus.GetConsensusStateRequestPB{
						DestUuid: host.Status.GetNodeInstance().GetPermanentUuid(),
						TabletId: []byte(tablet.GetTabletStatus().GetTabletId()),
						Type:     common.ConsensusConfigType_CONSENSUS_CONFIG_COMMITTED.Enum(),
					}
					consensusState, err := host.ConsensusService.GetConsensusState(&pb)
					if err != nil {
						ch <- Report{nil, err}
						return
					}
					tabletinfos = append(tabletinfos, &TabletInfo{
						Tablet:         tablet,
						ConsensusState: consensusState,
					})
				}

				filter := &strings.Builder{}
				if !options.ShowTombstonedFilter {
					addFilter(filter, fmt.Sprintf(`@.tablet.tablet_status.tabletDataState != "%s"`, common.TabletDataState_TABLET_DATA_TOMBSTONED.String()))
				}

				if options.TableFilter != "" {
					addFilter(filter, fmt.Sprintf(`@.tablet.tablet_status.tableName == "%s"`, options.TableFilter))
				}

				if options.NamespaceFilter != "" {
					addFilter(filter, fmt.Sprintf(`@.tablet.tablet_status.namespaceName == "%s"`, options.NamespaceFilter))
				}

				if options.LeadersOnlyFilter {
					addFilter(filter, fmt.Sprintf(`@.consensus_state.leaderLeaseStatus == "%s"`, consensus.LeaderLeaseStatus_HAS_LEASE.String()))
				}

				tabletReport := &format.Output{
					OutputMessage: fmt.Sprintf("Tablet Report: %s (%s)", host.Status.GetBoundRpcAddresses(), string(host.Status.GetNodeInstance().GetPermanentUuid())),
					JSONObject:    tabletinfos,
					OutputType:    ctx.GlobalOptions.Output,
					TableColumns: []format.Column{
						{Name: "TABLET", JSONPath: "$.tablet.tablet_status.tabletId"},
						{Name: "TABLE", JSONPath: "$.tablet.tablet_status.tableName"},
						{Name: "NAMESPACE", JSONPath: "$.tablet.tablet_status.namespaceName"},
						{Name: "STATE", JSONPath: "$.tablet.tablet_status.state"},
						{Name: "STATUS", JSONPath: "$.tablet.tablet_status.tabletDataState"},
						{Name: "START_KEY", Expr: "base64_to_hex(@.tablet.tablet_status.partition.partitionKeyStart)"},
						{Name: "END_KEY", Expr: "base64_to_hex(@.tablet.tablet_status.partition.partitionKeyEnd)"},
						{Name: "SST_SIZE", Expr: "size_pretty(@.tablet.tablet_status.sstFilesDiskSize)"},
						{Name: "WAL_SIZE", Expr: "size_pretty(@.tablet.tablet_status.walFilesDiskSize)"},
						{Name: "CTERM", JSONPath: "$.consensus_state.cstate.currentTerm"},
						{Name: "CIDX", JSONPath: "$.consensus_state.cstate.config.opidIndex"},
						{Name: "LEADER", JSONPath: "$.consensus_state.cstate.leaderUuid"},
						{Name: "LEASE_STATUS", JSONPath: "$.consensus_state.leaderLeaseStatus"},
					},
					Filter: filter.String(),
				}

				log.V(1).Info("returning report")
				ch <- Report{tabletReport, nil}
			}(server)
		}

		go func() {
			ctx.Log.V(1).Info("waiting for tablet reports to complete")
			wg.Wait()
			ctx.Log.V(1).Info("tablet reports have finished, closing channel")
			close(ch)
		}()

		for tabletReport := range ch {
			if tabletReport.err != nil {
				ctx.Log.Error(err, "tablet report failed")
			} else if tabletReport.output != nil {
				err = tabletReport.output.Println()
				if err != nil {
					ctx.Log.Error(err, "tablet report returned error")
				}
			} else {
				ctx.Log.Error(nil, "empty tablet report")
			}
		}
	}
	return nil
}

func addFilter(buf *strings.Builder, filter string) {
	if buf.Len() > 0 {
		buf.WriteString(" && ")
	}
	buf.WriteString(filter)
}
