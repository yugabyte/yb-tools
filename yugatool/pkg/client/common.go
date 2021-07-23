package client

import (
	"github.com/yugabyte/yb-tools/yugatool/api/yb/cdc"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/consensus"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/server"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/tserver"
	config "github.com/yugabyte/yb-tools/yugatool/pkg/client/config"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/message"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/session"
)

type HostState struct {
	session *session.Session

	config config.UniverseConfig

	Status                   *server.ServerStatusPB
	GenericService           server.GenericService
	MasterService            master.MasterService
	TabletServerService      tserver.TabletServerService
	TabletServerAdminService tserver.TabletServerAdminService
	ConsensusService         consensus.ConsensusService
	CDCService               cdc.CDCService
}

func NewHostState(host string, universeConfig *config.UniverseConfig) (*HostState, error) {
	s, err := session.NewSession(host, universeConfig, ping)
	if err != nil {
		return nil, err
	}
	hostState := &HostState{
		session: s,
		config:  config.UniverseConfig{},
	}
	hostState.GenericService = &server.GenericServiceImpl{
		Log:       universeConfig.Log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	hostState.MasterService = &master.MasterServiceImpl{
		Log:       universeConfig.Log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	hostState.TabletServerService = &tserver.TabletServerServiceImpl{
		Log:       universeConfig.Log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	hostState.TabletServerAdminService = &tserver.TabletServerAdminServiceImpl{
		Log:       universeConfig.Log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	hostState.ConsensusService = &consensus.ConsensusServiceImpl{
		Log:       universeConfig.Log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	hostState.CDCService = &cdc.CDCServiceImpl{
		Log:       universeConfig.Log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	status, err := hostState.GenericService.GetStatus(&server.GetStatusRequestPB{})
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	hostState.Status = status.GetStatus()

	return hostState, nil
}

func (h *HostState) Close() error {
	return h.session.Close()
}

func ping(s *session.Session) error {
	service := server.GenericServiceImpl{
		Log:       s.Log,
		Messenger: &message.MessengerImpl{Session: s},
	}
	_, err := service.Ping(&server.PingRequestPB{})
	return err
}
