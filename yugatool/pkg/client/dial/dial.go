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
	Timeout time.Duration
}

func (d *NetDialer) Dial(network, address string) (io.ReadWriteCloser, error) {
	netDialer := &net.Dialer{Timeout: d.Timeout}
	return netDialer.Dial(network, address)
}

type TLSDialer struct {
	Timeout time.Duration
	Config  *tls.Config
}

func (d *TLSDialer) Dial(network, address string) (io.ReadWriteCloser, error) {
	sslDialer := tls.Dialer{
		NetDialer: &net.Dialer{Timeout: d.Timeout},
		Config:    d.Config,
	}
	return sslDialer.Dial(network, address)
}
