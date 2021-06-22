package client

import "crypto/tls"

type UniverseConfig struct {
	Masters     []string // Masters to get initial connection information from
	MasterPort  int
	TserverPort int

	SslOpts *SslOptions
}

type ServerInfo struct {
	UUID     string
	Host     string
	Port     int
	IsMaster bool
}

type SslOptions struct {
	*tls.Config

	CertPath   string
	KeyPath    string
	CaCertPath string
}
