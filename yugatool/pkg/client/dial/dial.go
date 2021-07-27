package dial

import (
	"crypto/tls"
	"io"
	"net"
	"time"
)

type Dialer interface {
	Dial(network, address string) (io.ReadWriteCloser, error)
}

type NetDialer struct {
	TimeoutSeconds int64
}

func (d *NetDialer) Dial(network, address string) (io.ReadWriteCloser, error) {
	netDialer := &net.Dialer{Timeout: time.Duration(d.TimeoutSeconds) * time.Second}
	return netDialer.Dial(network, address)
}

type TLSDialer struct {
	TimeoutSeconds int64
	Config         *tls.Config
}

func (d *TLSDialer) Dial(network, address string) (io.ReadWriteCloser, error) {
	sslDialer := tls.Dialer{
		NetDialer: &net.Dialer{Timeout: time.Duration(d.TimeoutSeconds) * time.Second},
		Config:    d.Config,
	}
	return sslDialer.Dial(network, address)
}
