package util

import (
	"github.com/pkg/errors"
	"strings"
)

func ValidateMastersFlag(masterAddresses string) ([]string, error) {
	// TODO: how to structure the flag validation logic?
	// TODO: Should we test each address to see if it is a valid hostname?
	hosts := strings.Split(masterAddresses, ",")
	if len(hosts) == 1 {
		if hosts[0] == "" {
			return hosts, errors.New("master host list empty")
		}
	}
	return hosts, nil
}
