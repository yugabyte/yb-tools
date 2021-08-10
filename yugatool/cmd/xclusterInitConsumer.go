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
	"bytes"
	fmt "fmt"

	"github.com/blang/vfs"
	"github.com/go-logr/logr"
	. "github.com/icza/gox/gox"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	cmdutil "github.com/yugabyte/yb-tools/yugatool/cmd/util"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"github.com/yugabyte/yb-tools/yugatool/pkg/util"
)

// clusterInfoCmd represents the clusterInfo command
var xclusterInitConsumerCmd = &cobra.Command{
	Use:   "xcluster_init_consumer",
	Short: "Generate commands to init xCluster replication",
	Long:  `Generate commands to init xCluster replication`,
	RunE:  xclusterInitConsumer,
}

func init() {
	rootCmd.AddCommand(xclusterInitConsumerCmd)
}

func xclusterInitConsumer(_ *cobra.Command, _ []string) error {
	log, err := cmdutil.GetLogger("xcluster_init_consumer", debug)
	if err != nil {
		return err
	}

	hosts, err := cmdutil.ValidateHostnameList(masterAddresses, client.DefaultMasterPort)
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

	return RunInitXclusterConsumer(log, c)
}

func RunInitXclusterConsumer(log logr.Logger, c *client.YBClient) error {
	tables, err := c.Master.MasterService.ListTables(&master.ListTablesRequestPB{
		ExcludeSystemTables: NewBool(true),
		RelationTypeFilter:  []master.RelationType{master.RelationType_USER_TABLE_RELATION},
	})
	if err != nil {
		return err
	}

	clusterInfoCmd, err := c.Master.MasterService.GetMasterClusterConfig(&master.GetMasterClusterConfigRequestPB{})
	if err != nil {
		return err
	}

	//masters, err := c.Master.MasterService.ListMasters(&master.ListMastersRequestPB{})
	//if err != nil {
	//	return err
	//}

	var initCDCCommand bytes.Buffer

	initCDCCommand.WriteString("yb-admin -master_addresses ")
	initCDCCommand.WriteString("$CONSUMER_MASTERS")
	//for i, master := range masters.GetMasters() {
	//	hostports := master.Registration.GetPrivateRpcAddresses()
	//	initCDCCommand.WriteString(hostports[0].GetHost())
	//	initCDCCommand.WriteRune(':')
	//	initCDCCommand.WriteString(strconv.Itoa(int(hostports[0].GetPort())))
	//	if i+1 < len(masters.GetMasters()) {
	//		initCDCCommand.WriteRune(',')
	//	}
	//}

	initCDCCommand.WriteRune(' ')

	if util.HasTLS(c.Config.GetTlsOpts()) {
		initCDCCommand.WriteString("-certs_dir_name $CERTS_DIR ")
	}

	initCDCCommand.WriteString("setup_universe_replication ")

	initCDCCommand.WriteString(clusterInfoCmd.ClusterConfig.GetClusterUuid())

	initCDCCommand.WriteString(" $PRODUCER_MASTERS ")
	for i, table := range tables.GetTables() {
		if table.TableType.Number() == common.TableType_YQL_TABLE_TYPE.Number() {
			initCDCCommand.Write(table.GetId())
			if i+1 < len(tables.GetTables()) {
				initCDCCommand.WriteRune(',')
			}
		}
	}

	initCDCCommand.WriteRune(' ')

	for i, table := range tables.GetTables() {
		if table.TableType.Number() == common.TableType_YQL_TABLE_TYPE.Number() {
			streams, err := c.Master.MasterService.ListCDCStreams(&master.ListCDCStreamsRequestPB{
				TableId: NewString(string(table.GetId())),
			})
			if err != nil {
				return err
			}
			if streams.Error != nil {
				return errors.Errorf("error getting stream table %s: %s", table, streams.Error)
			}
			if len(streams.GetStreams()) != 1 {
				return errors.Errorf("found too many streams for table %s: %s", table, streams)
			}

			initCDCCommand.Write(streams.Streams[0].StreamId)
			if i+1 < len(tables.GetTables()) {
				initCDCCommand.WriteRune(',')
			}
		}
	}

	fmt.Println(initCDCCommand.String())
	return nil
}
