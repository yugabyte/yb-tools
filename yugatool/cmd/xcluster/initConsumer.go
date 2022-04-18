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
package xcluster

import (
	"bytes"
	fmt "fmt"

	. "github.com/icza/gox/gox"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/pkg/cmdutil"
	"github.com/yugabyte/yb-tools/yugatool/pkg/util"
)

func InitConsumerCmd(ctx *cmdutil.YugatoolContext) *cobra.Command {
	options := &InitConsumerOptions{}
	cmd := &cobra.Command{
		Use:   "init_consumer",
		Short: "Generate commands to init xCluster replication",
		Long:  `Generate commands to init xCluster replication`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}
			defer ctx.Client.Close()

			return runInitConsumer(ctx, options)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

type InitConsumerOptions struct {
	KeyspaceName   string `mapstructure:"keyspace"`
	SkipBootstraps bool   `mapstructure:"skip_bootstraps"`
	BatchSize      int    `mapstructure:"batch_size"`
}

func (o *InitConsumerOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&o.KeyspaceName, "keyspace", "", "keyspace to replicate")
	flags.BoolVar(&o.SkipBootstraps, "skip-bootstraps", false, "initialize replication without bootstrap IDs")
        // Use a sufficiently large number for default. Can't use MaxInt as we add this number
        // to other Ints below.
	flags.IntVar(&o.BatchSize, "batch-size", 100000, "number of tables per batch, defaults to no batching")
}

func (o *InitConsumerOptions) Validate() error {
	return nil
}

var _ cmdutil.CommandOptions = &InitConsumerOptions{}

func runInitConsumer(ctx *cmdutil.YugatoolContext, options *InitConsumerOptions) error {
	tables, err := getTablesToBootstrap(ctx, options.KeyspaceName)
	if err != nil {
		return err
	}

	clusterInfoCmd, err := ctx.Client.Master.MasterService.GetMasterClusterConfig(&master.GetMasterClusterConfigRequestPB{})
	if err != nil {
		return err
	}

	var initCDCCommandPrefix bytes.Buffer

	initCDCCommandPrefix.WriteString("yb-admin -master_addresses ")
	initCDCCommandPrefix.WriteString("$CONSUMER_MASTERS")

	initCDCCommandPrefix.WriteRune(' ')

	if util.HasTLS(ctx.Client.Config.GetTlsOpts()) {
		initCDCCommandPrefix.WriteString("-certs_dir_name $CERTS_DIR ")
	}

	initCDCCommandPrefix.WriteString("setup_universe_replication ")

	initCDCCommandPrefix.WriteString(clusterInfoCmd.ClusterConfig.GetClusterUuid())

	initCDCCommandPrefix.WriteString(" $PRODUCER_MASTERS ")

	var tablesArr = tables.GetTables()
	for tablesArrIdx := 0; tablesArrIdx < len(tablesArr); tablesArrIdx += options.BatchSize {
		var initCDCCommand = initCDCCommandPrefix
		batch := tablesArr[tablesArrIdx:min(tablesArrIdx+options.BatchSize, len(tablesArr))]

		for i, table := range batch {
			if table.TableType.Number() == common.TableType_YQL_TABLE_TYPE.Number() {
				initCDCCommand.Write(table.GetId())
				if i+1 < len(batch) {
					initCDCCommand.WriteRune(',')
				}
			}
		}

		if !options.SkipBootstraps {
			initCDCCommand.WriteRune(' ')

			for i, table := range batch {
				if table.TableType.Number() == common.TableType_YQL_TABLE_TYPE.Number() {
					streams, err := ctx.Client.Master.MasterService.ListCDCStreams(&master.ListCDCStreamsRequestPB{
						TableId: NewString(string(table.GetId())),
					})
					if err != nil {
						return err
					}
					if streams.Error != nil {
						return errors.Errorf("error getting stream table %s: %s", table, streams.Error)
					}

					if len(streams.GetStreams()) == 0 {
						return errors.Errorf("bootstrap IDs not found for table %s: %s", table, streams)
					}

					if len(streams.GetStreams()) > 1 {
						return errors.Errorf("found too many streams for table %s: %s", table, streams)
					}

					initCDCCommand.Write(streams.Streams[0].StreamId)
					if i+1 < len(batch) {
						initCDCCommand.WriteRune(',')
					}
				}
			}
		}

		fmt.Println(initCDCCommand.String())
	}
	return nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
