package util

import (
	"fmt"
	"net"

	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
)

func HostPortString(host *common.HostPortPB) string {
	if IsBasicIPv6(host.GetHost()) {
		return fmt.Sprintf("[%s]:%d", host.GetHost(), host.GetPort())
	}
	return fmt.Sprintf("%s:%d", host.GetHost(), host.GetPort())
}

// Has basic form. i.e "2001:db8::68"
func IsBasicIPv6(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	if v4 := ip.To4(); v4 != nil {
		return false
	}

	return true
}

func HasTLS(tlsOptions *config.TlsOptionsPB) bool {
	if tlsOptions == nil {
		return false
	}

	if tlsOptions.GetSkipHostVerification() ||
		tlsOptions.GetCaCertPath() != "" ||
		tlsOptions.GetCertPath() != "" ||
		tlsOptions.GetCaCertPath() != "" {
		return true
	}

	return false
}
