package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/blang/vfs"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	. "github.com/icza/gox/gox"
	"github.com/pkg/errors"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/dial"
	"github.com/yugabyte/yb-tools/yugatool/pkg/util"
)

const DefaultMasterPort = 7100
const DefaultTserverPort = 9100
const DefaultPostgresPort = 5433
const DefaultCsqlPort = 9042

type YBClient struct {
	Log logr.Logger
	Fs  vfs.Filesystem

	Config *config.UniverseConfigPB

	Master *HostState

	tServersUUIDMap map[uuid.UUID]*HostState
	dialer          dial.Dialer

	tabletServers *master.ListTabletServersResponsePB
}

func (c *YBClient) Connect() error {
	c.tServersUUIDMap = make(map[uuid.UUID]*HostState)

	dialer, err := c.GetDialer()
	if err != nil {
		return err
	}

	var hostState *HostState

	for _, m := range c.Config.Masters {
		// Connect to a master address
		hostState, err = NewHostState(c.Log, m, dialer)
		if err != nil {
			if hostState != nil {
				_ = hostState.Close()
			}
			c.Log.V(1).Info("could not connect", "host", m, "error", err)
			continue
		}
		tabletServers, err := hostState.MasterService.ListTabletServers(&master.ListTabletServersRequestPB{PrimaryOnly: NewBool(false)})
		if err != nil {
			return err
		}
		if tabletServers.Error != nil {
			if tabletServers.Error.GetCode() == master.MasterErrorPB_NOT_THE_LEADER {
				_ = hostState.Close()
				continue
			}
			return errors.Errorf("ListTabletServers returned error: %s", tabletServers.Error)
		}

		c.Master = hostState
		c.tabletServers = tabletServers

	}
	if c.Master == nil {
		return errors.Errorf("could not connect to master leader")
	}
	return err
}

func (c *YBClient) AllTservers() ([]*HostState, []error) {
	var hostStates []*HostState
	var errors []error
	for _, host := range c.tabletServers.GetServers() {
		hostState, err := c.GetHostByUUID(host.GetInstanceId().GetPermanentUuid())
		if err != nil {
			errors = append(errors, err)
		} else {
			hostStates = append(hostStates, hostState)
		}
	}
	return hostStates, errors
}

func (c *YBClient) TserverCount() int {
	return len(c.tabletServers.GetServers())
}

func (c *YBClient) GetHostByUUID(permanentUUID []byte) (*HostState, error) {
	tserverUUID, err := uuid.ParseBytes(permanentUUID)
	if err != nil {
		return nil, err
	}

	hoststate, ok := c.tServersUUIDMap[tserverUUID]
	if !ok {
		return c.dialTserver(tserverUUID)
	}

	return hoststate, nil
}

func (c *YBClient) dialTserver(tserverUUID uuid.UUID) (*HostState, error) {
	dialer, err := c.GetDialer()
	if err != nil {
		return nil, err
	}

	for _, server := range c.tabletServers.GetServers() {
		tsuuid, err := uuid.ParseBytes(server.GetInstanceId().GetPermanentUuid())
		if err != nil {
			return nil, err
		}

		if tsuuid.String() == tserverUUID.String() {
			rpcAddress := server.GetRegistration().Common.GetPrivateRpcAddresses()[0]

			hostState, err := NewHostState(c.Log, rpcAddress, dialer)
			if err != nil {
				return nil, err
			}

			c.tServersUUIDMap[tserverUUID] = hostState

			return hostState, nil
		}
	}

	return nil, fmt.Errorf("host %s not found in known tserver list", tserverUUID.String())
}

// TODO: Log errors
func (c *YBClient) Close() {
	c.Master.Close()
	for _, tserver := range c.tServersUUIDMap {
		tserver.Close()
	}
}
func (c *YBClient) OverrideDialer(dialer dial.Dialer) {
	c.dialer = dialer
}

func (c *YBClient) GetDialer() (dial.Dialer, error) {
	if c.dialer != nil {
		return c.dialer, nil
	}
	if !util.HasTLS(c.Config.GetTlsOpts()) {
		netDialer := &dial.NetDialer{
			TimeoutSeconds: c.Config.GetTimeoutSeconds(),
		}
		c.dialer = netDialer
		return netDialer, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.Config.GetTlsOpts().GetSkipHostVerification(),
	}

	if c.Config.GetTlsOpts().GetCaCertPath() != "" {
		f, err := vfs.ReadFile(c.Fs, c.Config.GetTlsOpts().GetCaCertPath())
		if err != nil {
			return nil, err
		}
		if tlsConfig.RootCAs == nil {
			tlsConfig.RootCAs = x509.NewCertPool()
		}
		if ok := tlsConfig.RootCAs.AppendCertsFromPEM(f); !ok {
			return nil, errors.Errorf("unable to add %s to the CA list", c.Config.GetTlsOpts().GetCaCertPath())
		}
	}

	if c.Config.GetTlsOpts().GetCertPath() != "" || c.Config.GetTlsOpts().GetKeyPath() != "" {
		if c.Config.GetTlsOpts().GetKeyPath() == "" || c.Config.GetTlsOpts().GetCertPath() == "" {
			return nil, errors.New("client certificate and key must both be set")
		}
		tlsCert, err := vfs.ReadFile(c.Fs, c.Config.GetTlsOpts().GetCertPath())
		if err != nil {
			return c.dialer, fmt.Errorf("unable to read x509 certificate: %w", err)
		}

		tlsKey, err := vfs.ReadFile(c.Fs, c.Config.GetTlsOpts().GetKeyPath())
		if err != nil {
			return c.dialer, fmt.Errorf("unable to read client key: %w", err)
		}

		tlsCertificate, err := tls.X509KeyPair(tlsCert, tlsKey)
		if err != nil {
			return c.dialer, fmt.Errorf("unable to read x509 key pair: %w", err)
		}

		tlsConfig.Certificates = append(tlsConfig.Certificates, tlsCertificate)
	}

	tlsDialer := &dial.TLSDialer{TimeoutSeconds: c.Config.GetTimeoutSeconds(), Config: tlsConfig}

	c.dialer = tlsDialer
	return tlsDialer, nil
}
