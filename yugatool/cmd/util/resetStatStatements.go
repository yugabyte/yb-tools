package util

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	. "github.com/icza/gox/gox"
	_ "github.com/lib/pq" // NOLINT
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/server"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/session"
	"github.com/yugabyte/yb-tools/yugatool/pkg/cmdutil"
)

func ResetStatStatementsCmd(ctx *cmdutil.YugatoolContext) *cobra.Command {
	options := &ResetStatStatementsOptions{}
	cmd := &cobra.Command{
		Use:   "reset_stat_statements",
		Short: "Reset pg_stat_statements statistics on all nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}
			defer ctx.Client.Close()

			return resetAllStatStatements(ctx.Log, ctx.Client, options)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

type ResetStatStatementsOptions struct {
	User     string `mapstructure:"user"`
	Database string `mapstructure:"database"`
	Password string `mapstructure:"password"`
}

var _ cmdutil.CommandOptions = &ResetStatStatementsOptions{}

func (o *ResetStatStatementsOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.PersistentFlags()
	flags.StringVarP(&o.User, "user", "u", "postgres", "postgres username")
	flags.StringVar(&o.Database, "database", "postgres", "postgres password")
	flags.StringVarP(&o.Password, "password", "p", "", "postgres password")

	_ = viper.BindEnv("user", "PGUSER", "YB_USER")
	_ = viper.BindEnv("database", "PGDATABASE", "YB_DATABASE")
	_ = viper.BindEnv("password", "PGPASSWORD", "YB_PASSWORD")
}

func (o *ResetStatStatementsOptions) Validate() error {
	return nil
}

func getPostgresHostPort(log logr.Logger, ybclient *client.YBClient, uuid []byte) (*common.HostPortPB, error) {
	getHostError := func(err error) (*common.HostPortPB, error) {
		return &common.HostPortPB{}, fmt.Errorf("could not get postgres bind address: %w", err)
	}

	host, err := ybclient.GetHostByUUID(uuid)
	if err != nil {
		return getHostError(err)
	}

	flag, err := host.GenericService.GetFlag(&server.GetFlagRequestPB{
		Flag: NewString("pgsql_proxy_bind_address"),
	})
	if err != nil {
		return getHostError(err)
	}
	if !flag.GetValid() {
		return getHostError(fmt.Errorf("invalid flag request: %s", flag))
	}

	hostname, port, err := cmdutil.SplitHostPort(flag.GetValue(), client.DefaultPostgresPort)
	if err != nil {
		return getHostError(err)
	}

	// TODO: what does it look like to listen on all IPv6 interfaces?
	if hostname == "0.0.0.0" {
		log.V(1).Info("psql proxy is listening on all addresses, using internal rpc interface instead", "rpc_bind_address", host.Status.BoundRpcAddresses[0])
		hostname = host.Status.BoundRpcAddresses[0].GetHost()
	}

	return &common.HostPortPB{
		Host: NewString(hostname),
		Port: NewUint32(port),
	}, nil
}

func resetAllStatStatements(log logr.Logger, ybclient *client.YBClient, options *ResetStatStatementsOptions) error {
	log.Info("getting postgres hosts...")
	hosts, errors := ybclient.AllTservers()
	if len(errors) > 0 {
		for _, err := range errors {
			if x, ok := err.(session.DialError); ok {
				log.Error(x.Err, "could not dial host", "hostport", x.Host)
			} else {
				return err
			}
		}
	}

	for _, host := range hosts {
		hostport, err := getPostgresHostPort(log, ybclient, host.Status.NodeInstance.PermanentUuid)
		if err != nil {
			log.Error(err, "could not find postgres hostport")
			continue
		}

		encryptionEnabled, err := isPostgresEncryptionEnabled(ybclient, host.Status.NodeInstance.PermanentUuid)
		if err != nil {
			// not the end of the world, try to connect anyway
			log.Error(err, "could not determine if encryption is enabled")
		}

		err = resetStatStatements(log, hostport, options, encryptionEnabled)
		if err != nil {
			log.Error(err, "could not reset pg_stat_statements")
		}
	}

	return nil
}

func resetStatStatements(log logr.Logger, host *common.HostPortPB, options *ResetStatStatementsOptions, encryptionEnabled bool) error {
	sslMode := "disable"
	if encryptionEnabled {
		sslMode = "require"
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		host.GetHost(), host.GetPort(), options.User, options.Database, sslMode)

	log = log.WithValues("host", host.GetHost(), "port", host.GetPort(), "user", options.User, "database", options.Database, "ssl", encryptionEnabled)
	log.Info("connecting to postgres host")

	if options.Password != "" {
		psqlInfo = fmt.Sprintf("%s password=%s", psqlInfo, options.Password)
	}
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}
	defer db.Close()

	log.Info("resetting pg_stat_statements")
	_, err = db.Exec(`SELECT pg_stat_statements_reset();`)

	return err
}

func isPostgresEncryptionEnabled(ybclient *client.YBClient, uuid []byte) (bool, error) {
	host, err := ybclient.GetHostByUUID(uuid)
	if err != nil {
		return false, err
	}

	flag, err := host.GenericService.GetFlag(&server.GetFlagRequestPB{
		Flag: NewString("use_client_to_server_encryption"),
	})
	if err != nil {
		return false, err
	}
	if !flag.GetValid() {
		return false, fmt.Errorf("invalid flag request: %s", flag)
	}

	if strings.ToLower(flag.GetValue()) == "true" {
		return true, nil
	}
	return false, nil
}
