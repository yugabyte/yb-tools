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
	"fmt"
	"strings"

	. "github.com/icza/gox/gox"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/pkg/cmdutil"
	"github.com/yugabyte/yb-tools/yugatool/pkg/util"
)

func InitProducerCmd(ctx *cmdutil.YugatoolContext) *cobra.Command {
	options := &InitProducerOptions{}
	cmd := &cobra.Command{
		Use:   "init_producer",
		Short: "Bootstrap Replication for xCluster replication producer",
		Long:  `Bootstrap Replication for xCluster replication producer`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}
			defer ctx.Client.Close()
			return initProducer(ctx, options)
		},
	}
	options.AddFlags(cmd)
	return cmd
}

type InitProducerOptions struct {
	KeyspaceName string `mapstructure:"keyspace"`
}

func (o *InitProducerOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&o.KeyspaceName, "keyspace", "", "keyspace target for replication bootstrap")
}

func (o *InitProducerOptions) Validate() error {
	return nil
}

var _ cmdutil.CommandOptions = &InitProducerOptions{}

func getTablesToBootstrap(ctx *cmdutil.YugatoolContext, keyspaceName string) (*master.ListTablesResponsePB, error) {
	var namespacevar *master.NamespaceIdentifierPB

	if keyspaceName != "" {
		namespacevar = &master.NamespaceIdentifierPB{
			Name:         NewString(keyspaceName),
			DatabaseType: common.YQLDatabase_YQL_DATABASE_CQL.Enum(),
		}
	}

	tables, err := ctx.Client.Master.MasterService.ListTables(&master.ListTablesRequestPB{
		Namespace:           namespacevar,
		ExcludeSystemTables: NewBool(true),
		RelationTypeFilter: []master.RelationType{
			master.RelationType_USER_TABLE_RELATION,
			master.RelationType_INDEX_TABLE_RELATION,
		},
	})
	if err != nil {
		return tables, err
	}
	if tables.Error != nil {
		return tables, fmt.Errorf("List tables returned Error: %s", tables.Error)
	}

	return tables, nil
}

func initProducer(ctx *cmdutil.YugatoolContext, o *InitProducerOptions) error {
	tables, err := getTablesToBootstrap(ctx, o.KeyspaceName)
	if err != nil {
		return err
	}

	var initCDCCommand strings.Builder

	initCDCCommand.WriteString("yb-admin -master_addresses ")
	initCDCCommand.WriteString("$PRODUCER_MASTERS")

	initCDCCommand.WriteRune(' ')

	if util.HasTLS(ctx.Client.Config.GetTlsOpts()) {
		initCDCCommand.WriteString("-certs_dir_name $CERTS_DIR")
	}

	initCDCCommand.WriteRune(' ')

	initCDCCommand.WriteString("bootstrap_cdc_producer")

	initCDCCommand.WriteRune(' ')

	for i, table := range tables.GetTables() {
		if table.TableType.Number() == common.TableType_YQL_TABLE_TYPE.Number() {
			initCDCCommand.Write(table.GetId())
			if i+1 < len(tables.GetTables()) {
				initCDCCommand.WriteRune(',')
			}
		}
	}

	fmt.Println(initCDCCommand.String())

	return nil
}
