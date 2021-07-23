package client

import (
	"context"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	. "github.com/icza/gox/gox"
	"github.com/pkg/errors"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/config"
)

type YBClient struct {
	Log     logr.Logger
	Context context.Context
	Config  *config.UniverseConfig

	Master *HostState

	TServersUUIDMap map[uuid.UUID]*HostState
	TServersHostMap map[string]*HostState
}

func Connect(config *config.UniverseConfig) (*YBClient, error) {
	c := &YBClient{
		Log:    config.Log,
		Config: config,
		Master: nil,

		TServersUUIDMap: make(map[uuid.UUID]*HostState),
		TServersHostMap: make(map[string]*HostState),
	}

	var hostState *HostState
	var err error

	for _, m := range c.Config.Masters {
		// Connect to a master address
		hostState, err = NewHostState(m, config)
		if err != nil {
			if hostState != nil {
				_ = hostState.Close()
			}
			c.Log.V(1).Info("could not connect", "host", m, "error", err)
			continue
		}
		tabletServers, err := hostState.MasterService.ListTabletServers(&master.ListTabletServersRequestPB{PrimaryOnly: NewBool(false)})
		if err != nil {
			return c, err
		}
		if tabletServers.Error != nil {
			if tabletServers.Error.GetCode() == master.MasterErrorPB_NOT_THE_LEADER {
				_ = hostState.Close()
				continue
			}
			return c, errors.Errorf("ListTabletServers returned error: %s", tabletServers.Error)
		}

		c.Master = hostState

		for _, server := range tabletServers.GetServers() {
			rpcAddress := server.GetRegistration().Common.GetPrivateRpcAddresses()[0]

			hostState, err := NewHostState(rpcAddress.GetHost()+":"+strconv.Itoa(int(rpcAddress.GetPort())), config)
			if err != nil {
				return c, err
			}

			tserverUUID, err := uuid.ParseBytes(server.GetInstanceId().GetPermanentUuid())
			if err != nil {
				return c, err
			}

			c.TServersUUIDMap[tserverUUID] = hostState
			c.TServersHostMap[rpcAddress.GetHost()] = hostState
		}
	}
	if c.Master == nil {
		return c, errors.Errorf("could not connect to master leader")
	}
	return c, err
}

// TODO: Log errors
func (c *YBClient) Close() {
	c.Master.Close()
	for _, tserver := range c.TServersUUIDMap {
		tserver.Close()
	}
}
