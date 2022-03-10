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

package util

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	. "github.com/icza/gox/gox"
	"github.com/spf13/cobra"
	"github.com/yugabyte/gocql"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/server"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/session"
	"github.com/yugabyte/yb-tools/yugatool/pkg/cmdutil"
	"github.com/yugabyte/yb-tools/yugatool/pkg/util"
)

func TableCreateCmd(ctx *cmdutil.YugatoolContext) *cobra.Command {
	options := &CreateOptions{}
	cmd := &cobra.Command{
		Use:   "table_create",
		Short: "Create a bunch of tables",
		Long:  `Create a bunch of tables, and optionally write to them`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}
			defer ctx.Client.Close()

			return tableCreate(ctx.Log, ctx.Client, options, ctx.GlobalOptions)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

type CreateOptions struct {
	TableCount     uint   `mapstructure:"tables_count"`
	TabletCount    uint   `mapstructure:"tablet_count"`
	TupleCount     uint   `mapstructure:"tuple_count"`
	User           string `mapstructure:"user"`
	Password       string `mapstructure:"password"`
	PopulateTables bool   `mapstructure:"populate"`
	CreateIndexes  bool   `mapstructure:"create_indexes"`
}

var _ cmdutil.CommandOptions = &CreateOptions{}

func (o *CreateOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.PersistentFlags()
	flags.UintVarP(&o.TableCount, "table-count", "t", 100, "number of tables to create/populate")
	flags.UintVarP(&o.TabletCount, "tablet-count", "T", 1, "number of tablets of to create per tserver")
	flags.UintVarP(&o.TupleCount, "tuple-count", "n", 1, "number of tuples to insert")
	flags.StringVarP(&o.User, "user", "u", "", "ycql username")
	flags.StringVarP(&o.Password, "password", "p", "", "ycql password")
	flags.BoolVar(&o.PopulateTables, "populate", false, "populate tables")
	flags.BoolVar(&o.CreateIndexes, "create-indexes", false, "create indexes")
}

func (o *CreateOptions) Validate() error {
	return nil
}

func tableCreate(log logr.Logger, c *client.YBClient, options *CreateOptions, globalOptions *cmdutil.GlobalOptions) error {
	log.Info("getting cql hosts...")
	cqlHosts, err := getCqlHosts(log, c)
	if err != nil {
		return err
	}

	ycqlClient := gocql.NewCluster(cqlHosts...)

	if options.Password != "" {
		ycqlClient.Authenticator = gocql.PasswordAuthenticator{
			Username: options.User,
			Password: options.Password,
		}
	}

	ycqlClient.Timeout = time.Duration(globalOptions.DialTimeout) * time.Second

	if globalOptions.ClientCert != "" || globalOptions.ClientKey != "" || globalOptions.CACert != "" || globalOptions.SkipHostVerification {
		ycqlClient.SslOpts = &gocql.SslOptions{
			CertPath:               globalOptions.ClientCert,
			KeyPath:                globalOptions.ClientKey,
			CaPath:                 globalOptions.CACert,
			EnableHostVerification: !globalOptions.SkipHostVerification,
		}
	}

	ycqlClient.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	log.Info("connecting to ycql", "hosts", cqlHosts)
	session, err := ycqlClient.CreateSession()
	if err != nil {
		log.Error(err, "error creating cluster session, cannot continue")
		return err
	}
	defer session.Close()

	err = RunTableCreate(log, session, options.TableCount, options.TabletCount*uint(c.TserverCount()))
	if err != nil {
		return err
	}

	if options.CreateIndexes {
		err = RunCreateIndexes(log, session, options.TableCount, options.TabletCount*uint(c.TserverCount()))
		if err != nil {
			return err
		}
	}

	if options.PopulateTables {
		err = RunPopulateTables(log, session, options.TableCount, options.TupleCount)
	}
	return err
}

func getCqlHosts(log logr.Logger, ybClient *client.YBClient) ([]string, error) {
	getHostsError := func(err error) ([]string, error) {
		return []string{}, fmt.Errorf("could not get cql bind address: %w", err)
	}

	var cqlHosts []string
	hosts, errors := ybClient.AllTservers()
	if len(errors) > 0 {
		for _, err := range errors {
			if x, ok := err.(session.DialError); ok {
				log.Error(x.Err, "could not dial host", "hostport", x.Host)
			} else {
				return getHostsError(err)
			}
		}
	}

	for _, host := range hosts {
		flag, err := host.GenericService.GetFlag(&server.GetFlagRequestPB{
			Flag: NewString("cql_proxy_bind_address"),
		})
		if err != nil {
			return getHostsError(err)
		}
		if !flag.GetValid() {
			return getHostsError(fmt.Errorf("invalid flag request: %s", flag))
		}

		hostname, port, err := cmdutil.SplitHostPort(flag.GetValue(), client.DefaultCsqlPort)
		if err != nil {
			return getHostsError(err)
		}

		hostPort := &common.HostPortPB{
			Host: NewString(hostname),
			Port: NewUint32(port),
		}

		cqlHosts = append(cqlHosts, util.HostPortString(hostPort))
	}

	return cqlHosts, nil
}

func RunTableCreate(log logr.Logger, session *gocql.Session, numberOfTables, numberOfTablets uint) error {
	CreateKeyspaceStatement := `CREATE KEYSPACE IF NOT EXISTS test;`
	log.Info("creating keyspace if not exists...", "query", CreateKeyspaceStatement)

	err := session.Query(CreateKeyspaceStatement).Exec()
	if err != nil {
		log.Error(err, "create keyspace failed", "query", CreateKeyspaceStatement)
		return err
	}

	for i := uint(0); i < numberOfTables; i++ {
		tableName := fmt.Sprintf("test.table%d", i)
		CreateTableStatement := `CREATE TABLE IF NOT EXISTS %s(a int primary key, b int) WITH transactions = { 'enabled' : true } AND tablets = %d;`
		query := fmt.Sprintf(CreateTableStatement, tableName, numberOfTablets)
		log.Info("creating table if not exists...", "query", query)

		err = session.Query(query).Exec()
		if err != nil {
			log.Error(err, "create table failed", "query", query)
			return err
		}
	}

	return nil
}

func RunCreateIndexes(log logr.Logger, session *gocql.Session, numberOfTables, numberOfTablets uint) error {
	for i := uint(0); i < numberOfTables; i++ {
		tableName := fmt.Sprintf("test.table%d", i)
		CreateIndexStatement := `CREATE INDEX IF NOT EXISTS ON %s (b) WITH tablets = %d;`
		query := fmt.Sprintf(CreateIndexStatement, tableName, numberOfTablets)
		log.Info("creating index if not exists...", "query", query)

		err := session.Query(query).Exec()
		if err != nil {
			log.Error(err, "create index failed", "query", query)
			return err
		}
	}

	return nil
}

func RunPopulateTables(log logr.Logger, session *gocql.Session, numberOfTables, numberOfTuples uint) error {

	wg := &sync.WaitGroup{}
	for i := uint(0); i < numberOfTables; i++ {
		tableName := fmt.Sprintf("test.table%d", i)
		wg.Add(1)
		log.Info("inserting into table", "table", tableName)
		go func(log logr.Logger, tableName string, numberOfTuples uint, wg *sync.WaitGroup) {
			query := fmt.Sprintf("insert into %s(a, b) VALUES (?, ?)", tableName)
			for j := uint(1); j <= numberOfTuples; j++ {

				err := session.Query(query).Bind(j, j).Exec()
				if err != nil {
					log.Error(err, "insert failed", "query", query, "values", []uint{j, j})
				}
				if j%1000 == 0 {
					log.Info("inserted tuples", "table", tableName, "count", j)
				}
			}
			log.Info("done inserting into table", "table", tableName, "count", numberOfTuples)
			wg.Done()
		}(log, tableName, numberOfTuples, wg)
	}
	wg.Wait()
	return nil
}
