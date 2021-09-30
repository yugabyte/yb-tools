package cmdutil

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	. "github.com/icza/gox/gox"
	"github.com/pkg/errors"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/pkg/util"
)

func ValidateHostnameList(masterAddresses string, defaultPort int) ([]*common.HostPortPB, error) {
	// TODO: how to structure the flag validation logic?
	var hostports []*common.HostPortPB
	validateError := func(err error) ([]*common.HostPortPB, error) {
		return []*common.HostPortPB{}, fmt.Errorf("unable to validate master address: %w", err)
	}

	hosts := strings.Split(masterAddresses, ",")
	if len(hosts) == 1 {
		if hosts[0] == "" {
			return validateError(errors.New("master host list empty"))
		}
	}

	for _, h := range hosts {
		host, port, err := SplitHostPort(h, defaultPort)
		if err != nil {
			return validateError(err)
		}

		hostports = append(hostports, &common.HostPortPB{
			Host: &host,
			Port: NewUint32(uint32(port)),
		})
	}

	return hostports, nil
}

func SplitHostPort(h string, defaultPort int) (string, uint32, error) {
	validateError := func(err error) (string, uint32, error) {
		return "", 0, err
	}

	if util.IsBasicIPv6(h) {
		return h, uint32(defaultPort), nil
	}
	var port int

	host, p, err := net.SplitHostPort(h)
	if err != nil {
		switch x := err.(type) {
		case *net.AddrError:
			if x.Err == "missing port in address" {
				return SplitHostPort(fmt.Sprintf("%s:%d", h, defaultPort), defaultPort)
			}
		default:
			return validateError(err)
		}

	}
	port, err = strconv.Atoi(p)
	if err != nil {
		return validateError(err)
	}
	return host, uint32(port), nil
}
