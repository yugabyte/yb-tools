package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"reflect"
	"time"

	"github.com/blang/vfs"
	"github.com/pkg/errors"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/dial"
)

type UniverseConfig struct {
	dialer dial.Dialer
	Fs     vfs.Filesystem

	Masters []string // Masters to get initial connection information from
	Timeout time.Duration

	SslOpts *SslOptions
}

func (c *UniverseConfig) OverrideDialer(dialer dial.Dialer) {
	c.dialer = dialer
}

func (c *UniverseConfig) GetDialer() (dial.Dialer, error) {
	if c.dialer != nil {
		return c.dialer, nil
	}
	if c.SslOpts == nil || reflect.DeepEqual(c.SslOpts, &SslOptions{}) {
		netDialer := &dial.NetDialer{
			Timeout: c.Timeout,
		}
		return netDialer, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.SslOpts.SkipHostVerification,
	}

	if c.SslOpts.CaCertPath != "" {
		f, err := vfs.ReadFile(c.Fs, c.SslOpts.CaCertPath)
		if err != nil {
			return nil, err
		}
		if tlsConfig.RootCAs == nil {
			tlsConfig.RootCAs = x509.NewCertPool()
		}
		if ok := tlsConfig.RootCAs.AppendCertsFromPEM(f); !ok {
			return nil, errors.Errorf("unable to add %s to the CA list", c.SslOpts.CaCertPath)
		}
	}

	if c.SslOpts.CertPath != "" || c.SslOpts.KeyPath != "" {
		if c.SslOpts.CertPath == "" || c.SslOpts.KeyPath == "" {
			return nil, errors.New("client certificate and key must both be set")
		}
		sslCert, err := vfs.ReadFile(c.Fs, c.SslOpts.CertPath)
		if err != nil {
			return c.dialer, fmt.Errorf("unable to read x509 certificate: %w", err)
		}

		sslKey, err := vfs.ReadFile(c.Fs, c.SslOpts.KeyPath)
		if err != nil {
			return c.dialer, fmt.Errorf("unable to read client key: %w", err)
		}

		tlsCertificate, err := tls.X509KeyPair(sslCert, sslKey)
		if err != nil {
			return c.dialer, fmt.Errorf("unable to read x509 key pair: %w", err)
		}

		tlsConfig.Certificates = append(tlsConfig.Certificates, tlsCertificate)
	}

	tlsDialer := &dial.TLSDialer{Timeout: c.Timeout, Config: tlsConfig}
	return tlsDialer, nil
}

type SslOptions struct {
	SkipHostVerification bool
	CaCertPath           string
	CertPath             string
	KeyPath              string
}
