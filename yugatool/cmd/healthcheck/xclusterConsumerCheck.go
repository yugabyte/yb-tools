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

package healthcheck

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"github.com/yugabyte/yb-tools/yugatool/pkg/cmdutil"
	"github.com/yugabyte/yb-tools/yugatool/pkg/healthcheck"
	"google.golang.org/protobuf/encoding/prototext"
)

func XclusterConsumerCheck(ctx *cmdutil.YugatoolContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "xcluster_consumer_check",
		Short: "Check for xCluster replication configuration issues",
		Long:  `Check for xCluster replication configuration issues between yugabyte clusters.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).Setup()
			if err != nil {
				return err
			}
			defer ctx.Client.Close()

			return runXClusterConsumerCheck(cmd, ctx.Log, ctx.Client)
		},
	}

	return cmd
}

func runXClusterConsumerCheck(cmd *cobra.Command, log logr.Logger, consumerClient *client.YBClient) error {
	clusterConfig, err := consumerClient.Master.MasterService.GetMasterClusterConfig(&master.GetMasterClusterConfigRequestPB{})
	if err != nil {
		return err
	}

	if clusterConfig.GetError() != nil {
		return errors.Errorf("Failed to get cluster config: %s", clusterConfig.GetError())
	}
	// Connect to each producer and generate a report
	for producerID, producer := range clusterConfig.GetClusterConfig().GetConsumerRegistry().GetProducerMap() {
		consumerUniverseConfig := &config.UniverseConfigPB{
			Masters:        producer.GetMasterAddrs(),
			TimeoutSeconds: consumerClient.Config.TimeoutSeconds,
			TlsOpts:        consumerClient.Config.TlsOpts,
		}

		producerReport := healthcheck.NewCDCProducerReport(log, consumerClient, consumerUniverseConfig, clusterConfig.GetClusterConfig().GetClusterUuid(), producerID, producer)

		err := producerReport.RunCheck()
		if err != nil {
			return err
		}
		if producerReport.Errors != nil {
			fmt.Fprintln(cmd.OutOrStdout(), prototext.Format(producerReport))
		}

		// TODO: Check that all masters are actually valid on producer
		//        - The master is actually in the producer cluster, and not in another cluster
		//        - The master is reachable
		// TODO: Check that every master in the producer is valid in the producer map
		// TODO: Confirm that the UUID of the universe actually matches the name of the CDC stream (it is never actually checked when creating the stream)
		// TODO: Report if streams are disabled
		// TODO: Check for multiple copies of replication of the same table
		// TODO: Check if table was deleted
		// TODO: Check that the producer isn't on the same system as this one
		// TODO: Confirm that these flags are configured correctly:
		//        - log_max_seconds_to_retain - Number of seconds to retain WAL for
		//        - log_stop_retaining_min_disk_mb - Number of megabytes of WAL to retain
		//        - enable_log_retention_by_op_idx - Retain up to checkpointed index
	}
	return nil
}
