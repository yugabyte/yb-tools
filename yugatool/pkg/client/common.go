package client

import (
	"github.com/go-logr/logr"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/cdc"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/consensus"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/server"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/tserver"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/dial"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/message"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/session"
	"github.com/yugabyte/yb-tools/yugatool/pkg/util"
)

type HostState struct {
	session *session.Session

	Status                   *server.ServerStatusPB
	GenericService           server.GenericService
	MasterService            master.MasterService
	TabletServerService      tserver.TabletServerService
	TabletServerAdminService tserver.TabletServerAdminService
	ConsensusService         consensus.ConsensusService
	CDCService               cdc.CDCService
}

func NewHostState(log logr.Logger, host *common.HostPortPB, dialer dial.Dialer) (*HostState, error) {
	log = log.WithValues("host", util.HostPortString(host))

	s, err := session.NewSession(log, host, dialer, ping)
	if err != nil {
		return nil, err
	}
	hostState := &HostState{
		session: s,
	}
	hostState.GenericService = &server.GenericServiceImpl{
		Log:       log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	hostState.MasterService = &master.MasterServiceImpl{
		Log:       log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	hostState.TabletServerService = &tserver.TabletServerServiceImpl{
		Log:       log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	hostState.TabletServerAdminService = &tserver.TabletServerAdminServiceImpl{
		Log:       log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	hostState.ConsensusService = &consensus.ConsensusServiceImpl{
		Log:       log,
		Messenger: &message.MessengerImpl{Session: s},
	}

	hostState.CDCService = &cdc.CDCServiceImpl{
		Log:       log,
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
