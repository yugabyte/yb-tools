package client

import "github.com/pkg/errors"

type YBSession struct {
	config *UniverseConfig
}

func New(masters ...string) *YBSession {
	cfg := &UniverseConfig{
		Masters:     masters,
		MasterPort:  7100,
		TserverPort: 9100,
	}
	return &YBSession{config: cfg}
}

func (s *YBSession) MasterPort(port int) *YBSession {
	s.config.MasterPort = port
	return s
}

func (s *YBSession) TserverPort(port int) *YBSession {
	s.config.TserverPort = port
	return s
}

func (s *YBSession) Begin() (*YBSession, error) {
	return s, errors.New("not implemented")
}
