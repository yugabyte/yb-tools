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
	"sync"
	"time"

	"github.com/blang/vfs"
	"github.com/go-logr/logr"
	. "github.com/icza/gox/gox"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yugabyte/gocql"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/server"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	cmdutil "github.com/yugabyte/yb-tools/yugatool/cmd/util"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	util2 "github.com/yugabyte/yb-tools/yugatool/pkg/util"
)

// clusterInfoCmd represents the clusterInfo command
var tableCreateCmd = &cobra.Command{
	Use:   "table_create -m master-1[:port],master-2[:port]...",
	Short: "Create a bunch of tables",
	Long:  `Createa bunch of tables and optionally write to them`,
	RunE:  tableCreate,
}
var (
	numberOfTables, numberOfTabletsPerTserver, numberOfTuples int
	populateTables                                            bool
	user, password                                            string
)

func init() {
	rootCmd.AddCommand(tableCreateCmd)

	flags := tableCreateCmd.PersistentFlags()
	flags.IntVarP(&numberOfTables, "tables", "t", 100, "number of tables to create/populate")
	flags.IntVarP(&numberOfTabletsPerTserver, "tablets", "T", 1, "number of tablets of to create per tserver")
	flags.BoolVar(&populateTables, "populate", false, "populateTables")
	flags.IntVarP(&numberOfTuples, "tuples", "n", 1, "number of tuples to insert")
	flags.StringVarP(&user, "user", "u", "", "ycql username")
	flags.StringVarP(&password, "password", "p", "", "ycql password")
}

func tableCreate(cmd *cobra.Command, args []string) error {
	log, err := cmdutil.GetLogger("tableCreate", debug)
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

	log.Info("getting cql hosts...")
	cqlHosts, err := getCqlHosts(c)
	if err != nil {
		return err
	}

	ycqlClient := gocql.NewCluster(cqlHosts...)

	if password != "" {
		ycqlClient.Authenticator = gocql.PasswordAuthenticator{
			Username: user,
			Password: password,
		}
	}

	ycqlClient.Timeout = time.Duration(dialTimeout) * time.Second

	if clientCert != "" || clientKey != "" || caCert != "" || skipHostVerification {
		ycqlClient.SslOpts = &gocql.SslOptions{
			CertPath:               clientCert,
			KeyPath:                clientKey,
			CaPath:                 caCert,
			EnableHostVerification: skipHostVerification,
		}
	}

	ycqlClient.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	//ycqlClient.Keyspace = "testkeyspace"
	log.Info("connecting to ycql", "hosts", cqlHosts)
	session, err := ycqlClient.CreateSession()
	if err != nil {
		log.Error(err, "error creating cluster session, cannot continue")
		return err
	}
	defer session.Close()

	err = RunTableCreate(log, session, numberOfTables, numberOfTabletsPerTserver*len(c.TServersUUIDMap))
	if err != nil {
		return err
	}

	if populateTables {
		err = RunPopulateTables(log, session, numberOfTables, numberOfTuples)
	}
	return err
}

func getCqlHosts(ybClient *client.YBClient) ([]string, error) {
	getHostsError := func(err error) ([]string, error) {
		return []string{}, fmt.Errorf("could not get cql bind address: %w", err)
	}

	var cqlHosts []string
	for _, host := range ybClient.TServersUUIDMap {
		flag, err := host.GenericService.GetFlag(&server.GetFlagRequestPB{
			Flag: NewString("cql_proxy_bind_address"),
		})
		if err != nil {
			return getHostsError(err)
		}
		if !flag.GetValid() {
			return getHostsError(errors.Errorf("invalid flag request: %s", flag))
		}

		hostname, port, err := cmdutil.SplitHostPort(flag.GetValue(), client.DefaultCsqlPort)
		if err != nil {
			return getHostsError(err)
		}

		hostPort := &common.HostPortPB{
			Host: NewString(hostname),
			Port: NewUint32(port),
		}

		cqlHosts = append(cqlHosts, util2.HostPortString(hostPort))
	}

	return cqlHosts, nil
}

func RunTableCreate(log logr.Logger, session *gocql.Session, numberOfTables, numberOfTablets int) error {
	CreateKeyspaceStatement := `CREATE KEYSPACE IF NOT EXISTS test;`
	log.Info("creating keyspace if not exists...", "query", CreateKeyspaceStatement)

	err := session.Query(CreateKeyspaceStatement).Exec()
	if err != nil {
		log.Error(err, "create keyspace failed", "query", CreateKeyspaceStatement)
		return err
	}

	for i := 0; i < numberOfTables; i++ {
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

func RunPopulateTables(log logr.Logger, session *gocql.Session, numberOfTables, numberOfTuples int) error {

	wg := &sync.WaitGroup{}
	for i := 0; i < numberOfTables; i++ {
		tableName := fmt.Sprintf("test.table%d", i)
		wg.Add(1)
		log.Info("inserting into table", "table", tableName)
		go func(log logr.Logger, tableName string, numberOfTuples int, wg *sync.WaitGroup) {
			query := fmt.Sprintf("insert into %s(a, b) VALUES (?, ?)", tableName)
			for j := 1; j <= numberOfTuples; j++ {

				err := session.Query(query).Bind(j, j).Exec()
				if err != nil {
					log.Error(err, "insert failed", "query", query, "values", []int{j, j})
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
